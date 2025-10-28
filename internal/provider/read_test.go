package provider

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRead(t *testing.T) {
	// Create a temporary file for testing
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test.txt")
	content := "test content"
	err := os.WriteFile(tempFile, []byte(content), 0644)
	require.NoError(t, err)

	tests := []struct {
		name        string
		input       string
		expected    string
		wasPath     bool
		expectError bool
	}{
		{
			name:     "read from file path",
			input:    tempFile,
			expected: content,
			wasPath:  true,
		},
		{
			name:     "read content directly",
			input:    "direct content",
			expected: "direct content",
			wasPath:  false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
			wasPath:  false,
		},
		{
			name:        "non-existent file",
			input:       "/non/existent/file",
			expected:    "/non/existent/file",
			wasPath:     false,
			expectError: false, // Should return input as content
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, wasPath, err := Read(tt.input)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
				assert.Equal(t, tt.wasPath, wasPath)
			}
		})
	}
}
