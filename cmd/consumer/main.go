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
	"autonomous-task-management/pkg/discord"
	"autonomous-task-management/pkg/log"
)

// main is the entry point for the background consumer service.
// This binary consumes messages from Kafka and processes them via UseCases.
//
// Pattern:
//  1. Initialize infra (same as cmd/api/main.go)
//  2. Create UseCases
//  3. Create Kafka consumer groups, wire handlers
//  4. Run & graceful shutdown
func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Println("Failed to load config: ", err)
		return
	}

	logger := log.Init(log.ZapConfig{
		Level:        cfg.Logger.Level,
		Mode:         cfg.Logger.Mode,
		Encoding:     cfg.Logger.Encoding,
		ColorEnabled: cfg.Logger.ColorEnabled,
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.Info(ctx, "Starting consumer service...")

	// Infrastructure
	postgresDB, err := postgre.Connect(ctx, cfg.Postgres)
	if err != nil {
		logger.Error(ctx, "Failed to connect to PostgreSQL: ", err)
		return
	}
	defer postgre.Disconnect(ctx, postgresDB)

	_, err = redis.Connect(ctx, cfg.Redis)
	if err != nil {
		logger.Error(ctx, "Failed to connect to Redis: ", err)
		return
	}
	defer redis.Disconnect()

	// Kafka consumer groups (optional)
	kafkaProducer, err := kafka.ConnectProducer(cfg.Kafka)
	if err != nil {
		logger.Warnf(ctx, "Kafka not available (optional): %v", err)
	} else {
		defer kafka.DisconnectProducer()
	}
	_ = kafkaProducer

	// Optional Discord
	discordClient, err := discord.New(logger, cfg.Discord.WebhookURL)
	if err != nil {
		logger.Warnf(ctx, "Discord not configured (optional): %v", err)
		discordClient = nil
	}
	_ = discordClient

	// TODO: Wire Kafka consumer groups here.
	// Example:
	//   exampleRepo := exampleRepoPostgre.New(postgresDB, logger)
	//   exampleUC   := exampleUseCase.New(exampleRepo, logger)
	//   consumer    := exampleKafkaConsumer.New(cfg.Kafka, exampleUC, logger)
	//   go consumer.Start(ctx)

	logger.Info(ctx, "Consumer service running. Waiting for shutdown signal...")
	<-ctx.Done()
	logger.Info(ctx, "Consumer service stopped gracefully")
}
