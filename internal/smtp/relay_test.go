package smtp

import (
	"context"
	"log/slog"
	"net"
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
