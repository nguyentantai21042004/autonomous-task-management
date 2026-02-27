package orchestrator

import (
	"strings"
	"testing"
	"time"
)

func TestBuildTimeContext(t *testing.T) {
	timezone := "Asia/Ho_Chi_Minh"
	context := buildTimeContext(timezone)

	// Verify context contains key elements
	if !strings.Contains(context, "SYSTEM CONTEXT") {
		t.Error("Context should contain 'SYSTEM CONTEXT'")
	}
	if !strings.Contains(context, "Hôm nay:") {
		t.Error("Context should contain 'Hôm nay:'")
	}
	if !strings.Contains(context, "Tuần này:") {
		t.Error("Context should contain 'Tuần này:'")
	}
	if !strings.Contains(context, "Ngày mai:") {
		t.Error("Context should contain 'Ngày mai:'")
	}
	if !strings.Contains(context, "YYYY-MM-DD") {
		t.Error("Context should contain 'YYYY-MM-DD'")
	}

	// Verify date format
	now := time.Now()
	todayStr := now.Format("2006-01-02")
	if !strings.Contains(context, todayStr) {
		t.Errorf("Context should contain today's date: %s", todayStr)
	}
}

func TestBuildTimeContext_WeekBoundaries(t *testing.T) {
	context := buildTimeContext("Asia/Ho_Chi_Minh")

	// Should contain Monday and Sunday dates
	lines := strings.Split(context, "\n")
	var weekLine string
	for _, line := range lines {
		if strings.Contains(line, "Tuần này:") {
			weekLine = line
			break
		}
	}

	if weekLine == "" {
		t.Error("Should contain week line")
	}
	if !strings.Contains(weekLine, "từ") {
		t.Error("Week line should contain 'từ'")
	}
	if !strings.Contains(weekLine, "đến") {
		t.Error("Week line should contain 'đến'")
	}
}

func TestBuildTimeContext_InvalidTimezone(t *testing.T) {
	// Should fallback to UTC without crashing
	context := buildTimeContext("Invalid/Timezone")

	if !strings.Contains(context, "SYSTEM CONTEXT") {
		t.Error("Should still generate context with invalid timezone")
	}
}
