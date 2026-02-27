package httpserver

import (
	"errors"

	"github.com/gin-gonic/gin"

	tgDelivery "autonomous-task-management/internal/task/delivery/telegram"
	"autonomous-task-management/internal/test"
	"autonomous-task-management/pkg/log"
)

// HTTPServer holds all dependencies for the HTTP server.
type HTTPServer struct {
	// Server
	gin         *gin.Engine
	l           log.Logger
	port        int
	mode        string
	environment string

	// Phase 2: Task domain
	telegramHandler tgDelivery.Handler

	// Phase 3: Webhook sync
	webhookHandler interface {
		HandleMemosWebhook(c *gin.Context)
	}

	// Phase 4: Git webhooks
	gitWebhookHandler interface {
		HandleGitHubWebhook(c *gin.Context)
		HandleGitLabWebhook(c *gin.Context)
	}

	// Test domain
	testHandler test.Handler
}

// Config is the dependency bag passed to New().
type Config struct {
	Logger      log.Logger
	Port        int
	Mode        string
	Environment string

	// Phase 2: Task domain
	TelegramHandler tgDelivery.Handler

	// Phase 3: Webhook sync
	WebhookHandler interface {
		HandleMemosWebhook(c *gin.Context)
	}

	// Phase 4: Git webhooks
	GitWebhookHandler interface {
		HandleGitHubWebhook(c *gin.Context)
		HandleGitLabWebhook(c *gin.Context)
	}

	// Test domain
	TestHandler test.Handler
}

// New creates a new HTTPServer instance.
func New(logger log.Logger, cfg Config) (*HTTPServer, error) {
	gin.SetMode(cfg.Mode)

	srv := &HTTPServer{
		l:                 logger,
		gin:               gin.Default(),
		port:              cfg.Port,
		mode:              cfg.Mode,
		environment:       cfg.Environment,
		telegramHandler:   cfg.TelegramHandler,
		webhookHandler:    cfg.WebhookHandler,
		gitWebhookHandler: cfg.GitWebhookHandler,
		testHandler:       cfg.TestHandler,
	}

	if err := srv.validate(); err != nil {
		return nil, err
	}

	return srv, nil
}

func (srv HTTPServer) validate() error {
	if srv.l == nil {
		return errors.New("logger is required")
	}
	if srv.mode == "" {
		return errors.New("mode is required")
	}
	if srv.port == 0 {
		return errors.New("port is required")
	}
	return nil
}
