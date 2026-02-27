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
	"autonomous-task-management/internal/agent"
	"autonomous-task-management/internal/agent/orchestrator"
	"autonomous-task-management/internal/agent/tools"
	"autonomous-task-management/internal/automation"
	"autonomous-task-management/internal/checklist"
	"autonomous-task-management/internal/httpserver"
	"autonomous-task-management/internal/router"
	"autonomous-task-management/internal/sync"
	tgDelivery "autonomous-task-management/internal/task/delivery/telegram"
	"autonomous-task-management/internal/task/repository"
	memosRepo "autonomous-task-management/internal/task/repository/memos"
	qdrantRepo "autonomous-task-management/internal/task/repository/qdrant"
	"autonomous-task-management/internal/task/usecase"
	"autonomous-task-management/internal/test"
	"autonomous-task-management/internal/webhook"
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

	// 3. Phase 2 & 3: Domain initialization
	var telegramHandler tgDelivery.Handler
	var webhookHandler sync.Handler
	var gitWebhookHandler *webhook.Handler
	var testHandler test.Handler

	if cfg.Telegram.BotToken != "" && cfg.Memos.AccessToken != "" {
		logger.Info(ctx, "Initializing Phase 2 components...")

		// Telegram Bot client
		telegramBot := telegram.NewBot(cfg.Telegram.BotToken)

		// DateMath parser - get timezone from LLM config or default
		timezone := "Asia/Ho_Chi_Minh"
		dateMathParser, dtErr := datemath.NewParser(timezone)
		if dtErr != nil {
			logger.Warnf(ctx, "Invalid timezone %q, falling back to UTC: %v", timezone, dtErr)
			dateMathParser, _ = datemath.NewParser("UTC")
		}

		// Memos repository
		memosClient := memosRepo.NewClient(cfg.Memos.URL, cfg.Memos.AccessToken)
		taskRepo := memosRepo.New(memosClient, cfg.Memos.ExternalURL, logger)

		// Google Calendar client (optional)
		var calendarClient *gcalendar.Client
		if cfg.GoogleCalendar.CredentialsPath != "" {
			calendarClient, err = gcalendar.NewClientFromCredentialsFile(ctx, cfg.GoogleCalendar.CredentialsPath)
			if err != nil {
				logger.Warnf(ctx, "Google Calendar not available (optional): %v", err)
				logger.Warn(ctx, "→ Run `go run scripts/gcal-auth/main.go` to generate token.json")
			} else {
				logger.Info(ctx, "Google Calendar initialized")
			}
		}

		// Qdrant Vector repository (optional, but recommended for Phase 3)
		// Let's use the interface type properly
		var vectorRepoInterface repository.VectorRepository
		if cfg.Qdrant.URL != "" {
			logger.Info(ctx, "Initializing Qdrant...")

			qdrantClient := pkgQdrant.NewClient(cfg.Qdrant.URL)

			// Initialize Voyage AI embedding client
			embeddingClient, voyageErr := voyage.New(cfg.Voyage.APIKey)
			if voyageErr != nil {
				logger.Warnf(ctx, "Failed to initialize Voyage AI client: %v", voyageErr)
			}

			// Create collection if not exists
			collectionReq := pkgQdrant.CreateCollectionRequest{
				Name: cfg.Qdrant.CollectionName,
				Vectors: pkgQdrant.VectorConfig{
					Size:     cfg.Qdrant.VectorSize,
					Distance: "Cosine",
				},
			}

			if err := qdrantClient.CreateCollection(ctx, collectionReq); err != nil {
				logger.Warnf(ctx, "Qdrant collection creation warning: %v (may already exist)", err)
			} else {
				logger.Infof(ctx, "Qdrant collection %q created", cfg.Qdrant.CollectionName)
			}

			// Initialize Qdrant repository
			if embeddingClient != nil {
				vectorRepoInterface = qdrantRepo.New(qdrantClient, embeddingClient, cfg.Qdrant.CollectionName, logger)
				logger.Info(ctx, "Qdrant initialized successfully")
			} else {
				logger.Warn(ctx, "Qdrant initialization skipped because embedding client failed")
			}
		}

		// Task UseCase
		taskUC := usecase.New(logger, llmManager, calendarClient, taskRepo, vectorRepoInterface, dateMathParser, timezone, cfg.Memos.ExternalURL)

		// Agent Tool Registry & Orchestrator
		toolRegistry := agent.NewToolRegistry()
		toolRegistry.Register(tools.NewSearchTasksTool(taskUC))
		if calendarClient != nil {
			toolRegistry.Register(tools.NewCheckCalendarTool(calendarClient, logger))
		}

		agentOrchestrator := orchestrator.New(llmManager, toolRegistry, logger, "Asia/Ho_Chi_Minh")

		semanticRouter := router.New(llmManager, logger)
		logger.Info(ctx, "Semantic Router initialized")

		// Checklist Service
		checklistSvc := checklist.New()

		// Automation UseCase
		automationUC := automation.New(taskRepo, vectorRepoInterface, checklistSvc, logger)

		// Webhook Handler
		if cfg.Webhook.Enabled {
			webhookSecurityConfig := webhook.SecurityConfig{
				Secret:          cfg.Webhook.Secret,
				AllowedIPs:      cfg.Webhook.AllowedIPs,
				RateLimitPerMin: cfg.Webhook.RateLimitPerMin,
			}
			gitWebhookHandler = webhook.NewHandler(automationUC, webhookSecurityConfig, logger)
			logger.Info(ctx, "Git webhook handler initialized")
		}

		// Tool Registry Phase 4 Additions
		toolRegistry.Register(tools.NewGetChecklistProgressTool(taskRepo, checklistSvc, logger))
		if vectorRepoInterface != nil {
			toolRegistry.Register(tools.NewUpdateChecklistItemTool(taskRepo, vectorRepoInterface, checklistSvc, logger))
		}

		// Telegram Delivery handler
		telegramHandler = tgDelivery.New(logger, taskUC, telegramBot, agentOrchestrator, automationUC, checklistSvc, taskRepo, semanticRouter)

		// Test handler (for E2E testing)
		testHandler = test.New(logger, semanticRouter, agentOrchestrator)

		// Webhook Sync handler
		if vectorRepoInterface != nil {
			webhookHandler = sync.NewWebhookHandler(taskRepo, vectorRepoInterface, logger)
		}

		// Register webhook: auto-detect ngrok or fallback to manual config
		webhookURL := cfg.Telegram.WebhookURL
		if webhookURL == "" {
			ngrokURL, ngrokErr := detectNgrokURL(ctx, "http://ngrok:4040")
			if ngrokErr != nil {
				logger.Warnf(ctx, "Could not detect ngrok URL: %v", ngrokErr)
			} else {
				webhookURL = ngrokURL + "/webhook/telegram"
				logger.Infof(ctx, "Auto-detected ngrok URL: %s", webhookURL)
			}
		}

		if webhookURL != "" {
			if whErr := telegramBot.SetWebhook(webhookURL); whErr != nil {
				logger.Warnf(ctx, "Failed to set Telegram webhook: %v", whErr)
			} else {
				logger.Infof(ctx, "Telegram webhook registered at %s", webhookURL)
			}
		}

		logger.Info(ctx, "Phase 2 initialized successfully")
	} else {
		logger.Warn(ctx, "Phase 2 skipped: TELEGRAM_BOT_TOKEN, GEMINI_API_KEY, or MEMOS_ACCESS_TOKEN is missing")
	}

	// 4. HTTP Server
	httpServer, err := httpserver.New(logger, httpserver.Config{
		Logger:            logger,
		Port:              cfg.HTTPServer.Port,
		Mode:              cfg.HTTPServer.Mode,
		Environment:       cfg.Environment.Name,
		TelegramHandler:   telegramHandler,
		WebhookHandler:    webhookHandler,
		GitWebhookHandler: gitWebhookHandler,
		TestHandler:       testHandler,
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
