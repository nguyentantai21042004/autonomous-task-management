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

func (m *MockTool) Name() string                    { return m.name }
func (m *MockTool) Description() string             { return m.description }
func (m *MockTool) Parameters() map[string]interface{} { return m.params }
func (m *MockTool) Execute(_ context.Context, _ map[string]interface{}) (interface{}, error) {
	return map[string]string{"result": "success"}, nil
}

// ============================================================================
// TEST HELPERS
// ============================================================================

// setupTestUseCase tao agent UseCase cho testing.
// V2.0: khong can cleanup goroutine, LRU tu quan ly TTL.
func setupTestUseCase(t *testing.T) (agent.UseCase, *MockLLMManager, *agent.ToolRegistry) {
	t.Helper()
	mockLLM := new(MockLLMManager)
	registry := agent.NewToolRegistry()
	logger := pkgLog.Init(pkgLog.ZapConfig{Level: "error", Mode: "development"})
	return New(mockLLM, registry, logger, "Asia/Ho_Chi_Minh"), mockLLM, registry
}
