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
	}, nil
}

// A Session is returned after EHLO.
type Session struct {
	log *slog.Logger
}

func (s *Session) AuthPlain(username, password string) error {
	return errors.New("unimplemented")
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
