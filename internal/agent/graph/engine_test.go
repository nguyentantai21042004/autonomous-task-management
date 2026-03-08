package graph

import (
	"context"
	"testing"

	"autonomous-task-management/internal/agent"
	"autonomous-task-management/pkg/llmprovider"
	pkgLog "autonomous-task-management/pkg/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newTestEngine(llm llmprovider.IManager, registry *agent.ToolRegistry) *Engine {
	l := pkgLog.Init(pkgLog.ZapConfig{Level: "error", Mode: "development"})
	return NewEngine(llm, registry, l, "You are a helpful assistant.")
}

// ---------------------------------------------------------------------------
// Engine.Run tests
// ---------------------------------------------------------------------------

func TestEngine_Run_AlreadyFinished(t *testing.T) {
	llm := new(mockLLM)
	registry := agent.NewToolRegistry()
	engine := newTestEngine(llm, registry)

	state := NewGraphState("user")
	state.Status = StatusFinished

	err := engine.Run(context.Background(), state)

	assert.NoError(t, err)
	assert.Equal(t, StatusFinished, state.Status)
	llm.AssertNotCalled(t, "GenerateContent")
}

func TestEngine_Run_DirectTextAnswer(t *testing.T) {
	llm := new(mockLLM)
	registry := agent.NewToolRegistry()
	engine := newTestEngine(llm, registry)

	llm.On("GenerateContent", mock.Anything, mock.Anything).
		Return(makeTextResponse("Da tim thay 3 tasks."), nil)

	state := NewGraphState("user")
	state.Status = StatusRunning
	state.AppendMessage(llmprovider.Message{Role: "user", Parts: []llmprovider.Part{{Text: "tim kiem tasks"}}})

	err := engine.Run(context.Background(), state)

	assert.NoError(t, err)
	assert.Equal(t, StatusFinished, state.Status)
	llm.AssertNumberOfCalls(t, "GenerateContent", 1)
}

func TestEngine_Run_ToolCallThenAnswer(t *testing.T) {
	llm := new(mockLLM)
	registry := agent.NewToolRegistry()
	registry.Register(&mockAgentTool{name: "search_tasks", result: []string{"task1"}})
	engine := newTestEngine(llm, registry)

	// LLM call 1: muon goi tool
	llm.On("GenerateContent", mock.Anything, mock.Anything).
		Return(makeFunctionCallResponse("search_tasks", map[string]interface{}{"query": "meeting"}), nil).Once()

	// LLM call 2: tra loi cuoi
	llm.On("GenerateContent", mock.Anything, mock.Anything).
		Return(makeTextResponse("Tim thay task meeting ngay mai."), nil).Once()

	state := NewGraphState("user")
	state.Status = StatusRunning
	state.AppendMessage(llmprovider.Message{Role: "user", Parts: []llmprovider.Part{{Text: "tim meeting"}}})

	err := engine.Run(context.Background(), state)

	assert.NoError(t, err)
	assert.Equal(t, StatusFinished, state.Status)
	llm.AssertNumberOfCalls(t, "GenerateContent", 2)

	// Response cuoi chinh xac
	response := engine.GetLastResponse(state)
	assert.Contains(t, response, "meeting")
}

func TestEngine_Run_PauseWhenAskingUser(t *testing.T) {
	llm := new(mockLLM)
	registry := agent.NewToolRegistry()
	engine := newTestEngine(llm, registry)

	llm.On("GenerateContent", mock.Anything, mock.Anything).
		Return(makeTextResponse("Ban muon tao task vao ngay nao?"), nil)

	state := NewGraphState("user")
	state.Status = StatusRunning
	state.AppendMessage(llmprovider.Message{Role: "user", Parts: []llmprovider.Part{{Text: "tao task"}}})

	err := engine.Run(context.Background(), state)

	assert.NoError(t, err)
	assert.Equal(t, StatusWaitingForHuman, state.Status)
	llm.AssertNumberOfCalls(t, "GenerateContent", 1)
}

func TestEngine_Run_PauseAndResume(t *testing.T) {
	llm := new(mockLLM)
	registry := agent.NewToolRegistry()
	engine := newTestEngine(llm, registry)

	// Turn 1: bot hoi ngay
	llm.On("GenerateContent", mock.Anything, mock.Anything).
		Return(makeTextResponse("Ban muon hop vao ngay nao?"), nil).Once()

	state := NewGraphState("user")
	state.Status = StatusRunning
	state.AppendMessage(llmprovider.Message{Role: "user", Parts: []llmprovider.Part{{Text: "tao lich hop SMAP"}}})

	err := engine.Run(context.Background(), state)
	assert.NoError(t, err)
	assert.Equal(t, StatusWaitingForHuman, state.Status)

	// Simulate user reply
	state.Status = StatusRunning
	state.AppendMessage(llmprovider.Message{Role: "user", Parts: []llmprovider.Part{{Text: "Thu 2 tuan sau"}}})

	// Turn 2: sau khi co du info → tra loi xong
	llm.On("GenerateContent", mock.Anything, mock.Anything).
		Return(makeTextResponse("Da tao lich hop thu 2 tuan sau."), nil).Once()

	err = engine.Run(context.Background(), state)
	assert.NoError(t, err)
	assert.Equal(t, StatusFinished, state.Status)

	response := engine.GetLastResponse(state)
	assert.Contains(t, response, "tuan sau")
}

func TestEngine_Run_MaxSteps(t *testing.T) {
	llm := new(mockLLM)
	registry := agent.NewToolRegistry()
	// Tool khong ton tai → append error → RUNNING → loop tiep
	engine := newTestEngine(llm, registry)

	// Luon tra ve FunctionCall den tool khong ton tai
	llm.On("GenerateContent", mock.Anything, mock.Anything).
		Return(makeFunctionCallResponse("ghost_tool", map[string]interface{}{}), nil)

	state := NewGraphState("user")
	state.Status = StatusRunning
	state.AppendMessage(llmprovider.Message{Role: "user", Parts: []llmprovider.Part{{Text: "test"}}})

	err := engine.Run(context.Background(), state)

	assert.NoError(t, err)
	// Sau MaxGraphSteps, engine force FINISHED
	assert.Equal(t, StatusFinished, state.Status)
}

func TestEngine_Run_ErrorState(t *testing.T) {
	llm := new(mockLLM)
	registry := agent.NewToolRegistry()
	engine := newTestEngine(llm, registry)

	state := NewGraphState("user")
	state.Status = StatusError

	err := engine.Run(context.Background(), state)

	assert.NoError(t, err)
	llm.AssertNotCalled(t, "GenerateContent")
}

// ---------------------------------------------------------------------------
// Engine.GetLastResponse tests
// ---------------------------------------------------------------------------

func TestEngine_GetLastResponse_Empty(t *testing.T) {
	engine := newTestEngine(new(mockLLM), agent.NewToolRegistry())
	state := NewGraphState("user")

	resp := engine.GetLastResponse(state)
	assert.Empty(t, resp)
}

func TestEngine_GetLastResponse_OnlyFunctionMessages(t *testing.T) {
	engine := newTestEngine(new(mockLLM), agent.NewToolRegistry())
	state := NewGraphState("user")
	state.Messages = []llmprovider.Message{
		{Role: "function", Parts: []llmprovider.Part{{FunctionResponse: &llmprovider.FunctionResponse{Name: "tool"}}}},
	}

	resp := engine.GetLastResponse(state)
	assert.Empty(t, resp)
}

func TestEngine_GetLastResponse_ReturnsLastAssistantText(t *testing.T) {
	engine := newTestEngine(new(mockLLM), agent.NewToolRegistry())
	state := NewGraphState("user")
	state.Messages = []llmprovider.Message{
		{Role: "assistant", Parts: []llmprovider.Part{{Text: "First response"}}},
		{Role: "function", Parts: []llmprovider.Part{{FunctionResponse: &llmprovider.FunctionResponse{}}}},
		{Role: "assistant", Parts: []llmprovider.Part{{Text: "Final response"}}},
	}

	resp := engine.GetLastResponse(state)
	assert.Equal(t, "Final response", resp)
}

func TestEngine_GetLastResponse_SkipsEmptyText(t *testing.T) {
	engine := newTestEngine(new(mockLLM), agent.NewToolRegistry())
	state := NewGraphState("user")
	state.Messages = []llmprovider.Message{
		{Role: "assistant", Parts: []llmprovider.Part{{Text: "Real response"}}},
		// Assistant message voi FunctionCall (khong co text)
		{Role: "assistant", Parts: []llmprovider.Part{{FunctionCall: &llmprovider.FunctionCall{Name: "tool"}}}},
	}

	resp := engine.GetLastResponse(state)
	assert.Equal(t, "Real response", resp)
}
