package smtp

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/ethanpil/smog/internal/config"
	"github.com/ethanpil/smog/internal/gmail"
	"golang.org/x/oauth2"
	gapi "google.golang.org/api/gmail/v1"
)

func TestSession_MailRcptData(t *testing.T) {
	// 1. Setup
	mockGmail := &gmail.MockService{
		SendFunc: func(ctx context.Context, token *oauth2.Token, rawEmail []byte) (*gapi.Message, error) {
			// In a real test, you might inspect the rawEmail content
			return &gapi.Message{Id: "test-message-id"}, nil
		},
	}

	session := &Session{
		log:         slog.Default(),
		cfg:         &config.Config{},
		gmailClient: mockGmail,
		token:       &oauth2.Token{}, // Dummy token
	}

	// 2. Test Mail
	from := "sender@example.com"
	err := session.Mail(from, nil)
	if err != nil {
		t.Fatalf("Mail() returned an error: %v", err)
	}

	if session.from != from {
		t.Errorf("Expected from to be '%s', got '%s'", from, session.from)
	}

	// 2. Test Rcpt
	to1 := "recipient1@example.com"
	err = session.Rcpt(to1, nil)
	if err != nil {
		t.Fatalf("Rcpt() returned an error: %v", err)
	}
	if len(session.to) != 1 || session.to[0] != to1 {
		t.Errorf("Expected to to be ['%s'], got '%v'", to1, session.to)
	}

	to2 := "recipient2@example.com"
	err = session.Rcpt(to2, nil)
	if err != nil {
		t.Fatalf("Rcpt() returned an error: %v", err)
	}
	if len(session.to) != 2 || session.to[1] != to2 {
		t.Errorf("Expected to to be ['%s', '%s'], got '%v'", to1, to2, session.to)
	}

	// 3. Test Data
	dataContent := "This is the email body."
	dataReader := strings.NewReader(dataContent)
	err = session.Data(dataReader)
	if err != nil {
		t.Fatalf("Data() returned an error: %v", err)
	}
	if session.data.String() != dataContent {
		t.Errorf("Expected data to be '%s', got '%s'", dataContent, session.data.String())
	}

	// 4. Test Reset
	session.Reset()
	if session.from != "" {
		t.Errorf("Expected from to be empty after Reset(), got '%s'", session.from)
	}
	if len(session.to) != 0 {
		t.Errorf("Expected to to be empty after Reset(), got '%v'", session.to)
	}
	if session.data.Len() != 0 {
		t.Errorf("Expected data to be empty after Reset(), got '%d'", session.data.Len())
	}

	// 5. Test Mail again to ensure Reset worked
	from2 := "another.sender@example.com"
	err = session.Mail(from2, nil)
	if err != nil {
		t.Fatalf("Mail() returned an error: %v", err)
	}
	if session.from != from2 {
		t.Errorf("Expected from to be '%s', got '%s'", from2, session.from)
	}
	if len(session.to) != 0 {
		t.Errorf("Expected to to be empty after Mail(), got '%v'", session.to)
	}
}
