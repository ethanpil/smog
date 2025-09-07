package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"net/http"
	"net/http/httptest"
	"net/smtp"
	"sync"
	"testing"
	"time"

	"github.com/ethanpil/smog/internal/config"
	"github.com/ethanpil/smog/internal/gmail"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	gapi "google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// getFreePort asks the kernel for a free open port that is ready to use.
func getFreePort(t *testing.T) int {
	t.Helper()
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	require.NoError(t, err)

	l, err := net.ListenTCP("tcp", addr)
	require.NoError(t, err)
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// mockGmailService is a custom implementation of the gmail.Service that allows
// us to intercept the Send call and redirect it to our httptest.Server.
type mockGmailService struct {
	gmail.Service
	t           *testing.T
	mockGServer *httptest.Server
	httpClient  *http.Client
}

// newMockGmailService creates a gmail service that routes requests to a mock server.
func newMockGmailService(t *testing.T, mockGServer *httptest.Server) gmail.Service {
	return &mockGmailService{
		t:           t,
		mockGServer: mockGServer,
		httpClient:  mockGServer.Client(), // Use the test server's client
	}
}

// Send overrides the real Send method. It creates a temporary, real gmail.Service
// configured to point to the mock server's URL, then calls the real Send method
// on that temporary service. This avoids having to re-implement the base64 encoding
// and API call structure.
func (m *mockGmailService) Send(ctx context.Context, token *oauth2.Token, rawEmail []byte) (*gapi.Message, error) {
	// Create a real gmail service client, but force it to use our mock server's URL
	// and http client. This is the most reliable way to test the real client's behavior.
	realGmailService, err := gapi.NewService(ctx, option.WithHTTPClient(m.httpClient), option.WithEndpoint(m.mockGServer.URL))
	require.NoError(m.t, err)

	// Base64url-encode the raw email, as the real client would do.
	encodedEmail := base64.RawURLEncoding.EncodeToString(rawEmail)
	message := &gapi.Message{
		Raw: encodedEmail,
	}

	// Call the real "Send" method, which will be directed to our mock server.
	return realGmailService.Users.Messages.Send("me", message).Do()
}

// TestEndToEndMessageRelay simulates a full SMTP transaction and verifies that
// the message is correctly relayed to the mock Google API endpoint.
func TestEndToEndMessageRelay(t *testing.T) {
	// 1. Setup mock Google API server
	var receivedBodyBytes []byte
	var requestReceived sync.Mutex
	requestReceived.Lock() // Lock initially

	mockGoogleAPIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer requestReceived.Unlock() // Unlock when request is received

		// The Google API client's internal URL construction for uploads is complex.
		// The critical part of this test is to verify the body, not the exact URL.
		// We just need to ensure it's a POST request to our mock server.
		require.Equal(t, http.MethodPost, r.Method, "http method should be POST")

		var err error
		receivedBodyBytes, err = io.ReadAll(r.Body)
		require.NoError(t, err, "failed to read request body")

		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(&gapi.Message{Id: "test-message-id"})
		require.NoError(t, err, "failed to encode google api response")
	}))
	defer mockGoogleAPIServer.Close()

	// 2. Configure Smog to use the mock server
	smtpPort := getFreePort(t)

	cfg := &config.Config{
		LogLevel:           "Verbose", // Force verbose logging for tests
		SMTPUser:           "testuser",
		SMTPPassword:       "testpass",
		SMTPPort:           smtpPort,
		MessageSizeLimitMB: 5,
		ReadTimeout:        15,
		WriteTimeout:       15,
		MaxRecipients:      25,
		AllowInsecureAuth:  true,
	}

	// Create a logger that writes to the test's output.
	// The `go test` command will capture stdout and print it on failure
	// or with the -v flag.
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// Create the mock Gmail service that points to our test server
	mockService := newMockGmailService(t, mockGoogleAPIServer)

	// 3. Start the smog server in a goroutine, injecting the mock service
	serverErrChan := make(chan error, 1)
	go func() {
		serverErrChan <- Run(cfg, logger, mockService)
	}()

	// Give the server a moment to start up.
	time.Sleep(100 * time.Millisecond)

	// 4. Use an SMTP client to send a message
	smtpAddr := fmt.Sprintf("localhost:%d", cfg.SMTPPort)
	auth := smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPassword, "localhost")

	from := "sender@example.com"
	to := []string{"recipient@example.com"}
	msg := []byte("To: recipient@example.com\r\n" +
		"Subject: Hello from Smog Test!\r\n" +
		"\r\n" +
		"This is a test message body.\r\n")

	err := smtp.SendMail(smtpAddr, auth, from, to, msg)
	require.NoError(t, err, "smtp.SendMail should succeed")

	// 5. Assertions
	// Wait for the mock API to receive the request, with a timeout.
	success := make(chan bool)
	go func() {
		requestReceived.Lock() // This will block until the handler unlocks it
		close(success)
	}()

	select {
	case <-success:
		// The request was received and the lock was released.
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for mock API to receive request")
	}

	require.NotNil(t, receivedBodyBytes, "mock google api should have received a body")

	// The Google API client sends a multipart request. A simple string search
	// is the most robust way to check for our message's presence.
	// The real gmail.Client double-encodes the message in a multipart body,
	// so we check for the raw message content.
	// The `Send` method in our mock service encodes it, so we need to find the encoded version.
	encodedMsg := base64.RawURLEncoding.EncodeToString(msg)
	bodyStr := string(receivedBodyBytes)

	// The google api client creates a multipart/related request.
	// One part has the metadata (in JSON) and the other has the raw message.
	// The JSON part refers to the raw part.
	// We need to find the base64 encoded message in the body.
	assert.Contains(t, bodyStr, encodedMsg, "request body should contain the base64url encoded message")

	// Check that the server goroutine hasn't exited with an error
	select {
	case err := <-serverErrChan:
		require.NoError(t, err, "server should not have exited with an error")
	default:
		// Server is still running, which is expected.
	}
}
