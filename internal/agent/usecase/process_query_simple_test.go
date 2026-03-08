package usecase

import (
	"context"
	"errors"
	"testing"

	"autonomous-task-management/internal/agent"
	"autonomous-task-management/internal/agent/graph"
	"autonomous-task-management/internal/model"
	"autonomous-task-management/pkg/llmprovider"
	pkgLog "autonomous-task-management/pkg/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// helper: tao implUseCase truc tiep de test ma khong qua New()
func newTestImplUseCase(llm *MockLLMManager, registry *agent.ToolRegistry) *implUseCase {
	l := pkgLog.Init(pkgLog.ZapConfig{Level: "error", Mode: "development"})
	uc := New(llm, registry, l, "Asia/Ho_Chi_Minh")
	return uc.(*implUseCase)
}

func makeAssistantResp(text string) *llmprovider.Response {
	return &llmprovider.Response{
		Content: llmprovider.Message{
			Role:  "assistant",
			Parts: []llmprovider.Part{{Text: text}},
		},
	}
}

func makeFuncCallResp(toolName string) *llmprovider.Response {
	return &llmprovider.Response{
		Content: llmprovider.Message{
			Role: "assistant",
			Parts: []llmprovider.Part{{
				FunctionCall: &llmprovider.FunctionCall{Name: toolName, Args: map[string]interface{}{}},
			}},
		},
	}
}

// ---------------------------------------------------------------------------
// Basic functionality
// ---------------------------------------------------------------------------

func TestProcessQuery_DirectAnswer(t *testing.T) {
	mockLLM := new(MockLLMManager)
	uc := newTestImplUseCase(mockLLM, agent.NewToolRegistry())

	mockLLM.On("GenerateContent", mock.Anything, mock.Anything).
		Return(makeAssistantResp("Xin chao! Toi la tro ly."), nil)

	resp, err := uc.ProcessQuery(context.Background(), model.Scope{UserID: "u1"}, "Xin chao")

	assert.NoError(t, err)
	assert.Contains(t, resp, "Xin chao")
}

func TestProcessQuery_ToolCallThenAnswer(t *testing.T) {
	mockLLM := new(MockLLMManager)
	registry := agent.NewToolRegistry()
	registry.Register(&MockTool{name: "search_tasks"})
	uc := newTestImplUseCase(mockLLM, registry)

	mockLLM.On("GenerateContent", mock.Anything, mock.Anything).
		Return(makeFuncCallResp("search_tasks"), nil).Once()
	mockLLM.On("GenerateContent", mock.Anything, mock.Anything).
		Return(makeAssistantResp("Tim thay 2 tasks."), nil).Once()

	resp, err := uc.ProcessQuery(context.Background(), model.Scope{UserID: "u1"}, "tim kiem")

	assert.NoError(t, err)
	assert.Contains(t, resp, "Tim thay")
	mockLLM.AssertNumberOfCalls(t, "GenerateContent", 2)
}

func TestProcessQuery_LLMError(t *testing.T) {
	mockLLM := new(MockLLMManager)
	uc := newTestImplUseCase(mockLLM, agent.NewToolRegistry())

	mockLLM.On("GenerateContent", mock.Anything, mock.Anything).
		Return(nil, errors.New("network error"))

	resp, err := uc.ProcessQuery(context.Background(), model.Scope{UserID: "u1"}, "test")

	assert.Error(t, err)
	assert.Empty(t, resp)
}

func TestProcessQuery_EmptyResponse(t *testing.T) {
	mockLLM := new(MockLLMManager)
	uc := newTestImplUseCase(mockLLM, agent.NewToolRegistry())

	mockLLM.On("GenerateContent", mock.Anything, mock.Anything).
		Return(&llmprovider.Response{
			Content: llmprovider.Message{Role: "assistant", Parts: []llmprovider.Part{}},
		}, nil)

	resp, err := uc.ProcessQuery(context.Background(), model.Scope{UserID: "u1"}, "test")

	assert.Error(t, err)
	assert.Empty(t, resp)
}

func TestProcessQuery_MaxSteps(t *testing.T) {
	mockLLM := new(MockLLMManager)
	uc := newTestImplUseCase(mockLLM, agent.NewToolRegistry())

	// Tool khong ton tai → loop tiep → max steps
	mockLLM.On("GenerateContent", mock.Anything, mock.Anything).
		Return(makeFuncCallResp("ghost_tool"), nil)

	_, err := uc.ProcessQuery(context.Background(), model.Scope{UserID: "u1"}, "test")

	// Max steps khong phai loi fatal, engine force FINISHED
	assert.NoError(t, err)
	mockLLM.AssertNumberOfCalls(t, "GenerateContent", graph.MaxGraphSteps)
}

// ---------------------------------------------------------------------------
// Session management
// ---------------------------------------------------------------------------

func TestProcessQuery_SessionPersistsAcrossCalls(t *testing.T) {
	mockLLM := new(MockLLMManager)
	uc := newTestImplUseCase(mockLLM, agent.NewToolRegistry())
	sc := model.Scope{UserID: "u_session"}

	mockLLM.On("GenerateContent", mock.Anything, mock.Anything).
		Return(makeAssistantResp("Turn 1"), nil).Once()
	mockLLM.On("GenerateContent", mock.Anything, mock.Anything).
		Return(makeAssistantResp("Turn 2"), nil).Once()

	uc.ProcessQuery(context.Background(), sc, "message 1")
	uc.ProcessQuery(context.Background(), sc, "message 2")

	// Session phai co ca 2 turns
	messages := uc.GetSessionMessages(sc.UserID)
	assert.NotNil(t, messages)
	assert.Greater(t, len(messages), 2)
}

func TestClearSession(t *testing.T) {
	mockLLM := new(MockLLMManager)
	uc := newTestImplUseCase(mockLLM, agent.NewToolRegistry())
	sc := model.Scope{UserID: "u_clear"}

	mockLLM.On("GenerateContent", mock.Anything, mock.Anything).
		Return(makeAssistantResp("Hello"), nil)

	uc.ProcessQuery(context.Background(), sc, "hello")
	assert.NotNil(t, uc.GetSessionMessages(sc.UserID))

	uc.ClearSession(sc.UserID)
	assert.Nil(t, uc.GetSessionMessages(sc.UserID))
}

func TestGetSessionMessages_EmptyForNewUser(t *testing.T) {
	uc := newTestImplUseCase(new(MockLLMManager), agent.NewToolRegistry())
	assert.Nil(t, uc.GetSessionMessages("brand_new_user"))
}

// ---------------------------------------------------------------------------
// Pause & Resume (V2.0 core feature)
// ---------------------------------------------------------------------------

func TestProcessQuery_PauseAndResume(t *testing.T) {
	mockLLM := new(MockLLMManager)
	uc := newTestImplUseCase(mockLLM, agent.NewToolRegistry())
	sc := model.Scope{UserID: "u_pause"}

	// Turn 1: LLM hoi ngay gio
	mockLLM.On("GenerateContent", mock.Anything, mock.Anything).
		Return(makeAssistantResp("Ban muon tao task vao ngay nao?"), nil).Once()

	resp1, err := uc.ProcessQuery(context.Background(), sc, "tao task review code")
	assert.NoError(t, err)
	assert.Contains(t, resp1, "ngay nao")

	// State phai la WAITING_FOR_HUMAN
	state, ok := uc.stateCache.Get(sc.UserID)
	assert.True(t, ok)
	assert.Equal(t, graph.StatusWaitingForHuman, state.Status)

	// Turn 2: user reply → agent resume, tra loi cuoi
	mockLLM.On("GenerateContent", mock.Anything, mock.Anything).
		Return(makeAssistantResp("Da tao task review code vao ngay mai."), nil).Once()

	resp2, err := uc.ProcessQuery(context.Background(), sc, "Ngay mai")
	assert.NoError(t, err)
	assert.Contains(t, resp2, "ngay mai")

	// State phai la FINISHED
	state, _ = uc.stateCache.Get(sc.UserID)
	assert.Equal(t, graph.StatusFinished, state.Status)
}

func TestProcessQuery_DangerousOpConfirm(t *testing.T) {
	mockLLM := new(MockLLMManager)
	uc := newTestImplUseCase(mockLLM, agent.NewToolRegistry())
	sc := model.Scope{UserID: "u_danger"}

	// Turn 1: LLM muon goi delete_all_tasks → WAITING_FOR_HUMAN
	mockLLM.On("GenerateContent", mock.Anything, mock.Anything).
		Return(makeFuncCallResp("delete_all_tasks"), nil).Once()

	uc.ProcessQuery(context.Background(), sc, "xoa tat ca tasks")

	state, _ := uc.stateCache.Get(sc.UserID)
	assert.Equal(t, graph.StatusWaitingForHuman, state.Status)
	assert.NotNil(t, state.PendingTool)

	// Turn 2: user xac nhan → chay tiep
	mockLLM.On("GenerateContent", mock.Anything, mock.Anything).
		Return(makeAssistantResp("Da xoa tat ca tasks."), nil).Once()

	resp, err := uc.ProcessQuery(context.Background(), sc, "ok")
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestProcessQuery_DangerousOpCancel(t *testing.T) {
	mockLLM := new(MockLLMManager)
	uc := newTestImplUseCase(mockLLM, agent.NewToolRegistry())
	sc := model.Scope{UserID: "u_cancel"}

	// Turn 1: LLM muon goi delete_all_tasks → WAITING
	mockLLM.On("GenerateContent", mock.Anything, mock.Anything).
		Return(makeFuncCallResp("delete_all_tasks"), nil).Once()

	uc.ProcessQuery(context.Background(), sc, "xoa tat ca tasks")

	// Turn 2: user tu choi → huy bo
	resp, err := uc.ProcessQuery(context.Background(), sc, "thoi dung xoa")
	assert.NoError(t, err)
	assert.Contains(t, resp, "huy")

	// PendingTool phai duoc xoa
	state, _ := uc.stateCache.Get(sc.UserID)
	assert.Nil(t, state.PendingTool)
}

// ---------------------------------------------------------------------------
// isUserConfirmed helper
// ---------------------------------------------------------------------------

func TestIsUserConfirmed(t *testing.T) {
	yes := []string{"ok", "OK", "yes", "dong y", "xac nhan", "co", "duoc", "chac chan"}
	no := []string{"no", "thoi", "khong", "dung lai", "cancel", "huy"}

	for _, s := range yes {
		assert.True(t, isUserConfirmed(s), "expected %q to be confirmed", s)
	}
	for _, s := range no {
		assert.False(t, isUserConfirmed(s), "expected %q to NOT be confirmed", s)
	}
}
