package smtp

import (
	"errors"
	"io"
	"log/slog"

	"github.com/emersion/go-smtp"
	"github.com/ethanpil/smog/internal/config"
)

// The Backend implements SMTP server methods.
type Backend struct {
	Cfg *config.Config
	Log *slog.Logger
}

func (be *Backend) NewSession(c *smtp.Conn) (smtp.Session, error) {
	return &Session{
		log: be.Log,
		cfg: be.Cfg,
	}, nil
}

// A Session is returned after EHLO.
type Session struct {
	log *slog.Logger
	cfg *config.Config
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
	s.log.Info("MAIL FROM:", "from", from)
	return nil
}

func (s *Session) Rcpt(to string, opts *smtp.RcptOptions) error {
	s.log.Info("RCPT TO:", "to", to)
	return nil
}

func (s *Session) Data(r io.Reader) error {
	if _, err := io.Copy(io.Discard, r); err != nil {
		return err
	}
	return nil
}

func (s *Session) Reset() {}

func (s *Session) Logout() error {
	return nil
}
