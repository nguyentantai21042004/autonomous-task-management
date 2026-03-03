package usecase

import (
	"context"
	"testing"

	"autonomous-task-management/internal/router"
	"autonomous-task-management/pkg/llmprovider"
	pkgLog "autonomous-task-management/pkg/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockLLMManager is a mock implementation of llmprovider.IManager
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

func TestClassify_CreateTask(t *testing.T) {
	ctx := context.Background()
	mockLLM := new(MockLLMManager)
	logger := pkgLog.Init(pkgLog.ZapConfig{Level: "info", Mode: "development"})

	uc := New(mockLLM, logger)

	// Mock LLM response for CREATE_TASK intent
	mockLLM.On("GenerateContent", ctx, mock.Anything).Return(&llmprovider.Response{
		Content: llmprovider.Message{
			Parts: []llmprovider.Part{
				{Text: `{"intent":"CREATE_TASK","confidence":95,"reasoning":"User wants to create a new task"}`},
			},
		},
	}, nil)

	output, err := uc.Classify(ctx, "Tạo task mới: Hoàn thành báo cáo", nil)

	assert.NoError(t, err)
	assert.Equal(t, router.IntentCreateTask, output.Intent)
	assert.Greater(t, output.Confidence, 0)
	mockLLM.AssertExpectations(t)
}

func TestClassify_SearchTask(t *testing.T) {
	ctx := context.Background()
	mockLLM := new(MockLLMManager)
	logger := pkgLog.Init(pkgLog.ZapConfig{Level: "info", Mode: "development"})

	uc := New(mockLLM, logger)

	// Mock LLM response for SEARCH_TASK intent
	mockLLM.On("GenerateContent", ctx, mock.Anything).Return(&llmprovider.Response{
		Content: llmprovider.Message{
			Parts: []llmprovider.Part{
				{Text: `{"intent":"SEARCH_TASK","confidence":90,"reasoning":"User wants to search for tasks"}`},
			},
		},
	}, nil)

	output, err := uc.Classify(ctx, "Tìm task về báo cáo", nil)

	assert.NoError(t, err)
	assert.Equal(t, router.IntentSearchTask, output.Intent)
	assert.Greater(t, output.Confidence, 0)
	mockLLM.AssertExpectations(t)
}

func TestClassify_ManageChecklist(t *testing.T) {
	ctx := context.Background()
	mockLLM := new(MockLLMManager)
	logger := pkgLog.Init(pkgLog.ZapConfig{Level: "info", Mode: "development"})

	uc := New(mockLLM, logger)

	// Mock LLM response for MANAGE_CHECKLIST intent
	mockLLM.On("GenerateContent", ctx, mock.Anything).Return(&llmprovider.Response{
		Content: llmprovider.Message{
			Parts: []llmprovider.Part{
				{Text: `{"intent":"MANAGE_CHECKLIST","confidence":88,"reasoning":"User wants to manage checklist items"}`},
			},
		},
	}, nil)

	output, err := uc.Classify(ctx, "Đánh dấu item 1 là hoàn thành", nil)

	assert.NoError(t, err)
	assert.Equal(t, router.IntentManageChecklist, output.Intent)
	assert.Greater(t, output.Confidence, 0)
	mockLLM.AssertExpectations(t)
}

func TestClassify_Conversation(t *testing.T) {
	ctx := context.Background()
	mockLLM := new(MockLLMManager)
	logger := pkgLog.Init(pkgLog.ZapConfig{Level: "info", Mode: "development"})

	uc := New(mockLLM, logger)

	// Mock LLM response for CONVERSATION intent
	mockLLM.On("GenerateContent", ctx, mock.Anything).Return(&llmprovider.Response{
		Content: llmprovider.Message{
			Parts: []llmprovider.Part{
				{Text: `{"intent":"CONVERSATION","confidence":85,"reasoning":"General conversation"}`},
			},
		},
	}, nil)

	output, err := uc.Classify(ctx, "Xin chào, bạn khỏe không?", nil)

	assert.NoError(t, err)
	assert.Equal(t, router.IntentConversation, output.Intent)
	assert.Greater(t, output.Confidence, 0)
	mockLLM.AssertExpectations(t)
}

func TestClassify_WithConversationHistory(t *testing.T) {
	ctx := context.Background()
	mockLLM := new(MockLLMManager)
	logger := pkgLog.Init(pkgLog.ZapConfig{Level: "info", Mode: "development"})

	uc := New(mockLLM, logger)

	history := []string{
		"User: Tạo task mới",
		"Bot: Task đã được tạo",
	}

	mockLLM.On("GenerateContent", ctx, mock.Anything).Return(&llmprovider.Response{
		Content: llmprovider.Message{
			Parts: []llmprovider.Part{
				{Text: `{"intent":"SEARCH_TASK","confidence":92,"reasoning":"User wants to view the created task"}`},
			},
		},
	}, nil)

	output, err := uc.Classify(ctx, "Cho tôi xem task vừa tạo", history)

	assert.NoError(t, err)
	assert.Equal(t, router.IntentSearchTask, output.Intent)
	mockLLM.AssertExpectations(t)
}

func TestClassify_EmptyResponse(t *testing.T) {
	ctx := context.Background()
	mockLLM := new(MockLLMManager)
	logger := pkgLog.Init(pkgLog.ZapConfig{Level: "info", Mode: "development"})

	uc := New(mockLLM, logger)

	// Mock empty response
	mockLLM.On("GenerateContent", ctx, mock.Anything).Return(&llmprovider.Response{
		Content: llmprovider.Message{
			Parts: []llmprovider.Part{},
		},
	}, nil)

	output, err := uc.Classify(ctx, "Test message", nil)

	assert.NoError(t, err)
	assert.Equal(t, RouterFallbackIntent, output.Intent)
	assert.Equal(t, RouterFallbackConfidence, output.Confidence)
	mockLLM.AssertExpectations(t)
}

func TestClassify_InvalidJSON(t *testing.T) {
	ctx := context.Background()
	mockLLM := new(MockLLMManager)
	logger := pkgLog.Init(pkgLog.ZapConfig{Level: "info", Mode: "development"})

	uc := New(mockLLM, logger)

	// Mock invalid JSON response
	mockLLM.On("GenerateContent", ctx, mock.Anything).Return(&llmprovider.Response{
		Content: llmprovider.Message{
			Parts: []llmprovider.Part{
				{Text: `invalid json response`},
			},
		},
	}, nil)

	output, err := uc.Classify(ctx, "Test message", nil)

	assert.NoError(t, err)
	assert.Equal(t, RouterFallbackIntent, output.Intent)
	assert.Equal(t, RouterFallbackConfidence, output.Confidence)
	mockLLM.AssertExpectations(t)
}

func TestClassify_JSONWithCodeBlock(t *testing.T) {
	ctx := context.Background()
	mockLLM := new(MockLLMManager)
	logger := pkgLog.Init(pkgLog.ZapConfig{Level: "info", Mode: "development"})

	uc := New(mockLLM, logger)

	// Mock response with markdown code block
	mockLLM.On("GenerateContent", ctx, mock.Anything).Return(&llmprovider.Response{
		Content: llmprovider.Message{
			Parts: []llmprovider.Part{
				{Text: "```json\n{\"intent\":\"CREATE_TASK\",\"confidence\":95,\"reasoning\":\"Test\"}\n```"},
			},
		},
	}, nil)

	output, err := uc.Classify(ctx, "Test message", nil)

	assert.NoError(t, err)
	assert.Equal(t, router.IntentCreateTask, output.Intent)
	assert.Equal(t, 95, output.Confidence)
	mockLLM.AssertExpectations(t)
}
