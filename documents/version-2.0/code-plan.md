# 💻 CODE PLAN VERSION 2.0 - STATEFUL ORCHESTRATION (GRAPH ENGINE)

## 🚨 CRITICAL FIXES APPLIED

> **Mục tiêu**: Chuyển đổi từ ReAct Loop đơn giản sang Graph-based State Machine với 2 bản vá logic quan trọng từ System Architect Review.

---

## ⚠️ CRITICAL FIXES (Áp dụng từ Architecture Review)

### 🐞 Fix #1: Lỗi Routing khi Resume (Panic khi thức dậy)

**Vấn đề**: Khi graph pause ở `StatusWaitingForUser`, switch case trong `Run()` không có `case NodeHumanInput` → crash với "unknown node type".

**Giải pháp**: Reset state trong Telegram Handler trước khi resume.

### 🐞 Fix #2: Heuristic Trap của isAskingUser

**Vấn đề**: Dùng regex/heuristics để detect câu hỏi rất dễ vỡ. LLM có thể hỏi mà không có dấu `?`.

**Giải pháp**: Chuyển "hỏi user" thành một Tool `request_human_input` - để LLM tự quyết định khi nào cần hỏi.

---

## PHẦN 1: GRAPH FOUNDATION (Nền móng Đồ thị)

### 1.1. Graph State Definition

#### 📁 File: `internal/agent/graph/types.go` (🆕 NEW)

```go
package graph

import (
	"time"

	"autonomous-task-management/pkg/llmprovider"
)

// GraphStatus represents the current state of the graph execution
type GraphStatus string

const (
	StatusRunning        GraphStatus = "RUNNING"
	StatusWaitingForUser GraphStatus = "WAITING_FOR_USER"
	StatusFinished       GraphStatus = "FINISHED"
	StatusError          GraphStatus = "ERROR"
)

// NodeType represents different types of nodes in the graph
type NodeType string

const (
	NodeAgent NodeType = "AGENT"
	NodeTool  NodeType = "TOOL"
	// NodeHumanInput is NOT a real node - it's a state marker
	// Graph will pause and wait for handler to resume
)

// GraphState represents the complete state of a graph execution
type GraphState struct {
	UserID      string                    `json:"user_id"`
	Status      GraphStatus               `json:"status"`
	CurrentNode NodeType                  `json:"current_node"`
	StepCount   int                       `json:"step_count"`
	Messages    []llmprovider.Message     `json:"messages"`
	PendingTool *PendingToolCall          `json:"pending_tool,omitempty"`

	// 🆕 Store question when waiting for user
	PendingQuestion string `json:"pending_question,omitempty"`

	CreatedAt   time.Time `json:"created_at"`
	LastUpdated time.Time `json:"last_updated"`
	LastError   string    `json:"last_error,omitempty"`
}

// PendingToolCall represents a tool call that needs to be executed
type PendingToolCall struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}
```

---

### 1.2. Graph Engine Core (WITH FIXES)

#### 📁 File: `internal/agent/graph/engine.go` (🆕 NEW)

```go
package graph

import (
	"context"
	"fmt"
	"time"

	"autonomous-task-management/internal/agent"
	"autonomous-task-management/pkg/llmprovider"
	pkgLog "autonomous-task-management/pkg/log"
)

const (
	MaxGraphSteps = 10
)

type Engine struct {
	llm      llmprovider.IManager
	registry *agent.ToolRegistry
	l        pkgLog.Logger
	timezone string
}

// Run executes the graph until it reaches a terminal state
// 🔧 FIX #1: Only handle NodeAgent and NodeTool in switch
// NodeHumanInput is handled by Telegram Handler before calling Run()
func (e *Engine) Run(ctx context.Context, state *GraphState) (*GraphState, error) {
	e.l.Infof(ctx, "graph: Starting execution for user %s (status: %s)", state.UserID, state.Status)

	// Main graph loop
	for state.Status == StatusRunning && state.StepCount < MaxGraphSteps {
		e.l.Infof(ctx, "graph: Step %d - Current node: %s", state.StepCount, state.CurrentNode)

		var err error
		switch state.CurrentNode {
		case NodeAgent:
			state, err = e.executeAgentNode(ctx, state)
		case NodeTool:
			state, err = e.executeToolNode(ctx, state)
		default:
			// This should never happen if handler resets state correctly
			return state, fmt.Errorf("unknown node type: %s (handler must reset before resume)", state.CurrentNode)
		}

		if err != nil {
			state.Status = StatusError
			state.LastError = err.Error()
			e.l.Errorf(ctx, "graph: Execution failed at step %d: %v", state.StepCount, err)
			return state, err
		}

		state.StepCount++
		state.LastUpdated = time.Now()
	}

	if state.StepCount >= MaxGraphSteps && state.Status == StatusRunning {
		state.Status = StatusError
		state.LastError = "exceeded maximum graph steps"
		e.l.Warnf(ctx, "graph: Exceeded max steps (%d) for user %s", MaxGraphSteps, state.UserID)
	}

	e.l.Infof(ctx, "graph: Execution completed with status: %s", state.Status)
	return state, nil
}

// executeAgentNode calls LLM to decide next action
// 🔧 FIX #2: Remove isAskingUser heuristic, use request_human_input tool instead
func (e *Engine) executeAgentNode(ctx context.Context, state *GraphState) (*GraphState, error) {
	e.l.Infof(ctx, "graph: Executing Agent node")

	// Build system prompt
	systemPrompt := e.buildSystemPrompt()

	// Call LLM
	resp, err := e.llm.GenerateContent(ctx, &llmprovider.Request{
		Messages:          state.Messages,
		Tools:             e.registry.ToFunctionDefinitions(),
		SystemInstruction: systemPrompt,
	})
	if err != nil {
		return state, fmt.Errorf("agent node: LLM call failed: %w", err)
	}

	// Add assistant response to history
	state.Messages = append(state.Messages, llmprovider.Message{
		Role:  "assistant",
		Parts: []llmprovider.Part{{Text: resp.Content.Parts[0].Text}},
	})

	// Check if LLM wants to call a tool
	if len(resp.Content.Parts) > 0 && resp.Content.Parts[0].FunctionCall != nil {
		fc := resp.Content.Parts[0].FunctionCall
		e.l.Infof(ctx, "graph: Agent decided to call tool: %s", fc.Name)

		// 🔧 FIX #2: Special handling for request_human_input tool
		if fc.Name == "request_human_input" {
			question, ok := fc.Args["question"].(string)
			if !ok {
				return state, fmt.Errorf("request_human_input: missing 'question' parameter")
			}

			e.l.Infof(ctx, "graph: Agent requesting human input: %s", question)
			state.PendingQuestion = question
			state.Status = StatusWaitingForUser
			// Keep CurrentNode as NodeAgent so handler knows to resume here
			return state, nil
		}

		// Regular tool call
		state.PendingTool = &PendingToolCall{
			Name:      fc.Name,
			Arguments: fc.Args,
		}
		state.CurrentNode = NodeTool
		state.Status = StatusRunning
		return state, nil
	}

	// LLM provided final answer (no tool call, no question)
	e.l.Infof(ctx, "graph: Agent provided final answer")
	state.Status = StatusFinished
	return state, nil
}

// executeToolNode executes the pending tool call
func (e *Engine) executeToolNode(ctx context.Context, state *GraphState) (*GraphState, error) {
	if state.PendingTool == nil {
		return state, fmt.Errorf("tool node: no pending tool call")
	}

	e.l.Infof(ctx, "graph: Executing tool: %s", state.PendingTool.Name)

	// Get tool from registry
	tool := e.registry.Get(state.PendingTool.Name)
	if tool == nil {
		return state, fmt.Errorf("tool node: tool not found: %s", state.PendingTool.Name)
	}

	// Execute tool
	result, err := tool.Execute(ctx, state.PendingTool.Arguments)
	if err != nil {
		e.l.Errorf(ctx, "graph: Tool execution failed: %v", err)
		state.Messages = append(state.Messages, llmprovider.Message{
			Role: "function",
			Parts: []llmprovider.Part{{
				FunctionResponse: &llmprovider.FunctionResponse{
					Name:     state.PendingTool.Name,
					Response: map[string]interface{}{"error": err.Error()},
				},
			}},
		})
	} else {
		state.Messages = append(state.Messages, llmprovider.Message{
			Role: "function",
			Parts: []llmprovider.Part{{
				FunctionResponse: &llmprovider.FunctionResponse{
					Name:     state.PendingTool.Name,
					Response: result,
				},
			}},
		})
	}

	// Clear pending tool and go back to agent
	state.PendingTool = nil
	state.CurrentNode = NodeAgent
	state.Status = StatusRunning

	return state, nil
}

// buildSystemPrompt creates graph-aware system instructions
func (e *Engine) buildSystemPrompt() string {
	return `Bạn là AI Assistant thông minh với khả năng sử dụng tools.

QUY TẮC QUAN TRỌNG:
1. Khi thiếu thông tin (ngày giờ, xác nhận), GỌI TOOL "request_human_input" với câu hỏi cụ thể
2. KHÔNG BAO GIỜ tự đoán thông tin quan trọng (ngày tháng, số tiền, tên người)
3. Với hành động nguy hiểm (xóa, sửa), LUÔN hỏi xác nhận qua "request_human_input"

TOOLS AVAILABLE:
- search_tasks: Tìm kiếm tasks
- check_calendar: Kiểm tra lịch
- update_checklist_item: Cập nhật checkbox
- request_human_input: Hỏi user khi thiếu thông tin (QUAN TRỌNG!)

VÍ DỤ SỬ DỤNG request_human_input:
User: "Tạo task họp"
→ Gọi request_human_input(question="Bạn muốn họp lúc mấy giờ và ngày nào?")

User: "Xóa tất cả tasks"
→ Gọi request_human_input(question="Bạn chắc chắn muốn xóa TẤT CẢ tasks? Nhập 'XÁC NHẬN' để tiếp tục.")
`
}
```

---

### 1.3. Request Human Input Tool (THE MAGIC TOOL)

#### 📁 File: `internal/agent/tools/request_human_input.go` (🆕 NEW)

```go
package tools

import (
	"context"

	"autonomous-task-management/internal/agent"
)

// RequestHumanInputTool is a special tool that pauses execution to ask user
// This is NOT a real tool - it's a signal to the graph engine
type RequestHumanInputTool struct{}

// NewRequestHumanInput creates the request_human_input tool
func NewRequestHumanInput() agent.Tool {
	return &RequestHumanInputTool{}
}

func (t *RequestHumanInputTool) Name() string {
	return "request_human_input"
}

func (t *RequestHumanInputTool) Description() string {
	return `Sử dụng tool này khi bạn THIẾU THÔNG TIN và cần hỏi người dùng.

QUAN TRỌNG: Đây là cách DUY NHẤT để hỏi user trong graph execution.

KHI NÀO SỬ DỤNG:
- Thiếu ngày giờ cụ thể (user nói "tạo task họp" nhưng không nói khi nào)
- Cần xác nhận hành động nguy hiểm (xóa, sửa nhiều tasks)
- Thông tin mơ hồ cần làm rõ

VÍ DỤ:
User: "Tạo task deadline dự án"
→ request_human_input(question="Deadline dự án vào ngày nào?")

User: "Xóa tất cả tasks hoàn thành"
→ request_human_input(question="Bạn chắc chắn muốn xóa? Nhập 'OK' để xác nhận.")
`
}

func (t *RequestHumanInputTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"question": map[string]interface{}{
				"type":        "string",
				"description": "Câu hỏi cần hỏi người dùng. Phải rõ ràng và cụ thể.",
			},
		},
		"required": []string{"question"},
	}
}

// Execute is never actually called - this tool is handled specially in executeAgentNode
func (t *RequestHumanInputTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// This should never be called because executeAgentNode intercepts it
	return map[string]string{
		"status": "This tool is handled specially by the graph engine",
	}, nil
}
```

---

## PHẦN 2: TELEGRAM HANDLER INTEGRATION (WITH FIX #1)

### 2.1. Update Handler to Support Graph

#### 📁 File: `internal/task/delivery/telegram/handler.go` (UPDATE)

```go
// 🔧 FIX #1: Proper state reset when resuming from WAITING_FOR_USER

func (h *handler) processMessage(ctx context.Context, msg *pkgTelegram.Message) error {
	sc := model.Scope{UserID: fmt.Sprintf("telegram_%d", msg.From.ID)}

	// Handle slash commands first (backward compatibility)
	switch {
	case msg.Text == "/start":
		return h.handleStart(ctx, msg.Chat.ID)
	case msg.Text == "/help":
		return h.handleHelp(ctx, msg.Chat.ID)
	case msg.Text == "/reset":
		h.graphEngine.ClearState(sc.UserID)
		return h.bot.SendMessage(msg.Chat.ID, "✅ Đã xóa lịch sử. Bắt đầu lại từ đầu!")
	// ... other commands
	}

	// Get or create graph state
	state := h.graphEngine.GetState(sc.UserID)
	if state == nil {
		// New conversation
		state = &graph.GraphState{
			UserID:      sc.UserID,
			Status:      graph.StatusRunning,
			CurrentNode: graph.NodeAgent,
			StepCount:   0,
			Messages:    []llmprovider.Message{},
			CreatedAt:   time.Now(),
			LastUpdated: time.Now(),
		}
	}

	// 🔧 FIX #1: Handle resume from WAITING_FOR_USER
	if state.Status == graph.StatusWaitingForUser {
		h.l.Infof(ctx, "handler: Resuming graph from WAITING_FOR_USER state")

		// Add user's answer to conversation
		state.Messages = append(state.Messages, llmprovider.Message{
			Role:  "user",
			Parts: []llmprovider.Part{{Text: msg.Text}},
		})

		// 🚨 CRITICAL: Reset state to resume execution
		state.CurrentNode = graph.NodeAgent
		state.Status = graph.StatusRunning
		state.PendingQuestion = "" // Clear the question

		h.l.Infof(ctx, "handler: State reset - CurrentNode: %s, Status: %s", state.CurrentNode, state.Status)
	} else {
		// New message in ongoing conversation
		state.Messages = append(state.Messages, llmprovider.Message{
			Role:  "user",
			Parts: []llmprovider.Part{{Text: msg.Text}},
		})
		state.Status = graph.StatusRunning
		state.CurrentNode = graph.NodeAgent
	}

	// Run graph engine
	state, err := h.graphEngine.Run(ctx, state)
	if err != nil {
		h.l.Errorf(ctx, "handler: Graph execution failed: %v", err)
		return h.bot.SendMessage(msg.Chat.ID, fmt.Sprintf("❌ Lỗi: %v", err))
	}

	// Save state back to cache
	h.graphEngine.SaveState(state)

	// Send response based on final status
	return h.sendGraphResponse(ctx, msg.Chat.ID, state)
}

// sendGraphResponse sends appropriate message based on graph state
func (h *handler) sendGraphResponse(ctx context.Context, chatID int64, state *graph.GraphState) error {
	switch state.Status {
	case graph.StatusFinished:
		// Get last assistant message
		lastMsg := h.getLastAssistantMessage(state.Messages)
		return h.bot.SendMessageHTML(chatID, lastMsg)

	case graph.StatusWaitingForUser:
		// Send the question LLM asked
		if state.PendingQuestion != "" {
			return h.bot.SendMessageHTML(chatID, state.PendingQuestion)
		}
		// Fallback: send last assistant message
		lastMsg := h.getLastAssistantMessage(state.Messages)
		return h.bot.SendMessageHTML(chatID, lastMsg)

	case graph.StatusError:
		return h.bot.SendMessage(chatID, fmt.Sprintf("❌ Lỗi: %s", state.LastError))

	default:
		return h.bot.SendMessage(chatID, "⚠️ Trạng thái không xác định")
	}
}

func (h *handler) getLastAssistantMessage(messages []llmprovider.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "assistant" && len(messages[i].Parts) > 0 {
			return messages[i].Parts[0].Text
		}
	}
	return "Không có phản hồi"
}
```

---

## PHẦN 3: GRAPH ENGINE FACTORY & CACHE

### 3.1. Engine Factory

#### 📁 File: `internal/agent/graph/new.go` (🆕 NEW)

```go
package graph

import (
	"sync"
	"time"

	"autonomous-task-management/internal/agent"
	"autonomous-task-management/pkg/llmprovider"
	pkgLog "autonomous-task-management/pkg/log"
)

// New creates a new Graph Engine with state cache
func New(
	llm llmprovider.IManager,
	registry *agent.ToolRegistry,
	l pkgLog.Logger,
	timezone string,
	cacheTTL time.Duration,
) *EngineWithCache {
	engine := &Engine{
		llm:      llm,
		registry: registry,
		l:        l,
		timezone: timezone,
	}

	ewc := &EngineWithCache{
		engine:      engine,
		stateCache:  make(map[string]*GraphState),
		cacheMutex:  sync.RWMutex{},
		cacheTTL:    cacheTTL,
		l:           l,
	}

	// Start cleanup goroutine
	go ewc.cleanupExpiredStates()

	return ewc
}

// EngineWithCache wraps Engine with state caching
type EngineWithCache struct {
	engine      *Engine
	stateCache  map[string]*GraphState
	cacheMutex  sync.RWMutex
	cacheTTL    time.Duration
	l           pkgLog.Logger
}

// Run executes graph with the engine
func (ewc *EngineWithCache) Run(ctx context.Context, state *GraphState) (*GraphState, error) {
	return ewc.engine.Run(ctx, state)
}

// GetState retrieves state from cache
func (ewc *EngineWithCache) GetState(userID string) *GraphState {
	ewc.cacheMutex.RLock()
	defer ewc.cacheMutex.RUnlock()

	state, exists := ewc.stateCache[userID]
	if !exists {
		return nil
	}

	return state
}

// SaveState saves state to cache
func (ewc *EngineWithCache) SaveState(state *GraphState) {
	ewc.cacheMutex.Lock()
	defer ewc.cacheMutex.Unlock()

	state.LastUpdated = time.Now()
	ewc.stateCache[state.UserID] = state
}

// ClearState removes state from cache
func (ewc *EngineWithCache) ClearState(userID string) {
	ewc.cacheMutex.Lock()
	defer ewc.cacheMutex.Unlock()

	delete(ewc.stateCache, userID)
}

// cleanupExpiredStates runs periodically to remove old states
func (ewc *EngineWithCache) cleanupExpiredStates() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		ewc.cacheMutex.Lock()
		now := time.Now()
		removed := 0

		for userID, state := range ewc.stateCache {
			if now.Sub(state.LastUpdated) > ewc.cacheTTL {
				delete(ewc.stateCache, userID)
				removed++
			}
		}

		ewc.cacheMutex.Unlock()

		if removed > 0 {
			ewc.l.Infof(context.Background(), "graph: Cleaned up %d expired states", removed)
		}
	}
}
```

---

## PHẦN 4: DEPENDENCY INJECTION

### 4.1. Update main.go

#### 📁 File: `cmd/api/main.go` (UPDATE)

```go
func main() {
	// ... existing setup ...

	// Initialize LLM Manager
	llmManager := llmprovider.InitializeProviders(cfg.LLM)

	// Initialize Tool Registry
	toolRegistry := agent.NewToolRegistry()

	// 🆕 Register request_human_input tool FIRST (most important)
	toolRegistry.Register(tools.NewRequestHumanInput())

	// Register other tools
	toolRegistry.Register(tools.NewSearchTasks(taskUC))
	toolRegistry.Register(tools.NewCheckCalendar(calendarClient))
	toolRegistry.Register(tools.NewGetChecklistProgress(checklistSvc, memosRepo))
	toolRegistry.Register(tools.NewUpdateChecklistItem(checklistSvc, memosRepo))

	// 🆕 Initialize Graph Engine (replaces old Orchestrator)
	graphEngine := graph.New(
		llmManager,
		toolRegistry,
		logger,
		cfg.LLM.Timezone,
		10*time.Minute, // Cache TTL
	)

	// Initialize Telegram Handler with Graph Engine
	telegramHandler := telegram.NewHandler(
		logger,
		taskUC,
		telegramBot,
		graphEngine, // 🆕 Pass graph engine instead of orchestrator
		automationUC,
		checklistSvc,
		memosRepo,
		semanticRouter,
	)

	// ... rest of setup ...
}
```

---

## PHẦN 5: TESTING STRATEGY

### 5.1. Unit Tests for Graph Engine

#### 📁 File: `internal/agent/graph/engine_test.go` (🆕 NEW)

```go
package graph_test

import (
	"context"
	"testing"
	"time"

	"autonomous-task-management/internal/agent"
	"autonomous-task-management/internal/agent/graph"
	"autonomous-task-management/pkg/llmprovider"
	pkgLog "autonomous-task-management/pkg/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Test Fix #1: Resume from WAITING_FOR_USER
func TestGraphEngine_ResumeFromWaitingForUser(t *testing.T) {
	mockLLM := new(MockLLMManager)
	registry := agent.NewToolRegistry()
	logger := pkgLog.NewLogger()

	engine := &graph.Engine{
		LLM:      mockLLM,
		Registry: registry,
		L:        logger,
		Timezone: "Asia/Ho_Chi_Minh",
	}

	// Create state that was paused
	state := &graph.GraphState{
		UserID:          "test_user",
		Status:          graph.StatusWaitingForUser,
		CurrentNode:     graph.NodeAgent, // Should be reset by handler
		StepCount:       2,
		Messages:        []llmprovider.Message{},
		PendingQuestion: "Bạn muốn họp lúc mấy giờ?",
		CreatedAt:       time.Now(),
		LastUpdated:     time.Now(),
	}

	// Simulate handler reset (THIS IS THE FIX)
	state.Status = graph.StatusRunning
	state.CurrentNode = graph.NodeAgent
	state.Messages = append(state.Messages, llmprovider.Message{
		Role:  "user",
		Parts: []llmprovider.Part{{Text: "3pm ngày mai"}},
	})

	// Mock LLM to provide final answer
	mockLLM.On("GenerateContent", mock.Anything, mock.Anything).Return(&llmprovider.Response{
		Content: llmprovider.Content{
			Parts: []llmprovider.Part{{Text: "Đã tạo lịch họp lúc 3pm ngày mai"}},
		},
	}, nil)

	// Run should complete successfully
	result, err := engine.Run(context.Background(), state)

	assert.NoError(t, err)
	assert.Equal(t, graph.StatusFinished, result.Status)
	mockLLM.AssertExpectations(t)
}

// Test Fix #2: request_human_input tool
func TestGraphEngine_RequestHumanInputTool(t *testing.T) {
	mockLLM := new(MockLLMManager)
	registry := agent.NewToolRegistry()
	registry.Register(tools.NewRequestHumanInput())
	logger := pkgLog.NewLogger()

	engine := &graph.Engine{
		LLM:      mockLLM,
		Registry: registry,
		L:        logger,
		Timezone: "Asia/Ho_Chi_Minh",
	}

	state := &graph.GraphState{
		UserID:      "test_user",
		Status:      graph.StatusRunning,
		CurrentNode: graph.NodeAgent,
		StepCount:   0,
		Messages: []llmprovider.Message{
			{Role: "user", Parts: []llmprovider.Part{{Text: "Tạo task họp"}}},
		},
		CreatedAt:   time.Now(),
		LastUpdated: time.Now(),
	}

	// Mock LLM to call request_human_input
	mockLLM.On("GenerateContent", mock.Anything, mock.Anything).Return(&llmprovider.Response{
		Content: llmprovider.Content{
			Parts: []llmprovider.Part{{
				FunctionCall: &llmprovider.FunctionCall{
					Name: "request_human_input",
					Args: map[string]interface{}{
						"question": "Bạn muốn họp lúc mấy giờ?",
					},
				},
			}},
		},
	}, nil)

	// Run should pause and wait for user
	result, err := engine.Run(context.Background(), state)

	assert.NoError(t, err)
	assert.Equal(t, graph.StatusWaitingForUser, result.Status)
	assert.Equal(t, "Bạn muốn họp lúc mấy giờ?", result.PendingQuestion)
	mockLLM.AssertExpectations(t)
}
```

---

## PHẦN 6: MIGRATION STRATEGY

### 6.1. Parallel Running (A/B Testing)

**Phase 1**: Chạy song song cả ReAct Loop và Graph Engine

```go
// In handler.go
if cfg.Features.UseGraphEngine {
	return h.handleWithGraphEngine(ctx, sc, msg)
} else {
	return h.handleWithReActLoop(ctx, sc, msg)
}
```

**Phase 2**: Sau 1 tuần testing, chuyển 100% sang Graph Engine

**Phase 3**: Remove old ReAct Loop code

---

## PHẦN 7: MILESTONES NGHIỆM THU

### 🏆 Milestone 1: "Pause & Resume" (Human-in-the-loop)

**Test Case**:

```
User: "Tạo task review code"
Bot: "Bạn muốn review code nào? Vui lòng cung cấp tên PR hoặc branch."
[User đợi 1 tiếng]
User: "PR #123"
Bot: "✅ Đã tạo task review code PR #123"
```

**Verify**:

- Graph pause ở `StatusWaitingForUser`
- State được lưu trong cache
- Resume đúng context sau 1 tiếng

### 🏆 Milestone 2: "Safe Execution" (Xác nhận trước khi xóa)

**Test Case**:

```
User: "Xóa tất cả tasks hoàn thành"
Bot: "Bạn chắc chắn muốn xóa? Nhập 'XÁC NHẬN' để tiếp tục."
User: "XÁC NHẬN"
Bot: "✅ Đã xóa 15 tasks hoàn thành"
```

**Verify**:

- LLM tự động gọi `request_human_input` cho hành động nguy hiểm
- Không xóa cho đến khi user xác nhận

### 🏆 Milestone 3: "Multi-step Workflow"

**Test Case**:

```
User: "Tạo task và đặt lịch họp về dự án SMAP"
Bot: "Bạn muốn họp lúc mấy giờ?"
User: "3pm ngày mai"
Bot: [Calls search_tasks → check_calendar → create_task → create_event]
Bot: "✅ Đã tạo task và đặt lịch họp lúc 3pm ngày mai"
```

**Verify**:

- Graph chạy qua nhiều nodes (Agent → Tool → Agent → Tool → ...)
- Không vượt quá MaxGraphSteps
- State được maintain xuyên suốt workflow

---

## 📊 SUMMARY: Key Changes

| Component             | Old (V1.2)    | New (V2.0)          | Fix Applied |
| --------------------- | ------------- | ------------------- | ----------- |
| Execution Model       | ReAct Loop    | Graph State Machine | -           |
| State Persistence     | SessionMemory | GraphState          | -           |
| Human Input Detection | Heuristics    | Tool-based          | ✅ Fix #2   |
| Resume Logic          | N/A           | Handler reset       | ✅ Fix #1   |
| Max Steps             | 5             | 10                  | -           |
| Pause/Resume          | ❌ No         | ✅ Yes              | -           |

---

**Document Version:** 2.0-FIXED  
**Last Updated:** 2026-03-03  
**Critical Fixes Applied:** 2  
**Status:** Ready for Implementation  
**Estimated Effort:** 3.5-5.5 days
