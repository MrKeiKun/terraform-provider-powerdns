package provider

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name                string
		serverURL           string
		recursorServerURL   string
		apiKey              string
		cacheEnable         bool
		cacheSizeMB         string
		cacheTTL            int
		expectError         bool
		expectedErrorMsg    string
	}{
		{
			name:              "valid client creation",
			serverURL:         "https://example.com",
			recursorServerURL: "https://recursor.example.com",
			apiKey:            "test-key",
			cacheEnable:       false,
			cacheSizeMB:       "10",
			cacheTTL:          60,
			expectError:       false,
		},
		{
			name:             "empty serverURL",
			serverURL:        "",
			recursorServerURL: "https://recursor.example.com",
			apiKey:           "test-key",
			expectError:      true,
			expectedErrorMsg: "serverURL cannot be empty",
		},
		{
			name:             "empty recursorServerURL",
			serverURL:        "https://example.com",
			recursorServerURL: "",
			apiKey:           "test-key",
			expectError:      true,
			expectedErrorMsg: "recursorServerURL cannot be empty",
		},
		{
			name:             "empty apiKey",
			serverURL:        "https://example.com",
			recursorServerURL: "https://recursor.example.com",
			apiKey:           "",
			expectError:      true,
			expectedErrorMsg: "apiKey cannot be empty",
		},
		{
			name:             "negative cacheTTL",
			serverURL:        "https://example.com",
			recursorServerURL: "https://recursor.example.com",
			apiKey:           "test-key",
			cacheTTL:         -1,
			expectError:      true,
			expectedErrorMsg: "cacheTTL cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(ctx, tt.serverURL, tt.recursorServerURL, tt.apiKey, nil, tt.cacheEnable, tt.cacheSizeMB, tt.cacheTTL)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrorMsg)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, client)
				assert.Equal(t, tt.serverURL, client.ServerURL)
				assert.Equal(t, tt.recursorServerURL, client.RecursorServerURL)
				assert.Equal(t, tt.apiKey, client.APIKey)
			}
		})
	}
}

func TestParseCacheSizeMB(t *testing.T) {
	tests := []struct {
		name        string
		cacheSizeMB string
		expected    int
		expectError bool
	}{
		{
			name:        "valid cache size",
			cacheSizeMB: "10",
			expected:    10 * 1024 * 1024,
			expectError: false,
		},
		{
			name:        "invalid cache size",
			cacheSizeMB: "invalid",
			expectError: true,
		},
		{
			name:        "zero cache size",
			cacheSizeMB: "0",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseCacheSizeMB(tt.cacheSizeMB)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestSanitizeURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		hasError bool
	}{
		{
			name:     "valid HTTPS URL",
			input:    "https://example.com",
			expected: "https://example.com",
			hasError: false,
		},
		{
			name:     "valid HTTP URL",
			input:    "http://example.com",
			expected: "http://example.com",
			hasError: false,
		},
		{
			name:     "URL without scheme",
			input:    "example.com",
			expected: "https://example.com",
			hasError: false,
		},
		{
			name:     "URL with port",
			input:    "https://example.com:8080",
			expected: "https://example.com:8080",
			hasError: false,
		},
		{
			name:     "empty URL",
			input:    "",
			hasError: true,
		},
		{
			name:     "invalid scheme",
			input:    "ftp://example.com",
			expected: "https://example.com",
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := sanitizeURL(tt.input)

			if tt.hasError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestRecord_ID(t *testing.T) {
	record := Record{
		Name: "example.com",
		Type: "A",
	}

	expected := "example.com:::A"
	assert.Equal(t, expected, record.ID())
}

func TestResourceRecordSet_ID(t *testing.T) {
	rrSet := ResourceRecordSet{
		Name: "example.com",
		Type: "A",
	}

	expected := "example.com:::A"
	assert.Equal(t, expected, rrSet.ID())
}

func TestParseID(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectedName string
		expectedType string
		hasError     bool
	}{
		{
			name:         "valid ID",
			input:        "example.com:::A",
			expectedName: "example.com",
			expectedType: "A",
			hasError:     false,
		},
		{
			name:    "invalid ID",
			input:   "invalid",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, typ, err := parseID(tt.input)

			if tt.hasError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedName, name)
				assert.Equal(t, tt.expectedType, typ)
			}
		})
	}
}