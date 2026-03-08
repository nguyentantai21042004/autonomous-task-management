package usecase

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"

	"autonomous-task-management/internal/model"
	"autonomous-task-management/internal/webhook"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// Mock logger
// ---------------------------------------------------------------------------

type mockLogger struct{}

func (m *mockLogger) Debug(_ context.Context, _ ...any)             {}
func (m *mockLogger) Debugf(_ context.Context, _ string, _ ...any)  {}
func (m *mockLogger) Info(_ context.Context, _ ...any)              {}
func (m *mockLogger) Infof(_ context.Context, _ string, _ ...any)   {}
func (m *mockLogger) Warn(_ context.Context, _ ...any)              {}
func (m *mockLogger) Warnf(_ context.Context, _ string, _ ...any)   {}
func (m *mockLogger) Error(_ context.Context, _ ...any)             {}
func (m *mockLogger) Errorf(_ context.Context, _ string, _ ...any)  {}
func (m *mockLogger) DPanic(_ context.Context, _ ...any)            {}
func (m *mockLogger) DPanicf(_ context.Context, _ string, _ ...any) {}
func (m *mockLogger) Panic(_ context.Context, _ ...any)             {}
func (m *mockLogger) Panicf(_ context.Context, _ string, _ ...any)  {}
func (m *mockLogger) Fatal(_ context.Context, _ ...any)             {}
func (m *mockLogger) Fatalf(_ context.Context, _ string, _ ...any)  {}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const testSecret = "test-webhook-secret-2024"

func newTestWebhookUC() *implUseCase {
	uc := New(webhook.SecurityConfig{
		Secret:          testSecret,
		RateLimitPerMin: 1000,
	}, &mockLogger{})
	return uc.(*implUseCase)
}

func signPayload(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// ---------------------------------------------------------------------------
// ParseGitHubEvent — Push events
// ---------------------------------------------------------------------------

func TestParseGitHubEvent_Push(t *testing.T) {
	uc := newTestWebhookUC()

	payload, _ := json.Marshal(map[string]interface{}{
		"ref": "refs/heads/main",
		"repository": map[string]interface{}{
			"full_name": "org/my-repo",
		},
		"head_commit": map[string]interface{}{
			"id":      "abc123",
			"message": "feat: add login",
			"author": map[string]interface{}{
				"name": "developer",
			},
		},
	})

	sig := signPayload(payload, testSecret)
	event, err := uc.ParseGitHubEvent(context.Background(), payload, "push", sig)

	assert.NoError(t, err)
	assert.Equal(t, model.SourceGitHub, event.Source)
	assert.Equal(t, "push", event.EventType)
	assert.Equal(t, "org/my-repo", event.Repository)
	assert.Equal(t, "main", event.Branch)
	assert.Equal(t, "abc123", event.Commit)
	assert.Equal(t, "developer", event.Author)
	assert.Equal(t, "feat: add login", event.Message)
}

func TestParseGitHubEvent_Push_BranchExtraction(t *testing.T) {
	uc := newTestWebhookUC()

	tests := []struct {
		ref            string
		expectedBranch string
	}{
		{"refs/heads/main", "main"},
		{"refs/heads/feature/auth", "feature/auth"},
		{"refs/heads/release/v1.0", "release/v1.0"},
		{"refs/tags/v1.0", "refs/tags/v1.0"},
		{"short", "short"},
	}

	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			payload, _ := json.Marshal(map[string]interface{}{
				"ref":         tt.ref,
				"repository":  map[string]interface{}{"full_name": "org/repo"},
				"head_commit": map[string]interface{}{"id": "abc", "message": "", "author": map[string]interface{}{"name": ""}},
			})
			event, err := uc.ParseGitHubEvent(context.Background(), payload, "push", signPayload(payload, testSecret))
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedBranch, event.Branch)
		})
	}
}

// ---------------------------------------------------------------------------
// ParseGitHubEvent — Pull Request events
// ---------------------------------------------------------------------------

func TestParseGitHubEvent_PR_Opened(t *testing.T) {
	uc := newTestWebhookUC()

	payload, _ := json.Marshal(map[string]interface{}{
		"action": "opened",
		"number": 42,
		"pull_request": map[string]interface{}{
			"title":  "Add authentication",
			"head":   map[string]interface{}{"ref": "feature/auth", "sha": "deadbeef"},
			"user":   map[string]interface{}{"login": "dev"},
			"merged": false,
		},
		"repository": map[string]interface{}{"full_name": "org/my-repo"},
	})

	event, err := uc.ParseGitHubEvent(context.Background(), payload, "pull_request", signPayload(payload, testSecret))

	assert.NoError(t, err)
	assert.Equal(t, "pull_request", event.EventType)
	assert.Equal(t, "opened", event.Action)
	assert.Equal(t, 42, event.PRNumber)
	assert.Equal(t, "Add authentication", event.Message)
	assert.Equal(t, "feature/auth", event.Branch)
}

func TestParseGitHubEvent_PR_ClosedAndMerged(t *testing.T) {
	uc := newTestWebhookUC()

	payload, _ := json.Marshal(map[string]interface{}{
		"action": "closed",
		"number": 99,
		"pull_request": map[string]interface{}{
			"title":  "Fix bug",
			"head":   map[string]interface{}{"ref": "fix/bug", "sha": "abc"},
			"user":   map[string]interface{}{"login": "dev"},
			"merged": true,
		},
		"repository": map[string]interface{}{"full_name": "org/repo"},
	})

	event, err := uc.ParseGitHubEvent(context.Background(), payload, "pull_request", signPayload(payload, testSecret))

	assert.NoError(t, err)
	assert.Equal(t, "merged", event.Action)
	assert.Equal(t, 99, event.PRNumber)
}

func TestParseGitHubEvent_PR_ClosedNotMerged(t *testing.T) {
	uc := newTestWebhookUC()

	payload, _ := json.Marshal(map[string]interface{}{
		"action": "closed",
		"number": 10,
		"pull_request": map[string]interface{}{
			"title":  "WIP Feature",
			"head":   map[string]interface{}{"ref": "wip", "sha": "abc"},
			"user":   map[string]interface{}{"login": "dev"},
			"merged": false,
		},
		"repository": map[string]interface{}{"full_name": "org/repo"},
	})

	event, err := uc.ParseGitHubEvent(context.Background(), payload, "pull_request", signPayload(payload, testSecret))

	assert.NoError(t, err)
	assert.Equal(t, "closed", event.Action)
}

// ---------------------------------------------------------------------------
// ParseGitHubEvent — Issue events
// ---------------------------------------------------------------------------

func TestParseGitHubEvent_Issue(t *testing.T) {
	uc := newTestWebhookUC()

	payload, _ := json.Marshal(map[string]interface{}{
		"action": "opened",
		"issue": map[string]interface{}{
			"number": 15,
			"title":  "Bug: login fails",
			"user":   map[string]interface{}{"login": "reporter"},
		},
		"repository": map[string]interface{}{"full_name": "org/my-repo"},
	})

	event, err := uc.ParseGitHubEvent(context.Background(), payload, "issues", signPayload(payload, testSecret))

	assert.NoError(t, err)
	assert.Equal(t, "issue", event.EventType)
	assert.Equal(t, 15, event.IssueNumber)
	assert.Equal(t, "Bug: login fails", event.Message)
	assert.Equal(t, "opened", event.Action)
}

// ---------------------------------------------------------------------------
// Security validation
// ---------------------------------------------------------------------------

func TestParseGitHubEvent_InvalidSignature(t *testing.T) {
	uc := newTestWebhookUC()
	payload := []byte(`{"ref":"refs/heads/main"}`)

	_, err := uc.ParseGitHubEvent(context.Background(), payload, "push", "sha256=invalid")
	assert.Error(t, err)
}

func TestParseGitHubEvent_WrongSecret(t *testing.T) {
	uc := newTestWebhookUC()
	payload := []byte(`{"ref":"refs/heads/main"}`)
	wrongSig := signPayload(payload, "wrong-secret")

	_, err := uc.ParseGitHubEvent(context.Background(), payload, "push", wrongSig)
	assert.ErrorIs(t, err, webhook.ErrInvalidSignature)
}

func TestParseGitHubEvent_MissingSignaturePrefix(t *testing.T) {
	uc := newTestWebhookUC()
	payload := []byte(`{"ref":"refs/heads/main"}`)

	_, err := uc.ParseGitHubEvent(context.Background(), payload, "push", "invalid-format")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid signature format")
}

func TestParseGitHubEvent_UnsupportedEventType(t *testing.T) {
	uc := newTestWebhookUC()
	payload := []byte(`{}`)
	sig := signPayload(payload, testSecret)

	_, err := uc.ParseGitHubEvent(context.Background(), payload, "deployment", sig)
	assert.ErrorIs(t, err, webhook.ErrUnsupportedEvent)
}

func TestParseGitHubEvent_InvalidJSON(t *testing.T) {
	uc := newTestWebhookUC()
	payload := []byte(`{invalid json}`)
	sig := signPayload(payload, testSecret)

	_, err := uc.ParseGitHubEvent(context.Background(), payload, "push", sig)
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// Rate limiting
// ---------------------------------------------------------------------------

func TestRateLimiter_AllowsNormalTraffic(t *testing.T) {
	rl := newRateLimiter(60)
	for i := 0; i < 5; i++ {
		assert.True(t, rl.Allow("source-1"))
	}
}

func TestRateLimiter_BlocksExcessiveTraffic(t *testing.T) {
	rl := newRateLimiter(1) // 1 req/min

	allowed := 0
	for i := 0; i < 100; i++ {
		if rl.Allow("source-flood") {
			allowed++
		}
	}
	assert.Less(t, allowed, 10)
}

func TestRateLimiter_IndependentSources(t *testing.T) {
	rl := newRateLimiter(60)
	assert.True(t, rl.Allow("source-A"))
	assert.True(t, rl.Allow("source-B"))
	assert.True(t, rl.Allow("source-C"))
}

// ---------------------------------------------------------------------------
// GitLab token validation
// ---------------------------------------------------------------------------

func TestValidateGitLabToken_ValidToken(t *testing.T) {
	uc := newTestWebhookUC()
	err := uc.validateGitLabToken(testSecret)
	assert.NoError(t, err)
}

func TestValidateGitLabToken_InvalidToken(t *testing.T) {
	uc := newTestWebhookUC()
	err := uc.validateGitLabToken("wrong-token")
	assert.ErrorIs(t, err, webhook.ErrInvalidSignature)
}

func TestValidateGitLabToken_EmptySecret(t *testing.T) {
	uc := New(webhook.SecurityConfig{Secret: ""}, &mockLogger{}).(*implUseCase)
	err := uc.validateGitLabToken("any-token")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

// ---------------------------------------------------------------------------
// GitHub signature edge cases
// ---------------------------------------------------------------------------

func TestValidateGitHubSignature_EmptySecret(t *testing.T) {
	uc := New(webhook.SecurityConfig{Secret: ""}, &mockLogger{}).(*implUseCase)
	err := uc.validateGitHubSignature([]byte("payload"), "sha256=abc")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

func TestValidateGitHubSignature_ValidSignature(t *testing.T) {
	uc := newTestWebhookUC()
	payload := []byte("test payload")
	sig := signPayload(payload, testSecret)
	err := uc.validateGitHubSignature(payload, sig)
	assert.NoError(t, err)
}
