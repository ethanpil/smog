package app

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"net/http"

	"github.com/emersion/go-smtp"
	"github.com/ethanpil/smog/internal/config"
	"github.com/ethanpil/smog/internal/gmail"
	smog_smtp "github.com/ethanpil/smog/internal/smtp"
)

func Run(cfg *config.Config, logger *slog.Logger, client *http.Client) error {
	// If a specific gmail service isn't provided, create the default one.
	// This is the standard operational path.
	if client == nil {
		// This branch is now primarily for tests that don't need a live http client.
		// A nil client will cause the backend to fail if it tries to send an email.
		logger.Warn("running with a nil http client; email sending will fail")
	}

	gmailService := gmail.New(logger, client)

	be := &smog_smtp.Backend{
		Cfg:         cfg,
		Log:         logger,
		GmailClient: gmailService,
	}

	s := smtp.NewServer(be)

	s.Addr = fmt.Sprintf(":%d", cfg.SMTPPort)
	s.Domain = "localhost"
	s.ReadTimeout = time.Duration(cfg.ReadTimeout) * time.Second
	s.WriteTimeout = time.Duration(cfg.WriteTimeout) * time.Second
	s.MaxMessageBytes = int64(cfg.MessageSizeLimitMB) * 1024 * 1024
	s.MaxRecipients = cfg.MaxRecipients
	s.AllowInsecureAuth = cfg.AllowInsecureAuth

	// Channel to hold errors from the server goroutine
	serverErrors := make(chan error, 1)

	// Goroutine to run the server
	go func() {
		logger.Info("starting smog smtp relay", "address", s.Addr)
		// `ListenAndServe` blocks until an error occurs. `ErrServerClosed` is expected
		// on a graceful shutdown, so we ignore it.
		if err := s.ListenAndServe(); err != nil && err != smtp.ErrServerClosed {
			serverErrors <- fmt.Errorf("smtp server error: %w", err)
		}
	}()

	// Wait for an interrupt signal or a server error
	logger.Info("smog is running. press ctrl-c to exit.")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		// This case handles errors during server startup or runtime.
		logger.Error("server failed to start or encountered a fatal error", "err", err)
		// Attempt a clean shutdown anyway, logging any further errors.
		if closeErr := s.Close(); closeErr != nil {
			logger.Error("failed to close smtp server during error handling", "err", closeErr)
		}
		return err // Return the original error that caused the server to fail.

	case sig := <-quit:
		// This case handles a graceful shutdown signal from the OS.
		logger.Info("shutting down smog smtp relay", "signal", sig.String())
		if err := s.Close(); err != nil {
			// This error means the graceful shutdown failed.
			return fmt.Errorf("failed to gracefully shutdown smtp server: %w", err)
		}
		logger.Info("smog smtp relay shut down gracefully")
	}

	return nil
}
