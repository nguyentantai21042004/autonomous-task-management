package graph

import (
	"context"
	"errors"
	"testing"

	"autonomous-task-management/pkg/llmprovider"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockLLM mock implementation for llmprovider.IManager
type mockLLM struct {
	mock.Mock
}

func (m *mockLLM) GenerateContent(ctx context.Context, req *llmprovider.Request) (*llmprovider.Response, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*llmprovider.Response), args.Error(1)
}

func makeTextResponse(text string) *llmprovider.Response {
	return &llmprovider.Response{
		Content: llmprovider.Message{
			Role:  "assistant",
			Parts: []llmprovider.Part{{Text: text}},
		},
	}
}

func makeFunctionCallResponse(toolName string, args map[string]interface{}) *llmprovider.Response {
	return &llmprovider.Response{
		Content: llmprovider.Message{
			Role: "assistant",
			Parts: []llmprovider.Part{{
				FunctionCall: &llmprovider.FunctionCall{
					Name: toolName,
					Args: args,
				},
			}},
		},
	}
}

func TestNodeAgent_FinalTextAnswer(t *testing.T) {
	llm := new(mockLLM)
	state := NewGraphState("user")
	state.Status = StatusRunning

	llm.On("GenerateContent", mock.Anything, mock.Anything).
		Return(makeTextResponse("Toi da tim thay 3 tasks."), nil)

	err := NodeAgent(context.Background(), state, llm, nil, "system prompt")

	assert.NoError(t, err)
	assert.Equal(t, StatusFinished, state.Status)
	assert.Nil(t, state.PendingTool)
	assert.Equal(t, 1, state.CurrentStep)
	assert.Len(t, state.Messages, 1)
}

func TestNodeAgent_QuestionToUser(t *testing.T) {
	llm := new(mockLLM)
	state := NewGraphState("user")
	state.Status = StatusRunning

	llm.On("GenerateContent", mock.Anything, mock.Anything).
		Return(makeTextResponse("Ban muon tao task vao ngay nao?"), nil)

	err := NodeAgent(context.Background(), state, llm, nil, "system prompt")

	assert.NoError(t, err)
	assert.Equal(t, StatusWaitingForHuman, state.Status)
	assert.Nil(t, state.PendingTool)
}

func TestNodeAgent_SafeFunctionCall(t *testing.T) {
	llm := new(mockLLM)
	state := NewGraphState("user")
	state.Status = StatusRunning

	llm.On("GenerateContent", mock.Anything, mock.Anything).
		Return(makeFunctionCallResponse("search_tasks", map[string]interface{}{"query": "PR 123"}), nil)

	err := NodeAgent(context.Background(), state, llm, nil, "system prompt")

	assert.NoError(t, err)
	assert.Equal(t, StatusRunning, state.Status) // safe tool → RUNNING
	assert.NotNil(t, state.PendingTool)
	assert.Equal(t, "search_tasks", state.PendingTool.Name)
}

func TestNodeAgent_DangerousFunctionCall(t *testing.T) {
	llm := new(mockLLM)
	state := NewGraphState("user")
	state.Status = StatusRunning

	llm.On("GenerateContent", mock.Anything, mock.Anything).
		Return(makeFunctionCallResponse("delete_all_tasks", map[string]interface{}{}), nil)

	err := NodeAgent(context.Background(), state, llm, nil, "system prompt")

	assert.NoError(t, err)
	assert.Equal(t, StatusWaitingForHuman, state.Status) // dangerous → WAITING
	assert.NotNil(t, state.PendingTool)
	assert.Equal(t, "delete_all_tasks", state.PendingTool.Name)
}

func TestNodeAgent_EmptyResponse(t *testing.T) {
	llm := new(mockLLM)
	state := NewGraphState("user")
	state.Status = StatusRunning

	llm.On("GenerateContent", mock.Anything, mock.Anything).
		Return(&llmprovider.Response{
			Content: llmprovider.Message{Role: "assistant", Parts: []llmprovider.Part{}},
		}, nil)

	err := NodeAgent(context.Background(), state, llm, nil, "system prompt")

	assert.ErrorIs(t, err, ErrEmptyResponse)
	assert.Equal(t, StatusError, state.Status)
}

func TestNodeAgent_LLMError(t *testing.T) {
	llm := new(mockLLM)
	state := NewGraphState("user")
	state.Status = StatusRunning

	llm.On("GenerateContent", mock.Anything, mock.Anything).
		Return(nil, errors.New("LLM connection failed"))

	err := NodeAgent(context.Background(), state, llm, nil, "system prompt")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "LLM connection failed")
	assert.Equal(t, StatusError, state.Status)
}

func TestNodeAgent_IncrementsStep(t *testing.T) {
	llm := new(mockLLM)
	state := NewGraphState("user")
	state.CurrentStep = 3

	llm.On("GenerateContent", mock.Anything, mock.Anything).
		Return(makeTextResponse("Done."), nil)

	NodeAgent(context.Background(), state, llm, nil, "system prompt")

	assert.Equal(t, 4, state.CurrentStep)
}

func TestIsDangerousOperation(t *testing.T) {
	dangerous := []string{"delete_task", "delete_all_tasks", "complete_all", "bulk_delete"}
	safe := []string{"search_tasks", "create_task", "check_calendar", "get_task"}

	for _, name := range dangerous {
		assert.True(t, isDangerousOperation(name), "expected %q to be dangerous", name)
	}
	for _, name := range safe {
		assert.False(t, isDangerousOperation(name), "expected %q to be safe", name)
	}
}

func TestIsAskingUser(t *testing.T) {
	questions := []string{
		"Ban muon tao task vao ngay nao?",
		"Vui long cho biet thoi gian?",
		"What time do you want?",
		"Ngay nao ban ranh?",
	}
	answers := []string{
		"Toi da tim thay 3 tasks.",
		"Da tao task thanh cong.",
		"Khong co task nao.",
		"Done.", // Khong co dau hoi → false
	}

	for _, q := range questions {
		assert.True(t, isAskingUser(q), "expected %q to be a question", q)
	}
	for _, a := range answers {
		assert.False(t, isAskingUser(a), "expected %q not to be a question", a)
	}
}
