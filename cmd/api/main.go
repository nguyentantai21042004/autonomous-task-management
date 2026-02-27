package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

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
	"autonomous-task-management/internal/webhook"
	"autonomous-task-management/pkg/datemath"
	"autonomous-task-management/pkg/gcalendar"
	"autonomous-task-management/pkg/gemini"
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

	// 3. Phase 2 & 3: Domain initialization
	var telegramHandler tgDelivery.Handler
	var webhookHandler sync.Handler
	var gitWebhookHandler *webhook.Handler

	if cfg.Telegram.BotToken != "" && cfg.Gemini.APIKey != "" && cfg.Memos.AccessToken != "" {
		logger.Info(ctx, "Initializing Phase 2 components...")

		// Telegram Bot client
		telegramBot := telegram.NewBot(cfg.Telegram.BotToken)

		// Gemini LLM client
		geminiClient := gemini.NewClient(cfg.Gemini.APIKey)

		// DateMath parser
		timezone := cfg.Gemini.Timezone
		if timezone == "" {
			timezone = "Asia/Ho_Chi_Minh"
		}
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
		taskUC := usecase.New(logger, geminiClient, calendarClient, taskRepo, vectorRepoInterface, dateMathParser, timezone, cfg.Memos.ExternalURL)

		// Agent Tool Registry & Orchestrator
		toolRegistry := agent.NewToolRegistry()
		toolRegistry.Register(tools.NewSearchTasksTool(taskUC))
		if calendarClient != nil {
			toolRegistry.Register(tools.NewCheckCalendarTool(calendarClient, logger))
		}

		agentOrchestrator := orchestrator.New(geminiClient, toolRegistry, logger, cfg.Gemini.Timezone)

		// 🆕 Initialize Semantic Router
		semanticRouter := router.New(geminiClient, logger)
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
