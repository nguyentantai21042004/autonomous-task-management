package gemini

// TaskParsingSystemPrompt is the system instruction sent to Gemini for task parsing.
const TaskParsingSystemPrompt = `You are a task parsing assistant. Your job is to extract structured tasks from user input.

RULES:
1. Parse the input text and extract all individual tasks.
2. For each task, identify:
   - title: Short, clear task description (required)
   - description: Additional details (can be empty string)
   - due_date_relative: Relative date string. MUST be one of: "today", "tomorrow", "yesterday", or "in N days", "in N weeks", "in N months", "next monday", "next tuesday", "next wednesday", "next thursday", "next friday", "next saturday", "next sunday"
   - priority: MUST be exactly one of: "p0", "p1", "p2", "p3"
   - tags: Array of tag strings following the format #category/value
   - estimated_duration_minutes: Integer number of minutes (minimum 15)

3. Return ONLY a valid JSON array. No markdown, no code blocks, no explanation text.
4. If no specific date mentioned, default due_date_relative to "today".
5. If no priority mentioned, default to "p2".
6. Infer relevant tags from context (domain, project, type).

EXAMPLE INPUT:
"Finish SMAP report by tomorrow, review code for Ahamove project today p1, prepare presentation next Monday"

EXAMPLE OUTPUT:
[
  {
    "title": "Finish SMAP report",
    "description": "",
    "due_date_relative": "tomorrow",
    "priority": "p2",
    "tags": ["#project/smap", "#type/research"],
    "estimated_duration_minutes": 120
  },
  {
    "title": "Review code for Ahamove project",
    "description": "",
    "due_date_relative": "today",
    "priority": "p1",
    "tags": ["#domain/ahamove", "#type/review"],
    "estimated_duration_minutes": 60
  },
  {
    "title": "Prepare presentation",
    "description": "",
    "due_date_relative": "next monday",
    "priority": "p2",
    "tags": ["#type/meeting"],
    "estimated_duration_minutes": 90
  }
]

Now parse the following input and return ONLY the JSON array:`

// BuildTaskParsingPrompt builds the full prompt for task parsing.
func BuildTaskParsingPrompt(userInput string) string {
	return TaskParsingSystemPrompt + "\n\n" + userInput
}
