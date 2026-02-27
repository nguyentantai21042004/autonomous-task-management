package llmprovider_test

import (
	"testing"
	"time"

	"autonomous-task-management/config"
	"autonomous-task-management/pkg/llmprovider"
	"autonomous-task-management/pkg/log"
)

// TestIntegration_ConfigToManagerFlow verifies that configuration loading,
// provider initialization, and manager work together correctly
func TestIntegration_ConfigToManagerFlow(t *testing.T) {
	// Step 1: Create a configuration (simulating config loading)
	cfg := &config.LLMConfig{
		Providers: []config.ProviderConfig{
			{
				Name:     "qwen",
				Enabled:  true,
				Priority: 1,
				APIKey:   "test-qwen-key",
				Model:    "qwen-plus",
				Timeout:  "30s",
			},
			{
				Name:     "gemini",
				Enabled:  true,
				Priority: 2,
				APIKey:   "test-gemini-key",
				Model:    "gemini-2.5-flash",
				Timeout:  "30s",
			},
		},
		FallbackEnabled: true,
		RetryAttempts:   3,
		RetryDelay:      "1s",
	}

	// Step 2: Initialize providers from configuration
	providers, err := llmprovider.InitializeProviders(cfg)
	if err != nil {
		t.Fatalf("Failed to initialize providers: %v", err)
	}

	// Verify providers are initialized correctly
	if len(providers) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(providers))
	}

	// Verify provider order (by priority)
	if providers[0].Name() != "qwen" {
		t.Errorf("Expected first provider to be qwen, got %s", providers[0].Name())
	}
	if providers[1].Name() != "gemini" {
		t.Errorf("Expected second provider to be gemini, got %s", providers[1].Name())
	}

	// Step 3: Create manager with providers
	retryDelay, _ := time.ParseDuration(cfg.RetryDelay)
	managerConfig := &llmprovider.Config{
		FallbackEnabled: cfg.FallbackEnabled,
		RetryAttempts:   cfg.RetryAttempts,
		RetryDelay:      retryDelay,
	}

	logger := log.Init(log.ZapConfig{
		Level:        "info",
		Mode:         "development",
		Encoding:     "console",
		ColorEnabled: false,
	})
	manager := llmprovider.NewManager(providers, managerConfig, logger)

	// Step 4: Verify manager is created successfully
	if manager == nil {
		t.Fatal("Manager should not be nil")
	}

	// Note: We don't actually call GenerateContent here because it would
	// require real API keys and make real API calls. The unit tests for
	// manager already verify the behavior with mock providers.
	t.Log("Integration test passed: Config -> Factory -> Manager flow works correctly")
}

// TestIntegration_ConfigValidation verifies that invalid configurations
// are caught during initialization
func TestIntegration_ConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.LLMConfig
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &config.LLMConfig{
				Providers: []config.ProviderConfig{
					{
						Name:     "qwen",
						Enabled:  true,
						Priority: 1,
						APIKey:   "test-key",
						Model:    "qwen-plus",
					},
				},
				FallbackEnabled: true,
				RetryAttempts:   3,
				RetryDelay:      "1s",
			},
			wantErr: false,
		},
		{
			name: "no providers",
			cfg: &config.LLMConfig{
				Providers:       []config.ProviderConfig{},
				FallbackEnabled: true,
				RetryAttempts:   3,
				RetryDelay:      "1s",
			},
			wantErr: true,
		},
		{
			name: "all providers disabled",
			cfg: &config.LLMConfig{
				Providers: []config.ProviderConfig{
					{
						Name:     "qwen",
						Enabled:  false,
						Priority: 1,
						APIKey:   "test-key",
						Model:    "qwen-plus",
					},
				},
				FallbackEnabled: true,
				RetryAttempts:   3,
				RetryDelay:      "1s",
			},
			wantErr: true,
		},
		{
			name: "missing API key",
			cfg: &config.LLMConfig{
				Providers: []config.ProviderConfig{
					{
						Name:     "qwen",
						Enabled:  true,
						Priority: 1,
						APIKey:   "", // Missing
						Model:    "qwen-plus",
					},
				},
				FallbackEnabled: true,
				RetryAttempts:   3,
				RetryDelay:      "1s",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := llmprovider.InitializeProviders(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("InitializeProviders() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestIntegration_ProviderPriorityOrdering verifies that providers
// are ordered correctly by priority
func TestIntegration_ProviderPriorityOrdering(t *testing.T) {
	cfg := &config.LLMConfig{
		Providers: []config.ProviderConfig{
			{
				Name:     "gemini",
				Enabled:  true,
				Priority: 10, // Higher priority number
				APIKey:   "test-gemini-key",
				Model:    "gemini-2.5-flash",
			},
			{
				Name:     "qwen",
				Enabled:  true,
				Priority: 1, // Lower priority number (should come first)
				APIKey:   "test-qwen-key",
				Model:    "qwen-plus",
			},
		},
		FallbackEnabled: true,
		RetryAttempts:   3,
		RetryDelay:      "1s",
	}

	providers, err := llmprovider.InitializeProviders(cfg)
	if err != nil {
		t.Fatalf("Failed to initialize providers: %v", err)
	}

	// Verify ascending priority order
	if providers[0].Name() != "qwen" {
		t.Errorf("Expected first provider (priority 1) to be qwen, got %s", providers[0].Name())
	}
	if providers[1].Name() != "gemini" {
		t.Errorf("Expected second provider (priority 10) to be gemini, got %s", providers[1].Name())
	}
}
