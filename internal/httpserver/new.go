package httpserver

import (
	"errors"

	"github.com/gin-gonic/gin"

	"autonomous-task-management/config"
	"autonomous-task-management/internal/agent"
	"autonomous-task-management/internal/automation"
	"autonomous-task-management/internal/checklist"
	"autonomous-task-management/internal/router"
	"autonomous-task-management/internal/sync"
	"autonomous-task-management/internal/task"
	tgDelivery "autonomous-task-management/internal/task/delivery/telegram"
	"autonomous-task-management/internal/task/repository"
	"autonomous-task-management/internal/test"
	"autonomous-task-management/internal/webhook"
	"autonomous-task-management/pkg/datemath"
	"autonomous-task-management/pkg/llmprovider"
	"autonomous-task-management/pkg/log"
	"autonomous-task-management/pkg/telegram"
)

// HTTPServer holds all dependencies for the HTTP server.
type HTTPServer struct {
	// Server
	gin         *gin.Engine
	l           log.Logger
	cfg         *config.Config
	port        int
	mode        string
	environment string

	// Infrastructure
	llmManager     llmprovider.IManager
	memosRepo      repository.MemosRepository
	vectorRepo     repository.VectorRepository
	calendarClient task.CalendarClient
	telegramBot    telegram.IBot
	dateMathParser datemath.IParser

	// Domain UseCases
	taskUC       task.UseCase
	agentUC      agent.UseCase
	automationUC automation.UseCase
	checklistUC  checklist.UseCase
	routerUC     router.UseCase
	syncUC       sync.UseCase
	webhookUC    webhook.UseCase

	// Domain Handlers
	telegramHandler tgDelivery.Handler
	syncHandler     sync.Handler
	webhookHandler  webhook.Handler
	testHandler     test.Handler
}

// Config is the dependency bag passed to New().
type Config struct {
	Logger      log.Logger
	Config      *config.Config
	Port        int
	Mode        string
	Environment string

	// Infrastructure
	LLMManager     llmprovider.IManager
	MemosRepo      repository.MemosRepository
	VectorRepo     repository.VectorRepository
	CalendarClient task.CalendarClient
	TelegramBot    telegram.IBot
	DateMathParser datemath.IParser
}

// New creates a new HTTPServer instance.
func New(logger log.Logger, cfg Config) (*HTTPServer, error) {
	gin.SetMode(cfg.Mode)

	srv := &HTTPServer{
		l:              logger,
		gin:            gin.Default(),
		cfg:            cfg.Config,
		port:           cfg.Port,
		mode:           cfg.Mode,
		environment:    cfg.Environment,
		llmManager:     cfg.LLMManager,
		memosRepo:      cfg.MemosRepo,
		vectorRepo:     cfg.VectorRepo,
		calendarClient: cfg.CalendarClient,
		telegramBot:    cfg.TelegramBot,
		dateMathParser: cfg.DateMathParser,
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
