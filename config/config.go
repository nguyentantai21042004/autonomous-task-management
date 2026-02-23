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

	// Databases
	Postgres PostgresConfig
	Redis    RedisConfig

	// Event Streaming (optional)
	Kafka KafkaConfig

	// Auth & Security
	JWT            JWTConfig
	Cookie         CookieConfig
	Encrypter      EncrypterConfig
	InternalConfig InternalConfig

	// Monitoring
	Discord DiscordConfig

	// Autonomous Task Management specifics
	Memos          MemosConfig
	Qdrant         QdrantConfig
	Telegram       TelegramConfig
	GoogleCalendar GoogleCalendarConfig
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

type PostgresConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
	Schema   string
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

type KafkaConfig struct {
	Brokers []string
	Topic   string
	GroupID string
}

type JWTConfig struct {
	SecretKey string
}

type CookieConfig struct {
	Domain         string
	Secure         bool
	SameSite       string
	MaxAge         int
	MaxAgeRemember int
	Name           string
}

type EncrypterConfig struct {
	Key string
}

type InternalConfig struct {
	InternalKey string
}

type DiscordConfig struct {
	WebhookURL string
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

	// PostgreSQL
	cfg.Postgres.Host = viper.GetString("postgres.host")
	cfg.Postgres.Port = viper.GetInt("postgres.port")
	cfg.Postgres.User = viper.GetString("postgres.user")
	cfg.Postgres.Password = viper.GetString("postgres.password")
	cfg.Postgres.DBName = viper.GetString("postgres.dbname")
	cfg.Postgres.SSLMode = viper.GetString("postgres.sslmode")
	cfg.Postgres.Schema = viper.GetString("postgres.schema")

	// Redis
	cfg.Redis.Host = viper.GetString("redis.host")
	cfg.Redis.Port = viper.GetInt("redis.port")
	cfg.Redis.Password = viper.GetString("redis.password")
	cfg.Redis.DB = viper.GetInt("redis.db")

	// Kafka (optional)
	cfg.Kafka.Brokers = viper.GetStringSlice("kafka.brokers")
	cfg.Kafka.Topic = viper.GetString("kafka.topic")
	cfg.Kafka.GroupID = viper.GetString("kafka.group_id")

	// Auth & Security
	cfg.JWT.SecretKey = viper.GetString("jwt.secret_key")
	cfg.Cookie.Domain = viper.GetString("cookie.domain")
	cfg.Cookie.Secure = viper.GetBool("cookie.secure")
	cfg.Cookie.SameSite = viper.GetString("cookie.samesite")
	cfg.Cookie.MaxAge = viper.GetInt("cookie.max_age")
	cfg.Cookie.MaxAgeRemember = viper.GetInt("cookie.max_age_remember")
	cfg.Cookie.Name = viper.GetString("cookie.name")
	cfg.Encrypter.Key = viper.GetString("encrypter.key")
	cfg.InternalConfig.InternalKey = viper.GetString("internal.internal_key")

	// Monitoring
	// Monitoring
	cfg.Discord.WebhookURL = viper.GetString("discord.webhook_url")

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
	if googleCreds := viper.GetString("google_service_account_json"); googleCreds != "" {
		cfg.GoogleCalendar.CredentialsPath = googleCreds
	}

	if err := validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func setDefaults() {
	viper.SetDefault("environment.name", "production")
	viper.SetDefault("http_server.port", 8080)
	viper.SetDefault("http_server.mode", "debug")
	viper.SetDefault("logger.level", "debug")
	viper.SetDefault("logger.mode", "debug")
	viper.SetDefault("logger.encoding", "console")
	viper.SetDefault("logger.color_enabled", true)

	viper.SetDefault("postgres.host", "localhost")
	viper.SetDefault("postgres.port", 5432)
	viper.SetDefault("postgres.user", "postgres")
	viper.SetDefault("postgres.password", "postgres")
	viper.SetDefault("postgres.dbname", "postgres")
	viper.SetDefault("postgres.sslmode", "prefer")
	viper.SetDefault("postgres.schema", "public")

	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)

	viper.SetDefault("kafka.brokers", []string{"localhost:9092"})
	viper.SetDefault("kafka.topic", "app.events")

	viper.SetDefault("cookie.domain", "localhost")
	viper.SetDefault("cookie.secure", false)
	viper.SetDefault("cookie.samesite", "Lax")
	viper.SetDefault("cookie.max_age", 28800)
	viper.SetDefault("cookie.max_age_remember", 604800)
	viper.SetDefault("cookie.name", "auth_token")
}

func validate(cfg *Config) error {
	if cfg.JWT.SecretKey == "" {
		return fmt.Errorf("jwt.secret_key is required")
	}
	if len(cfg.JWT.SecretKey) < 32 {
		return fmt.Errorf("jwt.secret_key must be at least 32 characters")
	}
	if cfg.Encrypter.Key == "" {
		return fmt.Errorf("encrypter.key is required")
	}
	if len(cfg.Encrypter.Key) < 32 {
		return fmt.Errorf("encrypter.key must be at least 32 characters")
	}
	if cfg.Postgres.Host == "" {
		return fmt.Errorf("postgres.host is required")
	}
	if cfg.Postgres.Port == 0 {
		return fmt.Errorf("postgres.port is required")
	}
	if cfg.Postgres.DBName == "" {
		return fmt.Errorf("postgres.dbname is required")
	}
	if cfg.Redis.Host == "" {
		return fmt.Errorf("redis.host is required")
	}
	if cfg.Redis.Port == 0 {
		return fmt.Errorf("redis.port is required")
	}
	if cfg.Cookie.Name == "" {
		return fmt.Errorf("cookie.name is required")
	}
	return nil
}
