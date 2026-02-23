package httpserver

import (
	"database/sql"
	"errors"

	"github.com/gin-gonic/gin"

	"autonomous-task-management/config"
	"autonomous-task-management/pkg/discord"
	"autonomous-task-management/pkg/encrypter"
	pkgJWT "autonomous-task-management/pkg/jwt"
	"autonomous-task-management/pkg/log"
	pkgRedis "autonomous-task-management/pkg/redis"
)

// HTTPServer holds all dependencies for the HTTP server.
type HTTPServer struct {
	// Server
	gin         *gin.Engine
	l           log.Logger
	port        int
	mode        string
	environment string

	// Database
	postgresDB *sql.DB

	// Auth & Security
	config       *config.Config
	jwtManager   pkgJWT.IManager
	redisClient  pkgRedis.IRedis
	cookieConfig config.CookieConfig
	encrypter    encrypter.Encrypter

	// Monitoring
	discord discord.IDiscord
}

// Config is the dependency bag passed to New().
type Config struct {
	// Server
	Logger      log.Logger
	Port        int
	Mode        string
	Environment string

	// Database
	PostgresDB *sql.DB

	// Auth & Security
	Config       *config.Config
	JWTManager   pkgJWT.IManager
	RedisClient  pkgRedis.IRedis
	CookieConfig config.CookieConfig
	Encrypter    encrypter.Encrypter

	// Monitoring
	Discord discord.IDiscord
}

// New creates a new HTTPServer instance.
func New(logger log.Logger, cfg Config) (*HTTPServer, error) {
	gin.SetMode(cfg.Mode)

	srv := &HTTPServer{
		l:           logger,
		gin:         gin.Default(),
		port:        cfg.Port,
		mode:        cfg.Mode,
		environment: cfg.Environment,

		postgresDB: cfg.PostgresDB,

		config:       cfg.Config,
		jwtManager:   cfg.JWTManager,
		redisClient:  cfg.RedisClient,
		cookieConfig: cfg.CookieConfig,
		encrypter:    cfg.Encrypter,

		discord: cfg.Discord,
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
	if srv.postgresDB == nil {
		return errors.New("postgresDB is required")
	}
	if srv.config == nil {
		return errors.New("config is required")
	}
	if srv.jwtManager == nil {
		return errors.New("jwtManager is required")
	}
	if srv.redisClient == nil {
		return errors.New("redisClient is required")
	}
	if srv.encrypter == nil {
		return errors.New("encrypter is required")
	}
	return nil
}
