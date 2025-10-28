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
				ServerURL:         "http://localhost:8081",
				RecursorServerURL: "http://localhost:8082",
				APIKey:            "secret",
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
				ServerURL:         "http://localhost:8081",
				RecursorServerURL: "http://localhost:8082",
				APIKey:            "secret",
				CACertificate:     "",
				InsecureHTTPS:     false,
				CacheEnable:       false,
				CacheMemorySize:   "10",
				CacheTTL:          60,
			},
			expectError: false,
		},
		{
			name: "config with client cert",
			config: Config{
				ServerURL:         "http://localhost:8081",
				RecursorServerURL: "http://localhost:8082",
				APIKey:            "secret",
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
				ServerURL:         "http://localhost:8081",
				RecursorServerURL: "http://localhost:8082",
				APIKey:            "secret",
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
