package httpserver

import (
	"context"

	"autonomous-task-management/internal/model"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func (srv HTTPServer) mapHandlers() error {
	srv.registerMiddlewares()
	srv.registerSystemRoutes()

	if err := srv.registerDomainRoutes(); err != nil {
		return err
	}

	return nil
}

func (srv HTTPServer) registerMiddlewares() {
	// CORS recovery
	srv.gin.Use(gin.Recovery())

	ctx := context.Background()
	if srv.environment == string(model.EnvironmentProduction) {
		srv.l.Infof(ctx, "CORS mode: production")
	} else {
		srv.l.Infof(ctx, "CORS mode: %s", srv.environment)
	}
}

func (srv HTTPServer) registerSystemRoutes() {
	srv.gin.GET("/health", srv.healthCheck)
	srv.gin.GET("/ready", srv.readyCheck)
	srv.gin.GET("/live", srv.liveCheck)

	srv.gin.GET("/swagger/*any", ginSwagger.WrapHandler(
		swaggerFiles.Handler,
		ginSwagger.URL("doc.json"),
		ginSwagger.DefaultModelsExpandDepth(-1),
	))
}

// registerDomainRoutes registers all domain routes.
func (srv HTTPServer) registerDomainRoutes() error {
	ctx := context.Background()

	// Phase 2: Telegram webhook
	if srv.telegramHandler != nil {
		srv.gin.POST("/webhook/telegram", srv.telegramHandler.HandleWebhook)
		srv.l.Infof(ctx, "Telegram webhook route registered at POST /webhook/telegram")
	} else {
		srv.l.Infof(ctx, "Telegram handler not configured, skipping webhook route")
	}

	// Phase 3: Memos webhook
	if srv.webhookHandler != nil {
		srv.gin.POST("/webhook/memos", srv.webhookHandler.HandleMemosWebhook)
		srv.l.Infof(ctx, "Memos webhook route registered at POST /webhook/memos")
	} else {
		srv.l.Infof(ctx, "Webhook handler not configured, skipping Memos webhook route")
	}

	return nil
}
