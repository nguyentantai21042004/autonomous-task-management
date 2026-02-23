package main

import (
	"context"
	"fmt"
	"os"

	"autonomous-task-management/config"
	"autonomous-task-management/internal/task/repository"
	memosRepo "autonomous-task-management/internal/task/repository/memos"
	qdrantRepo "autonomous-task-management/internal/task/repository/qdrant"
	"autonomous-task-management/pkg/log"
	pkgQdrant "autonomous-task-management/pkg/qdrant"
	"autonomous-task-management/pkg/voyage"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run scripts/backfill-embeddings/main.go <path/to/config.yaml>")
		fmt.Println("Example: go run scripts/backfill-embeddings/main.go config/config.yaml")
		os.Exit(1)
	}
	configPath := os.Args[1]

	// Load config
	os.Setenv("CONFIG_PATH", configPath)
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize Logger
	logger := log.Init(log.ZapConfig{
		Level:        "info",
		Mode:         "development",
		ColorEnabled: true,
	})

	ctx := context.Background()

	// Initialize clients
	memosClient := memosRepo.NewClient(cfg.Memos.URL, cfg.Memos.AccessToken)
	memosRepository := memosRepo.New(memosClient, cfg.Memos.URL, logger)

	qdrantClient := pkgQdrant.NewClient(cfg.Qdrant.URL)
	embeddingClient, err := voyage.New(cfg.Voyage.APIKey)
	if err != nil {
		logger.Fatalf(ctx, "Failed to initialize Voyage API: %v", err)
	}
	vectorRepo := qdrantRepo.New(qdrantClient, embeddingClient, cfg.Qdrant.CollectionName, logger)

	logger.Info(ctx, "Starting backfill process...")

	// Fetch all tasks from Memos
	tasks, err := memosRepository.ListTasks(ctx, repository.ListTasksOptions{
		Limit: 1000, // Adjust as needed
	})
	if err != nil {
		logger.Fatalf(ctx, "Failed to list tasks: %v", err)
	}

	logger.Infof(ctx, "Found %d tasks to backfill to Qdrant", len(tasks))

	successCount := 0
	// Embed each task
	for i, task := range tasks {
		if err := vectorRepo.EmbedTask(ctx, task); err != nil {
			logger.Errorf(ctx, "Failed to embed task %s: %v", task.ID, err)
			continue
		}
		logger.Infof(ctx, "Embedded task %d/%d: %s", i+1, len(tasks), task.ID)
		successCount++
	}

	logger.Infof(ctx, "Backfill complete! %d/%d tasks successfully embedded.", successCount, len(tasks))
}
