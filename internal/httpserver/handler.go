package httpserver

import (
	"context"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"autonomous-task-management/internal/agent"
	agentUC "autonomous-task-management/internal/agent/usecase"
	automationUC "autonomous-task-management/internal/automation/usecase"
	checklistUC "autonomous-task-management/internal/checklist/usecase"
	routerUC "autonomous-task-management/internal/router/usecase"
	syncHttp "autonomous-task-management/internal/sync/delivery/http"
	syncUC "autonomous-task-management/internal/sync/usecase"
	tgDelivery "autonomous-task-management/internal/task/delivery/telegram"
	taskUC "autonomous-task-management/internal/task/usecase"
	"autonomous-task-management/internal/test"
	"autonomous-task-management/internal/webhook"
	webhookHttp "autonomous-task-management/internal/webhook/delivery/http"
	webhookUC "autonomous-task-management/internal/webhook/usecase"
)

func (srv *HTTPServer) mapHandlers() error {
	srv.registerMiddlewares()
	srv.registerSystemRoutes()

	// Initialize domains in order of dependency
	srv.setupChecklistDomain()
	srv.setupRouterDomain()
	srv.setupTaskDomain()
	srv.setupSyncDomain()
	srv.setupAutomationDomain()
	srv.setupAgentDomain()
	srv.setupWebhookDomain()
	srv.setupTestDomain()

	return nil
}

func (srv *HTTPServer) registerMiddlewares() {
	srv.gin.Use(gin.Recovery())
	ctx := context.Background()
	srv.l.Infof(ctx, "Middlewares registered (Recovery enabled)")
}

func (srv *HTTPServer) registerSystemRoutes() {
	srv.gin.GET("/health", srv.healthCheck)
	srv.gin.GET("/ready", srv.readyCheck)
	srv.gin.GET("/live", srv.liveCheck)

	srv.gin.GET("/swagger/*any", ginSwagger.WrapHandler(
		swaggerFiles.Handler,
		ginSwagger.URL("doc.json"),
		ginSwagger.DefaultModelsExpandDepth(-1),
	))
}

func (srv *HTTPServer) setupChecklistDomain() {
	srv.checklistUC = checklistUC.New(srv.memosRepo, srv.vectorRepo, srv.l)
}

func (srv *HTTPServer) setupRouterDomain() {
	srv.routerUC = routerUC.New(srv.llmManager, srv.l)
}

func (srv *HTTPServer) setupTaskDomain() {
	srv.taskUC = taskUC.New(
		srv.l,
		srv.llmManager,
		srv.calendarClient,
		srv.memosRepo,
		srv.vectorRepo,
		srv.dateMathParser,
		nil, // reranker: optional, wired externally via srv.reranker if configured
		srv.cfg.LLM.Timezone,
		srv.cfg.Memos.ExternalURL,
	)

	// Register Telegram Webhook if token exists
	if srv.cfg.Telegram.BotToken != "" {
		// Note: we need agentUC, automationUC for Telegram handler,
		// so we'll finish telegram setup in setupAgentDomain or a separate step
	}
}

func (srv *HTTPServer) setupSyncDomain() {
	if srv.vectorRepo != nil {
		srv.syncUC = syncUC.New(srv.memosRepo, srv.vectorRepo, srv.l)
		srv.syncHandler = syncHttp.NewHandler(srv.syncUC, srv.l)
		srv.gin.POST("/webhook/memos", srv.syncHandler.HandleMemosWebhook)
		srv.l.Infof(context.Background(), "Sync domain routes registered at POST /webhook/memos")
	}
}

func (srv *HTTPServer) setupAutomationDomain() {
	srv.automationUC = automationUC.New(srv.memosRepo, srv.vectorRepo, srv.checklistUC, srv.l)
}

func (srv *HTTPServer) setupAgentDomain() {
	// Each domain self-registers its own tools — no cross-domain coupling here.
	registry := agent.NewToolRegistry()
	srv.taskUC.RegisterAgentTools(registry)
	srv.checklistUC.RegisterAgentTools(registry)

	srv.agentUC = agentUC.New(srv.llmManager, registry, srv.l, srv.cfg.LLM.Timezone)

	// Now we can finish Telegram Handler setup
	if srv.cfg.Telegram.BotToken != "" {
		srv.telegramHandler = tgDelivery.New(
			srv.l,
			srv.taskUC,
			srv.telegramBot,
			srv.agentUC,
			srv.automationUC,
			srv.checklistUC,
			srv.memosRepo,
			srv.routerUC,
		)
		srv.gin.POST("/webhook/telegram", srv.telegramHandler.HandleWebhook)
		srv.l.Infof(context.Background(), "Telegram webhook route registered at POST /webhook/telegram")
	}
}

func (srv *HTTPServer) setupWebhookDomain() {
	if srv.cfg.Webhook.Enabled {
		webhookConfig := webhook.SecurityConfig{
			Secret:          srv.cfg.Webhook.Secret,
			AllowedIPs:      srv.cfg.Webhook.AllowedIPs,
			RateLimitPerMin: srv.cfg.Webhook.RateLimitPerMin,
		}
		srv.webhookUC = webhookUC.New(webhookConfig, srv.l)
		srv.webhookHandler = webhookHttp.NewHandler(srv.webhookUC, srv.automationUC, srv.l)

		srv.gin.POST("/webhook/github", srv.webhookHandler.HandleGitHubWebhook)
		srv.gin.POST("/webhook/gitlab", srv.webhookHandler.HandleGitLabWebhook)
		srv.l.Infof(context.Background(), "Webhook domain routes registered (GitHub/GitLab)")
	}
}

func (srv *HTTPServer) setupTestDomain() {
	// Note: test domain might need direct orchestrator access if it's strictly for testing ReAct loop
	// For now, let's just make a mock or skip if not needed for production
	srv.testHandler = test.New(srv.l, srv.routerUC, srv.agentUC)
	srv.gin.POST("/test/message", srv.testHandler.HandleTestMessage)
	srv.gin.POST("/test/reset", srv.testHandler.HandleResetSession)
	srv.gin.GET("/test/health", srv.testHandler.HandleHealthCheck)
}
