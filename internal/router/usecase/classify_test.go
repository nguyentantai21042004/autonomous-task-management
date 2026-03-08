package usecase

import (
	"context"
	"errors"
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

// ---------------------------------------------------------------------------
// Tests for LLM slow path (ambiguous messages that bypass rule-based)
// ---------------------------------------------------------------------------

func TestClassify_CreateTask_LLMPath(t *testing.T) {
	ctx := context.Background()
	mockLLM := new(MockLLMManager)
	logger := pkgLog.Init(pkgLog.ZapConfig{Level: "info", Mode: "development"})
	uc := New(mockLLM, logger)

	// Ambiguous: không có signal rõ ràng → phải gọi LLM
	mockLLM.On("GenerateContent", ctx, mock.Anything).Return(&llmprovider.Response{
		Content: llmprovider.Message{
			Parts: []llmprovider.Part{
				{Text: `{"intent":"CREATE_TASK","confidence":75,"reasoning":"Context implies task creation"}`},
			},
		},
	}, nil)

	output, err := uc.Classify(ctx, "Nhớ là còn việc báo cáo tuần này chưa xong", nil)

	assert.NoError(t, err)
	assert.Equal(t, router.IntentCreateTask, output.Intent)
	assert.Greater(t, output.Confidence, 0)
	mockLLM.AssertExpectations(t)
}

func TestClassify_SearchTask_LLMPath(t *testing.T) {
	ctx := context.Background()
	mockLLM := new(MockLLMManager)
	logger := pkgLog.Init(pkgLog.ZapConfig{Level: "info", Mode: "development"})
	uc := New(mockLLM, logger)

	mockLLM.On("GenerateContent", ctx, mock.Anything).Return(&llmprovider.Response{
		Content: llmprovider.Message{
			Parts: []llmprovider.Part{
				{Text: `{"intent":"SEARCH_TASK","confidence":80,"reasoning":"User asking about PR status"}`},
			},
		},
	}, nil)

	// Ambiguous: không có keyword search rõ ràng
	output, err := uc.Classify(ctx, "PR 123 đang ở đâu rồi?", nil)

	assert.NoError(t, err)
	assert.Equal(t, router.IntentSearchTask, output.Intent)
	mockLLM.AssertExpectations(t)
}

func TestClassify_ManageChecklist_LLMPath(t *testing.T) {
	ctx := context.Background()
	mockLLM := new(MockLLMManager)
	logger := pkgLog.Init(pkgLog.ZapConfig{Level: "info", Mode: "development"})
	uc := New(mockLLM, logger)

	mockLLM.On("GenerateContent", ctx, mock.Anything).Return(&llmprovider.Response{
		Content: llmprovider.Message{
			Parts: []llmprovider.Part{
				{Text: `{"intent":"MANAGE_CHECKLIST","confidence":88,"reasoning":"User wants to manage checklist"}`},
			},
		},
	}, nil)

	// Ambiguous: không chắc là checklist hay conversation
	output, err := uc.Classify(ctx, "item số 2 trong memo đó xong rồi", nil)

	assert.NoError(t, err)
	assert.Equal(t, router.IntentManageChecklist, output.Intent)
	mockLLM.AssertExpectations(t)
}

func TestClassify_Conversation_LLMPath(t *testing.T) {
	ctx := context.Background()
	mockLLM := new(MockLLMManager)
	logger := pkgLog.Init(pkgLog.ZapConfig{Level: "info", Mode: "development"})
	uc := New(mockLLM, logger)

	mockLLM.On("GenerateContent", ctx, mock.Anything).Return(&llmprovider.Response{
		Content: llmprovider.Message{
			Parts: []llmprovider.Part{
				{Text: `{"intent":"CONVERSATION","confidence":85,"reasoning":"General question"}`},
			},
		},
	}, nil)

	// Ambiguous: câu hỏi chung không rõ intent
	output, err := uc.Classify(ctx, "tôi không biết phải làm sao với project này", nil)

	assert.NoError(t, err)
	assert.Equal(t, router.IntentConversation, output.Intent)
	mockLLM.AssertExpectations(t)
}

func TestClassify_WithConversationHistory_LLMPath(t *testing.T) {
	ctx := context.Background()
	mockLLM := new(MockLLMManager)
	logger := pkgLog.Init(pkgLog.ZapConfig{Level: "info", Mode: "development"})
	uc := New(mockLLM, logger)

	history := []string{"User: Tạo task mới", "Bot: Task đã được tạo"}

	mockLLM.On("GenerateContent", ctx, mock.Anything).Return(&llmprovider.Response{
		Content: llmprovider.Message{
			Parts: []llmprovider.Part{
				{Text: `{"intent":"SEARCH_TASK","confidence":92,"reasoning":"User wants to view the created task"}`},
			},
		},
	}, nil)

	// Ambiguous without history context
	output, err := uc.Classify(ctx, "cái vừa tạo đó đâu rồi", history)

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

// ---------------------------------------------------------------------------
// Additional coverage: LLM error, plain code block, edge cases
// ---------------------------------------------------------------------------

func TestClassify_LLMError(t *testing.T) {
	ctx := context.Background()
	mockLLM := new(MockLLMManager)
	logger := pkgLog.Init(pkgLog.ZapConfig{Level: "error", Mode: "development"})
	uc := New(mockLLM, logger)

	mockLLM.On("GenerateContent", ctx, mock.Anything).Return(nil, errors.New("LLM timeout"))

	_, err := uc.Classify(ctx, "ambiguous message", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "LLM")
	mockLLM.AssertExpectations(t)
}

func TestClassify_PlainCodeBlock(t *testing.T) {
	ctx := context.Background()
	mockLLM := new(MockLLMManager)
	logger := pkgLog.Init(pkgLog.ZapConfig{Level: "error", Mode: "development"})
	uc := New(mockLLM, logger)

	mockLLM.On("GenerateContent", ctx, mock.Anything).Return(&llmprovider.Response{
		Content: llmprovider.Message{
			Parts: []llmprovider.Part{
				{Text: "```\n{\"intent\":\"SEARCH_TASK\",\"confidence\":80,\"reasoning\":\"Test\"}\n```"},
			},
		},
	}, nil)

	output, err := uc.Classify(ctx, "some ambiguous query", nil)
	assert.NoError(t, err)
	assert.Equal(t, router.IntentSearchTask, output.Intent)
	assert.Equal(t, 80, output.Confidence)
	mockLLM.AssertExpectations(t)
}

func TestClassify_EmptyTextInParts(t *testing.T) {
	ctx := context.Background()
	mockLLM := new(MockLLMManager)
	logger := pkgLog.Init(pkgLog.ZapConfig{Level: "error", Mode: "development"})
	uc := New(mockLLM, logger)

	mockLLM.On("GenerateContent", ctx, mock.Anything).Return(&llmprovider.Response{
		Content: llmprovider.Message{
			Parts: []llmprovider.Part{
				{Text: ""},
			},
		},
	}, nil)

	output, err := uc.Classify(ctx, "ambiguous message", nil)
	assert.NoError(t, err)
	assert.Equal(t, RouterFallbackIntent, output.Intent)
	assert.Equal(t, RouterFallbackConfidence, output.Confidence)
	mockLLM.AssertExpectations(t)
}
