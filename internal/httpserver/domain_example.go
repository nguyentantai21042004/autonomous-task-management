package httpserver

import (
	"context"

	exampleHTTP "autonomous-task-management/internal/example/delivery/http"
	exampleRepo "autonomous-task-management/internal/example/repository/postgre"
	exampleUC "autonomous-task-management/internal/example/usecase"
	"autonomous-task-management/internal/middleware"

	"github.com/gin-gonic/gin"
)

// setupExampleDomain initializes the example domain and registers its routes.
//
// Pattern to follow when adding a new domain:
//  1. Create Repository:   repo := mydomainRepo.New(srv.postgresDB, srv.l)
//  2. Create UseCase:      uc := mydomainUC.New(repo, srv.l)
//  3. Create HTTP Handler: h := mydomainHTTP.New(srv.l, uc, srv.discord)
//  4. Register Routes:     mydomainHTTP.RegisterRoutes(rg.Group("/myresource"), h, mw)
func (srv HTTPServer) setupExampleDomain(ctx context.Context, api *gin.RouterGroup, mw middleware.Middleware) error {
	// 1. Repository
	repo := exampleRepo.New(srv.postgresDB, srv.l)

	// 2. UseCase
	uc := exampleUC.New(repo, srv.l)

	// 3. HTTP Handler
	h := exampleHTTP.New(srv.l, uc, srv.discord)

	// 4. Routes: registers /api/v1/example/items
	exampleHTTP.RegisterRoutes(api.Group("/example"), h, mw)

	srv.l.Infof(ctx, "Example domain registered")
	return nil
}
