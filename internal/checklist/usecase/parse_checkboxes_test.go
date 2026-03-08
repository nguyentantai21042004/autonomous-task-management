package usecase

import (
	"autonomous-task-management/internal/checklist"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// newTestUseCase creates a usecase for testing with nil/minimal dependencies.
// The ParseCheckboxes, GetStats, etc. methods don't use these dependencies.
func newTestUseCase() checklist.UseCase {
	return New(nil, nil, &noopLogger{})
}

// noopLogger is a minimal no-op logger for testing.
type noopLogger struct{}

func (noopLogger) Debug(context.Context, ...interface{})                        {}
func (noopLogger) Debugf(context.Context, string, ...interface{})              {}
func (noopLogger) Info(context.Context, ...interface{})                         {}
func (noopLogger) Infof(context.Context, string, ...interface{})               {}
func (noopLogger) Warn(context.Context, ...interface{})                         {}
func (noopLogger) Warnf(context.Context, string, ...interface{})               {}
func (noopLogger) Error(context.Context, ...interface{})                        {}
func (noopLogger) Errorf(context.Context, string, ...interface{})              {}
func (noopLogger) Fatal(context.Context, ...interface{})                        {}
func (noopLogger) Fatalf(context.Context, string, ...interface{})              {}
func (noopLogger) DPanic(context.Context, ...interface{})                       {}
func (noopLogger) DPanicf(context.Context, string, ...interface{})             {}
func (noopLogger) Panic(context.Context, ...interface{})                        {}
func (noopLogger) Panicf(context.Context, string, ...interface{})              {}

func TestParseCheckboxes_EmptyContent(t *testing.T) {
	uc := newTestUseCase()

	checkboxes := uc.ParseCheckboxes("")

	assert.Empty(t, checkboxes)
}

func TestParseCheckboxes_NoCheckboxes(t *testing.T) {
	uc := newTestUseCase()

	content := `# Task Title
This is a regular task without checkboxes.
Just some text content.`

	checkboxes := uc.ParseCheckboxes(content)

	assert.Empty(t, checkboxes)
}

func TestParseCheckboxes_UncheckedBoxes(t *testing.T) {
	uc := newTestUseCase()

	content := `# Task with Checklist
- [ ] Item 1
- [ ] Item 2
- [ ] Item 3`

	checkboxes := uc.ParseCheckboxes(content)

	assert.Len(t, checkboxes, 3)
	assert.False(t, checkboxes[0].Checked)
	assert.Equal(t, "Item 1", checkboxes[0].Text)
	assert.False(t, checkboxes[1].Checked)
	assert.Equal(t, "Item 2", checkboxes[1].Text)
	assert.False(t, checkboxes[2].Checked)
	assert.Equal(t, "Item 3", checkboxes[2].Text)
}

func TestParseCheckboxes_CheckedBoxes(t *testing.T) {
	uc := newTestUseCase()

	content := `# Completed Tasks
- [x] Done item 1
- [x] Done item 2`

	checkboxes := uc.ParseCheckboxes(content)

	assert.Len(t, checkboxes, 2)
	assert.True(t, checkboxes[0].Checked)
	assert.Equal(t, "Done item 1", checkboxes[0].Text)
	assert.True(t, checkboxes[1].Checked)
	assert.Equal(t, "Done item 2", checkboxes[1].Text)
}

func TestParseCheckboxes_MixedState(t *testing.T) {
	uc := newTestUseCase()

	content := `# Mixed Checklist
- [x] Completed item
- [ ] Pending item
- [x] Another completed
- [ ] Another pending`

	checkboxes := uc.ParseCheckboxes(content)

	assert.Len(t, checkboxes, 4)
	assert.True(t, checkboxes[0].Checked)
	assert.False(t, checkboxes[1].Checked)
	assert.True(t, checkboxes[2].Checked)
	assert.False(t, checkboxes[3].Checked)
}

func TestParseCheckboxes_WithIndentation(t *testing.T) {
	uc := newTestUseCase()

	content := `# Nested Checklist
- [ ] Parent item
  - [ ] Child item 1
  - [ ] Child item 2
- [ ] Another parent`

	checkboxes := uc.ParseCheckboxes(content)

	assert.Len(t, checkboxes, 4)
	assert.Equal(t, "", checkboxes[0].Indent)
	assert.Equal(t, "  ", checkboxes[1].Indent)
	assert.Equal(t, "  ", checkboxes[2].Indent)
	assert.Equal(t, "", checkboxes[3].Indent)
}

func TestParseCheckboxes_SpecialCharacters(t *testing.T) {
	uc := newTestUseCase()

	content := `# Special Characters
- [ ] Item with @mention
- [ ] Item with #hashtag
- [ ] Item with $special
- [ ] Item with "quotes"
- [ ] Item with 'apostrophe'`

	checkboxes := uc.ParseCheckboxes(content)

	assert.Len(t, checkboxes, 5)
	assert.Contains(t, checkboxes[0].Text, "@mention")
	assert.Contains(t, checkboxes[1].Text, "#hashtag")
	assert.Contains(t, checkboxes[2].Text, "$special")
}

func TestParseCheckboxes_Unicode(t *testing.T) {
	uc := newTestUseCase()

	content := `# Vietnamese Checklist
- [ ] Hoàn thành báo cáo
- [x] Gửi email cho khách hàng
- [ ] Cập nhật tài liệu`

	checkboxes := uc.ParseCheckboxes(content)

	assert.Len(t, checkboxes, 3)
	assert.Contains(t, checkboxes[0].Text, "Hoàn thành")
	assert.Contains(t, checkboxes[1].Text, "Gửi email")
	assert.True(t, checkboxes[1].Checked)
}

func TestParseCheckboxes_EmptyCheckboxText(t *testing.T) {
	uc := newTestUseCase()

	content := `# Empty Checkboxes
- [ ] 
- [x] 
- [ ] Valid item`

	checkboxes := uc.ParseCheckboxes(content)

	// Should handle empty checkbox text gracefully
	assert.GreaterOrEqual(t, len(checkboxes), 1)
}

func TestParseCheckboxes_MultilineContent(t *testing.T) {
	uc := newTestUseCase()

	content := `# Task
Some description here.

## Checklist
- [ ] First item
- [ ] Second item

Some notes below.
- [ ] Third item

More text.`

	checkboxes := uc.ParseCheckboxes(content)

	assert.Len(t, checkboxes, 3)
}

func TestGetStats_EmptyContent(t *testing.T) {
	uc := newTestUseCase()

	stats := uc.GetStats("")

	assert.Equal(t, 0, stats.Total)
	assert.Equal(t, 0, stats.Completed)
	assert.Equal(t, 0, stats.Pending)
	assert.Equal(t, 0.0, stats.Progress)
}

func TestGetStats_NoCheckboxes(t *testing.T) {
	uc := newTestUseCase()

	content := "Just regular text without checkboxes"
	stats := uc.GetStats(content)

	assert.Equal(t, 0, stats.Total)
	assert.Equal(t, 0.0, stats.Progress)
}

func TestGetStats_AllUnchecked(t *testing.T) {
	uc := newTestUseCase()

	content := `- [ ] Item 1
- [ ] Item 2
- [ ] Item 3`

	stats := uc.GetStats(content)

	assert.Equal(t, 3, stats.Total)
	assert.Equal(t, 0, stats.Completed)
	assert.Equal(t, 3, stats.Pending)
	assert.Equal(t, 0.0, stats.Progress)
}

func TestGetStats_AllChecked(t *testing.T) {
	uc := newTestUseCase()

	content := `- [x] Item 1
- [x] Item 2
- [x] Item 3`

	stats := uc.GetStats(content)

	assert.Equal(t, 3, stats.Total)
	assert.Equal(t, 3, stats.Completed)
	assert.Equal(t, 0, stats.Pending)
	assert.Equal(t, 100.0, stats.Progress)
}

func TestGetStats_PartiallyCompleted(t *testing.T) {
	uc := newTestUseCase()

	content := `- [x] Item 1
- [ ] Item 2
- [x] Item 3
- [ ] Item 4`

	stats := uc.GetStats(content)

	assert.Equal(t, 4, stats.Total)
	assert.Equal(t, 2, stats.Completed)
	assert.Equal(t, 2, stats.Pending)
	assert.Equal(t, 50.0, stats.Progress)
}

func TestIsFullyCompleted_EmptyContent(t *testing.T) {
	uc := newTestUseCase()

	completed := uc.IsFullyCompleted("")

	assert.False(t, completed)
}

func TestIsFullyCompleted_NoCheckboxes(t *testing.T) {
	uc := newTestUseCase()

	completed := uc.IsFullyCompleted("Regular text")

	assert.False(t, completed)
}

func TestIsFullyCompleted_AllChecked(t *testing.T) {
	uc := newTestUseCase()

	content := `- [x] Item 1
- [x] Item 2`

	completed := uc.IsFullyCompleted(content)

	assert.True(t, completed)
}

func TestIsFullyCompleted_PartiallyChecked(t *testing.T) {
	uc := newTestUseCase()

	content := `- [x] Item 1
- [ ] Item 2`

	completed := uc.IsFullyCompleted(content)

	assert.False(t, completed)
}

func TestUpdateAllCheckboxes_CheckAll(t *testing.T) {
	uc := newTestUseCase()

	content := `- [ ] Item 1
- [ ] Item 2
- [ ] Item 3`

	updated := uc.UpdateAllCheckboxes(content, true)

	assert.Contains(t, updated, "[x] Item 1")
	assert.Contains(t, updated, "[x] Item 2")
	assert.Contains(t, updated, "[x] Item 3")
	assert.NotContains(t, updated, "[ ]")
}

func TestUpdateAllCheckboxes_UncheckAll(t *testing.T) {
	uc := newTestUseCase()

	content := `- [x] Item 1
- [x] Item 2
- [x] Item 3`

	updated := uc.UpdateAllCheckboxes(content, false)

	assert.Contains(t, updated, "[ ] Item 1")
	assert.Contains(t, updated, "[ ] Item 2")
	assert.Contains(t, updated, "[ ] Item 3")
	assert.NotContains(t, updated, "[x]")
}

func TestUpdateAllCheckboxes_EmptyContent(t *testing.T) {
	uc := newTestUseCase()

	updated := uc.UpdateAllCheckboxes("", true)

	assert.Equal(t, "", updated)
}

func TestUpdateAllCheckboxes_PreservesOtherContent(t *testing.T) {
	uc := newTestUseCase()

	content := `# Task Title
Some description.

- [ ] Item 1
- [ ] Item 2

Some notes.`

	updated := uc.UpdateAllCheckboxes(content, true)

	assert.Contains(t, updated, "# Task Title")
	assert.Contains(t, updated, "Some description")
	assert.Contains(t, updated, "Some notes")
	assert.Contains(t, updated, "[x] Item 1")
	assert.Contains(t, updated, "[x] Item 2")
}

func TestUpdateCheckbox_ValidMatch(t *testing.T) {
	uc := newTestUseCase()

	content := `- [ ] Item 1
- [ ] Item 2
- [ ] Item 3`

	output, err := uc.UpdateCheckbox(context.Background(), checklist.UpdateCheckboxInput{
		Content:      content,
		CheckboxText: "Item 2",
		Checked:      true,
	})

	assert.NoError(t, err)
	assert.True(t, output.Updated)
	assert.Equal(t, 1, output.Count)
	assert.Contains(t, output.Content, "[ ] Item 1")
	assert.Contains(t, output.Content, "[x] Item 2")
	assert.Contains(t, output.Content, "[ ] Item 3")
}

func TestUpdateCheckbox_UncheckItem(t *testing.T) {
	uc := newTestUseCase()

	content := `- [x] Item 1
- [x] Item 2
- [x] Item 3`

	output, err := uc.UpdateCheckbox(context.Background(), checklist.UpdateCheckboxInput{
		Content:      content,
		CheckboxText: "Item 1",
		Checked:      false,
	})

	assert.NoError(t, err)
	assert.True(t, output.Updated)
	assert.Contains(t, output.Content, "[ ] Item 1")
	assert.Contains(t, output.Content, "[x] Item 2")
	assert.Contains(t, output.Content, "[x] Item 3")
}

func TestUpdateCheckbox_PartialMatch(t *testing.T) {
	uc := newTestUseCase()

	content := `- [ ] Complete the report
- [ ] Send email to client
- [ ] Update documentation`

	output, err := uc.UpdateCheckbox(context.Background(), checklist.UpdateCheckboxInput{
		Content:      content,
		CheckboxText: "email",
		Checked:      true,
	})

	assert.NoError(t, err)
	assert.True(t, output.Updated)
	assert.Equal(t, 1, output.Count)
	assert.Contains(t, output.Content, "[x] Send email")
}

func TestUpdateCheckbox_NoMatch(t *testing.T) {
	uc := newTestUseCase()

	content := `- [ ] Item 1
- [ ] Item 2`

	output, err := uc.UpdateCheckbox(context.Background(), checklist.UpdateCheckboxInput{
		Content:      content,
		CheckboxText: "Item 999",
		Checked:      true,
	})

	assert.NoError(t, err)
	assert.False(t, output.Updated)
	assert.Equal(t, 0, output.Count)
}

func TestUpdateCheckbox_EmptyContent(t *testing.T) {
	uc := newTestUseCase()

	output, err := uc.UpdateCheckbox(context.Background(), checklist.UpdateCheckboxInput{
		Content:      "",
		CheckboxText: "Item",
		Checked:      true,
	})

	assert.NoError(t, err)
	assert.False(t, output.Updated)
	assert.Equal(t, "", output.Content)
}

func TestUpdateCheckbox_NoCheckboxes(t *testing.T) {
	uc := newTestUseCase()

	content := "Regular text without checkboxes"

	output, err := uc.UpdateCheckbox(context.Background(), checklist.UpdateCheckboxInput{
		Content:      content,
		CheckboxText: "text",
		Checked:      true,
	})

	assert.NoError(t, err)
	assert.False(t, output.Updated)
	assert.Equal(t, 0, output.Count)
}

func TestUpdateCheckbox_PreservesFormatting(t *testing.T) {
	uc := newTestUseCase()

	content := `# Task Title

Some description.

- [ ] Item 1
- [ ] Item 2

Notes below.`

	output, err := uc.UpdateCheckbox(context.Background(), checklist.UpdateCheckboxInput{
		Content:      content,
		CheckboxText: "Item 1",
		Checked:      true,
	})

	assert.NoError(t, err)
	assert.True(t, output.Updated)
	assert.Contains(t, output.Content, "# Task Title")
	assert.Contains(t, output.Content, "Some description")
	assert.Contains(t, output.Content, "Notes below")
	assert.Contains(t, output.Content, "[x] Item 1")
}

func TestUpdateCheckbox_WithVietnamese(t *testing.T) {
	uc := newTestUseCase()

	content := `- [ ] Hoàn thành báo cáo
- [ ] Gửi email
- [ ] Cập nhật tài liệu`

	output, err := uc.UpdateCheckbox(context.Background(), checklist.UpdateCheckboxInput{
		Content:      content,
		CheckboxText: "Gửi email",
		Checked:      true,
	})

	assert.NoError(t, err)
	assert.True(t, output.Updated)
	assert.Contains(t, output.Content, "[ ] Hoàn thành")
	assert.Contains(t, output.Content, "[x] Gửi email")
	assert.Contains(t, output.Content, "[ ] Cập nhật")
}

func TestUpdateCheckbox_CaseInsensitive(t *testing.T) {
	uc := newTestUseCase()

	content := `- [ ] Complete Task
- [ ] Send Email
- [ ] Update Docs`

	output, err := uc.UpdateCheckbox(context.Background(), checklist.UpdateCheckboxInput{
		Content:      content,
		CheckboxText: "send email",
		Checked:      true,
	})

	assert.NoError(t, err)
	assert.True(t, output.Updated)
	assert.Contains(t, output.Content, "[x] Send Email")
}

func TestUpdateCheckbox_MultipleMatches(t *testing.T) {
	uc := newTestUseCase()

	content := `- [ ] Task 1
- [ ] Task 2
- [ ] Task 3`

	// "Task" matches all items
	output, err := uc.UpdateCheckbox(context.Background(), checklist.UpdateCheckboxInput{
		Content:      content,
		CheckboxText: "Task",
		Checked:      true,
	})

	assert.NoError(t, err)
	assert.True(t, output.Updated)
	assert.Equal(t, 3, output.Count)
	assert.Contains(t, output.Content, "[x] Task 1")
	assert.Contains(t, output.Content, "[x] Task 2")
	assert.Contains(t, output.Content, "[x] Task 3")
}

func TestUpdateCheckbox_NestedCheckboxes(t *testing.T) {
	uc := newTestUseCase()

	content := `- [ ] Parent 1
  - [ ] Child 1
  - [ ] Child 2
- [ ] Parent 2`

	output, err := uc.UpdateCheckbox(context.Background(), checklist.UpdateCheckboxInput{
		Content:      content,
		CheckboxText: "Child 1",
		Checked:      true,
	})

	assert.NoError(t, err)
	assert.True(t, output.Updated)
	assert.Contains(t, output.Content, "[ ] Parent 1")
	assert.Contains(t, output.Content, "[x] Child 1")
	assert.Contains(t, output.Content, "[ ] Child 2")
}
