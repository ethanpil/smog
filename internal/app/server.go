package app

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/emersion/go-smtp"
	"github.com/ethanpil/smog/internal/config"
	smog_smtp "github.com/ethanpil/smog/internal/smtp"
)

func Run(cfg *config.Config, logger *slog.Logger) error {
	be := &smog_smtp.Backend{
		Cfg: cfg,
		Log: logger,
	}

	s := smtp.NewServer(be)

	s.Addr = fmt.Sprintf(":%d", cfg.SMTPPort)
	s.Domain = "localhost"
	s.ReadTimeout = 10 * time.Second
	s.WriteTimeout = 10 * time.Second
	s.MaxMessageBytes = int64(cfg.MessageSizeLimitMB) * 1024 * 1024
	s.MaxRecipients = 50
	s.AllowInsecureAuth = true

	logger.Info("starting smog smtp relay", "address", s.Addr)
	return s.ListenAndServe()
}
