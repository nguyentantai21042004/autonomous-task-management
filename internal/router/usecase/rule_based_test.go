package usecase

import (
	"context"
	"testing"

	"autonomous-task-management/internal/router"
	pkgLog "autonomous-task-management/pkg/log"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// classifyByRules unit tests
// ---------------------------------------------------------------------------

func TestClassifyByRules_CreateTask_Strong(t *testing.T) {
	cases := []string{
		"tạo task mới cho tôi",
		"thêm task review PR",
		"đặt lịch họp vào thứ 2",
		"nhắc tôi deadline lúc 9h",
		"nhắc nhở cuộc họp SMAP",
		"tạo công việc deploy",
		"lên kế hoạch sprint tháng này",
	}
	for _, msg := range cases {
		t.Run(msg, func(t *testing.T) {
			result, ok := classifyByRules(msg)
			assert.True(t, ok, "should be confident for: %q", msg)
			assert.Equal(t, router.IntentCreateTask, result.Intent)
			assert.GreaterOrEqual(t, result.Confidence, ruleBasedThreshold)
		})
	}
}

func TestClassifyByRules_SearchTask_Strong(t *testing.T) {
	cases := []string{
		"tìm task về PR",
		"tìm kiếm công việc tuần này",
		"search task deadline",
		"xem task hôm nay",
		"danh sách task tháng này",
		"task nào đang pending",
		"tìm kiếm báo cáo tuần này", // tìm kiếm = 85
	}
	for _, msg := range cases {
		t.Run(msg, func(t *testing.T) {
			result, ok := classifyByRules(msg)
			assert.True(t, ok, "should be confident for: %q", msg)
			assert.Equal(t, router.IntentSearchTask, result.Intent)
			assert.GreaterOrEqual(t, result.Confidence, ruleBasedThreshold)
		})
	}
}

func TestClassifyByRules_Checklist_Strong(t *testing.T) {
	cases := []string{
		"đánh dấu hoàn thành task 1",
		"mark done item 3",
		"/check 2",
		"/uncheck item 1",
		"/complete abc123",
		"/progress task123",
		"tick checkbox số 2",
		"uncheck item cuối",
	}
	for _, msg := range cases {
		t.Run(msg, func(t *testing.T) {
			result, ok := classifyByRules(msg)
			assert.True(t, ok, "should be confident for: %q", msg)
			assert.Equal(t, router.IntentManageChecklist, result.Intent)
		})
	}
}

func TestClassifyByRules_Conversation_Strong(t *testing.T) {
	cases := []string{
		"xin chào bot",
		"hello",
		"hi bot",
		"chào bot bạn khỏe không",
		"/start",
		"/help",
		"cảm ơn bạn",
		"thanks",
		"bạn là ai",
	}
	for _, msg := range cases {
		t.Run(msg, func(t *testing.T) {
			result, ok := classifyByRules(msg)
			assert.True(t, ok, "should be confident for: %q", msg)
			assert.Equal(t, router.IntentConversation, result.Intent)
		})
	}
}

func TestClassifyByRules_Ambiguous_ReturnsNotConfident(t *testing.T) {
	ambiguous := []string{
		"PR 123 thế nào rồi?",                    // no clear signal
		"tôi không biết phải làm sao",             // vague
		"project SMAP đang ở trạng thái nào",      // could be search or conversation
		"nhớ là còn việc báo cáo chưa xong",       // soft create signal
		"cái task đó xong chưa",                   // soft search
		"item số 2 trong memo đó xong rồi",        // soft checklist
	}
	for _, msg := range ambiguous {
		t.Run(msg, func(t *testing.T) {
			_, ok := classifyByRules(msg)
			assert.False(t, ok, "should NOT be confident (needs LLM) for: %q", msg)
		})
	}
}

// ---------------------------------------------------------------------------
// Integration: verify LLM is NOT called for rule-based messages
// ---------------------------------------------------------------------------

func TestClassify_RuleBasedMessages_SkipsLLM(t *testing.T) {
	ctx := context.Background()
	mockLLM := new(MockLLMManager)
	logger := pkgLog.Init(pkgLog.ZapConfig{Level: "error", Mode: "development"})
	uc := New(mockLLM, logger)

	// These messages should be handled by rule-based — LLM must NOT be called
	cases := []struct {
		msg            string
		expectedIntent router.Intent
	}{
		{"tạo task mới: review PR 123", router.IntentCreateTask},
		{"tìm task về deployment", router.IntentSearchTask},
		{"đánh dấu hoàn thành task 1", router.IntentManageChecklist},
		{"xin chào", router.IntentConversation},
		{"/check abc", router.IntentManageChecklist},
		{"nhắc tôi họp lúc 3h", router.IntentCreateTask},
		{"danh sách task hôm nay", router.IntentSearchTask},
	}

	for _, c := range cases {
		t.Run(c.msg, func(t *testing.T) {
			output, err := uc.Classify(ctx, c.msg, nil)
			assert.NoError(t, err)
			assert.Equal(t, c.expectedIntent, output.Intent)
		})
	}

	// LLM should NEVER have been called
	mockLLM.AssertNotCalled(t, "GenerateContent")
}

// ---------------------------------------------------------------------------
// containsToken tests
// ---------------------------------------------------------------------------

func TestContainsToken_MatchesWholeWord(t *testing.T) {
	assert.True(t, containsToken("tìm task hôm nay", "task"))
	assert.True(t, containsToken("tạo task", "tạo task"))
}

func TestContainsToken_DoesNotMatchPartial(t *testing.T) {
	// "task" should not match inside "retask" or "taskbar"
	assert.False(t, containsToken("retask something", "task"))
	assert.False(t, containsToken("taskbar open", "task"))
}

func TestContainsToken_CaseInsensitive(t *testing.T) {
	// normalize() lowercases before calling containsToken
	lower := normalize("Tạo Task Mới")
	assert.True(t, containsToken(lower, "tạo task"))
}

func TestContainsToken_SlashCommands(t *testing.T) {
	// Slash commands start with / which is not a letter/digit → boundary ok
	assert.True(t, containsToken("/check abc", "/check"))
	assert.True(t, containsToken("/progress task123", "/progress"))
}

// ---------------------------------------------------------------------------
// ScorePatterns edge cases
// ---------------------------------------------------------------------------

func TestScorePatterns_CapsAt100(t *testing.T) {
	// Multiple strong signals should not exceed 100
	score := scorePatterns("tạo task mới thêm task thêm công việc", createSignals)
	assert.LessOrEqual(t, score, 100)
}

func TestScorePatterns_ZeroForNoMatch(t *testing.T) {
	score := scorePatterns("completely unrelated text", createSignals)
	assert.Equal(t, 0, score)
}

// ---------------------------------------------------------------------------
// Benchmark: rule-based should be fast (no I/O)
// ---------------------------------------------------------------------------

func BenchmarkClassifyByRules(b *testing.B) {
	ctx := context.Background()
	mockLLM := new(MockLLMManager)
	logger := pkgLog.Init(pkgLog.ZapConfig{Level: "error", Mode: "development"})
	uc := New(mockLLM, logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = uc.Classify(ctx, "tạo task mới deadline ngày mai", nil)
	}
}
