package gmail

import (
	"io"
	"log/slog"
	"net/mail"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReplaceToHeader(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	testCases := []struct {
		name               string
		originalTo         string
		newRecipients      []string
		expectedToHeader   string
		expectHeaderExists bool
	}{
		{
			name:               "Single recipient replacement",
			originalTo:         "original@example.com",
			newRecipients:      []string{"new@example.com"},
			expectedToHeader:   "new@example.com",
			expectHeaderExists: true,
		},
		{
			name:               "Multiple recipient replacement",
			originalTo:         "original@example.com",
			newRecipients:      []string{"new1@example.com", "new2@example.com"},
			expectedToHeader:   "new1@example.com, new2@example.com",
			expectHeaderExists: true,
		},
		{
			name:               "Replacing multiple recipients with one",
			originalTo:         "old1@example.com, old2@example.com",
			newRecipients:      []string{"new@example.com"},
			expectedToHeader:   "new@example.com",
			expectHeaderExists: true,
		},
		{
			name:               "Empty recipient list removes header",
			originalTo:         "original@example.com",
			newRecipients:      []string{},
			expectedToHeader:   "",
			expectHeaderExists: false,
		},
		{
			name:               "Email with no initial To header",
			originalTo:         "", // No "To" header will be added
			newRecipients:      []string{"new@example.com"},
			expectedToHeader:   "new@example.com",
			expectHeaderExists: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 1. Construct raw email
			var rawEmail strings.Builder
			rawEmail.WriteString("From: sender@example.com\r\n")
			if tc.originalTo != "" {
				rawEmail.WriteString("To: " + tc.originalTo + "\r\n")
			}
			rawEmail.WriteString("Subject: Test\r\n")
			rawEmail.WriteString("\r\n")
			rawEmail.WriteString("This is the body.")

			// 2. Call the function
			modifiedEmailBytes, err := replaceToHeader(logger, tc.newRecipients, []byte(rawEmail.String()))
			require.NoError(t, err)

			// 3. Parse the result and assert
			msg, err := mail.ReadMessage(strings.NewReader(string(modifiedEmailBytes)))
			require.NoError(t, err)

			toHeader, ok := msg.Header["To"]
			if tc.expectHeaderExists {
				require.True(t, ok, "Expected 'To' header to exist")
				assert.Equal(t, tc.expectedToHeader, toHeader[0])
			} else {
				assert.False(t, ok, "Expected 'To' header to be removed")
			}

			// Ensure body is preserved
			body, err := io.ReadAll(msg.Body)
			require.NoError(t, err)
			assert.Equal(t, "This is the body.", string(body))
		})
	}
}
