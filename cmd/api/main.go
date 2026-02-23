package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"autonomous-task-management/config"
	_ "autonomous-task-management/docs" // Swagger docs
	"autonomous-task-management/internal/httpserver"
	tgDelivery "autonomous-task-management/internal/task/delivery/telegram"
	memosRepo "autonomous-task-management/internal/task/repository/memos"
	"autonomous-task-management/internal/task/usecase"
	"autonomous-task-management/pkg/datemath"
	"autonomous-task-management/pkg/gcalendar"
	"autonomous-task-management/pkg/gemini"
	"autonomous-task-management/pkg/log"
	"autonomous-task-management/pkg/telegram"
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

	// 3. Phase 2: Task domain
	var telegramHandler tgDelivery.Handler

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
		taskRepo := memosRepo.New(memosClient, cfg.Memos.URL, logger)

		// Google Calendar client (optional)
		var calendarClient *gcalendar.Client
		if cfg.GoogleCalendar.CredentialsPath != "" {
			calendarClient, err = gcalendar.NewClientFromCredentialsFile(ctx, cfg.GoogleCalendar.CredentialsPath)
			if err != nil {
				logger.Warnf(ctx, "Google Calendar not available (optional): %v", err)
				logger.Warn(ctx, "→ Run `go run scripts/gcal-auth/main.go` to generate token.json")
			} else {
				logger.Info(ctx, "✅ Google Calendar initialized")
			}
		}

		// Task UseCase
		taskUC := usecase.New(logger, geminiClient, calendarClient, taskRepo, dateMathParser, timezone, cfg.Memos.URL)

		// Telegram Delivery handler
		telegramHandler = tgDelivery.New(logger, taskUC, telegramBot)

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
				logger.Infof(ctx, "✅ Telegram webhook registered at %s", webhookURL)
			}
		}

		logger.Info(ctx, "Phase 2 initialized successfully")
	} else {
		logger.Warn(ctx, "Phase 2 skipped: TELEGRAM_BOT_TOKEN, GEMINI_API_KEY, or MEMOS_ACCESS_TOKEN is missing")
	}

	// 4. HTTP Server
	httpServer, err := httpserver.New(logger, httpserver.Config{
		Logger:          logger,
		Port:            cfg.HTTPServer.Port,
		Mode:            cfg.HTTPServer.Mode,
		Environment:     cfg.Environment.Name,
		TelegramHandler: telegramHandler,
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
