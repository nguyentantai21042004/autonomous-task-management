package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"autonomous-task-management/config"
	"autonomous-task-management/config/kafka"
	"autonomous-task-management/config/postgre"
	"autonomous-task-management/config/redis"
	_ "autonomous-task-management/docs" // Swagger docs
	"autonomous-task-management/internal/httpserver"
	"autonomous-task-management/pkg/discord"
	"autonomous-task-management/pkg/encrypter"
	"autonomous-task-management/pkg/jwt"
	"autonomous-task-management/pkg/log"
)

// @title       Golang Boilerplate API
// @description Generic Go service boilerplate. Replace this with your service description.
// @version     1
// @host        localhost:8080
// @schemes     http
//
// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
// @description Bearer token authentication. Format: "Bearer {token}"
func main() {
	// 1. Configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Println("Failed to load config: ", err)
		return
	}

	// 2. Logger
	logger := log.Init(log.ZapConfig{
		Level:        cfg.Logger.Level,
		Mode:         cfg.Logger.Mode,
		Encoding:     cfg.Logger.Encoding,
		ColorEnabled: cfg.Logger.ColorEnabled,
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.Info(ctx, "Starting service...")
	logger.Infof(ctx, "Environment: %s", cfg.Environment.Name)
	logger.Infof(ctx, "Memos URL: %s", cfg.Memos.URL)
	logger.Infof(ctx, "Qdrant URL: %s", cfg.Qdrant.URL)

	// 3. Infrastructure
	encrypterInstance := encrypter.New(cfg.Encrypter.Key)

	postgresDB, err := postgre.Connect(ctx, cfg.Postgres)
	if err != nil {
		logger.Warnf(ctx, "Failed to connect to PostgreSQL (optional): %v", err)
		postgresDB = nil
	} else {
		defer postgre.Disconnect(ctx, postgresDB)
		logger.Info(ctx, "PostgreSQL connected")
	}

	redisClient, err := redis.Connect(ctx, cfg.Redis)
	if err != nil {
		logger.Warnf(ctx, "Failed to connect to Redis (optional): %v", err)
		redisClient = nil
	} else {
		defer redis.Disconnect()
		logger.Info(ctx, "Redis connected")
	}

	// 4. Optional: Kafka producer
	kafkaProducer, err := kafka.ConnectProducer(cfg.Kafka)
	if err != nil {
		logger.Warnf(ctx, "Kafka not configured or unavailable (optional): %v", err)
		kafkaProducer = nil
	} else {
		defer kafka.DisconnectProducer()
		logger.Info(ctx, "Kafka producer connected")
	}
	// Kafka producer is available as kafkaProducer if needed
	_ = kafkaProducer

	// 5. Utilities
	discordClient, err := discord.New(logger, cfg.Discord.WebhookURL)
	if err != nil {
		logger.Warnf(ctx, "Discord webhook not configured (optional): %v", err)
		discordClient = nil
	} else {
		logger.Info(ctx, "Discord client initialized")
	}

	jwtManager, err := jwt.New(jwt.Config{SecretKey: cfg.JWT.SecretKey})
	if err != nil {
		logger.Error(ctx, "Failed to initialize JWT manager: ", err)
		return
	}

	// 6. HTTP Server (wires all domains internally)
	httpServer, err := httpserver.New(logger, httpserver.Config{
		Logger:      logger,
		Port:        cfg.HTTPServer.Port,
		Mode:        cfg.HTTPServer.Mode,
		Environment: cfg.Environment.Name,

		PostgresDB: postgresDB,

		Config:       cfg,
		JWTManager:   jwtManager,
		RedisClient:  redisClient,
		CookieConfig: cfg.Cookie,
		Encrypter:    encrypterInstance,

		Discord: discordClient,
	})
	if err != nil {
		logger.Error(ctx, "Failed to initialize HTTP server: ", err)
		return
	}

	// 7. Run
	if err := httpServer.Run(); err != nil {
		logger.Error(ctx, "Failed to run server: ", err)
		return
	}

	logger.Info(ctx, "Server stopped gracefully")
}
