package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccProvider_Configure(t *testing.T) {
	// Set up environment variables for testing
	os.Setenv("PDNS_API_KEY", "test-api-key")
	os.Setenv("PDNS_SERVER_URL", "https://test.example.com")
	os.Setenv("PDNS_RECURSOR_SERVER_URL", "https://recursor.test.example.com")
	defer func() {
		os.Unsetenv("PDNS_API_KEY")
		os.Unsetenv("PDNS_SERVER_URL")
		os.Unsetenv("PDNS_RECURSOR_SERVER_URL")
	}()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: map[string]func() (tfprotov6.ProviderServer, error){
			"powerdns": func() (tfprotov6.ProviderServer, error) {
				return providerserver.NewProtocol6(New("test")())(), nil
			},
		},
		Steps: []resource.TestStep{
			{
				Config: `provider "powerdns" {}`,
				Check:  resource.ComposeTestCheckFunc(
				// Add checks here if needed
				),
			},
		},
	})
}

func TestGetConfigValueWithEnvFallback(t *testing.T) {
	// Set up environment variable
	os.Setenv("TEST_ENV_VAR", "env-value")
	defer os.Unsetenv("TEST_ENV_VAR")

	tests := []struct {
		name        string
		configValue string
		envVar      string
		expected    string
	}{
		{
			name:        "config value provided",
			configValue: "config-value",
			envVar:      "TEST_ENV_VAR",
			expected:    "config-value",
		},
		{
			name:        "fallback to env var",
			configValue: "",
			envVar:      "TEST_ENV_VAR",
			expected:    "env-value",
		},
		{
			name:        "no env var set",
			configValue: "",
			envVar:      "NON_EXISTENT_VAR",
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getConfigValueWithEnvFallback(tt.configValue, tt.envVar)
			if result != tt.expected {
				t.Errorf("getConfigValueWithEnvFallback() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetConfigBoolWithEnvFallback(t *testing.T) {
	// Set up environment variables
	os.Setenv("TEST_BOOL_TRUE", "true")
	os.Setenv("TEST_BOOL_FALSE", "false")
	os.Setenv("TEST_BOOL_INVALID", "invalid")
	defer func() {
		os.Unsetenv("TEST_BOOL_TRUE")
		os.Unsetenv("TEST_BOOL_FALSE")
		os.Unsetenv("TEST_BOOL_INVALID")
	}()

	tests := []struct {
		name        string
		configValue bool
		isNull      bool
		isUnknown   bool
		envVar      string
		expected    bool
	}{
		{
			name:        "config value provided",
			configValue: true,
			isNull:      false,
			isUnknown:   false,
			envVar:      "TEST_BOOL_TRUE",
			expected:    true,
		},
		{
			name:        "fallback to env var true",
			configValue: false,
			isNull:      true,
			isUnknown:   false,
			envVar:      "TEST_BOOL_TRUE",
			expected:    true,
		},
		{
			name:        "fallback to env var false",
			configValue: true,
			isNull:      true,
			isUnknown:   false,
			envVar:      "TEST_BOOL_FALSE",
			expected:    false,
		},
		{
			name:        "invalid env var",
			configValue: true,
			isNull:      true,
			isUnknown:   false,
			envVar:      "TEST_BOOL_INVALID",
			expected:    true, // Should return config value on parse error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getConfigBoolWithEnvFallback(tt.configValue, tt.isNull, tt.isUnknown, tt.envVar)
			if result != tt.expected {
				t.Errorf("getConfigBoolWithEnvFallback() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetConfigIntWithEnvFallback(t *testing.T) {
	// Set up environment variables
	os.Setenv("TEST_INT_42", "42")
	os.Setenv("TEST_INT_INVALID", "invalid")
	defer func() {
		os.Unsetenv("TEST_INT_42")
		os.Unsetenv("TEST_INT_INVALID")
	}()

	tests := []struct {
		name        string
		configValue int
		isNull      bool
		isUnknown   bool
		envVar      string
		expected    int
	}{
		{
			name:        "config value provided",
			configValue: 10,
			isNull:      false,
			isUnknown:   false,
			envVar:      "TEST_INT_42",
			expected:    10,
		},
		{
			name:        "fallback to env var",
			configValue: 0,
			isNull:      true,
			isUnknown:   false,
			envVar:      "TEST_INT_42",
			expected:    42,
		},
		{
			name:        "invalid env var",
			configValue: 10,
			isNull:      true,
			isUnknown:   false,
			envVar:      "TEST_INT_INVALID",
			expected:    10, // Should return config value on parse error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getConfigIntWithEnvFallback(tt.configValue, tt.isNull, tt.isUnknown, tt.envVar)
			if result != tt.expected {
				t.Errorf("getConfigIntWithEnvFallback() = %v, want %v", result, tt.expected)
			}
		})
	}
}
