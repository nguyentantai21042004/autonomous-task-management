package usecase

import (
	"autonomous-task-management/internal/task/repository"
	"autonomous-task-management/pkg/datemath"
	"autonomous-task-management/pkg/gcalendar"
	"autonomous-task-management/pkg/llmprovider"
	pkgLog "autonomous-task-management/pkg/log"
	"context"
)

// CalendarClient abstracts the Google Calendar API.
type CalendarClient interface {
	CreateEvent(ctx context.Context, req gcalendar.CreateEventRequest) (*gcalendar.Event, error)
}

type implUseCase struct {
	l          pkgLog.Logger
	llm        *llmprovider.Manager
	calendar   CalendarClient
	repo       repository.MemosRepository
	vectorRepo repository.VectorRepository
	dateMath   *datemath.Parser
	timezone   string
	memosURL   string
}

// New creates a new task UseCase instance.
func New(
	l pkgLog.Logger,
	llm *llmprovider.Manager,
	calendar CalendarClient,
	repo repository.MemosRepository,
	vectorRepo repository.VectorRepository,
	dateMath *datemath.Parser,
	timezone string,
	memosURL string,
) *implUseCase {
	return &implUseCase{
		l:          l,
		llm:        llm,
		calendar:   calendar,
		repo:       repo,
		vectorRepo: vectorRepo,
		dateMath:   dateMath,
		timezone:   timezone,
		memosURL:   memosURL,
	}
}
