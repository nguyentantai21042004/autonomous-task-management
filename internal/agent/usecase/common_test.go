package usecase

import (
	"context"
	"testing"

	"autonomous-task-management/internal/agent"
	"autonomous-task-management/pkg/llmprovider"
	pkgLog "autonomous-task-management/pkg/log"

	"github.com/stretchr/testify/mock"
)

// ============================================================================
// MOCK IMPLEMENTATIONS
// ============================================================================

type MockLLMManager struct {
	mock.Mock
}

func (m *MockLLMManager) GenerateContent(ctx context.Context, req *llmprovider.Request) (*llmprovider.Response, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*llmprovider.Response), args.Error(1)
}

type MockTool struct {
	name        string
	description string
	params      map[string]interface{}
}

func (m *MockTool) Name() string {
	return m.name
}

func (m *MockTool) Description() string {
	return m.description
}

func (m *MockTool) Parameters() map[string]interface{} {
	return m.params
}

func (m *MockTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	return map[string]string{"result": "success"}, nil
}

// ============================================================================
// TEST HELPERS
// ============================================================================

// setupTestUseCase creates a new usecase for testing with cleanup
func setupTestUseCase(t *testing.T) (agent.UseCase, *MockLLMManager, *agent.ToolRegistry) {
	mockLLM := new(MockLLMManager)
	registry := agent.NewToolRegistry()
	logger := pkgLog.Init(pkgLog.ZapConfig{Level: "info", Mode: "development"})

	uc := New(mockLLM, registry, logger, "Asia/Ho_Chi_Minh")

	// Cleanup goroutine after test
	t.Cleanup(func() {
		uc.(*implUseCase).stopCleanupForTest()
	})

	return uc, mockLLM, registry
}
