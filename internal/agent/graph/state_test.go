package graph

import (
	"testing"
	"time"

	"autonomous-task-management/pkg/llmprovider"

	"github.com/stretchr/testify/assert"
)

func TestNewGraphState(t *testing.T) {
	state := NewGraphState("user123")

	assert.Equal(t, "user123", state.UserID)
	assert.Equal(t, StatusFinished, state.Status)
	assert.Empty(t, state.Messages)
	assert.Empty(t, state.RecentTurns)
	assert.Nil(t, state.PendingTool)
	assert.Equal(t, 0, state.CurrentStep)
	assert.Equal(t, defaultStateTTL, state.TTL)
	assert.WithinDuration(t, time.Now(), state.LastUpdated, time.Second)
}

func TestGraphState_IsExpired(t *testing.T) {
	t.Run("not expired when just created", func(t *testing.T) {
		state := NewGraphState("user")
		assert.False(t, state.IsExpired())
	})

	t.Run("expired when past TTL", func(t *testing.T) {
		state := NewGraphState("user")
		state.TTL = 1 * time.Millisecond
		time.Sleep(5 * time.Millisecond)
		assert.True(t, state.IsExpired())
	})

	t.Run("not expired when TTL not reached", func(t *testing.T) {
		state := NewGraphState("user")
		state.TTL = 1 * time.Hour
		assert.False(t, state.IsExpired())
	})
}

func TestGraphState_AppendMessage(t *testing.T) {
	state := NewGraphState("user")

	msg1 := llmprovider.Message{Role: "user", Parts: []llmprovider.Part{{Text: "hello"}}}
	msg2 := llmprovider.Message{Role: "assistant", Parts: []llmprovider.Part{{Text: "world"}}}

	state.AppendMessage(msg1)
	state.AppendMessage(msg2)

	assert.Len(t, state.Messages, 2)
	assert.Len(t, state.RecentTurns, 2)
	assert.Equal(t, "user", state.Messages[0].Role)
	assert.Equal(t, "assistant", state.Messages[1].Role)
}

func TestGraphState_AppendMessage_LimitsRecentTurns(t *testing.T) {
	state := NewGraphState("user")

	// Append 8 messages (vuot qua maxRecentTurns=6)
	for i := 0; i < 8; i++ {
		state.AppendMessage(llmprovider.Message{
			Role:  "user",
			Parts: []llmprovider.Part{{Text: "msg"}},
		})
	}

	// Messages giu tat ca
	assert.Len(t, state.Messages, 8)
	// RecentTurns chi giu 6
	assert.Len(t, state.RecentTurns, maxRecentTurns)
}

func TestGraphState_TrimHistory(t *testing.T) {
	state := NewGraphState("user")

	// Them 25 messages
	for i := 0; i < 25; i++ {
		state.Messages = append(state.Messages, llmprovider.Message{
			Role:  "user",
			Parts: []llmprovider.Part{{Text: "msg"}},
		})
	}

	state.TrimHistory()

	assert.Len(t, state.Messages, maxHistoryLength)
}

func TestGraphState_TrimHistory_NoTrimWhenUnderLimit(t *testing.T) {
	state := NewGraphState("user")

	for i := 0; i < 10; i++ {
		state.Messages = append(state.Messages, llmprovider.Message{Role: "user"})
	}

	state.TrimHistory()
	assert.Len(t, state.Messages, 10)
}

func TestGraphState_Touch(t *testing.T) {
	state := NewGraphState("user")
	original := state.LastUpdated
	time.Sleep(5 * time.Millisecond)

	state.Touch()

	assert.True(t, state.LastUpdated.After(original))
}

// ---------------------------------------------------------------------------
// CompressIfNeeded tests
// ---------------------------------------------------------------------------

func TestCompressIfNeeded_NoOpWhenUnderThreshold(t *testing.T) {
	state := NewGraphState("user")
	for i := 0; i < compressionThreshold; i++ {
		state.Messages = append(state.Messages, llmprovider.Message{
			Role:  "user",
			Parts: []llmprovider.Part{{Text: "message"}},
		})
	}

	state.CompressIfNeeded()

	assert.Empty(t, state.OlderSummary)
	assert.Len(t, state.Messages, compressionThreshold)
}

func TestCompressIfNeeded_CompressesWhenOverThreshold(t *testing.T) {
	state := NewGraphState("user")
	for i := 0; i < compressionThreshold+5; i++ {
		state.Messages = append(state.Messages, llmprovider.Message{
			Role:  "user",
			Parts: []llmprovider.Part{{Text: "old message"}},
		})
	}

	state.CompressIfNeeded()

	assert.NotEmpty(t, state.OlderSummary)
	// Only maxRecentTurns messages kept
	assert.Len(t, state.Messages, maxRecentTurns)
}

func TestCompressIfNeeded_SummaryContainsOlderContent(t *testing.T) {
	state := NewGraphState("user")
	// Add enough messages to trigger compression
	for i := 0; i < compressionThreshold+2; i++ {
		state.Messages = append(state.Messages, llmprovider.Message{
			Role:  "user",
			Parts: []llmprovider.Part{{Text: "important context"}},
		})
	}

	state.CompressIfNeeded()

	assert.Contains(t, state.OlderSummary, "important context")
}

func TestCompressIfNeeded_FunctionMessagesSkipped(t *testing.T) {
	state := NewGraphState("user")
	for i := 0; i < compressionThreshold+2; i++ {
		state.Messages = append(state.Messages, llmprovider.Message{
			Role:  "function",
			Parts: []llmprovider.Part{{FunctionResponse: &llmprovider.FunctionResponse{Name: "tool"}}},
		})
	}

	state.CompressIfNeeded()

	// Function messages are skipped in summary
	assert.Empty(t, state.OlderSummary)
	assert.Len(t, state.Messages, maxRecentTurns)
}

func TestCompressIfNeeded_AppendsToPreviousSummary(t *testing.T) {
	state := NewGraphState("user")
	state.OlderSummary = "previous summary"

	for i := 0; i < compressionThreshold+2; i++ {
		state.Messages = append(state.Messages, llmprovider.Message{
			Role:  "assistant",
			Parts: []llmprovider.Part{{Text: "new content"}},
		})
	}

	state.CompressIfNeeded()

	assert.Contains(t, state.OlderSummary, "previous summary")
	assert.Contains(t, state.OlderSummary, "new content")
}
