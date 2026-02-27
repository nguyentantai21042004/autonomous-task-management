package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all service configuration.
type Config struct {
	// Environment
	Environment EnvironmentConfig

	// Server
	HTTPServer HTTPServerConfig
	Logger     LoggerConfig

	// Autonomous Task Management specifics
	Memos          MemosConfig
	Qdrant         QdrantConfig
	Telegram       TelegramConfig
	GoogleCalendar GoogleCalendarConfig
	Voyage         VoyageConfig

	// LLM Provider Abstraction
	LLM LLMConfig

	// Webhooks
	Webhook WebhookConfig
}

type EnvironmentConfig struct {
	Name string
}

type HTTPServerConfig struct {
	Port int
	Mode string
}

type LoggerConfig struct {
	Level        string
	Mode         string
	Encoding     string
	ColorEnabled bool
}

type MemosConfig struct {
	URL         string
	APIVersion  string
	AccessToken string
	ExternalURL string // URL for generating user-facing links (e.g., http://localhost:5230)
}

type QdrantConfig struct {
	URL            string
	CollectionName string
	VectorSize     int
}

type TelegramConfig struct {
	BotToken   string
	WebhookURL string
}

type GoogleCalendarConfig struct {
	CredentialsPath string
	CalendarID      string
}

type VoyageConfig struct {
	APIKey string
}

// LLMConfig holds configuration for the LLM provider abstraction layer
type LLMConfig struct {
	Providers       []ProviderConfig `yaml:"providers"`
	FallbackEnabled bool             `yaml:"fallback_enabled"`
	RetryAttempts   int              `yaml:"retry_attempts"`
	RetryDelay      string           `yaml:"retry_delay"`
	MaxTotalTimeout string           `yaml:"max_total_timeout"` // NEW: Global timeout for entire fallback chain
}

// ProviderConfig holds configuration for a single LLM provider
type ProviderConfig struct {
	Name     string `yaml:"name"`
	Enabled  bool   `yaml:"enabled"`
	Priority int    `yaml:"priority"`
	APIKey   string `yaml:"api_key"`
	BaseURL  string `yaml:"base_url,omitempty"`
	Model    string `yaml:"model"`
	Timeout  string `yaml:"timeout"`
}

type WebhookConfig struct {
	Enabled         bool
	Secret          string
	AllowedIPs      []string
	RateLimitPerMin int
}

// Load loads configuration using Viper.
// Config file name: config.yaml — searched in ./config, ., /etc/app/
func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/app/")

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	cfg := &Config{}

	// Environment & Server
	cfg.Environment.Name = viper.GetString("environment.name")
	cfg.HTTPServer.Port = viper.GetInt("http_server.port")
	cfg.HTTPServer.Mode = viper.GetString("http_server.mode")
	cfg.Logger.Level = viper.GetString("logger.level")
	cfg.Logger.Mode = viper.GetString("logger.mode")
	cfg.Logger.Encoding = viper.GetString("logger.encoding")
	cfg.Logger.ColorEnabled = viper.GetBool("logger.color_enabled")

	// Autonomous Task Management specifics
	cfg.Memos.URL = viper.GetString("memos.url")
	cfg.Memos.APIVersion = viper.GetString("memos.api_version")
	cfg.Memos.AccessToken = viper.GetString("memos.access_token")
	cfg.Memos.ExternalURL = viper.GetString("memos.external_url")
	if memosURL := viper.GetString("memos_url"); memosURL != "" {
		cfg.Memos.URL = memosURL
	}
	if memosToken := viper.GetString("memos_access_token"); memosToken != "" {
		cfg.Memos.AccessToken = memosToken
	}
	// If external URL not set, default to internal URL
	if cfg.Memos.ExternalURL == "" {
		cfg.Memos.ExternalURL = cfg.Memos.URL
	}

	cfg.Qdrant.URL = viper.GetString("qdrant.url")
	cfg.Qdrant.CollectionName = viper.GetString("qdrant.collection_name")
	cfg.Qdrant.VectorSize = viper.GetInt("qdrant.vector_size")
	if qdrantURL := viper.GetString("qdrant_url"); qdrantURL != "" {
		cfg.Qdrant.URL = qdrantURL
	}

	cfg.Telegram.BotToken = viper.GetString("telegram.bot_token")
	cfg.Telegram.WebhookURL = viper.GetString("telegram.webhook_url")
	if tgToken := viper.GetString("telegram_bot_token"); tgToken != "" {
		cfg.Telegram.BotToken = tgToken
	}

	cfg.GoogleCalendar.CredentialsPath = viper.GetString("google_calendar.credentials_path")
	cfg.GoogleCalendar.CalendarID = viper.GetString("google_calendar.calendar_id")
	if googleCreds := viper.GetString("google_calendar_credentials"); googleCreds != "" {
		cfg.GoogleCalendar.CredentialsPath = googleCreds
	}

	// Voyage AI
	cfg.Voyage.APIKey = viper.GetString("voyage.api_key")
	if voyageKey := viper.GetString("voyage_api_key"); voyageKey != "" {
		cfg.Voyage.APIKey = voyageKey
	}

	// LLM Provider Abstraction
	cfg.LLM.FallbackEnabled = viper.GetBool("llm.fallback_enabled")
	cfg.LLM.RetryAttempts = viper.GetInt("llm.retry_attempts")
	cfg.LLM.RetryDelay = viper.GetString("llm.retry_delay")
	cfg.LLM.MaxTotalTimeout = viper.GetString("llm.max_total_timeout")

	// Load provider configurations
	if viper.IsSet("llm.providers") {
		providersRaw := viper.Get("llm.providers")
		if providersList, ok := providersRaw.([]interface{}); ok {
			for _, p := range providersList {
				if providerMap, ok := p.(map[string]interface{}); ok {
					provider := ProviderConfig{
						Name:     getStringFromMap(providerMap, "name"),
						Enabled:  getBoolFromMap(providerMap, "enabled"),
						Priority: getIntFromMap(providerMap, "priority"),
						APIKey:   expandEnvVar(getStringFromMap(providerMap, "api_key")),
						BaseURL:  getStringFromMap(providerMap, "base_url"),
						Model:    getStringFromMap(providerMap, "model"),
						Timeout:  getStringFromMap(providerMap, "timeout"),
					}
					cfg.LLM.Providers = append(cfg.LLM.Providers, provider)
				}
			}
		}
	}

	// Validate LLM config
	if len(cfg.LLM.Providers) == 0 {
		return nil, fmt.Errorf("no LLM providers configured - please add llm.providers section to config.yaml")
	}

	// Webhooks
	cfg.Webhook.Enabled = viper.GetBool("webhook.enabled")
	cfg.Webhook.Secret = viper.GetString("webhook.secret")
	if webhookSecret := viper.GetString("webhook_secret"); webhookSecret != "" {
		cfg.Webhook.Secret = webhookSecret
	}
	cfg.Webhook.RateLimitPerMin = viper.GetInt("webhook.rate_limit_per_min")

	// Split allowed IPs since viper might not parse array seamlessly from env
	var ips []string
	if rawIps := viper.GetString("webhook.allowed_ips"); rawIps != "" {
		for _, ip := range strings.Split(rawIps, ",") {
			ip = strings.TrimSpace(ip)
			if ip != "" {
				ips = append(ips, ip)
			}
		}
	}
	cfg.Webhook.AllowedIPs = ips

	return cfg, nil
}

func setDefaults() {
	viper.SetDefault("environment.name", "development")
	viper.SetDefault("http_server.port", 8080)
	viper.SetDefault("http_server.mode", "debug")
	viper.SetDefault("logger.level", "debug")
	viper.SetDefault("logger.mode", "debug")
	viper.SetDefault("logger.encoding", "console")
	viper.SetDefault("logger.color_enabled", true)
	viper.SetDefault("qdrant.collection_name", "tasks")
	viper.SetDefault("qdrant.vector_size", 1024)
	viper.SetDefault("webhook.rate_limit_per_min", 60)
	viper.SetDefault("webhook.enabled", true)

	// LLM defaults
	viper.SetDefault("llm.fallback_enabled", true)
	viper.SetDefault("llm.retry_attempts", 3)
	viper.SetDefault("llm.retry_delay", "1s")
	viper.SetDefault("llm.max_total_timeout", "60s") // Default: 60 seconds for entire fallback chain
}

// expandEnvVar expands environment variables in the format ${VAR_NAME}
func expandEnvVar(value string) string {
	if value == "" {
		return value
	}

	// Check if value is in format ${VAR_NAME}
	if strings.HasPrefix(value, "${") && strings.HasSuffix(value, "}") {
		envVar := value[2 : len(value)-1]
		// Try viper first (handles both env and config)
		if envValue := viper.GetString(envVar); envValue != "" {
			return envValue
		}
		// Try lowercase version
		if envValue := viper.GetString(strings.ToLower(envVar)); envValue != "" {
			return envValue
		}
		// Try direct os.Getenv as last resort
		if envValue := os.Getenv(envVar); envValue != "" {
			return envValue
		}
	}

	return value
}

// validateLLMConfig validates the LLM configuration
func validateLLMConfig(cfg *LLMConfig) error {
	if len(cfg.Providers) == 0 {
		return fmt.Errorf("no LLM providers configured")
	}

	enabledCount := 0
	priorityMap := make(map[int]bool)

	for i, provider := range cfg.Providers {
		// Check required fields
		if provider.Name == "" {
			return fmt.Errorf("provider %d: name is required", i)
		}
		if provider.Model == "" {
			return fmt.Errorf("provider %s: model is required", provider.Name)
		}

		if provider.Enabled {
			enabledCount++

			// Check priority is valid
			if provider.Priority <= 0 {
				return fmt.Errorf("provider %s: priority must be positive", provider.Name)
			}

			// Check for duplicate priorities
			if priorityMap[provider.Priority] {
				return fmt.Errorf("provider %s: duplicate priority %d", provider.Name, provider.Priority)
			}
			priorityMap[provider.Priority] = true

			// Check API key is set (warning only)
			if provider.APIKey == "" {
				fmt.Printf("Warning: provider %s has no API key configured\n", provider.Name)
			}
		}
	}

	if enabledCount == 0 {
		return fmt.Errorf("no enabled LLM providers")
	}

	return nil
}

// Helper functions to safely extract values from map[string]interface{}
func getStringFromMap(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getBoolFromMap(m map[string]interface{}, key string) bool {
	if val, ok := m[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

func getIntFromMap(m map[string]interface{}, key string) int {
	if val, ok := m[key]; ok {
		if i, ok := val.(int); ok {
			return i
		}
		// Handle float64 from JSON unmarshaling
		if f, ok := val.(float64); ok {
			return int(f)
		}
	}
	return 0
}
