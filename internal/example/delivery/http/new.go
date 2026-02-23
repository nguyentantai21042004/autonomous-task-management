package http

import (
	"autonomous-task-management/internal/example"
	"autonomous-task-management/pkg/discord"
	"autonomous-task-management/pkg/log"
)

// Handler is the public interface for the example HTTP delivery layer.
type Handler interface {
	Create(c interface{})
	List(c interface{})
	Detail(c interface{})
	Update(c interface{})
	Delete(c interface{})
}

type handler struct {
	l       log.Logger
	uc      example.UseCase
	discord discord.IDiscord
}

// New creates a new HTTP handler for the example domain.
func New(l log.Logger, uc example.UseCase, disc discord.IDiscord) *handler {
	return &handler{
		l:       l,
		uc:      uc,
		discord: disc,
	}
}
