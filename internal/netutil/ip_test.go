package netutil

import (
	"io"
	"log/slog"
	"net"
	"testing"
)

func TestIsAllowed(t *testing.T) {
	// Create a dummy logger that discards output.
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	testCases := []struct {
		name           string
		clientIP       string
		allowedSubnets []string
		expected       bool
	}{
		{
			name:           "Empty allowed list",
			clientIP:       "192.168.1.10",
			allowedSubnets: []string{},
			expected:       true,
		},
		{
			name:           "IP in CIDR block",
			clientIP:       "192.168.1.10",
			allowedSubnets: []string{"192.168.1.0/24"},
			expected:       true,
		},
		{
			name:           "IP not in CIDR block",
			clientIP:       "192.168.2.10",
			allowedSubnets: []string{"192.168.1.0/24"},
			expected:       false,
		},
		{
			name:           "IP matches single allowed IP",
			clientIP:       "10.0.0.5",
			allowedSubnets: []string{"10.0.0.5"},
			expected:       true,
		},
		{
			name:           "IP does not match single allowed IP",
			clientIP:       "10.0.0.6",
			allowedSubnets: []string{"10.0.0.5"},
			expected:       false,
		},
		{
			name:           "IP matches one of multiple entries",
			clientIP:       "172.16.0.5",
			allowedSubnets: []string{"10.0.0.0/8", "172.16.0.0/16"},
			expected:       true,
		},
		{
			name:           "IP matches neither of multiple entries",
			clientIP:       "192.168.1.1",
			allowedSubnets: []string{"10.0.0.0/8", "172.16.0.0/16"},
			expected:       false,
		},
		{
			name:           "Invalid entry in list is ignored",
			clientIP:       "192.168.1.1",
			allowedSubnets: []string{"10.0.0.0/8", "not-an-ip", "192.168.1.1"},
			expected:       true,
		},
		{
			name:           "IPv6 in CIDR block",
			clientIP:       "2001:db8::1",
			allowedSubnets: []string{"2001:db8::/32"},
			expected:       true,
		},
		{
			name:           "IPv6 not in CIDR block",
			clientIP:       "2001:db9::1",
			allowedSubnets: []string{"2001:db8::/32"},
			expected:       false,
		},
		{
			name:           "IPv6 matches single allowed IP",
			clientIP:       "fe80::1",
			allowedSubnets: []string{"fe80::1"},
			expected:       true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			clientIP := net.ParseIP(tc.clientIP)
			if clientIP == nil {
				t.Fatalf("Invalid IP address in test case: %s", tc.clientIP)
			}

			result := IsAllowed(logger, clientIP, tc.allowedSubnets)
			if result != tc.expected {
				t.Errorf("IsAllowed(%s, %v) = %v; want %v", tc.clientIP, tc.allowedSubnets, result, tc.expected)
			}
		})
	}
}
