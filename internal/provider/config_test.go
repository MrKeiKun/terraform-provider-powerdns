package provider

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_Client(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		config      Config
		expectError bool
	}{
		{
			name: "valid config",
			config: Config{
				ServerURL:         "https://example.com",
				RecursorServerURL: "https://recursor.example.com",
				APIKey:            "test-key",
				InsecureHTTPS:     false,
				CacheEnable:       false,
				CacheMemorySize:   "10",
				CacheTTL:          60,
			},
			expectError: false,
		},
		{
			name: "config with CA certificate",
			config: Config{
				ServerURL:         "https://example.com",
				RecursorServerURL: "https://recursor.example.com",
				APIKey:            "test-key",
				CACertificate:     "test-ca-cert",
				InsecureHTTPS:     false,
				CacheEnable:       false,
				CacheMemorySize:   "10",
				CacheTTL:          60,
			},
			expectError: true, // Will fail because Read function is not mocked
		},
		{
			name: "config with client cert",
			config: Config{
				ServerURL:         "https://example.com",
				RecursorServerURL: "https://recursor.example.com",
				APIKey:            "test-key",
				ClientCertFile:    "test-cert.pem",
				ClientCertKeyFile: "test-key.pem",
				InsecureHTTPS:     false,
				CacheEnable:       false,
				CacheMemorySize:   "10",
				CacheTTL:          60,
			},
			expectError: true, // Will fail because file loading is not mocked
		},
		{
			name: "config with insecure HTTPS",
			config: Config{
				ServerURL:         "https://example.com",
				RecursorServerURL: "https://recursor.example.com",
				APIKey:            "test-key",
				InsecureHTTPS:     true,
				CacheEnable:       false,
				CacheMemorySize:   "10",
				CacheTTL:          60,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := tt.config.Client(ctx)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, client)
				assert.Equal(t, tt.config.ServerURL, client.ServerURL)
				assert.Equal(t, tt.config.RecursorServerURL, client.RecursorServerURL)
				assert.Equal(t, tt.config.APIKey, client.APIKey)
			}
		})
	}
}
