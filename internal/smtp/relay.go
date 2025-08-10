package smtp

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"

	"github.com/emersion/go-smtp"
	"github.com/ethanpil/smog/internal/config"
	"github.com/ethanpil/smog/internal/gmail"
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
	return &Session{
		log:         be.Log,
		cfg:         be.Cfg,
		gmailClient: be.GmailClient,
		token:       be.Token,
	}, nil
}

// A Session is returned after EHLO.
type Session struct {
	log         *slog.Logger
	cfg         *config.Config
	gmailClient gmail.Service
	token       *oauth2.Token
	from        string
	to          []string
	data        bytes.Buffer
}

// Login is called by the go-smtp library to authenticate a user.
// It handles both AUTH PLAIN and AUTH LOGIN.
func (s *Session) Login(username, password string) error {
	s.log.Info("AUTH attempt", "username", username)

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
}

// AuthPlain is a wrapper around Login to satisfy the go-smtp interface.
func (s *Session) AuthPlain(username, password string) error {
	return s.Login(username, password)
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
	if _, err := io.Copy(&s.data, r); err != nil {
		s.log.Error("error reading data stream", "err", err)
		return err
	}

	s.log.Info("message data received, preparing to send via gmail", "from", s.from, "to", s.to)

	ctx := context.Background()
	if _, err := s.gmailClient.Send(ctx, s.token, s.data.Bytes()); err != nil {
		s.log.Error("failed to send email via gmail", "err", err)
		// Return a generic error to the SMTP client
		return errors.New("failed to relay message")
	}

	s.log.Info("message relayed successfully", "from", s.from, "to", s.to)
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
