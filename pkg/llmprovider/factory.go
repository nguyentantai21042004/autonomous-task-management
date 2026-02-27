package llmprovider

import (
	"fmt"
	"sort"
	"strings"

	"autonomous-task-management/config"
	"autonomous-task-management/pkg/deepseek"
	"autonomous-task-management/pkg/gemini"
	"autonomous-task-management/pkg/qwen"
)

// InitializeProviders creates Provider instances from config.LLMConfig
// Returns providers sorted by priority (ascending) with disabled providers filtered out
// Skips providers that fail to initialize instead of failing the entire service
func InitializeProviders(cfg *config.LLMConfig) ([]Provider, error) {
	if cfg == nil {
		return nil, fmt.Errorf("LLM config is nil")
	}

	if len(cfg.Providers) == 0 {
		return nil, ErrNoProvidersConfigured
	}

	// Filter enabled providers
	var enabledProviders []config.ProviderConfig
	for _, p := range cfg.Providers {
		if p.Enabled {
			enabledProviders = append(enabledProviders, p)
		}
	}

	if len(enabledProviders) == 0 {
		return nil, ErrNoProvidersConfigured
	}

	// Sort by priority (ascending order)
	sort.Slice(enabledProviders, func(i, j int) bool {
		return enabledProviders[i].Priority < enabledProviders[j].Priority
	})

	// Build provider instances - skip failed ones instead of failing entirely
	var providers []Provider
	var initErrors []string

	for _, p := range enabledProviders {
		provider, err := createProvider(p)
		if err != nil {
			// Log error but continue with other providers
			errMsg := fmt.Sprintf("failed to initialize provider %s (priority %d): %v", p.Name, p.Priority, err)
			initErrors = append(initErrors, errMsg)
			fmt.Printf("Warning: %s\n", errMsg)
			continue
		}
		providers = append(providers, provider)
	}

	// If no providers were successfully initialized, return error
	if len(providers) == 0 {
		return nil, fmt.Errorf("no providers successfully initialized: %s", strings.Join(initErrors, "; "))
	}

	// If some providers failed, log warning but continue
	if len(initErrors) > 0 {
		fmt.Printf("Warning: %d provider(s) failed to initialize but continuing with %d working provider(s)\n",
			len(initErrors), len(providers))
	}

	return providers, nil
}

// createProvider creates a concrete provider instance based on the provider config
func createProvider(cfg config.ProviderConfig) (Provider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("provider %s: API key is required", cfg.Name)
	}
	if cfg.Model == "" {
		return nil, fmt.Errorf("provider %s: model is required", cfg.Name)
	}

	switch cfg.Name {
	case "deepseek":
		client, err := deepseek.New(deepseek.Config{
			APIKey:  cfg.APIKey,
			Model:   cfg.Model,
			BaseURL: cfg.BaseURL,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create deepseek client: %w", err)
		}
		return NewDeepSeekAdapter(client), nil

	case "qwen", "alibaba":
		client, err := qwen.New(qwen.Config{
			APIKey: cfg.APIKey,
			Model:  cfg.Model,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create qwen client: %w", err)
		}
		return NewQwenAdapter(client), nil

	case "gemini":
		client, err := gemini.New(gemini.Config{
			APIKey: cfg.APIKey,
			Model:  cfg.Model,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create gemini client: %w", err)
		}
		return NewGeminiAdapter(client), nil

	default:
		return nil, fmt.Errorf("unknown provider: %s", cfg.Name)
	}
}
