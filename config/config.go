package config

import (
	"fmt"
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
	Gemini         GeminiConfig
	Voyage         VoyageConfig

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

type GeminiConfig struct {
	APIKey   string
	Timezone string // IANA timezone, e.g. "Asia/Ho_Chi_Minh"
}

type VoyageConfig struct {
	APIKey string
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
	if memosURL := viper.GetString("memos_url"); memosURL != "" {
		cfg.Memos.URL = memosURL
	}
	if memosToken := viper.GetString("memos_access_token"); memosToken != "" {
		cfg.Memos.AccessToken = memosToken
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

	// Gemini LLM
	cfg.Gemini.APIKey = viper.GetString("gemini.api_key")
	cfg.Gemini.Timezone = viper.GetString("gemini.timezone")
	if apiKey := viper.GetString("gemini_api_key"); apiKey != "" {
		cfg.Gemini.APIKey = apiKey
	}
	if tz := viper.GetString("gemini_timezone"); tz != "" {
		cfg.Gemini.Timezone = tz
	}

	// Voyage AI
	cfg.Voyage.APIKey = viper.GetString("voyage.api_key")
	if voyageKey := viper.GetString("voyage_api_key"); voyageKey != "" {
		cfg.Voyage.APIKey = voyageKey
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
	viper.SetDefault("gemini.timezone", "Asia/Ho_Chi_Minh")
	viper.SetDefault("qdrant.collection_name", "tasks")
	viper.SetDefault("qdrant.vector_size", 1024)
	viper.SetDefault("webhook.rate_limit_per_min", 60)
	viper.SetDefault("webhook.enabled", true)
}
