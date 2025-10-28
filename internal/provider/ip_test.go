package provider

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateCIDR(t *testing.T) {
	tests := []struct {
		name        string
		input       interface{}
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid IPv4 /8",
			input:       "10.0.0.0/8",
			expectError: false,
		},
		{
			name:        "valid IPv4 /16",
			input:       "172.16.0.0/16",
			expectError: false,
		},
		{
			name:        "valid IPv4 /24",
			input:       "192.168.1.0/24",
			expectError: false,
		},
		{
			name:        "invalid IPv4 prefix /32",
			input:       "192.168.1.1/32",
			expectError: true,
			errorMsg:    "IPv4 prefix length must be 8, 16, or 24",
		},
		{
			name:        "valid IPv6 /4",
			input:       "2000::/4",
			expectError: false,
		},
		{
			name:        "valid IPv6 /64",
			input:       "2001:db8::/64",
			expectError: false,
		},
		{
			name:        "invalid IPv6 prefix /3",
			input:       "2000::/3",
			expectError: true,
			errorMsg:    "IPv6 prefix length must be a multiple of 4 between 4 and 124",
		},
		{
			name:        "invalid IPv6 prefix /128",
			input:       "2001:db8::1/128",
			expectError: true,
			errorMsg:    "IPv6 prefix length must be a multiple of 4 between 4 and 124",
		},
		{
			name:        "invalid CIDR format",
			input:       "invalid",
			expectError: true,
			errorMsg:    "invalid CIDR format",
		},
		{
			name:        "non-string input",
			input:       123,
			expectError: true,
			errorMsg:    "expected string, got int",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws, errors := ValidateCIDR(tt.input, "test")

			if tt.expectError {
				require.NotEmpty(t, errors)
				assert.Contains(t, errors[0].Error(), tt.errorMsg)
			} else {
				assert.Empty(t, errors)
				assert.Empty(t, ws)
			}
		})
	}
}

func TestParsePTRRecordName(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    net.IP
		expectError bool
	}{
		{
			name:        "valid IPv4 PTR",
			input:       "4.3.2.1.in-addr.arpa.",
			expected:    net.ParseIP("1.2.3.4"),
			expectError: false,
		},
		{
			name:        "valid IPv6 PTR",
			input:       "b.a.9.8.7.6.5.4.3.2.1.0.f.e.d.c.b.a.9.8.ip6.arpa.",
			expected:    net.ParseIP("2001:db8::1"),
			expectError: false,
		},
		{
			name:        "invalid IPv4 PTR - wrong length",
			input:       "4.3.2.in-addr.arpa.",
			expectError: true,
		},
		{
			name:        "invalid IPv6 PTR - wrong length",
			input:       "b.a.9.8.ip6.arpa.",
			expectError: true,
		},
		{
			name:        "unsupported format",
			input:       "example.com.",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParsePTRRecordName(tt.input)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestGetPTRRecordName(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
	}{
		{
			name:        "valid IPv4",
			input:       "192.168.1.10",
			expected:    "10.1.168.192",
			expectError: false,
		},
		{
			name:        "valid IPv6",
			input:       "2001:db8::1",
			expected:    "1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2",
			expectError: false,
		},
		{
			name:        "invalid IP",
			input:       "invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetPTRRecordName(tt.input)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestParseReverseZoneName(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
	}{
		{
			name:        "valid IPv4 /8",
			input:       "10.in-addr.arpa.",
			expected:    "10.0.0.0/8",
			expectError: false,
		},
		{
			name:        "valid IPv4 /16",
			input:       "16.172.in-addr.arpa.",
			expected:    "172.16.0.0/16",
			expectError: false,
		},
		{
			name:        "valid IPv4 /24",
			input:       "1.168.192.in-addr.arpa.",
			expected:    "192.168.1.0/24",
			expectError: false,
		},
		{
			name:        "valid IPv6 /4",
			input:       "2.ip6.arpa.",
			expected:    "2000::/4",
			expectError: false,
		},
		{
			name:        "invalid IPv4 - too many parts",
			input:       "1.2.3.4.in-addr.arpa.",
			expectError: true,
		},
		{
			name:        "invalid IPv6 - invalid nibble",
			input:       "g.ip6.arpa.",
			expectError: true,
		},
		{
			name:        "unsupported format",
			input:       "example.com.",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseReverseZoneName(tt.input)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestGetReverseZoneName(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
	}{
		{
			name:        "valid IPv4 /8",
			input:       "10.0.0.0/8",
			expected:    "10.in-addr.arpa.",
			expectError: false,
		},
		{
			name:        "valid IPv4 /16",
			input:       "172.16.0.0/16",
			expected:    "16.172.in-addr.arpa.",
			expectError: false,
		},
		{
			name:        "valid IPv4 /24",
			input:       "192.168.1.0/24",
			expected:    "1.168.192.in-addr.arpa.",
			expectError: false,
		},
		{
			name:        "valid IPv6 /4",
			input:       "2000::/4",
			expected:    "2.ip6.arpa.",
			expectError: false,
		},
		{
			name:        "invalid CIDR",
			input:       "invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GetReverseZoneName(tt.input)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}