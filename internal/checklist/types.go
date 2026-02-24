package checklist

// Checkbox represents a single checkbox in markdown
type Checkbox struct {
	Line    int    // Line number in content
	Indent  string // Leading whitespace
	Checked bool   // true if [x], false if [ ]
	Text    string // Checkbox text content
	RawLine string // Original line
}

// ChecklistStats represents checklist progress
type ChecklistStats struct {
	Total     int     // Total checkboxes
	Completed int     // Checked checkboxes
	Pending   int     // Unchecked checkboxes
	Progress  float64 // Completion percentage (0-100)
}

// UpdateCheckboxInput is input for updating a checkbox
type UpdateCheckboxInput struct {
	Content      string // Original markdown content
	CheckboxText string // Text to match (partial match OK)
	Checked      bool   // New checked state
}

// UpdateCheckboxOutput is result of checkbox update
type UpdateCheckboxOutput struct {
	Content string // Updated markdown content
	Updated bool   // Whether any checkbox was updated
	Count   int    // Number of checkboxes updated
}
