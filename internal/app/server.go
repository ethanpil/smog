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
	"github.com/ethanpil/smog/internal/auth"
	"github.com/ethanpil/smog/internal/config"
	"github.com/ethanpil/smog/internal/gmail"
	smog_smtp "github.com/ethanpil/smog/internal/smtp"
	"golang.org/x/oauth2"
)

func Run(cfg *config.Config, logger *slog.Logger, gmailService gmail.Service) error {
	var token *oauth2.Token
	var err error

	// If a specific gmail service isn't provided, create the default one.
	// This is the standard operational path.
	if gmailService == nil {
		logger.Debug("creating default google api client")
		var httpClient *http.Client
		httpClient, token, err = auth.GetClient(logger, cfg)
		if err != nil {
			return fmt.Errorf("could not get google api client: %w", err)
		}
		gmailService = gmail.New(logger, httpClient)
	} else {
		// If a gmail service (likely a mock) is provided, we still need a placeholder token
		// for the backend, although it may not be used by the mock.
		logger.Debug("using provided gmail service")
		token = &oauth2.Token{AccessToken: "fake-test-token"}
	}

	be := &smog_smtp.Backend{
		Cfg:         cfg,
		Log:         logger,
		GmailClient: gmailService,
		Token:       token,
	}

	s := smtp.NewServer(be)

	s.Addr = fmt.Sprintf(":%d", cfg.SMTPPort)
	s.Domain = "localhost"
	s.ReadTimeout = 10 * time.Second
	s.WriteTimeout = 10 * time.Second
	s.MaxMessageBytes = int64(cfg.MessageSizeLimitMB) * 1024 * 1024
	s.MaxRecipients = 50
	s.AllowInsecureAuth = true

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
