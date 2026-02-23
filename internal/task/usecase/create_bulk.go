package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"autonomous-task-management/internal/model"
	"autonomous-task-management/internal/task"
	"autonomous-task-management/internal/task/repository"
	"autonomous-task-management/pkg/gcalendar"
)

// CreateBulk parses raw text, creates Memos tasks and Google Calendar events.
func (uc *implUseCase) CreateBulk(ctx context.Context, sc model.Scope, input task.CreateBulkInput) (task.CreateBulkOutput, error) {
	if strings.TrimSpace(input.RawText) == "" {
		return task.CreateBulkOutput{}, task.ErrEmptyInput
	}

	uc.l.Infof(ctx, "CreateBulk: user=%s input_length=%d", sc.UserID, len(input.RawText))

	// Step 1: Parse tasks from raw text via LLM
	parsedTasks, err := uc.parseInputWithLLM(ctx, input.RawText)
	if err != nil {
		return task.CreateBulkOutput{}, fmt.Errorf("failed to parse input with LLM: %w", err)
	}

	if len(parsedTasks) == 0 {
		return task.CreateBulkOutput{}, task.ErrNoTasksParsed
	}

	uc.l.Infof(ctx, "CreateBulk: LLM parsed %d tasks", len(parsedTasks))

	// Step 2: Resolve relative dates to absolute times
	tasksWithDates := uc.resolveDueDates(parsedTasks)

	// Step 3: Create each task in Memos and optionally in Google Calendar
	createdTasks := make([]task.CreatedTask, 0, len(tasksWithDates))

	for _, t := range tasksWithDates {
		// Build markdown content
		content := buildMarkdownContent(t)

		// Create in Memos
		memoTask, memoErr := uc.repo.CreateTask(ctx, repository.CreateTaskOptions{
			Content:    content,
			Tags:       allTags(t),
			Visibility: "PRIVATE",
		})
		if memoErr != nil {
			uc.l.Errorf(ctx, "CreateBulk: failed to create Memos task %q: %v", t.Title, memoErr)
			continue
		}

		// Attempt to create Google Calendar event (non-blocking on failure)
		calendarLink := uc.tryCreateCalendarEvent(ctx, t, memoTask)

		createdTasks = append(createdTasks, task.CreatedTask{
			MemoID:       memoTask.ID,
			MemoURL:      memoTask.MemoURL,
			CalendarLink: calendarLink,
			Title:        t.Title,
		})

		uc.l.Infof(ctx, "CreateBulk: created task %q memoID=%s", t.Title, memoTask.ID)
	}

	return task.CreateBulkOutput{
		Tasks:     createdTasks,
		TaskCount: len(createdTasks),
	}, nil
}

// tryCreateCalendarEvent attempts to create a Google Calendar event.
// Returns the event HTML link, or empty string on failure (graceful degradation).
func (uc *implUseCase) tryCreateCalendarEvent(ctx context.Context, t taskWithDate, memoTask model.Task) string {
	if uc.calendar == nil {
		return ""
	}

	startTime := t.DueDateAbsolute
	duration := t.EstimatedDurationMinutes
	if duration <= 0 {
		duration = 60 // default 1 hour
	}
	endTime := startTime.Add(time.Duration(duration) * time.Minute)

	description := t.Description
	if memoTask.MemoURL != "" {
		description += fmt.Sprintf("\n\n📝 Memos: %s", memoTask.MemoURL)
	}

	event, err := uc.calendar.CreateEvent(ctx, gcalendar.CreateEventRequest{
		CalendarID:  "primary",
		Summary:     t.Title,
		Description: strings.TrimSpace(description),
		StartTime:   startTime,
		EndTime:     endTime,
		Timezone:    uc.timezone,
	})
	if err != nil {
		uc.l.Warnf(ctx, "CreateBulk: calendar event creation failed for %q (non-fatal): %v", t.Title, err)
		return ""
	}

	return event.HtmlLink
}
