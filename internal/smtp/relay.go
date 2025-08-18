package smtp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"

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
	remoteAddr := c.Conn().RemoteAddr()
	ipStr, _, err := net.SplitHostPort(remoteAddr.String())
	if err != nil {
		be.Log.Error("could not parse remote address", "remoteAddr", remoteAddr.String(), "err", err)
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
		// Per AGENTS.md, reject default password. This is a safety check;
		// the main validation should be at startup.
		if s.cfg.SMTPPassword == "smoggmos" {
			s.log.Error("authentication failed: server is using the default insecure password")
			return errors.New("authentication failed: server misconfiguration")
		}

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
			s.log.Warn("message rejected: size exceeds limit", "from", s.from, "to", s.to, "size_bytes", s.data.Len(), "limit_bytes", limitBytes)
			return &smtp.SMTPError{
				Code:    552,
				Message: fmt.Sprintf("Message size exceeds fixed limit of %d MB", s.cfg.MessageSizeLimitMB),
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
	sentMsg, err := s.gmailClient.Send(ctx, s.token, s.data.Bytes())
	if err != nil {
		s.log.Error("failed to send email via gmail", "err", err)
		// Return a generic error to the SMTP client
		return errors.New("failed to relay message")
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
	s.to = nil
	s.data.Reset()
}

func (s *Session) Logout() error {
	return nil
}
