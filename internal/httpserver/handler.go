package httpserver

import (
	"context"

	"autonomous-task-management/internal/middleware"
	"autonomous-task-management/internal/model"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func (srv HTTPServer) mapHandlers() error {
	mw := middleware.New(srv.l, srv.jwtManager, srv.cookieConfig, srv.config.InternalConfig.InternalKey, srv.config, srv.encrypter)

	srv.registerMiddlewares(mw)
	srv.registerSystemRoutes()

	if err := srv.registerDomainRoutes(mw); err != nil {
		return err
	}

	return nil
}

func (srv HTTPServer) registerMiddlewares(mw middleware.Middleware) {
	srv.gin.Use(middleware.Recovery(srv.l, srv.discord))

	corsConfig := middleware.DefaultCORSConfig(srv.environment)
	srv.gin.Use(middleware.CORS(corsConfig))

	ctx := context.Background()
	if srv.environment == string(model.EnvironmentProduction) {
		srv.l.Infof(ctx, "CORS mode: production")
	} else {
		srv.l.Infof(ctx, "CORS mode: %s", srv.environment)
	}

	srv.gin.Use(mw.Locale())
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

// registerDomainRoutes initializes and registers all domain routes.
// Add new domains here following the same pattern as setupExampleDomain.
func (srv HTTPServer) registerDomainRoutes(mw middleware.Middleware) error {
	ctx := context.Background()

	api := srv.gin.Group("/api/v1")

	if err := srv.setupExampleDomain(ctx, api, mw); err != nil {
		return err
	}

	return nil
}
