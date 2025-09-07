package gmail

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"

	"golang.org/x/oauth2"
	gapi "google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// Service is the interface for the Gmail client.
type Service interface {
	Send(ctx context.Context, token *oauth2.Token, rawEmail []byte) (*gapi.Message, error)
}

// Client is a wrapper around the Gmail API client.
type Client struct {
	logger *slog.Logger
	client *http.Client
}

// New creates a new Gmail client.
func New(logger *slog.Logger, client *http.Client) Service {
	return &Client{
		logger: logger,
		client: client,
	}
}

// Send sends a raw email buffer to the Gmail API.
func (c *Client) Send(ctx context.Context, token *oauth2.Token, rawEmail []byte) (*gapi.Message, error) {
	c.logger.Info("sending email via gmail api")

	// Create a new Gmail service using the provided token
	srv, err := gapi.NewService(ctx,
		option.WithHTTPClient(c.client),
		option.WithTokenSource(oauth2.StaticTokenSource(token)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gmail service: %w", err)
	}

	// Base64url-encode the raw email
	encodedEmail := base64.RawURLEncoding.EncodeToString(rawEmail)

	// Create a new message
	message := &gapi.Message{
		Raw: encodedEmail,
	}

	// Send the message
	sentMsg, err := srv.Users.Messages.Send("me", message).Do()
	if err != nil {
		c.logger.Error("failed to send email", "error", err)
		return nil, fmt.Errorf("failed to send email: %w", err)
	}

	c.logger.Info("email sent successfully", "message_id", sentMsg.Id)
	return sentMsg, nil
}
