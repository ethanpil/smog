package smtp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
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
	log         *slog.Logger
	cfg         *config.Config
	gmailClient gmail.Service
	token       *oauth2.Token
	clientIP    string
	from        string
	to          []string
	data        bytes.Buffer
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

	// Enforce message size limit.
	if s.cfg.MessageSizeLimitMB > 0 {
		limitBytes := int64(s.cfg.MessageSizeLimitMB) * 1024 * 1024
		// Use a LimitReader to avoid reading the entire message into memory if it's too large.
		// We read up to limitBytes + 1 to see if the message is over the limit.
		lr := io.LimitReader(r, limitBytes+1)
		if _, err := io.Copy(&s.data, lr); err != nil {
			s.log.Error("error reading data stream", "err", err)
			return err
		}

		if int64(s.data.Len()) > limitBytes {
			s.log.Warn("message rejected: size exceeds limit",
				"size_mb", float64(s.data.Len())/(1024*1024),
				"limit_mb", s.cfg.MessageSizeLimitMB)
			return &smtp.SMTPError{
				Code:    552,
				Message: fmt.Sprintf("Message size %.2f MB exceeds limit of %d MB",
					float64(s.data.Len())/(1024*1024), s.cfg.MessageSizeLimitMB),
			}
		}
	} else {
		// If no limit is set, read the whole thing.
		if _, err := io.Copy(&s.data, r); err != nil {
			s.log.Error("error reading data stream", "err", err)
			return err
		}
	}

	s.log.Info("message data received, preparing to send via gmail", "from", s.from, "to", s.to)

	ctx := context.Background()
	sentMsg, err := s.gmailClient.Send(ctx, s.token, s.to, s.data.Bytes())
	if err != nil {
		s.log.Error("failed to send email via gmail", "err", err)
		// Return more specific errors based on error type
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
	s.from = ""
	s.to = s.to[:0] // Reuse slice capacity
	s.data.Reset()
}

func (s *Session) Logout() error {
	return nil
}
