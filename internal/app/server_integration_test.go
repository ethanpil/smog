package app

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"net/smtp"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ethanpil/smog/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	gogmail "google.golang.org/api/gmail/v1"
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

// Test helper to create a mock google api server and a client that talks to it.
func newMockGServer(t *testing.T) (*http.Client, *httptest.Server) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This handler can be expanded if tests need to simulate errors or specific responses.
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(&gogmail.Message{Id: "test-message-id"})
		require.NoError(t, err, "failed to encode google api response")
	}))

	// Create an http client that is configured to talk to our mock server.
	// It's also configured with a dummy token source.
	client := &http.Client{
		Transport: &oauth2.Transport{
			Source: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "dummy-token-for-test"}),
			Base:   mockServer.Client().Transport,
		},
	}

	return client, mockServer
}

// TestEndToEndMessageRelay simulates a full SMTP transaction and verifies that
// the message is correctly relayed to the mock Google API endpoint.
func TestEndToEndMessageRelay(t *testing.T) {
	// 1. Setup mock Google API server
	var receivedBodyBytes []byte
	var requestReceived sync.Mutex
	requestReceived.Lock() // Lock initially

	// This is the new part of the test setup. We create a mock server and a client that talks to it.
	mockGoogleAPIClient, mockGoogleAPIServer := newMockGServer(t)
	defer mockGoogleAPIServer.Close()

	// We need to override the handler to get the request body
	mockGoogleAPIServer.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer requestReceived.Unlock() // Unlock when request is received

		require.Equal(t, http.MethodPost, r.Method, "http method should be POST")

		var err error
		receivedBodyBytes, err = io.ReadAll(r.Body)
		require.NoError(t, err, "failed to read request body")

		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(&gogmail.Message{Id: "test-message-id"})
		require.NoError(t, err, "failed to encode google api response")
	})

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
		// These paths are needed for the call to auth.GetClient inside app.Run
		GoogleCredentialsPath: "testdata/dummy_credentials.json",
		GoogleTokenPath:       "testdata/dummy_token.json",
	}

	// Create dummy credential and token files for the test
	require.NoError(t, os.MkdirAll("testdata", 0755))
	require.NoError(t, os.WriteFile(cfg.GoogleCredentialsPath, []byte(`{"installed":{"client_id":"dummy"}}`), 0644))
	require.NoError(t, os.WriteFile(cfg.GoogleTokenPath, []byte(`{"access_token":"dummy","refresh_token":"dummy"}`), 0644))
	defer os.RemoveAll("testdata")

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// 3. Start the smog server in a goroutine, injecting the mock client
	serverErrChan := make(chan error, 1)
	go func() {
		// We pass the mock client here. The Run function will use this client
		// instead of creating a real one.
		serverErrChan <- Run(cfg, logger, mockGoogleAPIClient)
	}()

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
	success := make(chan bool)
	go func() {
		requestReceived.Lock()
		close(success)
	}()

	select {
	case <-success:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for mock API to receive request")
	}

	require.NotNil(t, receivedBodyBytes, "mock google api should have received a body")

	encodedMsg := base64.URLEncoding.EncodeToString(msg)
	bodyStr := string(receivedBodyBytes)

	assert.Contains(t, bodyStr, encodedMsg, "request body should contain the base64url encoded message")

	select {
	case err := <-serverErrChan:
		require.NoError(t, err, "server should not have exited with an error")
	default:
	}
}
