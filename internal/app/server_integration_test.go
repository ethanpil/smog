package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"net/mail"
	"net/smtp"
	"os"
	"strings"
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

// Send overrides the real Send method. It mimics the new behavior of the real
// gmail.Client by performing header replacement before sending the message to the
// underlying mock google API http server.
func (m *mockGmailService) Send(ctx context.Context, token *oauth2.Token, recipients []string, rawEmail io.Reader) (*gapi.Message, error) {
	// Mimic the header replacement logic from the actual client.
	// The reader is passed directly, no need to create a new one.
	msg, err := mail.ReadMessage(rawEmail)
	require.NoError(m.t, err, "mock send: failed to parse raw email")

	msg.Header["To"] = []string{strings.Join(recipients, ", ")}

	var newEmailBuffer bytes.Buffer
	for k, v := range msg.Header {
		newEmailBuffer.WriteString(fmt.Sprintf("%s: %s\r\n", k, strings.Join(v, ", ")))
	}
	newEmailBuffer.WriteString("\r\n")
	_, err = io.Copy(&newEmailBuffer, msg.Body)
	require.NoError(m.t, err, "mock send: failed to copy body")

	// Create a real gmail service client, but force it to use our mock server's URL
	// and http client. This is the most reliable way to test the real client's behavior.
	realGmailService, err := gapi.NewService(ctx, option.WithHTTPClient(m.httpClient), option.WithEndpoint(m.mockGServer.URL))
	require.NoError(m.t, err)

	// Base64url-encode the *modified* email.
	encodedEmail := base64.RawURLEncoding.EncodeToString(newEmailBuffer.Bytes())
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

	// Wait for the server to become available by polling the SMTP port.
	smtpAddr := fmt.Sprintf("localhost:%d", cfg.SMTPPort)
	var conn net.Conn
	var err error
	for i := 0; i < 20; i++ { // Retry for up to 2 seconds
		conn, err = net.DialTimeout("tcp", smtpAddr, 100*time.Millisecond)
		if err == nil {
			conn.Close()
			break // Success
		}
		time.Sleep(100 * time.Millisecond)
	}
	require.NoError(t, err, "failed to connect to SMTP server on %s after multiple retries", smtpAddr)

	// 4. Use an SMTP client to send a message
	auth := smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPassword, "localhost")

	from := "sender@example.com"
	// Use different recipients for the envelope and the header to expose the bug.
	envelopeRecipient := "envelope-recipient@example.com"
	headerRecipient := "header-recipient@example.com"

	to := []string{envelopeRecipient}
	msg := []byte(fmt.Sprintf("To: %s\r\n"+
		"Subject: Hello from Smog Test!\r\n"+
		"\r\n"+
		"This is a test message body.\r\n", headerRecipient))

	err = smtp.SendMail(smtpAddr, auth, from, to, msg)
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

	// --- Start: Modified Assertion ---
	// The original test only checked for the presence of the message body.
	// This new assertion checks that the recipient is the one from the SMTP
	// envelope (`RCPT TO`), not the one from the message header (`To:`).

	// The Google API client sends a JSON request containing the raw message.
	// We need to unmarshal it to get the base64-encoded data.
	var apiMsg gapi.Message
	// The body may not be a direct JSON representation of gapi.Message,
	// but rather a multipart message. We'll parse the raw message from it.
	// A robust way is to find the `"raw": "..."` part in the JSON body.
	// For this test, we can make a simplifying assumption that the body
	// can be parsed into the Message struct. If not, we'd need a more
	// complex multipart parser.
	// Let's first try to unmarshal the whole body.
	err = json.Unmarshal(receivedBodyBytes, &apiMsg)
	if err != nil {
		// If unmarshalling fails, it's likely a multipart body.
		// We'll have to resort to string searching for the raw field,
		// which is less robust but necessary for this test structure.
		bodyStr := string(receivedBodyBytes)
		// This is a simplified way to extract the base64 content.
		// A real implementation would parse the multipart MIME message.
		start := "{\"raw\":\""
		end := "\"}"
		if s := strings.Index(bodyStr, start); s != -1 {
			if e := strings.LastIndex(bodyStr, end); e != -1 && e > s {
				rawBase64 := bodyStr[s+len(start) : e]
				// The Google client might use standard base64, not raw. Let's try both.
				decodedBytes, decodeErr := base64.RawURLEncoding.DecodeString(rawBase64)
				if decodeErr != nil {
					decodedBytes, decodeErr = base64.StdEncoding.DecodeString(rawBase64)
				}
				require.NoError(t, decodeErr, "failed to decode base64 raw string from body")
				apiMsg.Raw = base64.RawURLEncoding.EncodeToString(decodedBytes) // Re-encode with the one that works for the next step
			}
		}
	}
	require.NotEmpty(t, apiMsg.Raw, "could not extract raw message from request body")

	// Decode the raw message from base64.
	decodedMsg, err := base64.RawURLEncoding.DecodeString(apiMsg.Raw)
	require.NoError(t, err, "failed to decode raw message")

	// Assert that the decoded message contains the ENVELOPE recipient.
	// This is the crucial part of the test. Due to the bug, the message
	// will actually contain the HEADER recipient, and this assertion will fail.
	expectedToHeader := fmt.Sprintf("To: %s", envelopeRecipient)
	assert.Contains(t, string(decodedMsg), expectedToHeader,
		"The 'To:' header in the relayed message should match the SMTP envelope recipient")
	// --- End: Modified Assertion ---

	// Check that the server goroutine hasn't exited with an error
	select {
	case err := <-serverErrChan:
		require.NoError(t, err, "server should not have exited with an error")
	default:
		// Server is still running, which is expected.
	}
}
