package llmprovider

import (
	"context"
	"errors"
	"testing"
	"time"
)

// mockProvider is a test implementation of the Provider interface
type mockProvider struct {
	name       string
	model      string
	shouldFail bool
	response   *Response
	callCount  int
}

func (m *mockProvider) GenerateContent(ctx context.Context, req *Request) (*Response, error) {
	m.callCount++
	if m.shouldFail {
		return nil, errors.New("mock provider error")
	}
	return m.response, nil
}

func (m *mockProvider) Name() string {
	return m.name
}

func (m *mockProvider) Model() string {
	return m.model
}

// mockLogger is a test implementation of the Logger interface
type mockLogger struct {
	infoMessages []string
	warnMessages []string
}

func (m *mockLogger) Debug(ctx context.Context, arg ...any)                   {}
func (m *mockLogger) Debugf(ctx context.Context, template string, arg ...any) {}
func (m *mockLogger) Info(ctx context.Context, arg ...any) {
	if len(arg) > 0 {
		if msg, ok := arg[0].(string); ok {
			m.infoMessages = append(m.infoMessages, msg)
		}
	}
}
func (m *mockLogger) Infof(ctx context.Context, template string, arg ...any) {}
func (m *mockLogger) Warn(ctx context.Context, arg ...any) {
	if len(arg) > 0 {
		if msg, ok := arg[0].(string); ok {
			m.warnMessages = append(m.warnMessages, msg)
		}
	}
}
func (m *mockLogger) Warnf(ctx context.Context, template string, arg ...any)   {}
func (m *mockLogger) Error(ctx context.Context, arg ...any)                    {}
func (m *mockLogger) Errorf(ctx context.Context, template string, arg ...any)  {}
func (m *mockLogger) DPanic(ctx context.Context, arg ...any)                   {}
func (m *mockLogger) DPanicf(ctx context.Context, template string, arg ...any) {}
func (m *mockLogger) Panic(ctx context.Context, arg ...any)                    {}
func (m *mockLogger) Panicf(ctx context.Context, template string, arg ...any)  {}
func (m *mockLogger) Fatal(ctx context.Context, arg ...any)                    {}
func (m *mockLogger) Fatalf(ctx context.Context, template string, arg ...any)  {}

func TestGenerateContent_SuccessWithPrimaryProvider(t *testing.T) {
	// Setup
	expectedResponse := &Response{
		Content: Message{
			Role: "assistant",
			Parts: []Part{
				{Text: "Hello from primary provider"},
			},
		},
		ProviderName: "primary",
		ModelName:    "primary-model",
		Usage: &Usage{
			InputTokens:  100,
			OutputTokens: 50,
			TotalTokens:  150,
		},
	}

	primary := &mockProvider{
		name:       "primary",
		model:      "primary-model",
		shouldFail: false,
		response:   expectedResponse,
	}

	logger := &mockLogger{}
	config := &Config{
		FallbackEnabled: true,
		RetryAttempts:   3,
		RetryDelay:      100 * time.Millisecond,
	}

	manager := NewManager([]Provider{primary}, config, logger)

	// Execute
	req := &Request{
		Messages: []Message{
			{
				Role: "user",
				Parts: []Part{
					{Text: "Hello"},
				},
			},
		},
	}

	resp, err := manager.GenerateContent(context.Background(), req)

	// Verify
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if resp.ProviderName != "primary" {
		t.Errorf("Expected provider name 'primary', got: %s", resp.ProviderName)
	}

	if primary.callCount != 1 {
		t.Errorf("Expected primary provider to be called once, got: %d", primary.callCount)
	}

	if len(logger.infoMessages) != 1 {
		t.Errorf("Expected 1 info log message, got: %d", len(logger.infoMessages))
	}

	if len(logger.warnMessages) != 0 {
		t.Errorf("Expected 0 warn log messages, got: %d", len(logger.warnMessages))
	}
}

func TestGenerateContent_FallbackToSecondaryProvider(t *testing.T) {
	// Setup
	expectedResponse := &Response{
		Content: Message{
			Role: "assistant",
			Parts: []Part{
				{Text: "Hello from secondary provider"},
			},
		},
		ProviderName: "secondary",
		ModelName:    "secondary-model",
		Usage: &Usage{
			InputTokens:  100,
			OutputTokens: 50,
			TotalTokens:  150,
		},
	}

	primary := &mockProvider{
		name:       "primary",
		model:      "primary-model",
		shouldFail: true,
	}

	secondary := &mockProvider{
		name:       "secondary",
		model:      "secondary-model",
		shouldFail: false,
		response:   expectedResponse,
	}

	logger := &mockLogger{}
	config := &Config{
		FallbackEnabled: true,
		RetryAttempts:   2,
		RetryDelay:      10 * time.Millisecond,
	}

	manager := NewManager([]Provider{primary, secondary}, config, logger)

	// Execute
	req := &Request{
		Messages: []Message{
			{
				Role: "user",
				Parts: []Part{
					{Text: "Hello"},
				},
			},
		},
	}

	resp, err := manager.GenerateContent(context.Background(), req)

	// Verify
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if resp.ProviderName != "secondary" {
		t.Errorf("Expected provider name 'secondary', got: %s", resp.ProviderName)
	}

	// Primary should be called RetryAttempts times (2)
	if primary.callCount != 2 {
		t.Errorf("Expected primary provider to be called 2 times, got: %d", primary.callCount)
	}

	// Secondary should be called once
	if secondary.callCount != 1 {
		t.Errorf("Expected secondary provider to be called once, got: %d", secondary.callCount)
	}

	// Should have 1 info (success) and 1 warn (primary failure)
	if len(logger.infoMessages) != 1 {
		t.Errorf("Expected 1 info log message, got: %d", len(logger.infoMessages))
	}

	if len(logger.warnMessages) != 1 {
		t.Errorf("Expected 1 warn log message, got: %d", len(logger.warnMessages))
	}
}

func TestGenerateContent_AllProvidersFail(t *testing.T) {
	// Setup
	primary := &mockProvider{
		name:       "primary",
		model:      "primary-model",
		shouldFail: true,
	}

	secondary := &mockProvider{
		name:       "secondary",
		model:      "secondary-model",
		shouldFail: true,
	}

	logger := &mockLogger{}
	config := &Config{
		FallbackEnabled: true,
		RetryAttempts:   2,
		RetryDelay:      10 * time.Millisecond,
	}

	manager := NewManager([]Provider{primary, secondary}, config, logger)

	// Execute
	req := &Request{
		Messages: []Message{
			{
				Role: "user",
				Parts: []Part{
					{Text: "Hello"},
				},
			},
		},
	}

	resp, err := manager.GenerateContent(context.Background(), req)

	// Verify
	if err == nil {
		t.Fatal("Expected error when all providers fail, got nil")
	}

	if !errors.Is(err, ErrAllProvidersFailed) {
		t.Errorf("Expected ErrAllProvidersFailed, got: %v", err)
	}

	if resp != nil {
		t.Errorf("Expected nil response, got: %v", resp)
	}

	// Both providers should be called RetryAttempts times (2)
	if primary.callCount != 2 {
		t.Errorf("Expected primary provider to be called 2 times, got: %d", primary.callCount)
	}

	if secondary.callCount != 2 {
		t.Errorf("Expected secondary provider to be called 2 times, got: %d", secondary.callCount)
	}

	// Should have 2 warn messages (one for each provider failure)
	if len(logger.warnMessages) != 2 {
		t.Errorf("Expected 2 warn log messages, got: %d", len(logger.warnMessages))
	}
}

func TestGenerateContent_NoFallbackWhenDisabled(t *testing.T) {
	// Setup
	primary := &mockProvider{
		name:       "primary",
		model:      "primary-model",
		shouldFail: true,
	}

	secondary := &mockProvider{
		name:       "secondary",
		model:      "secondary-model",
		shouldFail: false,
		response: &Response{
			ProviderName: "secondary",
			ModelName:    "secondary-model",
			Usage:        &Usage{},
		},
	}

	logger := &mockLogger{}
	config := &Config{
		FallbackEnabled: false, // Fallback disabled
		RetryAttempts:   2,
		RetryDelay:      10 * time.Millisecond,
	}

	manager := NewManager([]Provider{primary, secondary}, config, logger)

	// Execute
	req := &Request{
		Messages: []Message{
			{
				Role: "user",
				Parts: []Part{
					{Text: "Hello"},
				},
			},
		},
	}

	resp, err := manager.GenerateContent(context.Background(), req)

	// Verify
	if err == nil {
		t.Fatal("Expected error when primary fails and fallback is disabled, got nil")
	}

	if resp != nil {
		t.Errorf("Expected nil response, got: %v", resp)
	}

	// Primary should be called RetryAttempts times (2)
	if primary.callCount != 2 {
		t.Errorf("Expected primary provider to be called 2 times, got: %d", primary.callCount)
	}

	// Secondary should NOT be called
	if secondary.callCount != 0 {
		t.Errorf("Expected secondary provider to NOT be called, got: %d calls", secondary.callCount)
	}
}

func TestGenerateContent_NoProvidersConfigured(t *testing.T) {
	// Setup
	logger := &mockLogger{}
	config := &Config{
		FallbackEnabled: true,
		RetryAttempts:   3,
		RetryDelay:      100 * time.Millisecond,
	}

	manager := NewManager([]Provider{}, config, logger)

	// Execute
	req := &Request{
		Messages: []Message{
			{
				Role: "user",
				Parts: []Part{
					{Text: "Hello"},
				},
			},
		},
	}

	resp, err := manager.GenerateContent(context.Background(), req)

	// Verify
	if err == nil {
		t.Fatal("Expected error when no providers configured, got nil")
	}

	if !errors.Is(err, ErrNoProvidersConfigured) {
		t.Errorf("Expected ErrNoProvidersConfigured, got: %v", err)
	}

	if resp != nil {
		t.Errorf("Expected nil response, got: %v", resp)
	}
}
