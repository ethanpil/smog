package gmail

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/mail"
	"strings"

	"golang.org/x/oauth2"
	gapi "google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// Service is the interface for the Gmail client.
type Service interface {
	Send(ctx context.Context, token *oauth2.Token, recipients []string, rawEmail io.Reader) (*gapi.Message, error)
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

// replaceToHeader parses a raw email from a reader, replaces its 'To' header
// with the given recipients, and returns the reconstructed raw email as bytes.
func replaceToHeader(logger *slog.Logger, recipients []string, rawEmail io.Reader) ([]byte, error) {
	msg, err := mail.ReadMessage(rawEmail)
	if err != nil {
		logger.Error("failed to parse raw email for header replacement", "error", err)
		return nil, fmt.Errorf("failed to parse raw email: %w", err)
	}

	// Replace the 'To' header.
	if len(recipients) > 0 {
		msg.Header["To"] = []string{strings.Join(recipients, ", ")}
	} else {
		// If there are no recipients, remove the 'To' header to avoid ambiguity.
		delete(msg.Header, "To")
	}

	var newEmailBuffer bytes.Buffer
	for k, v := range msg.Header {
		// TODO: This is a simplified header writer. A more robust implementation
		// would handle multi-line headers and proper encoding.
		newEmailBuffer.WriteString(fmt.Sprintf("%s: %s\r\n", k, strings.Join(v, ", ")))
	}
	newEmailBuffer.WriteString("\r\n")

	if _, err := io.Copy(&newEmailBuffer, msg.Body); err != nil {
		logger.Error("failed to copy email body during reconstruction", "error", err)
		return nil, fmt.Errorf("failed to reconstruct email: %w", err)
	}

	return newEmailBuffer.Bytes(), nil
}

// Send sends a raw email stream to the Gmail API. It parses the raw email,
// replaces the "To" header with the provided recipients, and then sends it.
func (c *Client) Send(ctx context.Context, token *oauth2.Token, recipients []string, rawEmail io.Reader) (*gapi.Message, error) {
	c.logger.Info("sending email via gmail api", "recipients", recipients)

	// The `replaceToHeader` function now reads from the stream and returns bytes.
	modifiedEmail, err := replaceToHeader(c.logger, recipients, rawEmail)
	if err != nil {
		return nil, err // Error is already logged in replaceToHeader
	}

	// Create a new Gmail service using the provided token.
	srv, err := gapi.NewService(ctx,
		option.WithHTTPClient(c.client),
		option.WithTokenSource(oauth2.StaticTokenSource(token)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gmail service: %w", err)
	}

	// Base64url-encode the *new* raw email.
	encodedEmail := base64.RawURLEncoding.EncodeToString(modifiedEmail)

	// Create a new message.
	message := &gapi.Message{
		Raw: encodedEmail,
	}

	// Send the message.
	sentMsg, err := srv.Users.Messages.Send("me", message).Do()
	if err != nil {
		c.logger.Error("failed to send email", "error", err)
		return nil, fmt.Errorf("failed to send email: %w", err)
	}

	c.logger.Info("email sent successfully", "message_id", sentMsg.Id)
	return sentMsg, nil
}
