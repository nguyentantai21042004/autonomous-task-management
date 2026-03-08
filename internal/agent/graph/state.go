package graph

import (
	"fmt"
	"strings"
	"time"

	"autonomous-task-management/pkg/llmprovider"
)

// GraphStatus represents the current execution state of the graph.
type GraphStatus string

const (
	// StatusRunning: engine dang chay, can goi NodeAgent hoac NodeExecuteTool
	StatusRunning GraphStatus = "RUNNING"

	// StatusWaitingForHuman: engine dung lai, cho user reply
	// State duoc luu trong cache de resume sau
	StatusWaitingForHuman GraphStatus = "WAITING_FOR_HUMAN"

	// StatusFinished: engine hoan thanh, co the lay response
	StatusFinished GraphStatus = "FINISHED"

	// StatusError: co loi xay ra
	StatusError GraphStatus = "ERROR"
)

const (
	maxRecentTurns       = 6
	defaultStateTTL      = 30 * time.Minute
	maxHistoryLength     = 20
	compressionThreshold = 8 // compress when Messages exceeds this (reduces token cost earlier)
	summaryMaxChars      = 800
)

// GraphState thay the SessionMemory cua V1.2.
// Luu toan bo trang thai tien trinh, cho phep pause/resume giua cac tin nhan.
type GraphState struct {
	UserID string
	Status GraphStatus

	// Full conversation history (gui len LLM)
	Messages []llmprovider.Message

	// Execution context — KHONG co trong V1.2
	PendingTool   *llmprovider.FunctionCall // tool dang cho chay (khi WAITING)
	CurrentStep   int
	CurrentIntent string

	// Context compression (giam token cost)
	OlderSummary string               // cac turns cu duoc tom tat thanh 1 doan
	RecentTurns  []llmprovider.Message // chi giu maxRecentTurns turns gan nhat, raw

	// Metadata
	LastUpdated time.Time
	TTL         time.Duration
}

// NewGraphState tao GraphState moi cho mot user.
func NewGraphState(userID string) *GraphState {
	return &GraphState{
		UserID:      userID,
		Status:      StatusFinished,
		Messages:    make([]llmprovider.Message, 0),
		RecentTurns: make([]llmprovider.Message, 0),
		LastUpdated: time.Now(),
		TTL:         defaultStateTTL,
	}
}

// IsExpired tra ve true neu state qua TTL.
func (s *GraphState) IsExpired() bool {
	return time.Since(s.LastUpdated) > s.TTL
}

// AppendMessage them message vao Messages va RecentTurns.
// Tu dong gioi han RecentTurns khong vuot qua maxRecentTurns.
func (s *GraphState) AppendMessage(msg llmprovider.Message) {
	s.Messages = append(s.Messages, msg)
	s.RecentTurns = append(s.RecentTurns, msg)
	if len(s.RecentTurns) > maxRecentTurns {
		s.RecentTurns = s.RecentTurns[len(s.RecentTurns)-maxRecentTurns:]
	}
}

// TrimHistory gioi han Messages tranh context bloat.
// Giu lai maxHistoryLength messages gan nhat.
func (s *GraphState) TrimHistory() {
	if len(s.Messages) > maxHistoryLength {
		s.Messages = s.Messages[len(s.Messages)-maxHistoryLength:]
	}
}

// Touch cap nhat LastUpdated.
func (s *GraphState) Touch() {
	s.LastUpdated = time.Now()
}

// CompressIfNeeded chay context compression khi Messages vuot qua compressionThreshold.
// Cac messages cu duoc rut gon thanh OlderSummary; chi giu maxRecentTurns messages cuoi.
// Viec nay giam token cost cho cac cuoc hoi dai.
func (s *GraphState) CompressIfNeeded() {
	if len(s.Messages) <= compressionThreshold {
		return
	}

	// Tach phan cu (tat ca tru maxRecentTurns cuoi)
	cutoff := len(s.Messages) - maxRecentTurns
	older := s.Messages[:cutoff]

	// Build deterministic summary — khong can LLM
	var sb strings.Builder
	if s.OlderSummary != "" {
		sb.WriteString(s.OlderSummary)
		sb.WriteString(" | ")
	}
	for _, msg := range older {
		if msg.Role == "function" {
			continue // bo qua function results de giam noise
		}
		for _, part := range msg.Parts {
			if part.Text != "" {
				excerpt := part.Text
				if len([]rune(excerpt)) > 120 {
					excerpt = string([]rune(excerpt)[:120]) + "…"
				}
				sb.WriteString(fmt.Sprintf("[%s] %s | ", msg.Role, excerpt))
			}
		}
	}

	summary := strings.TrimRight(sb.String(), " |")
	// Truncate total summary length
	if len([]rune(summary)) > summaryMaxChars {
		summary = string([]rune(summary)[:summaryMaxChars]) + "…"
	}

	s.OlderSummary = summary
	s.Messages = s.Messages[cutoff:]
}
