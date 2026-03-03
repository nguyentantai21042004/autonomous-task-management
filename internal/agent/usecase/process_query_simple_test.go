package usecase

import (
	"context"
	"errors"
	"testing"

	"autonomous-task-management/internal/agent"
	"autonomous-task-management/internal/model"
	"autonomous-task-management/pkg/llmprovider"
	pkgLog "autonomous-task-management/pkg/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Simplified tests without goroutine issues

func TestProcessQuery_Simple_DirectAnswer(t *testing.T) {
	mockLLM := new(MockLLMManager)
	registry := agent.NewToolRegistry()
	logger := pkgLog.Init(pkgLog.ZapConfig{Level: "error", Mode: "development"}) // Error level to reduce noise

	// Create usecase WITHOUT starting cleanup goroutine
	uc := &implUseCase{
		llm:          mockLLM,
		registry:     registry,
		l:            logger,
		timezone:     "Asia/Ho_Chi_Minh",
		sessionCache: make(map[string]*agent.SessionMemory),
		cacheTTL:     10,
		stopCleanup:  make(chan struct{}),
	}

	mockLLM.On("GenerateContent", mock.Anything, mock.Anything).Return(&llmprovider.Response{
		Content: llmprovider.Message{
			Role:  "assistant",
			Parts: []llmprovider.Part{{Text: "Xin chào! Tôi là trợ lý."}},
		},
	}, nil)

	sc := model.Scope{UserID: "test_user"}
	response, err := uc.ProcessQuery(context.Background(), sc, "Xin chào")

	assert.NoError(t, err)
	assert.Contains(t, response, "Xin chào")
}

func TestProcessQuery_Simple_WithToolCall(t *testing.T) {
	mockLLM := new(MockLLMManager)
	registry := agent.NewToolRegistry()
	logger := pkgLog.Init(pkgLog.ZapConfig{Level: "error", Mode: "development"})

	mockTool := &MockTool{
		name:        "test_tool",
		description: "Test tool",
		params:      map[string]interface{}{},
	}
	registry.Register(mockTool)

	uc := &implUseCase{
		llm:          mockLLM,
		registry:     registry,
		l:            logger,
		timezone:     "Asia/Ho_Chi_Minh",
		sessionCache: make(map[string]*agent.SessionMemory),
		cacheTTL:     10,
		stopCleanup:  make(chan struct{}),
	}

	// First call - tool request
	mockLLM.On("GenerateContent", mock.Anything, mock.Anything).Return(&llmprovider.Response{
		Content: llmprovider.Message{
			Role: "assistant",
			Parts: []llmprovider.Part{
				{FunctionCall: &llmprovider.FunctionCall{Name: "test_tool", Args: map[string]interface{}{}}},
			},
		},
	}, nil).Once()

	// Second call - final answer
	mockLLM.On("GenerateContent", mock.Anything, mock.Anything).Return(&llmprovider.Response{
		Content: llmprovider.Message{
			Role:  "assistant",
			Parts: []llmprovider.Part{{Text: "Tool executed successfully"}},
		},
	}, nil).Once()

	sc := model.Scope{UserID: "test_user"}
	response, err := uc.ProcessQuery(context.Background(), sc, "Run tool")

	assert.NoError(t, err)
	assert.Contains(t, response, "successfully")
	mockLLM.AssertNumberOfCalls(t, "GenerateContent", 2)
}

func TestProcessQuery_Simple_MaxSteps(t *testing.T) {
	mockLLM := new(MockLLMManager)
	registry := agent.NewToolRegistry()
	logger := pkgLog.Init(pkgLog.ZapConfig{Level: "error", Mode: "development"})

	uc := &implUseCase{
		llm:          mockLLM,
		registry:     registry,
		l:            logger,
		timezone:     "Asia/Ho_Chi_Minh",
		sessionCache: make(map[string]*agent.SessionMemory),
		cacheTTL:     10,
		stopCleanup:  make(chan struct{}),
	}

	// Always return tool call (infinite loop)
	mockLLM.On("GenerateContent", mock.Anything, mock.Anything).Return(&llmprovider.Response{
		Content: llmprovider.Message{
			Role: "assistant",
			Parts: []llmprovider.Part{
				{FunctionCall: &llmprovider.FunctionCall{Name: "nonexistent", Args: map[string]interface{}{}}},
			},
		},
	}, nil)

	sc := model.Scope{UserID: "test_user"}
	response, err := uc.ProcessQuery(context.Background(), sc, "Test")

	assert.NoError(t, err)
	assert.Contains(t, response, "vượt quá")
	mockLLM.AssertNumberOfCalls(t, "GenerateContent", MaxAgentSteps)
}

func TestProcessQuery_Simple_LLMError(t *testing.T) {
	mockLLM := new(MockLLMManager)
	registry := agent.NewToolRegistry()
	logger := pkgLog.Init(pkgLog.ZapConfig{Level: "error", Mode: "development"})

	uc := &implUseCase{
		llm:          mockLLM,
		registry:     registry,
		l:            logger,
		timezone:     "Asia/Ho_Chi_Minh",
		sessionCache: make(map[string]*agent.SessionMemory),
		cacheTTL:     10,
		stopCleanup:  make(chan struct{}),
	}

	mockLLM.On("GenerateContent", mock.Anything, mock.Anything).Return(nil, errors.New("LLM error"))

	sc := model.Scope{UserID: "test_user"}
	response, err := uc.ProcessQuery(context.Background(), sc, "Test")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "LLM")
	assert.Empty(t, response)
}

func TestProcessQuery_Simple_EmptyResponse(t *testing.T) {
	mockLLM := new(MockLLMManager)
	registry := agent.NewToolRegistry()
	logger := pkgLog.Init(pkgLog.ZapConfig{Level: "error", Mode: "development"})

	uc := &implUseCase{
		llm:          mockLLM,
		registry:     registry,
		l:            logger,
		timezone:     "Asia/Ho_Chi_Minh",
		sessionCache: make(map[string]*agent.SessionMemory),
		cacheTTL:     10,
		stopCleanup:  make(chan struct{}),
	}

	mockLLM.On("GenerateContent", mock.Anything, mock.Anything).Return(&llmprovider.Response{
		Content: llmprovider.Message{
			Role:  "assistant",
			Parts: []llmprovider.Part{},
		},
	}, nil)

	sc := model.Scope{UserID: "test_user"}
	response, err := uc.ProcessQuery(context.Background(), sc, "Test")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
	assert.Empty(t, response)
}

func TestGetSessionMessages_Simple(t *testing.T) {
	logger := pkgLog.Init(pkgLog.ZapConfig{Level: "error", Mode: "development"})

	uc := &implUseCase{
		l:            logger,
		sessionCache: make(map[string]*agent.SessionMemory),
		cacheTTL:     10,
		stopCleanup:  make(chan struct{}),
	}

	// Empty session
	messages := uc.GetSessionMessages("new_user")
	assert.Nil(t, messages)

	// Create session
	uc.sessionCache["existing_user"] = &agent.SessionMemory{
		UserID: "existing_user",
		Messages: []llmprovider.Message{
			{Role: "user", Parts: []llmprovider.Part{{Text: "Hello"}}},
		},
	}

	messages = uc.GetSessionMessages("existing_user")
	assert.NotNil(t, messages)
	assert.Len(t, messages, 1)
}

func TestClearSession_Simple(t *testing.T) {
	logger := pkgLog.Init(pkgLog.ZapConfig{Level: "error", Mode: "development"})

	uc := &implUseCase{
		l:            logger,
		sessionCache: make(map[string]*agent.SessionMemory),
		cacheTTL:     10,
		stopCleanup:  make(chan struct{}),
	}

	// Create session
	uc.sessionCache["user1"] = &agent.SessionMemory{
		UserID:   "user1",
		Messages: []llmprovider.Message{{Role: "user", Parts: []llmprovider.Part{{Text: "Test"}}}},
	}

	// Clear
	uc.ClearSession("user1")

	// Verify cleared
	messages := uc.GetSessionMessages("user1")
	assert.Nil(t, messages)
}
