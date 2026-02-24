package datemath_test

import (
	"testing"
	"time"

	"autonomous-task-management/pkg/datemath"
)

func TestNewParser(t *testing.T) {
	_, err := datemath.NewParser("Asia/Ho_Chi_Minh")
	if err != nil {
		t.Fatalf("unexpected error creating valid parser: %v", err)
	}

	_, err = datemath.NewParser("Invalid/Timezone")
	if err == nil {
		t.Fatalf("expected error for invalid timezone")
	}
}

func TestParse(t *testing.T) {
	parser, _ := datemath.NewParser("UTC")
	baseTime := time.Date(2024, 5, 1, 15, 30, 0, 0, time.UTC) // Wednesday, May 1, 2024
	startOfBase := time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		relative string
		want     time.Time
		wantErr  bool
	}{
		{
			name:     "Today",
			relative: "today",
			want:     startOfBase,
		},
		{
			name:     "Tomorrow",
			relative: "tomorrow",
			want:     startOfBase.AddDate(0, 0, 1),
		},
		{
			name:     "Yesterday",
			relative: "yesterday",
			want:     startOfBase.AddDate(0, 0, -1),
		},
		{
			name:     "In 3 days",
			relative: "in 3 days",
			want:     startOfBase.AddDate(0, 0, 3),
		},
		{
			name:     "In 2 weeks",
			relative: "in 2 weeks",
			want:     startOfBase.AddDate(0, 0, 14),
		},
		{
			name:     "In 1 month",
			relative: "in 1 month",
			want:     startOfBase.AddDate(0, 1, 0),
		},
		{
			name:     "Invalid duration pattern",
			relative: "in a few days",
			want:     baseTime,
			wantErr:  true,
		},
		{
			name:     "Next Monday (from Wed)",
			relative: "next monday",
			want:     startOfBase.AddDate(0, 0, 5), // Wed(3) to Mon(1) is +5 days
		},
		{
			name:     "Next Wednesday (from Wed)",
			relative: "next wednesday",
			want:     startOfBase.AddDate(0, 0, 7), // 1 week later
		},
		{
			name:     "Unknown fallback",
			relative: "some random day",
			want:     startOfBase, // falls back to startOfDay(base)
		},
		{
			name:     "Invalid Next Weekday",
			relative: "next funday",
			want:     baseTime, // Error returns baseTime
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parser.Parse(tt.relative, baseTime)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !got.Equal(tt.want) {
				t.Errorf("Parse() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEndOfDay(t *testing.T) {
	parser, _ := datemath.NewParser("UTC")
	base := time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC)
	want := time.Date(2024, 5, 1, 23, 59, 59, 0, time.UTC)

	got := parser.EndOfDay(base)
	if !got.Equal(want) {
		t.Errorf("EndOfDay() got = %v, want %v", got, want)
	}
}
