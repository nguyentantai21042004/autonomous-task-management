package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"autonomous-task-management/config"
	_ "autonomous-task-management/docs" // Swagger docs
	"autonomous-task-management/internal/httpserver"
	"autonomous-task-management/internal/task/repository"
	memosRepo "autonomous-task-management/internal/task/repository/memos"
	qdrantRepo "autonomous-task-management/internal/task/repository/qdrant"
	"autonomous-task-management/pkg/datemath"
	"autonomous-task-management/pkg/gcalendar"
	"autonomous-task-management/pkg/llmprovider"
	"autonomous-task-management/pkg/log"
	pkgQdrant "autonomous-task-management/pkg/qdrant"
	"autonomous-task-management/pkg/telegram"
	"autonomous-task-management/pkg/voyage"
)

// @title       Autonomous Task Management API
// @description AI-powered task management with Telegram, Gemini LLM, Memos, and Google Calendar.
// @version     1
// @host        localhost:8080
// @schemes     http
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

	logger.Info(ctx, "Starting Autonomous Task Management...")
	logger.Infof(ctx, "Environment: %s", cfg.Environment.Name)
	logger.Infof(ctx, "Memos URL: %s", cfg.Memos.URL)

	// Initialize LLM Provider Manager
	providers, err := llmprovider.InitializeProviders(&cfg.LLM)
	if err != nil {
		logger.Fatalf(ctx, "Failed to initialize LLM providers: %v", err)
	}

	// Parse retry delay
	retryDelay, parseErr := time.ParseDuration(cfg.LLM.RetryDelay)
	if parseErr != nil {
		logger.Warnf(ctx, "Invalid retry delay %q, using default 1s: %v", cfg.LLM.RetryDelay, parseErr)
		retryDelay = time.Second
	}

	// Parse max total timeout
	maxTotalTimeout, parseErr := time.ParseDuration(cfg.LLM.MaxTotalTimeout)
	if parseErr != nil {
		logger.Warnf(ctx, "Invalid max_total_timeout %q, using default 60s: %v", cfg.LLM.MaxTotalTimeout, parseErr)
		maxTotalTimeout = 60 * time.Second
	}

	// Create Provider Manager
	managerConfig := &llmprovider.Config{
		FallbackEnabled: cfg.LLM.FallbackEnabled,
		RetryAttempts:   cfg.LLM.RetryAttempts,
		RetryDelay:      retryDelay,
		MaxTotalTimeout: maxTotalTimeout,
	}
	llmManager := llmprovider.NewManager(providers, managerConfig, logger)
	logger.Info(ctx, "LLM Provider Manager initialized",
		"providers", len(providers),
		"fallback_enabled", cfg.LLM.FallbackEnabled,
		"retry_attempts", cfg.LLM.RetryAttempts,
		"max_total_timeout", maxTotalTimeout,
	)

	// Log provider details
	for i, provider := range providers {
		logger.Infof(ctx, "  Provider %d: %s (model: %s)", i+1, provider.Name(), provider.Model())
	}

	// 3. Infrastructure initialization
	// DateMath parser
	timezone := cfg.LLM.Timezone
	if timezone == "" {
		timezone = "Asia/Ho_Chi_Minh"
	}
	dateMathParser, dtErr := datemath.NewParser(timezone)
	if dtErr != nil {
		logger.Warnf(ctx, "Invalid timezone %q, falling back to UTC: %v", timezone, dtErr)
		dateMathParser, _ = datemath.NewParser("UTC")
	}

	// Telegram Bot client
	var telegramBot telegram.IBot
	if cfg.Telegram.BotToken != "" {
		telegramBot = telegram.NewBot(cfg.Telegram.BotToken)
	}

	// Memos repository
	memosClient := memosRepo.NewClient(cfg.Memos.URL, cfg.Memos.AccessToken)
	taskRepo := memosRepo.New(memosClient, cfg.Memos.ExternalURL, logger)

	// Google Calendar client (optional)
	var calendarClient gcalendar.IGCalendar
	if cfg.GoogleCalendar.CredentialsPath != "" {
		calendarClient, err = gcalendar.NewClientFromCredentialsFile(ctx, cfg.GoogleCalendar.CredentialsPath)
		if err != nil {
			logger.Warnf(ctx, "Google Calendar not available: %v", err)
		}
	}

	// Qdrant Vector repository (optional)
	var vectorRepo repository.VectorRepository
	if cfg.Qdrant.URL != "" {
		qdrantClient := pkgQdrant.NewClient(cfg.Qdrant.URL)
		embeddingClient, voyageErr := voyage.New(cfg.Voyage.APIKey)
		if voyageErr == nil && embeddingClient != nil {
			vectorRepo = qdrantRepo.New(qdrantClient, embeddingClient, cfg.Qdrant.CollectionName, logger)
		}
	}

	// 4. HTTP Server
	httpServer, err := httpserver.New(logger, httpserver.Config{
		Logger:         logger,
		Config:         cfg,
		Port:           cfg.HTTPServer.Port,
		Mode:           cfg.HTTPServer.Mode,
		Environment:    cfg.Environment.Name,
		LLMManager:     llmManager,
		MemosRepo:      taskRepo,
		VectorRepo:     vectorRepo,
		CalendarClient: calendarClient,
		TelegramBot:    telegramBot,
		DateMathParser: dateMathParser,
	})
	if err != nil {
		logger.Error(ctx, "Failed to initialize HTTP server: ", err)
		return
	}

	// 5. Run
	if err := httpServer.Run(); err != nil {
		logger.Error(ctx, "Failed to run server: ", err)
		return
	}

	logger.Info(ctx, "Server stopped gracefully")
}
