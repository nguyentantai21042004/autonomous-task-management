package response_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"autonomous-task-management/pkg/response"
)

func TestDateMarshalJSON(t *testing.T) {
	tm := time.Date(2024, 5, 1, 15, 30, 0, 0, time.UTC)
	// Response type uses Local() time, so test depends on test runner timezone.
	// To avoid flaky tests, we just check if it gets wrapped in JSON quotes and isn't empty.
	d := response.Date(tm)

	b, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("unexpected error marshaling Date: %v", err)
	}

	str := string(b)
	if !strings.HasPrefix(str, `"`) || !strings.HasSuffix(str, `"`) {
		t.Errorf("expected string JSON format, got %s", str)
	}
	if len(str) < 10 {
		t.Errorf("marshaled string too short: %s", str)
	}
}

func TestDateTimeMarshalJSON(t *testing.T) {
	tm := time.Date(2024, 5, 1, 15, 30, 0, 0, time.UTC)
	dt := response.DateTime(tm)

	b, err := json.Marshal(dt)
	if err != nil {
		t.Fatalf("unexpected error marshaling DateTime: %v", err)
	}

	str := string(b)
	if !strings.HasPrefix(str, `"`) || !strings.HasSuffix(str, `"`) {
		t.Errorf("expected string JSON format, got %s", str)
	}
	if len(str) < 15 {
		t.Errorf("marshaled string too short: %s", str)
	}
}
