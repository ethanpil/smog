package gmail

import (
	"context"

	"golang.org/x/oauth2"
	"google.golang.org/api/gmail/v1"
)

// MockService is a mock of the Gmail Service interface.
type MockService struct {
	SendFunc func(ctx context.Context, token *oauth2.Token, rawEmail []byte) (*gmail.Message, error)
}

// Send calls the mock's SendFunc.
func (m *MockService) Send(ctx context.Context, token *oauth2.Token, rawEmail []byte) (*gmail.Message, error) {
	return m.SendFunc(ctx, token, rawEmail)
}
