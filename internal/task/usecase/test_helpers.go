package usecase

import (
	"context"

	"autonomous-task-management/pkg/gemini"
	"autonomous-task-management/pkg/llmprovider"
)

// Mock logger for testing
type mockLogger struct{}

func (m *mockLogger) Debug(ctx context.Context, arg ...any)                    {}
func (m *mockLogger) Debugf(ctx context.Context, template string, arg ...any)  {}
func (m *mockLogger) Info(ctx context.Context, arg ...any)                     {}
func (m *mockLogger) Infof(ctx context.Context, template string, arg ...any)   {}
func (m *mockLogger) Warn(ctx context.Context, arg ...any)                     {}
func (m *mockLogger) Warnf(ctx context.Context, template string, arg ...any)   {}
func (m *mockLogger) Error(ctx context.Context, arg ...any)                    {}
func (m *mockLogger) Errorf(ctx context.Context, template string, arg ...any)  {}
func (m *mockLogger) Fatal(ctx context.Context, arg ...any)                    {}
func (m *mockLogger) Fatalf(ctx context.Context, template string, arg ...any)  {}
func (m *mockLogger) DPanic(ctx context.Context, arg ...any)                   {}
func (m *mockLogger) DPanicf(ctx context.Context, template string, arg ...any) {}
func (m *mockLogger) Panic(ctx context.Context, arg ...any)                    {}
func (m *mockLogger) Panicf(ctx context.Context, template string, arg ...any)  {}

// createManagerFromGeminiClient creates a Provider Manager with a Gemini provider for testing
func createManagerFromGeminiClient(client gemini.IGemini, logger *mockLogger) *llmprovider.Manager {
	provider := llmprovider.NewGeminiAdapter(client)
	config := &llmprovider.Config{
		FallbackEnabled: false,
		RetryAttempts:   1,
	}
	return llmprovider.NewManager([]llmprovider.Provider{provider}, config, logger)
}

// Mock Gemini client for testing
type mockGeminiClient struct {
	response *gemini.Response
	err      error
}

func (m *mockGeminiClient) GenerateContent(ctx context.Context, req *gemini.Request) (*gemini.Response, error) {
	return m.response, m.err
}

func (m *mockGeminiClient) Model() string {
	return "gemini-test"
}
