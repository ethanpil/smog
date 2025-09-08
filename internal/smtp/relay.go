package smtp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"strings"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
	"github.com/ethanpil/smog/internal/config"
	"github.com/ethanpil/smog/internal/gmail"
	"github.com/ethanpil/smog/internal/netutil"
	"golang.org/x/oauth2"
)

// The Backend implements SMTP server methods.
type Backend struct {
	Cfg         *config.Config
	Log         *slog.Logger
	GmailClient gmail.Service
	Token       *oauth2.Token
}

func (be *Backend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	return be.newSession(c.Conn())
}

// newSession is the internal, testable implementation of NewSession.
func (be *Backend) newSession(conn net.Conn) (smtp.Session, error) {
	remoteAddr := conn.RemoteAddr()
	ipStr, _, err := net.SplitHostPort(remoteAddr.String())
	if err != nil {
		be.Log.Error("could not parse remote address", "remoteAddr", remoteAddr.String(), "network", remoteAddr.Network(), "err", err)
		return nil, fmt.Errorf("internal server error: could not parse address")
	}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		be.Log.Error("could not parse IP from remote address", "ipStr", ipStr)
		return nil, fmt.Errorf("internal server error: could not parse ip")
	}

	if !netutil.IsAllowed(be.Log, ip, be.Cfg.AllowedSubnets) {
		be.Log.Warn("rejecting connection from disallowed IP", "remoteIP", ip.String())
		return nil, &smtp.SMTPError{
			Code:    554,
			Message: "access denied",
		}
	}

	be.Log.Debug("accepted connection", "remoteIP", ip.String())

	return &Session{
		log:         be.Log,
		cfg:         be.Cfg,
		gmailClient: be.GmailClient,
		token:       be.Token,
		clientIP:    ip.String(),
	}, nil
}

// A Session is returned after EHLO.
type Session struct {
	log          *slog.Logger
	cfg          *config.Config
	gmailClient  gmail.Service
	token        *oauth2.Token
	clientIP     string
	from         string
	to           []string
	dataFilePath string // Path to the temporary file holding the message data
	dataSize     int64  // Size of the message data
}

// AuthMechanisms returns a slice of available auth mechanisms to satisfy the
// go-smtp server for AUTH support.
func (s *Session) AuthMechanisms() []string {
	return []string{sasl.Plain}
}

// Auth is called to authenticate a user.
func (s *Session) Auth(mech string) (sasl.Server, error) {
	s.log.Info("AUTH attempt", "mechanism", mech)
	if mech != sasl.Plain {
		s.log.Warn("unsupported auth mechanism", "mechanism", mech)
		return nil, errors.New("unsupported authentication mechanism")
	}

	return sasl.NewPlainServer(func(identity, username, password string) error {
		if username != s.cfg.SMTPUser || password != s.cfg.SMTPPassword {
			s.log.Warn("authentication failed: invalid credentials", "username", username)
			return errors.New("invalid username or password")
		}

		s.log.Info("AUTH successful", "username", username)
		return nil
	}), nil
}

func (s *Session) Mail(from string, opts *smtp.MailOptions) error {
	s.log.Info("MAIL FROM", "from", from)
	s.Reset()
	s.from = from
	return nil
}

func (s *Session) Rcpt(to string, opts *smtp.RcptOptions) error {
	s.log.Info("RCPT TO", "to", to)
	s.to = append(s.to, to)
	return nil
}

func (s *Session) Data(r io.Reader) error {
	s.log.Debug("DATA received")

	// Create a temporary file to store the message data.
	tmpFile, err := os.CreateTemp("", "smog-data-")
	if err != nil {
		s.log.Error("failed to create temporary file", "err", err)
		return &smtp.SMTPError{Code: 451, Message: "Temporary server error"}
	}
	s.dataFilePath = tmpFile.Name()
	// No need to defer removal here because Reset() will handle it.

	var reader io.Reader = r
	var rawLimit int64

	if s.cfg.MessageSizeLimitMB > 0 {
		limitBytes := int64(s.cfg.MessageSizeLimitMB) * 1024 * 1024
		// The message size limit is for the base64 encoded message.
		// We calculate the approximate raw size limit. The base64-encoded size is
		// roughly 4/3 of the original size.
		rawLimit = (limitBytes * 3) / 4
		// We read up to rawLimit + 1 to detect if the message is too large.
		reader = io.LimitReader(r, rawLimit+1)
	}

	// Copy the data from the reader to the temporary file.
	s.dataSize, err = io.Copy(tmpFile, reader)
	if err != nil {
		tmpFile.Close()
		s.log.Error("error reading data stream", "err", err)
		return &smtp.SMTPError{Code: 451, Message: "Error reading message data"}
	}

	// Check if the message size exceeds the raw limit.
	if rawLimit > 0 && s.dataSize > rawLimit {
		tmpFile.Close()
		s.log.Warn("message rejected: size exceeds limit",
			"raw_size_bytes", s.dataSize,
			"approx_encoded_size_mb", float64(s.dataSize*4/3)/(1024*1024),
			"limit_mb", s.cfg.MessageSizeLimitMB)
		return &smtp.SMTPError{
			Code: 552,
			Message: fmt.Sprintf(
				"Message raw size %.2f MB is too large. The limit of %d MB applies to the base64-encoded message.",
				float64(s.dataSize)/(1024*1024),
				s.cfg.MessageSizeLimitMB,
			),
		}
	}

	// Close the file handle we were using for writing.
	if err := tmpFile.Close(); err != nil {
		s.log.Error("failed to close temporary file after write", "err", err)
		return &smtp.SMTPError{Code: 451, Message: "Temporary server error"}
	}

	// Now, open the same temporary file for reading.
	readFile, err := os.Open(s.dataFilePath)
	if err != nil {
		s.log.Error("failed to open temporary file for reading", "err", err)
		return &smtp.SMTPError{Code: 451, Message: "Temporary server error"}
	}
	defer readFile.Close()

	s.log.Info("message data received, preparing to send via gmail", "from", s.from, "to", s.to, "size_bytes", s.dataSize)

	ctx := context.Background()
	sentMsg, err := s.gmailClient.Send(ctx, s.token, s.to, readFile)
	if err != nil {
		s.log.Error("failed to send email via gmail", "err", err)
		if strings.Contains(err.Error(), "quota") {
			return &smtp.SMTPError{Code: 452, Message: "Service temporarily unavailable due to quota limits"}
		}
		if strings.Contains(err.Error(), "authentication") {
			return &smtp.SMTPError{Code: 535, Message: "Authentication required - please run 'smog auth login'"}
		}
		return &smtp.SMTPError{Code: 451, Message: "Temporary failure relaying message"}
	}

	s.log.Info("message relayed successfully",
		"client_ip", s.clientIP,
		"from", s.from,
		"to", s.to,
		"message_id", sentMsg.Id,
	)
	return nil
}

func (s *Session) Reset() {
	if s.dataFilePath != "" {
		if err := os.Remove(s.dataFilePath); err != nil {
			s.log.Warn("failed to remove temporary data file", "path", s.dataFilePath, "err", err)
		}
	}
	s.from = ""
	s.to = s.to[:0] // Reuse slice capacity
	s.dataFilePath = ""
	s.dataSize = 0
}

func (s *Session) Logout() error {
	return nil
}
