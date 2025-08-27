package gmail

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"

	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// Service is the interface for the Gmail client.
type Service interface {
	Send(ctx context.Context, rawEmail []byte) (*gmail.Message, error)
}

// Client is a wrapper around the Gmail API client.
type Client struct {
	logger *slog.Logger
	client *http.Client // This client is pre-configured with authentication
}

// New creates a new Gmail client.
func New(logger *slog.Logger, client *http.Client) Service {
	return &Client{
		logger: logger,
		client: client,
	}
}

// Send sends a raw email buffer to the Gmail API.
func (c *Client) Send(ctx context.Context, rawEmail []byte) (*gmail.Message, error) {
	c.logger.Info("sending email via gmail api")

	// Create a new Gmail service. The underlying http.Client (`c.client`) is already
	// authenticated and handles token refreshes automatically. We no longer pass a
	// static token source, which was the cause of the refresh issue.
	srv, err := gmail.NewService(ctx, option.WithHTTPClient(c.client))
	if err != nil {
		return nil, fmt.Errorf("failed to create gmail service: %w", err)
	}

	// Base64url-encode the raw email
	encodedEmail := base64.URLEncoding.EncodeToString(rawEmail)

	// Create a new message
	message := &gmail.Message{
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
