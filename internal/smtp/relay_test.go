package smtp

import (
	"context"
	"io"
	"log/slog"
	"net"
	"os"
	"strings"
	"testing"

	"github.com/emersion/go-smtp"
	"github.com/ethanpil/smog/internal/config"
	"github.com/ethanpil/smog/internal/gmail"
	"golang.org/x/oauth2"
	gapi "google.golang.org/api/gmail/v1"
)

func TestSession_MailRcptData(t *testing.T) {
	// 1. Setup
	dataContent := "This is the email body."
	mockGmail := &gmail.MockService{
		SendFunc: func(ctx context.Context, token *oauth2.Token, recipients []string, rawEmail io.Reader) (*gapi.Message, error) {
			if len(recipients) != 2 {
				t.Errorf("expected 2 recipients, got %d", len(recipients))
			}
			if recipients[0] != "recipient1@example.com" {
				t.Errorf("expected recipient1 to be 'recipient1@example.com', got '%s'", recipients[0])
			}
			if recipients[1] != "recipient2@example.com" {
				t.Errorf("expected recipient2 to be 'recipient2@example.com', got '%s'", recipients[1])
			}
			// The gmail client is responsible for reading the email body.
			emailBytes, err := io.ReadAll(rawEmail)
			if err != nil {
				t.Fatalf("failed to read rawEmail: %v", err)
			}
			// The header is modified by the client, so we can't do a direct string comparison.
			// We just check if the body is present.
			if !strings.Contains(string(emailBytes), dataContent) {
				t.Errorf("expected email body to contain '%s', got '%s'", dataContent, string(emailBytes))
			}

			return &gapi.Message{Id: "test-message-id"}, nil
		},
	}

	session := &Session{
		log:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg:         &config.Config{},
		gmailClient: mockGmail,
		token:       &oauth2.Token{}, // Dummy token
	}
	defer session.Reset()

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
	dataReader := strings.NewReader("To: someone@else.com\r\n\r\n" + dataContent)
	err = session.Data(dataReader)
	if err != nil {
		t.Fatalf("Data() returned an error: %v", err)
	}
	if session.dataSize == 0 {
		t.Error("Expected dataSize to be non-zero after Data(), got 0")
	}
	if _, err := os.Stat(session.dataFilePath); os.IsNotExist(err) {
		t.Errorf("Expected temporary data file to exist, but it doesn't: %s", session.dataFilePath)
	}

	// 4. Test Reset
	session.Reset()
	if session.from != "" {
		t.Errorf("Expected from to be empty after Reset(), got '%s'", session.from)
	}
	if len(session.to) != 0 {
		t.Errorf("Expected to to be empty after Reset(), got '%v'", session.to)
	}
	if session.dataSize != 0 {
		t.Errorf("Expected dataSize to be 0 after Reset(), got '%d'", session.dataSize)
	}
	if session.dataFilePath != "" {
		t.Errorf("Expected dataFilePath to be empty after Reset(), got '%s'", session.dataFilePath)
	}
}

func TestSession_Data_SizeLimit(t *testing.T) {
	// Use a mock that does nothing, as we don't expect it to be called.
	mockGmail := &gmail.MockService{
		SendFunc: func(ctx context.Context, token *oauth2.Token, recipients []string, rawEmail io.Reader) (*gapi.Message, error) {
			t.Error("gmail.Send should not be called for oversized messages")
			return nil, nil
		},
	}

	session := &Session{
		log: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cfg: &config.Config{
			// Set a small limit, e.g., 1MB.
			// The raw limit will be ~0.75MB.
			MessageSizeLimitMB: 1,
		},
		gmailClient: mockGmail,
		token:       &oauth2.Token{},
	}
	defer session.Reset()

	// Test case 1: Message is too large
	t.Run("MessageTooLarge", func(t *testing.T) {
		// Create a large dummy message (e.g., 2MB)
		largeData := make([]byte, 2*1024*1024)
		dataReader := strings.NewReader(string(largeData))

		err := session.Data(dataReader)
		if err == nil {
			t.Fatal("Expected an error for oversized message, but got nil")
		}

		smtpErr, ok := err.(*smtp.SMTPError)
		if !ok {
			t.Fatalf("Expected error to be of type *smtp.SMTPError, but got %T", err)
		}

		if smtpErr.Code != 552 {
			t.Errorf("Expected SMTP error code 552, but got %d", smtpErr.Code)
		}

		if !strings.Contains(smtpErr.Message, "is too large") {
			t.Errorf("Expected error message to contain 'is too large', but got '%s'", smtpErr.Message)
		}
	})

	// Test case 2: Message is within the limit
	t.Run("MessageWithinLimit", func(t *testing.T) {
		// This time, the mock should succeed.
		mockGmail.SendFunc = func(ctx context.Context, token *oauth2.Token, recipients []string, rawEmail io.Reader) (*gapi.Message, error) {
			return &gapi.Message{Id: "success-id"}, nil
		}
		session.gmailClient = mockGmail

		// Create a small message
		smallData := make([]byte, 100) // 100 bytes is well within the limit
		dataReader := strings.NewReader(string(smallData))

		err := session.Data(dataReader)
		if err != nil {
			t.Fatalf("Expected no error for message within limit, but got: %v", err)
		}
	})
}

// mockNetConn is a mock implementation of net.Conn for testing Backend.newSession.
type mockNetConn struct {
	net.Conn
	remoteAddr net.Addr
}

func (m *mockNetConn) RemoteAddr() net.Addr {
	return m.remoteAddr
}

// mockAddr is a mock implementation of net.Addr for testing.
type mockAddr struct {
	network string
	address string
}

func (a *mockAddr) Network() string {
	return a.network
}

func (a *mockAddr) String() string {
	return a.address
}

func TestBackend_newSession(t *testing.T) {
	logger := slog.Default()
	cfg := &config.Config{
		AllowedSubnets: []string{"192.168.1.0/24", "2001:db8::/32"},
	}

	backend := &Backend{
		Cfg: cfg,
		Log: logger,
	}

	testCases := []struct {
		name          string
		remoteAddr    net.Addr
		expectError   bool
		errorContains string
	}{
		{
			name: "Allowed IPv4",
			remoteAddr: &mockAddr{
				network: "tcp",
				address: "192.168.1.10:12345",
			},
			expectError: false,
		},
		{
			name: "Disallowed IPv4",
			remoteAddr: &mockAddr{
				network: "tcp",
				address: "10.0.0.1:12345",
			},
			expectError:   true,
			errorContains: "access denied",
		},
		{
			name: "Allowed IPv6",
			remoteAddr: &mockAddr{
				network: "tcp",
				address: "[2001:db8::1]:12345",
			},
			expectError: false,
		},
		{
			name: "Disallowed IPv6",
			remoteAddr: &mockAddr{
				network: "tcp",
				address: "[fe80::1]:12344",
			},
			expectError:   true,
			errorContains: "access denied",
		},
		{
			name: "Invalid address format",
			remoteAddr: &mockAddr{
				network: "tcp",
				address: "invalid-address",
			},
			expectError:   true,
			errorContains: "internal server error: could not parse address",
		},
		{
			name: "Address without port",
			remoteAddr: &mockAddr{
				network: "tcp",
				address: "192.168.1.10",
			},
			expectError:   true,
			errorContains: "internal server error: could not parse address",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockConn := &mockNetConn{remoteAddr: tc.remoteAddr}
			session, err := backend.newSession(mockConn)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected an error, but got none")
				}
				if err != nil && tc.errorContains != "" && !strings.Contains(err.Error(), tc.errorContains) {
					t.Errorf("Expected error to contain '%s', but it was '%s'", tc.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Did not expect an error, but got: %v", err)
				}
				if session == nil {
					t.Errorf("Expected a session, but got nil")
				}
			}
		})
	}
}
