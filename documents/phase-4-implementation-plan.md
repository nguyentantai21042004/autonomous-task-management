# PHASE 4: AUTOMATION & CHECKLIST MANAGEMENT

## ✅ Phase 3 Advanced Verification

**Phase 3 Advanced Deliverables Completed:**

- ✅ ReAct Agent Orchestrator with 5-step limit
- ✅ RAG with Unicode-safe context truncation
- ✅ Webhook sync with exponential backoff retry
- ✅ Multi-mode Telegram interface (`/search`, `/ask`, default create)
- ✅ Calendar conflict detection tool
- ✅ Complete wiring with proper dependency injection
- ✅ Both critical bugs fixed (Unicode slicing, goroutine leak)

**Ready for Phase 4:**

- Agent framework operational (Phase 3 Advanced)
- Webhook infrastructure ready (Phase 3 Advanced)
- Memos integration stable (Phase 2)
- Foundation for automation

---

## 🎯 Mục tiêu Phase 4

Xây dựng hệ thống tự động hóa checklist và webhook integration:

1. **Markdown Checklist Parser:** Parse và manipulate checklist trong Memos
2. **Webhook Receiver:** Nhận webhook từ Git platforms (GitHub/GitLab)
3. **Auto-Completion Logic:** Tự động tick checklist dựa trên events
4. **Bidirectional Sync:** Cập nhật Memos ↔ Qdrant khi checklist thay đổi
5. **Agent Tool Extension:** Agent có thể query và update checklist

**Key Features:**

- Regex-based checklist detection và manipulation
- Git webhook integration (PR merged, commit pushed)
- Automatic task completion based on external events
- Checklist progress tracking
- Smart task closure (auto-archive completed tasks)

---

## Kiến trúc Phase 4

```
┌─────────────────────────────────────────────────────────────┐
│                    External Events                           │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │   GitHub     │  │   GitLab     │  │   Manual     │      │
│  │   Webhook    │  │   Webhook    │  │   Telegram   │      │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘      │
└─────────┼──────────────────┼──────────────────┼─────────────┘
          │                  │                  │
          │ 1. Event         │                  │
          ▼                  ▼                  ▼
┌─────────────────────────────────────────────────────────────┐
│              Golang Backend (Orchestrator)                   │
│  ┌─────────────────────────────────────────────────────────┐│
│  │  Webhook Handler (internal/webhook/)                    ││
│  │  - Signature verification                               ││
│  │  - Event type routing                                   ││
│  │  - Rate limiting                                        ││
│  └──────────┬──────────────────────────────────────────────┘│
│             │ 2. Parsed event                               │
│             ▼                                                │
│  ┌─────────────────────────────────────────────────────────┐│
│  │  Checklist Service (internal/checklist/)                ││
│  │  - Regex parser (find checkboxes)                       ││
│  │  - Markdown manipulator (tick/untick)                   ││
│  │  - Progress calculator                                  ││
│  └──────────┬──────────────────────────────────────────────┘│
│             │ 3. Checklist operations                       │
│             ▼                                                │
│  ┌─────────────────────────────────────────────────────────┐│
│  │  Automation UseCase (internal/automation/)              ││
│  │  - Match event → task                                   ││
│  │  - Update Memos content                                 ││
│  │  - Trigger re-embedding                                 ││
│  │  - Archive completed tasks                              ││
│  └──────────┬──────────────────────────────────────────────┘│
└─────────────┼──────────────────────────────────────────────┘
              │
       ┌──────┴──────┐
       ▼             ▼
┌────────────┐  ┌──────────────┐
│   Memos    │  │   Qdrant     │
│    API     │  │   Vector DB  │
│  (Update)  │  │  (Re-embed)  │
└────────────┘  └──────────────┘
```

---

## 🚨 CRITICAL BUG FIXES (Must Read Before Implementation)

### Expert Review Summary - Final Version

This section contains fixes for **5 critical issues** (3 original + 2 new):

**Original Bugs (Fixed):**

1. ✅ **Vector Search for Exact Tags** - False positive matching (PR 123 → PR 124)
2. ✅ **Double Re-embedding** - Wasted API cost + race condition
3. ✅ **Rate Limiter Memory Leak** - OOM after days of running

**New Bugs (Fixed in Second Review):**

1. ✅ **Syntax Error in matcher.go** - Duplicate code block causing compile error
2. ✅ **Business Logic Flaw** - "closed" PR should NOT auto-complete tasks (only "merged")

**Final Bugs (Fixed in Third Review):**

1. ✅ **Push Event Blocker** - Push events incorrectly blocked by Action check
2. ✅ **Missing sanitizeContent** - Code blocks not sanitized before regex matching

**Pro-Tips Applied:**

- ✅ **Enhanced HMAC Security** - Raw byte comparison instead of hex strings
- ✅ **Partial Match Warning** - User notification when multiple checkboxes matched

**Optimizations Applied:**

- ✅ **Skip Unnecessary Memos Update** - Don't call API when content unchanged

---

### 🐞 FINAL FIX 1: Syntax Error in matcher.go

**Problem:** Duplicate code block outside function causing compile error

```go
// ❌ WRONG: Duplicate code after function ends
func (m *TaskMatcher) mergeMatches(...) []TaskMatch {
 // ... code ...
 return merged
}  // Function ends here

// ❌ This code is OUTSIDE the function - compile error!
for _, match := range keywordMatches {
 merged = append(merged, match)
}
return merged  // ❌ return outside function!
```

**Solution:** Remove duplicate code block

```go
// ✅ CORRECT: Clean function with no duplicates
func (m *TaskMatcher) mergeMatches(tagMatches, keywordMatches []TaskMatch) []TaskMatch {
 seen := make(map[string]bool)
 merged := make([]TaskMatch, 0)

 // Add tag matches first (higher priority - exact match)
 for _, match := range tagMatches {
  if !seen[match.TaskID] {
   merged = append(merged, match)
   seen[match.TaskID] = true
  }
 }

 // Add keyword matches (lower priority - semantic match)
 for _, match := range keywordMatches {
  if !seen[match.TaskID] {
   merged = append(merged, match)
   seen[match.TaskID] = true
  }
 }

 return merged
}  // ✅ Function ends cleanly
```

---

### 🚨 FINAL FIX 2: Business Logic Flaw - PR Closed vs Merged

**Problem:** Auto-completing tasks when PR is closed WITHOUT merge

```go
// ❌ WRONG: Treats "closed" same as "merged"
if event.Action != "merged" && event.Action != "closed" {
 // Skip
}
// This will auto-complete tasks even when PR is REJECTED!
```

**Why This is Critical:**

- GitHub/GitLab parser already distinguishes: `merged` vs `closed`
- If PR has `merged=true`, action becomes `"merged"`
- If action is still `"closed"`, it means PR was **REJECTED/CANCELLED**
- Auto-completing rejected PR tasks = **project management disaster!**

**Solution:** Only allow "merged" action

```go
// ✅ CORRECT: Only process merged PRs
if event.Action != "merged" {
 uc.l.Infof(ctx, "Skipping event with action: %s (only 'merged' triggers auto-completion)", event.Action)
 return ProcessWebhookOutput{
  TasksUpdated: 0,
  Message:      fmt.Sprintf("Event action '%s' not processed", event.Action),
 }, nil
}
```

**Business Logic:**

- `merged` → Auto-complete tasks ✅
- `closed` (without merge) → Do nothing ❌
- `opened`, `synchronize`, etc. → Do nothing ❌

---

### 💡 PRO-TIP 1: Enhanced HMAC Security

**Original Code:** Comparing hex-encoded strings

```go
// ❌ LESS SECURE: Hex string comparison
actualSig := hex.EncodeToString(mac.Sum(nil))
if !hmac.Equal([]byte(expectedSig), []byte(actualSig)) {
 return fmt.Errorf("signature verification failed")
}
```

**Improved Code:** Raw byte comparison

```go
// ✅ MORE SECURE: Raw byte comparison
expectedSigHex := signature[7:] // Remove "sha256=" prefix

// Decode hex to bytes
expectedSig, err := hex.DecodeString(expectedSigHex)
if err != nil {
 return fmt.Errorf("invalid signature hex encoding: %w", err)
}

// Calculate HMAC
mac := hmac.New(sha256.New, []byte(v.config.Secret))
mac.Write(payload)
actualSig := mac.Sum(nil)

// Compare raw bytes (more secure)
if !hmac.Equal(expectedSig, actualSig) {
 return fmt.Errorf("signature verification failed")
}
```

**Why This is Better:**

- Avoids string conversion overhead
- More resistant to encoding attacks
- Industry best practice for cryptographic comparison

---

### 💡 PRO-TIP 2: Partial Match Warning

**Problem:** `/check abc123 code` matches ALL checkboxes containing "code"

```markdown
- [ ] Review code ← Matched!
- [ ] Write code test ← Also matched!
- [ ] Deploy code ← Also matched!
```

**Solution:** Warn user when multiple matches occur

```go
// ⚠️ PRO-TIP: Partial match warning
warningMsg := ""
if output.Count > 1 {
 warningMsg = fmt.Sprintf("\n\n⚠️ Lưu ý: %d checkboxes được cập nhật. Nếu không đúng ý, hãy gõ text cụ thể hơn.", output.Count)
}

return h.bot.SendMessage(chatID, fmt.Sprintf("%s Đã cập nhật %d checkbox(es) matching %q%s",
 emoji, output.Count, itemText, warningMsg))
```

**User Experience:**

```
✅ Đã cập nhật 3 checkbox(es) matching "code"

⚠️ Lưu ý: 3 checkboxes được cập nhật. Nếu không đúng ý, hãy gõ text cụ thể hơn.
```

This helps users understand they need to be more specific: `/check abc123 Review code` instead of just `/check abc123 code`.

---

### ⚠️ CRITICAL BUG 1: Vector Search for Exact Tags (HARD BLOCKER)

**Problem in Original Plan:** Using vector search for exact tag matching

```go
// ❌ WRONG: Vector search treats #pr/123 and #pr/124 as semantically similar
func (m *TaskMatcher) searchByTags(ctx context.Context, criteria MatchCriteria) {
 query := strings.Join(criteria.Tags, " ")  // "#pr/123"
 results, err := m.vectorRepo.SearchTasks(ctx, repository.SearchTasksOptions{
  Query: query,  // ❌ Creates embedding, finds similar PRs!
  Limit: 10,
 })
}
```

**Why This Fails:**

- LLM embeddings understand semantic meaning
- `#pr/123` and `#pr/124` are 99% semantically similar (both are PR IDs)
- Vector search will return BOTH, causing false positives
- Merging PR 123 will auto-complete tasks for PR 124!

**Solution:** Use Qdrant Payload Filter for exact matching

```go
// ✅ CORRECT: Exact match using Qdrant payload filter
func (m *TaskMatcher) searchByTags(ctx context.Context, criteria MatchCriteria) {
 // Use Qdrant's Must filter for exact tag matching
 results, err := m.vectorRepo.SearchTasksWithFilter(ctx, repository.SearchTasksOptions{
  Filter: repository.PayloadFilter{
   Must: []repository.Condition{
    {
     Key:   "tags",
     Match: repository.MatchAny{Values: criteria.Tags},
    },
   },
  },
  Limit: 10,
 })
}
```

**Required Changes:**

1. Update Phase 3 embedding to store tags in payload
2. Add `SearchTasksWithFilter` method to VectorRepository
3. Use exact matching for tags, semantic search for keywords only

---

### ⚠️ CRITICAL BUG 2: Double Re-embedding (Event Loop Trap)

**Problem in Original Plan:** Calling EmbedTask twice for same update

```go
// ❌ WRONG: Manual re-embedding in Phase 4
func (uc *usecase) updateTaskChecklist(ctx context.Context, taskID string, content string) error {
 // Update Memos
 uc.memosRepo.UpdateTask(ctx, taskID, updatedContent)

 // ❌ Manual re-embed
 task, _ := uc.memosRepo.GetTask(ctx, taskID)
 uc.vectorRepo.EmbedTask(ctx, task)  // ❌ DUPLICATE!
}
```

**Why This Fails:**

- Phase 3 Advanced already has Memos webhook sync
- When you update Memos, it triggers webhook → Phase 3 re-embeds
- Phase 4 also calls EmbedTask → **Double embedding**
- Wastes API cost (2x Voyage AI calls)
- Race condition: Which embedding wins?

**Solution:** Trust Phase 3 webhook sync (Single Source of Truth)

```go
// ✅ CORRECT: Let Phase 3 handle re-embedding
func (uc *usecase) updateTaskChecklist(ctx context.Context, taskID string, content string) error {
 // Update Memos
 if err := uc.memosRepo.UpdateTask(ctx, taskID, updatedContent); err != nil {
  return fmt.Errorf("failed to update Memos: %w", err)
 }

 // ✅ DO NOTHING - Phase 3 webhook will handle re-embedding automatically
 uc.l.Infof(ctx, "Updated task %s, Phase 3 webhook will re-embed", taskID)
 return nil
}
```

**Architecture Principle:** Phase 3 owns Memos ↔ Qdrant sync. Phase 4 only updates Memos.

---

### ⚠️ CRITICAL BUG 3: Rate Limiter Memory Leak

**Problem in Original Plan:** Custom rate limiter with unbounded map

```go
// ❌ WRONG: Map grows forever, never cleaned up
type rateLimiter struct {
 mu      sync.Mutex
 buckets map[string]*bucket  // ❌ Never deleted, grows forever
}

func (rl *rateLimiter) Allow(key string) error {
 b, exists := rl.buckets[key]
 if !exists {
  b = &bucket{...}
  rl.buckets[key] = b  // ❌ New entry for every unique IP
 }
 // No cleanup logic → OOM after days of running
}
```

**Why This Fails:**

- Every unique IP/source creates a new map entry
- Map never cleaned up → memory leak
- After days: thousands of IPs → OOM crash
- Integer seconds calculation causes jitter

**Solution:** Use `golang.org/x/time/rate` with LRU cache

```go
// ✅ CORRECT: Use standard library with LRU cleanup
import (
 "golang.org/x/time/rate"
 "github.com/hashicorp/golang-lru/v2/expirable"
)

type rateLimiter struct {
 limiters *expirable.LRU[string, *rate.Limiter]
 rate     rate.Limit
 burst    int
}

func newRateLimiter(requestsPerMin int) *rateLimiter {
 return &rateLimiter{
  // Auto-cleanup after 5 minutes of inactivity
  limiters: expirable.NewLRU[string, *rate.Limiter](1000, nil, time.Minute*5),
  rate:     rate.Limit(float64(requestsPerMin) / 60.0), // Per second
  burst:    requestsPerMin / 10, // Allow burst
 }
}

func (rl *rateLimiter) Allow(key string) error {
 limiter, ok := rl.limiters.Get(key)
 if !ok {
  limiter = rate.NewLimiter(rl.rate, rl.burst)
  rl.limiters.Add(key, limiter)
 }

 if !limiter.Allow() {
  return fmt.Errorf("rate limit exceeded for %s", key)
 }
 return nil
}
```

**Benefits:**

- Auto-cleanup after 5 minutes inactivity
- Microsecond precision
- Thread-safe
- Production-tested

---

### ⚠️ FINAL BUG 6: Push Event Blocker (Logical Flaw)

**Problem in Updated Plan:** Action check blocks ALL events including push

```go
// ❌ WRONG: Blocks push events which have empty Action field
if event.Action != "merged" {
 uc.l.Infof(ctx, "Skipping event...")
 return ProcessWebhookOutput{...}, nil
}
```

**Why This Fails:**

- Phase 4 goal: "Git webhook integration (PR merged, **commit pushed**)"
- Push events have **empty Action field** (`Action = ""`)
- Current condition blocks ALL push events!
- Only PR/MR events have Action field (`opened`, `closed`, `merged`)

**Real-World Scenario:**

```
User commits directly to main branch:
→ GitHub sends push event with Action=""
→ Your code checks: "" != "merged" → BLOCKED!
→ Automation never runs for direct commits
```

**Solution:** Only check Action for PR/MR events

```go
// ✅ CORRECT: Only validate Action for PR/MR events
if (event.EventType == "pull_request" || event.EventType == "merge_request") && event.Action != "merged" {
 uc.l.Infof(ctx, "Skipping PR/MR event with action: %s (only 'merged' triggers auto-completion)", event.Action)
 return ProcessWebhookOutput{
  TasksUpdated: 0,
  Message:      fmt.Sprintf("PR/MR action '%s' not processed", event.Action),
 }, nil
}

// Push events (Action="") will pass through and continue processing
uc.l.Infof(ctx, "Processing %s event", event.EventType)
```

**Event Type Matrix:**

| Event Type      | Action Field | Should Process?  |
| --------------- | ------------ | ---------------- |
| `push`          | `""` (empty) | ✅ YES           |
| `pull_request`  | `opened`     | ❌ NO            |
| `pull_request`  | `closed`     | ❌ NO (rejected) |
| `pull_request`  | `merged`     | ✅ YES           |
| `merge_request` | `merged`     | ✅ YES           |

---

### ⚠️ FINAL BUG 7: Missing sanitizeContent Implementation

**Problem in Updated Plan:** Optimization mentioned but not integrated

```go
// ❌ WRONG: ParseCheckboxes still uses raw content
func (s *service) ParseCheckboxes(content string) []Checkbox {
 matches := s.pattern.FindAllStringSubmatch(content, -1)  // ❌ No sanitization!
 // ... rest of logic
}
```

**Why This Fails:**

- Optimization section shows `sanitizeContent()` helper
- But `ParseCheckboxes()` doesn't call it!
- Code blocks still contain fake checkboxes
- Automation will match checkboxes in code examples

**Real-World Scenario:**

````markdown
## Task: Implement feature

- [ ] Write code
- [ ] Test code

Example usage:

```markdown
- [ ] This is a fake checkbox in documentation
```
````

**Without sanitization:** Regex finds 3 checkboxes (including fake one)
**With sanitization:** Regex finds 2 checkboxes (only real ones)

**Solution:** Implement and integrate sanitizeContent

````go
// ✅ STEP 1: Add helper function to service.go
func sanitizeContent(content string) string {
 // Remove fenced code blocks (```...```)
 codeBlockPattern := regexp.MustCompile("(?s)```.*?```")
 sanitized := codeBlockPattern.ReplaceAllString(content, "")

 // Remove inline code blocks (`...`)
 inlineCodePattern := regexp.MustCompile("`[^`]+`")
 sanitized = inlineCodePattern.ReplaceAllString(sanitized, "")

 return sanitized
}

// ✅ STEP 2: Use in ParseCheckboxes
func (s *service) ParseCheckboxes(content string) []Checkbox {
 // Sanitize before regex matching
 sanitized := sanitizeContent(content)
 matches := s.pattern.FindAllStringSubmatch(sanitized, -1)

 checkboxes := make([]Checkbox, 0, len(matches))
 for i, match := range matches {
  checkboxes = append(checkboxes, Checkbox{
   Index:     i,
   Indent:    match[1],
   Checked:   match[2] == "x",
   Text:      match[3],
   FullMatch: match[0],
  })
 }
 return checkboxes
}
````

**Note on UpdateCheckbox/UpdateAllCheckboxes:**

At MVP level, these functions can still use raw content (risk of updating fake checkboxes in code blocks is low since users rarely put checkboxes in code). For 100% correctness, you'd need a full markdown AST parser instead of regex.

---

### 💡 OPTIMIZATION: Skip Unnecessary Memos Update

**Problem:** Calling Memos API even when content unchanged

```go
// ❌ INEFFICIENT: Always calls UpdateTask even if nothing changed
func (uc *usecase) CompleteTask(ctx context.Context, taskID string) error {
 task, _ := uc.memosRepo.GetTask(ctx, taskID)
 updatedContent := uc.checklistSvc.UpdateAllCheckboxes(task.Content, true)

 // ❌ What if task already 100% completed? Wasted API call!
 if err := uc.memosRepo.UpdateTask(ctx, taskID, updatedContent); err != nil {
  return err
 }
 return nil
}
```

**Why This Matters:**

- Task already 100% completed → `updatedContent == task.Content`
- Calling `UpdateTask` with identical content wastes:
  - HTTP request to Memos
  - Database write operation
  - Webhook trigger (Phase 3 re-embedding)
- Multiplied by hundreds of automation events = significant overhead

**Solution:** Check for changes before updating

```go
// ✅ OPTIMIZED: Skip update if content unchanged
func (uc *usecase) CompleteTask(ctx context.Context, taskID string) error {
 task, err := uc.memosRepo.GetTask(ctx, taskID)
 if err != nil {
  return fmt.Errorf("failed to get task: %w", err)
 }

 updatedContent := uc.checklistSvc.UpdateAllCheckboxes(task.Content, true)

 // ✅ Skip update if nothing changed
 if updatedContent == task.Content {
  uc.l.Infof(ctx, "Task %s already fully completed, skipping update", taskID)
  return nil
 }

 // Only update when content actually changed
 if err := uc.memosRepo.UpdateTask(ctx, taskID, updatedContent); err != nil {
  return fmt.Errorf("failed to update task: %w", err)
 }

 uc.l.Infof(ctx, "Task %s updated successfully", taskID)
 return nil
}
```

**Performance Impact:**

- Saves ~50% of Memos API calls for already-completed tasks
- Reduces unnecessary webhook triggers
- Prevents redundant re-embedding operations

---

### Decision 1: Checklist Format Standard

**Problem:** Memos supports multiple checkbox formats:

- `- [ ] Task` (GitHub-style)
- `* [ ] Task` (Alternative)
- `+ [ ] Task` (Alternative)

**Solution:** Standardize on GitHub-style `- [ ]` and `- [x]`

**Rationale:**

- Most widely used format
- Compatible with GitHub/GitLab rendering
- Easier regex patterns
- Better user familiarity

**Implementation:**

```go
const (
 CheckboxUnchecked = `- [ ]`
 CheckboxChecked   = `- [x]`
 CheckboxPattern   = `(?m)^(\s*)- \[([ x])\] (.+)$`
)
```

---

### 💡 OPTIMIZATION: Protect Regex from Code Blocks

**Problem:** Regex matches checkboxes inside markdown code blocks

````markdown
## Task

- [ ] Real task

Example code:

```md
- [ ] Fake task in code block
```
````

````

**Impact:** Automation will tick fake checkboxes in code examples

**Solution:** Sanitize content before regex matching

```go
// sanitizeContent removes code blocks before checkbox parsing
func sanitizeContent(content string) string {
 // Remove fenced code blocks (```...```)
 codeBlockPattern := regexp.MustCompile("(?s)```.*?```")
 sanitized := codeBlockPattern.ReplaceAllString(content, "")

 // Remove inline code blocks (`...`)
 inlineCodePattern := regexp.MustCompile("`[^`]+`")
 sanitized = inlineCodePattern.ReplaceAllString(sanitized, "")

 return sanitized
}

// Use in ParseCheckboxes
func (s *service) ParseCheckboxes(content string) []Checkbox {
 sanitized := sanitizeContent(content)
 matches := s.pattern.FindAllStringSubmatch(sanitized, -1)
 // ... rest of logic
}
````

---

### Decision 2: Webhook Security

**Problem:** Public webhook endpoints are vulnerable to:

- Replay attacks
- Spoofed requests
- DDoS

**Solution:** Multi-layer security

1. **Signature Verification:** Validate HMAC signature from Git platforms
2. **Secret Token:** Shared secret in webhook configuration
3. **Rate Limiting:** Max 100 requests/minute per source
4. **IP Whitelist:** Optional IP restriction for known sources

**Implementation:**

```go
func (h *WebhookHandler) verifyGitHubSignature(payload []byte, signature string) bool {
 mac := hmac.New(sha256.New, []byte(h.webhookSecret))
 mac.Write(payload)
 expectedMAC := hex.EncodeToString(mac.Sum(nil))
 return hmac.Equal([]byte(signature), []byte("sha256="+expectedMAC))
}
```

---

### Decision 3: Task Matching Strategy

**Problem:** How to match webhook event → specific Memos task?

**Options:**

1. **Tag-based:** Match by tags (e.g., `#repo/myproject`)
2. **URL-based:** Embed Memos URL in PR description
3. **Keyword-based:** Search task content for PR/issue number

**Solution:** Hybrid approach (Tag + Keyword)

**Rationale:**

- Tag-based: Fast, reliable for project-level tasks
- Keyword-based: Flexible for specific PR/issue references
- Fallback: If no match, log and skip (no false positives)

**Example:**

```markdown
## Task in Memos

#repo/myproject #pr/123

- [ ] Implement user authentication
- [ ] Write unit tests
- [ ] Update documentation

PR: https://github.com/user/myproject/pull/123
```

When PR #123 merged → Auto-tick all checkboxes

---

### Decision 4: Completion Threshold

**Problem:** When to consider a task "completed"?

**Options:**

1. All checkboxes ticked (100%)
2. Majority ticked (>50%)
3. Manual trigger only

**Solution:** 100% completion + manual override

**Rationale:**

- Clear, unambiguous rule
- Prevents premature archival
- User can manually mark complete via Telegram `/complete <task_id>`

**Implementation:**

```go
func (s *ChecklistService) IsFullyCompleted(content string) bool {
 checkboxes := s.ParseCheckboxes(content)
 if len(checkboxes) == 0 {
  return false // No checkboxes = not a checklist task
 }

 for _, cb := range checkboxes {
  if !cb.Checked {
   return false
  }
 }
 return true
}
```

---

## Cấu trúc Module Mới

```
internal/
├── checklist/                  # Checklist parsing & manipulation
│   ├── service.go              # Core checklist logic
│   ├── parser.go               # Regex-based parser
│   ├── types.go                # Checkbox, ChecklistItem structs
│   └── new.go                  # Factory
├── automation/                 # Automation orchestration
│   ├── interface.go            # UseCase interface
│   ├── usecase.go              # Business logic
│   ├── types.go                # Input/Output structs
│   ├── matcher.go              # Event → Task matching
│   └── new.go                  # Factory
├── webhook/                    # Webhook receivers
│   ├── handler.go              # HTTP handlers
│   ├── github.go               # GitHub webhook parser
│   ├── gitlab.go               # GitLab webhook parser
│   ├── types.go                # Webhook event structs
│   ├── security.go             # Signature verification
│   └── new.go                  # Factory
└── agent/
    └── tools/
        ├── update_checklist.go # Agent tool for checklist updates
        └── get_progress.go     # Agent tool for progress query

pkg/
└── markdown/                   # Markdown utilities
    ├── parser.go               # Generic markdown parser
    ├── renderer.go             # Markdown renderer
    └── types.go                # AST types

config/
└── config.yaml                 # Add webhook config section
```

---

## Task Breakdown

### Task 4.1: Checklist Service (Core Logic)

**Mục tiêu:** Xây dựng service để parse và manipulate checklist trong Markdown

**Files:**

- `internal/checklist/service.go`
- `internal/checklist/parser.go`
- `internal/checklist/types.go`
- `internal/checklist/new.go`

**Types Definition:**

```go
// internal/checklist/types.go
package checklist

import "time"

// Checkbox represents a single checkbox in markdown
type Checkbox struct {
 Line      int       // Line number in content
 Indent    string    // Leading whitespace
 Checked   bool      // true if [x], false if [ ]
 Text      string    // Checkbox text content
 RawLine   string    // Original line
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
 Content  string // Updated markdown content
 Updated  bool   // Whether any checkbox was updated
 Count    int    // Number of checkboxes updated
}
```

**Service Interface:**

````go
// internal/checklist/service.go
package checklist

import (
 "context"
 "regexp"
 "strings"
)

const (
 CheckboxUnchecked = `- [ ]`
 CheckboxChecked   = `- [x]`
 // Regex pattern: captures indent, checkbox state, and text
 // Example: "  - [x] Task name" → groups: ["  ", "x", "Task name"]
 CheckboxPattern = `(?m)^(\s*)- \[([ xX])\] (.+)$`
)

type Service interface {
 // ParseCheckboxes extracts all checkboxes from markdown content
 ParseCheckboxes(content string) []Checkbox

 // GetStats calculates checklist statistics
 GetStats(content string) ChecklistStats

 // UpdateCheckbox updates checkbox state by text match
 UpdateCheckbox(ctx context.Context, input UpdateCheckboxInput) (UpdateCheckboxOutput, error)

 // UpdateAllCheckboxes sets all checkboxes to specified state
 UpdateAllCheckboxes(content string, checked bool) string

 // IsFullyCompleted checks if all checkboxes are checked
 IsFullyCompleted(content string) bool
}

type service struct {
 pattern *regexp.Regexp
}

func New() Service {
 return &service{
  pattern: regexp.MustCompile(CheckboxPattern),
 }
}

// sanitizeContent removes code blocks before checkbox parsing
// ✅ OPTIMIZATION: Prevents matching fake checkboxes in code examples
func sanitizeContent(content string) string {
 // Remove fenced code blocks (```...```)
 fencedCodeBlockPattern := regexp.MustCompile("(?s)```.*?```")
 sanitized := fencedCodeBlockPattern.ReplaceAllString(content, "")

 // Remove inline code blocks (`...`)
 inlineCodePattern := regexp.MustCompile("`[^`]+`")
 sanitized = inlineCodePattern.ReplaceAllString(sanitized, "")

 return sanitized
}

// ParseCheckboxes extracts all checkboxes from markdown
func (s *service) ParseCheckboxes(content string) []Checkbox {
 // ✅ Sanitize first to remove code blocks
 sanitized := sanitizeContent(content)

 matches := s.pattern.FindAllStringSubmatch(sanitized, -1)
 checkboxes := make([]Checkbox, 0, len(matches))

 lineNum := 0
 for _, match := range matches {
  if len(match) != 4 {
   continue
  }

  checkbox := Checkbox{
   Line:    lineNum,
   Indent:  match[1],
   Checked: strings.ToLower(match[2]) == "x",
   Text:    strings.TrimSpace(match[3]),
   RawLine: match[0],
  }
  checkboxes = append(checkboxes, checkbox)
  lineNum++
 }

 return checkboxes
}

// GetStats calculates checklist statistics
func (s *service) GetStats(content string) ChecklistStats {
 checkboxes := s.ParseCheckboxes(content)
 total := len(checkboxes)

 if total == 0 {
  return ChecklistStats{
   Total:     0,
   Completed: 0,
   Pending:   0,
   Progress:  0,
  }
 }

 completed := 0
 for _, cb := range checkboxes {
  if cb.Checked {
   completed++
  }
 }

 pending := total - completed
 progress := float64(completed) / float64(total) * 100

 return ChecklistStats{
  Total:     total,
  Completed: completed,
  Pending:   pending,
  Progress:  progress,
 }
}

// UpdateCheckbox updates checkbox state by text match (partial match)
func (s *service) UpdateCheckbox(ctx context.Context, input UpdateCheckboxInput) (UpdateCheckboxOutput, error) {
 if input.Content == "" {
  return UpdateCheckboxOutput{Content: input.Content, Updated: false}, nil
 }

 lines := strings.Split(input.Content, "\n")
 updated := false
 count := 0

 // Normalize search text for matching
 searchText := strings.ToLower(strings.TrimSpace(input.CheckboxText))

 for i, line := range lines {
  // Check if line is a checkbox
  if !strings.Contains(line, "- [") {
   continue
  }

  // Extract checkbox text
  matches := s.pattern.FindStringSubmatch(line)
  if len(matches) != 4 {
   continue
  }

  checkboxText := strings.ToLower(strings.TrimSpace(matches[3]))

  // Partial match: if search text is substring of checkbox text
  if !strings.Contains(checkboxText, searchText) {
   continue
  }

  // Update checkbox state
  indent := matches[1]
  text := matches[3]

  if input.Checked {
   lines[i] = indent + CheckboxChecked + " " + text
  } else {
   lines[i] = indent + CheckboxUnchecked + " " + text
  }

  updated = true
  count++
 }

 return UpdateCheckboxOutput{
  Content: strings.Join(lines, "\n"),
  Updated: updated,
  Count:   count,
 }, nil
}

// UpdateAllCheckboxes sets all checkboxes to specified state
func (s *service) UpdateAllCheckboxes(content string, checked bool) string {
 state := CheckboxUnchecked
 if checked {
  state = CheckboxChecked
 }

 // Replace all checkbox states
 result := s.pattern.ReplaceAllStringFunc(content, func(match string) string {
  matches := s.pattern.FindStringSubmatch(match)
  if len(matches) != 4 {
   return match
  }
  return matches[1] + state + " " + matches[3]
 })

 return result
}

// IsFullyCompleted checks if all checkboxes are checked
func (s *service) IsFullyCompleted(content string) bool {
 checkboxes := s.ParseCheckboxes(content)
 if len(checkboxes) == 0 {
  return false // No checkboxes = not a checklist task
 }

 for _, cb := range checkboxes {
  if !cb.Checked {
   return false
  }
 }
 return true
}
````

---

### Task 4.2: Webhook Security & Verification

**Mục tiêu:** Xây dựng layer bảo mật cho webhook endpoints

**Files:**

- `internal/webhook/security.go`
- `internal/webhook/types.go`

**Implementation:**

```go
// internal/webhook/types.go
package webhook

import "time"

// WebhookSource represents the source platform
type WebhookSource string

const (
 SourceGitHub WebhookSource = "github"
 SourceGitLab WebhookSource = "gitlab"
 SourceManual WebhookSource = "manual"
)

// WebhookEvent represents a parsed webhook event
type WebhookEvent struct {
 Source      WebhookSource          // Platform source
 EventType   string                 // Event type (push, pull_request, etc.)
 Repository  string                 // Repository name
 Branch      string                 // Branch name
 Commit      string                 // Commit SHA
 Author      string                 // Event author
 Message     string                 // Commit/PR message
 PRNumber    int                    // PR number (if applicable)
 IssueNumber int                    // Issue number (if applicable)
 Action      string                 // Action (opened, closed, merged, etc.)
 Metadata    map[string]interface{} // Additional metadata
 ReceivedAt  time.Time              // When webhook was received
}

// SecurityConfig holds webhook security settings
type SecurityConfig struct {
 Secret          string   // Shared secret for signature verification
 AllowedIPs      []string // IP whitelist (optional)
 RateLimitPerMin int      // Max requests per minute
}
```

```go
// internal/webhook/security.go
package webhook

import (
 "crypto/hmac"
 "crypto/sha256"
 "encoding/hex"
 "fmt"
 "net"
 "net/http"
 "strings"
 "sync"
 "time"
)

// SecurityValidator validates webhook requests
type SecurityValidator struct {
 config      SecurityConfig
 rateLimiter *rateLimiter
}

func NewSecurityValidator(config SecurityConfig) *SecurityValidator {
 return &SecurityValidator{
  config:      config,
  rateLimiter: newRateLimiter(config.RateLimitPerMin),
 }
}

// ValidateGitHubSignature verifies GitHub webhook signature
func (v *SecurityValidator) ValidateGitHubSignature(payload []byte, signature string) error {
 if v.config.Secret == "" {
  return fmt.Errorf("webhook secret not configured")
 }

 // GitHub sends signature as "sha256=<hex>"
 if !strings.HasPrefix(signature, "sha256=") {
  return fmt.Errorf("invalid signature format")
 }

 expectedSigHex := signature[7:] // Remove "sha256=" prefix

 // ✅ PRO-TIP: Decode hex to bytes for more secure comparison
 expectedSig, err := hex.DecodeString(expectedSigHex)
 if err != nil {
  return fmt.Errorf("invalid signature hex encoding: %w", err)
 }

 // Calculate HMAC
 mac := hmac.New(sha256.New, []byte(v.config.Secret))
 mac.Write(payload)
 actualSig := mac.Sum(nil)

 // ✅ Constant-time comparison on raw bytes (more secure than hex strings)
 if !hmac.Equal(expectedSig, actualSig) {
  return fmt.Errorf("signature verification failed")
 }

 return nil
}

// ValidateGitLabToken verifies GitLab webhook token
func (v *SecurityValidator) ValidateGitLabToken(token string) error {
 if v.config.Secret == "" {
  return fmt.Errorf("webhook secret not configured")
 }

 if token != v.config.Secret {
  return fmt.Errorf("invalid token")
 }

 return nil
}

// ValidateIPAddress checks if request IP is whitelisted
func (v *SecurityValidator) ValidateIPAddress(r *http.Request) error {
 if len(v.config.AllowedIPs) == 0 {
  return nil // No IP restriction
 }

 // Extract IP from request
 ip := extractIP(r)

 // Check against whitelist
 for _, allowedIP := range v.config.AllowedIPs {
  if ip == allowedIP {
   return nil
  }

  // Check CIDR range
  if strings.Contains(allowedIP, "/") {
   _, ipNet, err := net.ParseCIDR(allowedIP)
   if err != nil {
    continue
   }
   if ipNet.Contains(net.ParseIP(ip)) {
    return nil
   }
  }
 }

 return fmt.Errorf("IP %s not whitelisted", ip)
}

// CheckRateLimit enforces rate limiting
func (v *SecurityValidator) CheckRateLimit(source string) error {
 return v.rateLimiter.Allow(source)
}

// extractIP extracts client IP from request
func extractIP(r *http.Request) string {
 // Check X-Forwarded-For header (proxy/load balancer)
 if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
  ips := strings.Split(xff, ",")
  return strings.TrimSpace(ips[0])
 }

 // Check X-Real-IP header
 if xri := r.Header.Get("X-Real-IP"); xri != "" {
  return xri
 }

 // Fallback to RemoteAddr
 ip, _, _ := net.SplitHostPort(r.RemoteAddr)
 return ip
}

// ✅ FIXED: Production-grade rate limiter with auto-cleanup
// Uses golang.org/x/time/rate with LRU cache to prevent memory leaks
type rateLimiter struct {
 limiters *expirable.LRU[string, *rate.Limiter]
 rate     rate.Limit
 burst    int
}

func newRateLimiter(requestsPerMin int) *rateLimiter {
 return &rateLimiter{
  // Auto-cleanup after 5 minutes of inactivity
  limiters: expirable.NewLRU[string, *rate.Limiter](
   1000,          // Max 1000 unique sources
   nil,           // No eviction callback
   time.Minute*5, // TTL: 5 minutes
  ),
  rate:  rate.Limit(float64(requestsPerMin) / 60.0), // Per second
  burst: requestsPerMin / 10,                         // Allow burst
 }
}

func (rl *rateLimiter) Allow(key string) error {
 limiter, ok := rl.limiters.Get(key)
 if !ok {
  limiter = rate.NewLimiter(rl.rate, rl.burst)
  rl.limiters.Add(key, limiter)
 }

 if !limiter.Allow() {
  return fmt.Errorf("rate limit exceeded for %s", key)
 }
 return nil
}
```

**Required imports:**

```go
import (
 "fmt"
 "time"

 "golang.org/x/time/rate"
 "github.com/hashicorp/golang-lru/v2/expirable"
)
```

**Update go.mod:**

```bash
go get golang.org/x/time/rate
go get github.com/hashicorp/golang-lru/v2
```

---

### Task 4.3: GitHub Webhook Handler

**Mục tiêu:** Parse và xử lý GitHub webhook events

**Files:**

- `internal/webhook/github.go`

**Implementation:**

```go
// internal/webhook/github.go
package webhook

import (
 "encoding/json"
 "fmt"
 "time"
)

// GitHubWebhookParser parses GitHub webhook payloads
type GitHubWebhookParser struct{}

func NewGitHubParser() *GitHubWebhookParser {
 return &GitHubWebhookParser{}
}

// ParsePushEvent parses GitHub push event
func (p *GitHubWebhookParser) ParsePushEvent(payload []byte) (*WebhookEvent, error) {
 var event struct {
  Ref        string `json:"ref"`
  Repository struct {
   FullName string `json:"full_name"`
  } `json:"repository"`
  HeadCommit struct {
   ID      string `json:"id"`
   Message string `json:"message"`
   Author  struct {
    Name string `json:"name"`
   } `json:"author"`
  } `json:"head_commit"`
 }

 if err := json.Unmarshal(payload, &event); err != nil {
  return nil, fmt.Errorf("failed to parse push event: %w", err)
 }

 // Extract branch name from ref (refs/heads/main → main)
 branch := event.Ref
 if len(branch) > 11 && branch[:11] == "refs/heads/" {
  branch = branch[11:]
 }

 return &WebhookEvent{
  Source:     SourceGitHub,
  EventType:  "push",
  Repository: event.Repository.FullName,
  Branch:     branch,
  Commit:     event.HeadCommit.ID,
  Author:     event.HeadCommit.Author.Name,
  Message:    event.HeadCommit.Message,
  ReceivedAt: time.Now(),
 }, nil
}

// ParsePullRequestEvent parses GitHub pull request event
func (p *GitHubWebhookParser) ParsePullRequestEvent(payload []byte) (*WebhookEvent, error) {
 var event struct {
  Action      string `json:"action"` // opened, closed, merged, etc.
  Number      int    `json:"number"`
  PullRequest struct {
   Title string `json:"title"`
   Head  struct {
    Ref string `json:"ref"` // Branch name
    SHA string `json:"sha"` // Commit SHA
   } `json:"head"`
   User struct {
    Login string `json:"login"`
   } `json:"user"`
   Merged bool `json:"merged"`
  } `json:"pull_request"`
  Repository struct {
   FullName string `json:"full_name"`
  } `json:"repository"`
 }

 if err := json.Unmarshal(payload, &event); err != nil {
  return nil, fmt.Errorf("failed to parse pull request event: %w", err)
 }

 // Determine action (merged takes precedence over closed)
 action := event.Action
 if action == "closed" && event.PullRequest.Merged {
  action = "merged"
 }

 return &WebhookEvent{
  Source:     SourceGitHub,
  EventType:  "pull_request",
  Repository: event.Repository.FullName,
  Branch:     event.PullRequest.Head.Ref,
  Commit:     event.PullRequest.Head.SHA,
  Author:     event.PullRequest.User.Login,
  Message:    event.PullRequest.Title,
  PRNumber:   event.Number,
  Action:     action,
  ReceivedAt: time.Now(),
 }, nil
}

// ParseIssueEvent parses GitHub issue event
func (p *GitHubWebhookParser) ParseIssueEvent(payload []byte) (*WebhookEvent, error) {
 var event struct {
  Action string `json:"action"` // opened, closed, etc.
  Issue  struct {
   Number int    `json:"number"`
   Title  string `json:"title"`
   User   struct {
    Login string `json:"login"`
   } `json:"user"`
  } `json:"issue"`
  Repository struct {
   FullName string `json:"full_name"`
  } `json:"repository"`
 }

 if err := json.Unmarshal(payload, &event); err != nil {
  return nil, fmt.Errorf("failed to parse issue event: %w", err)
 }

 return &WebhookEvent{
  Source:      SourceGitHub,
  EventType:   "issue",
  Repository:  event.Repository.FullName,
  Author:      event.Issue.User.Login,
  Message:     event.Issue.Title,
  IssueNumber: event.Issue.Number,
  Action:      event.Action,
  ReceivedAt:  time.Now(),
 }, nil
}
```

---

### Task 4.4: GitLab Webhook Handler

**Mục tiêu:** Parse và xử lý GitLab webhook events

**Files:**

- `internal/webhook/gitlab.go`

**Implementation:**

```go
// internal/webhook/gitlab.go
package webhook

import (
 "encoding/json"
 "fmt"
 "time"
)

// GitLabWebhookParser parses GitLab webhook payloads
type GitLabWebhookParser struct{}

func NewGitLabParser() *GitLabWebhookParser {
 return &GitLabWebhookParser{}
}

// ParsePushEvent parses GitLab push event
func (p *GitLabWebhookParser) ParsePushEvent(payload []byte) (*WebhookEvent, error) {
 var event struct {
  ObjectKind string `json:"object_kind"`
  Ref        string `json:"ref"`
  Project    struct {
   PathWithNamespace string `json:"path_with_namespace"`
  } `json:"project"`
  Commits []struct {
   ID      string `json:"id"`
   Message string `json:"message"`
   Author  struct {
    Name string `json:"name"`
   } `json:"author"`
  } `json:"commits"`
 }

 if err := json.Unmarshal(payload, &event); err != nil {
  return nil, fmt.Errorf("failed to parse push event: %w", err)
 }

 // Extract branch name from ref
 branch := event.Ref
 if len(branch) > 11 && branch[:11] == "refs/heads/" {
  branch = branch[11:]
 }

 // Get last commit
 var commit, message, author string
 if len(event.Commits) > 0 {
  lastCommit := event.Commits[len(event.Commits)-1]
  commit = lastCommit.ID
  message = lastCommit.Message
  author = lastCommit.Author.Name
 }

 return &WebhookEvent{
  Source:     SourceGitLab,
  EventType:  "push",
  Repository: event.Project.PathWithNamespace,
  Branch:     branch,
  Commit:     commit,
  Author:     author,
  Message:    message,
  ReceivedAt: time.Now(),
 }, nil
}

// ParseMergeRequestEvent parses GitLab merge request event
func (p *GitLabWebhookParser) ParseMergeRequestEvent(payload []byte) (*WebhookEvent, error) {
 var event struct {
  ObjectKind       string `json:"object_kind"`
  ObjectAttributes struct {
   IID          int    `json:"iid"` // MR number
   Title        string `json:"title"`
   State        string `json:"state"` // opened, closed, merged
   Action       string `json:"action"`
   SourceBranch string `json:"source_branch"`
   LastCommit   struct {
    ID string `json:"id"`
   } `json:"last_commit"`
  } `json:"object_attributes"`
  User struct {
   Name string `json:"name"`
  } `json:"user"`
  Project struct {
   PathWithNamespace string `json:"path_with_namespace"`
  } `json:"project"`
 }

 if err := json.Unmarshal(payload, &event); err != nil {
  return nil, fmt.Errorf("failed to parse merge request event: %w", err)
 }

 return &WebhookEvent{
  Source:     SourceGitLab,
  EventType:  "merge_request",
  Repository: event.Project.PathWithNamespace,
  Branch:     event.ObjectAttributes.SourceBranch,
  Commit:     event.ObjectAttributes.LastCommit.ID,
  Author:     event.User.Name,
  Message:    event.ObjectAttributes.Title,
  PRNumber:   event.ObjectAttributes.IID,
  Action:     event.ObjectAttributes.Action,
  ReceivedAt: time.Now(),
 }, nil
}

// ParseIssueEvent parses GitLab issue event
func (p *GitLabWebhookParser) ParseIssueEvent(payload []byte) (*WebhookEvent, error) {
 var event struct {
  ObjectKind       string `json:"object_kind"`
  ObjectAttributes struct {
   IID    int    `json:"iid"` // Issue number
   Title  string `json:"title"`
   State  string `json:"state"`
   Action string `json:"action"`
  } `json:"object_attributes"`
  User struct {
   Name string `json:"name"`
  } `json:"user"`
  Project struct {
   PathWithNamespace string `json:"path_with_namespace"`
  } `json:"project"`
 }

 if err := json.Unmarshal(payload, &event); err != nil {
  return nil, fmt.Errorf("failed to parse issue event: %w", err)
 }

 return &WebhookEvent{
  Source:      SourceGitLab,
  EventType:   "issue",
  Repository:  event.Project.PathWithNamespace,
  Author:      event.User.Name,
  Message:     event.ObjectAttributes.Title,
  IssueNumber: event.ObjectAttributes.IID,
  Action:      event.ObjectAttributes.Action,
  ReceivedAt:  time.Now(),
 }, nil
}
```

---

### Task 4.5: Automation UseCase (Business Logic)

**Mục tiêu:** Orchestrate automation flow: Event → Match Task → Update Checklist → Re-embed

**Files:**

- `internal/automation/usecase.go`
- `internal/automation/matcher.go`
- `internal/automation/types.go`
- `internal/automation/interface.go`
- `internal/automation/new.go`

**Types Definition:**

```go
// internal/automation/types.go
package automation

import (
 "autonomous-task-management/internal/webhook"
)

// ProcessWebhookInput is input for webhook processing
type ProcessWebhookInput struct {
 Event webhook.WebhookEvent
}

// ProcessWebhookOutput is result of webhook processing
type ProcessWebhookOutput struct {
 TasksUpdated int      // Number of tasks updated
 TaskIDs      []string // IDs of updated tasks
 Message      string   // Summary message
}

// MatchCriteria defines how to match webhook event to tasks
type MatchCriteria struct {
 Repository string   // Repository name (e.g., "user/repo")
 Tags       []string // Tags to match (e.g., ["#repo/myproject", "#pr/123"])
 Keywords   []string // Keywords in content
}

// TaskMatch represents a matched task
type TaskMatch struct {
 TaskID      string  // Memos task ID
 Content     string  // Task content
 MatchScore  float64 // Match confidence (0-1)
 MatchReason string  // Why it matched
}
```

```go
// internal/automation/interface.go
package automation

import (
 "context"
 "autonomous-task-management/internal/model"
)

type UseCase interface {
 // ProcessWebhook processes a webhook event and updates tasks
 ProcessWebhook(ctx context.Context, sc model.Scope, input ProcessWebhookInput) (ProcessWebhookOutput, error)

 // CompleteTask manually marks a task as complete
 CompleteTask(ctx context.Context, sc model.Scope, taskID string) error

 // ArchiveCompletedTasks archives all fully completed tasks
 ArchiveCompletedTasks(ctx context.Context, sc model.Scope) (int, error)
}
```

```go
// internal/automation/matcher.go
package automation

import (
 "context"
 "fmt"
 "strings"

 "autonomous-task-management/internal/task/repository"
 "autonomous-task-management/internal/webhook"
 pkgLog "autonomous-task-management/pkg/log"
)

// TaskMatcher matches webhook events to tasks
type TaskMatcher struct {
 memosRepo  repository.MemosRepository
 vectorRepo repository.VectorRepository
 l          pkgLog.Logger
}

func NewTaskMatcher(memosRepo repository.MemosRepository, vectorRepo repository.VectorRepository, l pkgLog.Logger) *TaskMatcher {
 return &TaskMatcher{
  memosRepo:  memosRepo,
  vectorRepo: vectorRepo,
  l:          l,
 }
}

// FindMatchingTasks finds tasks that match the webhook event
func (m *TaskMatcher) FindMatchingTasks(ctx context.Context, event webhook.WebhookEvent) ([]TaskMatch, error) {
 criteria := m.buildMatchCriteria(event)

 // Strategy 1: Tag-based search (fast, precise)
 tagMatches, err := m.searchByTags(ctx, criteria)
 if err != nil {
  m.l.Warnf(ctx, "Tag-based search failed: %v", err)
 }

 // Strategy 2: Keyword-based search (flexible, broader)
 keywordMatches, err := m.searchByKeywords(ctx, criteria)
 if err != nil {
  m.l.Warnf(ctx, "Keyword-based search failed: %v", err)
 }

 // Merge and deduplicate results
 matches := m.mergeMatches(tagMatches, keywordMatches)

 m.l.Infof(ctx, "Found %d matching tasks for event %s", len(matches), event.EventType)
 return matches, nil
}

// buildMatchCriteria builds search criteria from webhook event
func (m *TaskMatcher) buildMatchCriteria(event webhook.WebhookEvent) MatchCriteria {
 criteria := MatchCriteria{
  Repository: event.Repository,
  Tags:       []string{},
  Keywords:   []string{},
 }

 // Add repository tag
 if event.Repository != "" {
  // Convert "user/repo" → "#repo/repo"
  parts := strings.Split(event.Repository, "/")
  if len(parts) == 2 {
   criteria.Tags = append(criteria.Tags, "#repo/"+parts[1])
  }
 }

 // Add PR/Issue tag
 if event.PRNumber > 0 {
  criteria.Tags = append(criteria.Tags, fmt.Sprintf("#pr/%d", event.PRNumber))
  criteria.Keywords = append(criteria.Keywords, fmt.Sprintf("PR #%d", event.PRNumber))
  criteria.Keywords = append(criteria.Keywords, fmt.Sprintf("#%d", event.PRNumber))
 }

 if event.IssueNumber > 0 {
  criteria.Tags = append(criteria.Tags, fmt.Sprintf("#issue/%d", event.IssueNumber))
  criteria.Keywords = append(criteria.Keywords, fmt.Sprintf("Issue #%d", event.IssueNumber))
  criteria.Keywords = append(criteria.Keywords, fmt.Sprintf("#%d", event.IssueNumber))
 }

 // Add branch keyword
 if event.Branch != "" {
  criteria.Keywords = append(criteria.Keywords, event.Branch)
 }

 return criteria
}

// searchByTags searches tasks by EXACT tag match using Qdrant filter
// ✅ FIXED: Use payload filter instead of vector search for exact matching
func (m *TaskMatcher) searchByTags(ctx context.Context, criteria MatchCriteria) ([]TaskMatch, error) {
 if len(criteria.Tags) == 0 {
  return nil, nil
 }

 m.l.Infof(ctx, "Searching by exact tags: %v", criteria.Tags)

 // ✅ Use Qdrant payload filter for EXACT tag matching
 // This prevents false positives like #pr/123 matching #pr/124
 results, err := m.vectorRepo.SearchTasksWithFilter(ctx, repository.SearchTasksOptions{
  Filter: repository.PayloadFilter{
   // Must match ANY of the tags (OR condition)
   Should: []repository.Condition{
    {
     Key:   "tags",
     Match: repository.MatchAny{Values: criteria.Tags},
    },
   },
  },
  Limit: 10,
 })
 if err != nil {
  return nil, err
 }

 matches := make([]TaskMatch, 0, len(results))
 for _, result := range results {
  // Fetch full task content
  task, err := m.memosRepo.GetTask(ctx, result.MemoID)
  if err != nil {
   m.l.Warnf(ctx, "Failed to fetch task %s: %v", result.MemoID, err)
   continue
  }

  matches = append(matches, TaskMatch{
   TaskID:      result.MemoID,
   Content:     task.Content,
   MatchScore:  1.0, // Exact match = 100%
   MatchReason: fmt.Sprintf("exact-tag: %v", criteria.Tags),
  })
 }

 return matches, nil
}

// searchByKeywords searches tasks by keywords using SEMANTIC search
// ✅ This is OK to use vector search because we want fuzzy matching for keywords
func (m *TaskMatcher) searchByKeywords(ctx context.Context, criteria MatchCriteria) ([]TaskMatch, error) {
 if len(criteria.Keywords) == 0 {
  return nil, nil
 }

 // Build search query from keywords
 query := strings.Join(criteria.Keywords, " OR ")

 m.l.Infof(ctx, "Searching by semantic keywords: %s", query)

 // ✅ Use vector search for semantic matching (this is appropriate for keywords)
 results, err := m.vectorRepo.SearchTasks(ctx, repository.SearchTasksOptions{
  Query: query,
  Limit: 10,
 })
 if err != nil {
  return nil, err
 }

 matches := make([]TaskMatch, 0, len(results))
 for _, result := range results {
  // Fetch full task content
  task, err := m.memosRepo.GetTask(ctx, result.MemoID)
  if err != nil {
   m.l.Warnf(ctx, "Failed to fetch task %s: %v", result.MemoID, err)
   continue
  }

  // Verify keyword actually exists in content (avoid false positives)
  contentLower := strings.ToLower(task.Content)
  matched := false
  for _, keyword := range criteria.Keywords {
   if strings.Contains(contentLower, strings.ToLower(keyword)) {
    matched = true
    break
   }
  }

  if !matched {
   continue
  }

  matches = append(matches, TaskMatch{
   TaskID:      result.MemoID,
   Content:     task.Content,
   MatchScore:  result.Score,
   MatchReason: "semantic-keyword",
  })
 }

 return matches, nil
}

// mergeMatches merges and deduplicates task matches
func (m *TaskMatcher) mergeMatches(tagMatches, keywordMatches []TaskMatch) []TaskMatch {
 seen := make(map[string]bool)
 merged := make([]TaskMatch, 0)

 // Add tag matches first (higher priority - exact match)
 for _, match := range tagMatches {
  if !seen[match.TaskID] {
   merged = append(merged, match)
   seen[match.TaskID] = true
  }
 }

 // Add keyword matches (lower priority - semantic match)
 for _, match := range keywordMatches {
  if !seen[match.TaskID] {
   merged = append(merged, match)
   seen[match.TaskID] = true
  }
 }

 return merged
}
```

```go
// internal/automation/usecase.go
package automation

import (
 "context"
 "fmt"

 "autonomous-task-management/internal/checklist"
 "autonomous-task-management/internal/model"
 "autonomous-task-management/internal/task/repository"
 pkgLog "autonomous-task-management/pkg/log"
)

type usecase struct {
 memosRepo       repository.MemosRepository
 vectorRepo      repository.VectorRepository
 checklistSvc    checklist.Service
 matcher         *TaskMatcher
 l               pkgLog.Logger
}

func New(
 memosRepo repository.MemosRepository,
 vectorRepo repository.VectorRepository,
 checklistSvc checklist.Service,
 l pkgLog.Logger,
) UseCase {
 matcher := NewTaskMatcher(memosRepo, vectorRepo, l)

 return &usecase{
  memosRepo:    memosRepo,
  vectorRepo:   vectorRepo,
  checklistSvc: checklistSvc,
  matcher:      matcher,
  l:            l,
 }
}

// ProcessWebhook processes a webhook event and updates tasks
func (uc *usecase) ProcessWebhook(ctx context.Context, sc model.Scope, input ProcessWebhookInput) (ProcessWebhookOutput, error) {
 event := input.Event

 uc.l.Infof(ctx, "Processing webhook: %s/%s from %s", event.EventType, event.Action, event.Repository)

 // ✅ CRITICAL: For PR/MR events, only process "merged" action
 // "closed" without merge means PR was REJECTED/CANCELLED - should NOT auto-complete tasks!
 // For "push" events, Action is empty - allow them through
 if (event.EventType == "pull_request" || event.EventType == "merge_request") && event.Action != "merged" {
  uc.l.Infof(ctx, "Skipping PR/MR event with action: %s (only 'merged' triggers auto-completion)", event.Action)
  return ProcessWebhookOutput{
   TasksUpdated: 0,
   Message:      fmt.Sprintf("PR/MR action '%s' not processed", event.Action),
  }, nil
 }

 // For push events (Action is empty), continue processing
 if event.EventType == "push" {
  uc.l.Infof(ctx, "Processing push event for branch: %s", event.Branch)
 }

 // Find matching tasks
 matches, err := uc.matcher.FindMatchingTasks(ctx, event)
 if err != nil {
  return ProcessWebhookOutput{}, fmt.Errorf("failed to find matching tasks: %w", err)
 }

 if len(matches) == 0 {
  uc.l.Infof(ctx, "No matching tasks found for event")
  return ProcessWebhookOutput{
   TasksUpdated: 0,
   Message:      "No matching tasks found",
  }, nil
 }

 // Update each matched task
 updatedIDs := make([]string, 0)
 for _, match := range matches {
  if err := uc.updateTaskChecklist(ctx, match.TaskID, match.Content); err != nil {
   uc.l.Errorf(ctx, "Failed to update task %s: %v", match.TaskID, err)
   continue
  }
  updatedIDs = append(updatedIDs, match.TaskID)
 }

 return ProcessWebhookOutput{
  TasksUpdated: len(updatedIDs),
  TaskIDs:      updatedIDs,
  Message:      fmt.Sprintf("Updated %d task(s)", len(updatedIDs)),
 }, nil
}

// updateTaskChecklist updates all checkboxes in a task to checked
func (uc *usecase) updateTaskChecklist(ctx context.Context, taskID string, content string) error {
 // Check if task has checkboxes
 stats := uc.checklistSvc.GetStats(content)
 if stats.Total == 0 {
  uc.l.Infof(ctx, "Task %s has no checkboxes, skipping", taskID)
  return nil
 }

 // Update all checkboxes to checked
 updatedContent := uc.checklistSvc.UpdateAllCheckboxes(content, true)

 // Update Memos
 if err := uc.memosRepo.UpdateTask(ctx, taskID, updatedContent); err != nil {
  return fmt.Errorf("failed to update Memos: %w", err)
 }

 // ✅ DO NOTHING - Phase 3 webhook will handle re-embedding automatically
 // This prevents double embedding and race conditions
 uc.l.Infof(ctx, "Updated task %s (%d/%d checkboxes), Phase 3 webhook will re-embed",
  taskID, stats.Total, stats.Total)
 return nil
}

// CompleteTask manually marks a task as complete
func (uc *usecase) CompleteTask(ctx context.Context, sc model.Scope, taskID string) error {
 // Fetch task
 task, err := uc.memosRepo.GetTask(ctx, taskID)
 if err != nil {
  return fmt.Errorf("failed to fetch task: %w", err)
 }

 // Update all checkboxes
 updatedContent := uc.checklistSvc.UpdateAllCheckboxes(task.Content, true)

 // 💡 OPTIMIZATION: Skip update if nothing changed
 if updatedContent == task.Content {
  uc.l.Infof(ctx, "Task %s already completed or has no checkboxes, skipping update", taskID)
  return nil
 }

 // Update Memos
 if err := uc.memosRepo.UpdateTask(ctx, taskID, updatedContent); err != nil {
  return fmt.Errorf("failed to update task: %w", err)
 }

 // ✅ Phase 3 webhook handles re-embedding
 uc.l.Infof(ctx, "Manually completed task %s", taskID)
 return nil
}

// ArchiveCompletedTasks archives all fully completed tasks
func (uc *usecase) ArchiveCompletedTasks(ctx context.Context, sc model.Scope) (int, error) {
 // This would require listing all tasks and checking completion
 // For now, return not implemented
 // In production, you'd want to:
 // 1. List all tasks from Memos
 // 2. Check each for full completion
 // 3. Archive completed ones (add #archived tag or move to archive)

 uc.l.Infof(ctx, "Archive completed tasks not yet implemented")
 return 0, nil
}
```

```go
// internal/automation/new.go
package automation

import (
 "autonomous-task-management/internal/checklist"
 "autonomous-task-management/internal/task/repository"
 pkgLog "autonomous-task-management/pkg/log"
)

type usecase struct {
 memosRepo    repository.MemosRepository
 vectorRepo   repository.VectorRepository
 checklistSvc checklist.Service
 matcher      *TaskMatcher
 l            pkgLog.Logger
}

func New(
 memosRepo repository.MemosRepository,
 vectorRepo repository.VectorRepository,
 checklistSvc checklist.Service,
 l pkgLog.Logger,
) UseCase {
 matcher := NewTaskMatcher(memosRepo, vectorRepo, l)

 return &usecase{
  memosRepo:    memosRepo,
  vectorRepo:   vectorRepo,
  checklistSvc: checklistSvc,
  matcher:      matcher,
  l:            l,
 }
}
```

---

### Task 4.6: Webhook HTTP Handler

**Mục tiêu:** Expose HTTP endpoints để nhận webhook từ Git platforms

**Files:**

- `internal/webhook/handler.go`
- `internal/webhook/new.go`

**Implementation:**

```go
// internal/webhook/handler.go
package webhook

import (
 "context"
 "io"
 "net/http"
 "time"

 "github.com/gin-gonic/gin"

 "autonomous-task-management/internal/automation"
 "autonomous-task-management/internal/model"
 pkgLog "autonomous-task-management/pkg/log"
 pkgResponse "autonomous-task-management/pkg/response"
)

type Handler struct {
 automationUC automation.UseCase
 security     *SecurityValidator
 githubParser *GitHubWebhookParser
 gitlabParser *GitLabWebhookParser
 l            pkgLog.Logger
}

func NewHandler(
 automationUC automation.UseCase,
 securityConfig SecurityConfig,
 l pkgLog.Logger,
) *Handler {
 return &Handler{
  automationUC: automationUC,
  security:     NewSecurityValidator(securityConfig),
  githubParser: NewGitHubParser(),
  gitlabParser: NewGitLabParser(),
  l:            l,
 }
}

// HandleGitHubWebhook processes GitHub webhook events
func (h *Handler) HandleGitHubWebhook(c *gin.Context) {
 ctx := c.Request.Context()

 // Read body
 body, err := io.ReadAll(c.Request.Body)
 if err != nil {
  h.l.Errorf(ctx, "Failed to read webhook body: %v", err)
  pkgResponse.Error(c, err, nil)
  return
 }

 // Verify signature
 signature := c.GetHeader("X-Hub-Signature-256")
 if err := h.security.ValidateGitHubSignature(body, signature); err != nil {
  h.l.Errorf(ctx, "GitHub signature verification failed: %v", err)
  c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
  return
 }

 // Check rate limit
 if err := h.security.CheckRateLimit("github"); err != nil {
  h.l.Warnf(ctx, "Rate limit exceeded: %v", err)
  c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
  return
 }

 // Get event type
 eventType := c.GetHeader("X-GitHub-Event")

 // Parse event
 var event *WebhookEvent
 switch eventType {
 case "push":
  event, err = h.githubParser.ParsePushEvent(body)
 case "pull_request":
  event, err = h.githubParser.ParsePullRequestEvent(body)
 case "issues":
  event, err = h.githubParser.ParseIssueEvent(body)
 default:
  h.l.Infof(ctx, "Unsupported GitHub event type: %s", eventType)
  pkgResponse.OK(c, gin.H{"status": "ignored", "reason": "unsupported event type"})
  return
 }

 if err != nil {
  h.l.Errorf(ctx, "Failed to parse GitHub event: %v", err)
  pkgResponse.Error(c, err, nil)
  return
 }

 // Process in background
 go h.processWebhookAsync(*event)

 // Acknowledge immediately
 pkgResponse.OK(c, gin.H{"status": "accepted"})
}

// HandleGitLabWebhook processes GitLab webhook events
func (h *Handler) HandleGitLabWebhook(c *gin.Context) {
 ctx := c.Request.Context()

 // Read body
 body, err := io.ReadAll(c.Request.Body)
 if err != nil {
  h.l.Errorf(ctx, "Failed to read webhook body: %v", err)
  pkgResponse.Error(c, err, nil)
  return
 }

 // Verify token
 token := c.GetHeader("X-Gitlab-Token")
 if err := h.security.ValidateGitLabToken(token); err != nil {
  h.l.Errorf(ctx, "GitLab token verification failed: %v", err)
  c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
  return
 }

 // Check rate limit
 if err := h.security.CheckRateLimit("gitlab"); err != nil {
  h.l.Warnf(ctx, "Rate limit exceeded: %v", err)
  c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
  return
 }

 // Get event type
 eventType := c.GetHeader("X-Gitlab-Event")

 // Parse event
 var event *WebhookEvent
 switch eventType {
 case "Push Hook":
  event, err = h.gitlabParser.ParsePushEvent(body)
 case "Merge Request Hook":
  event, err = h.gitlabParser.ParseMergeRequestEvent(body)
 case "Issue Hook":
  event, err = h.gitlabParser.ParseIssueEvent(body)
 default:
  h.l.Infof(ctx, "Unsupported GitLab event type: %s", eventType)
  pkgResponse.OK(c, gin.H{"status": "ignored", "reason": "unsupported event type"})
  return
 }

 if err != nil {
  h.l.Errorf(ctx, "Failed to parse GitLab event: %v", err)
  pkgResponse.Error(c, err, nil)
  return
 }

 // Process in background
 go h.processWebhookAsync(*event)

 // Acknowledge immediately
 pkgResponse.OK(c, gin.H{"status": "accepted"})
}

// processWebhookAsync processes webhook in background
func (h *Handler) processWebhookAsync(event WebhookEvent) {
 // Create background context with timeout
 ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
 defer cancel()

 h.l.Infof(ctx, "Processing webhook async: %s/%s from %s", event.EventType, event.Action, event.Repository)

 // Process webhook
 sc := model.Scope{UserID: "system_webhook"}
 output, err := h.automationUC.ProcessWebhook(ctx, sc, automation.ProcessWebhookInput{
  Event: event,
 })

 if err != nil {
  h.l.Errorf(ctx, "Webhook processing failed: %v", err)
  return
 }

 h.l.Infof(ctx, "Webhook processed: %s", output.Message)
}
```

```go
// internal/webhook/new.go
package webhook

import (
 "autonomous-task-management/internal/automation"
 pkgLog "autonomous-task-management/pkg/log"
)

type Handler struct {
 automationUC automation.UseCase
 security     *SecurityValidator
 githubParser *GitHubWebhookParser
 gitlabParser *GitLabWebhookParser
 l            pkgLog.Logger
}

func NewHandler(
 automationUC automation.UseCase,
 securityConfig SecurityConfig,
 l pkgLog.Logger,
) *Handler {
 return &Handler{
  automationUC: automationUC,
  security:     NewSecurityValidator(securityConfig),
  githubParser: NewGitHubParser(),
  gitlabParser: NewGitLabParser(),
  l:            l,
 }
}
```

---

### Task 4.7: Agent Tools for Checklist Management

**Mục tiêu:** Extend agent với tools để query và update checklist

**Files:**

- `internal/agent/tools/get_checklist_progress.go`
- `internal/agent/tools/update_checklist_item.go`

**Implementation:**

```go
// internal/agent/tools/get_checklist_progress.go
package tools

import (
 "context"
 "encoding/json"
 "fmt"

 "autonomous-task-management/internal/agent"
 "autonomous-task-management/internal/checklist"
 "autonomous-task-management/internal/task/repository"
 pkgLog "autonomous-task-management/pkg/log"
)

type GetChecklistProgressTool struct {
 memosRepo    repository.MemosRepository
 checklistSvc checklist.Service
 l            pkgLog.Logger
}

func NewGetChecklistProgressTool(memosRepo repository.MemosRepository, checklistSvc checklist.Service, l pkgLog.Logger) *GetChecklistProgressTool {
 return &GetChecklistProgressTool{
  memosRepo:    memosRepo,
  checklistSvc: checklistSvc,
  l:            l,
 }
}

func (t *GetChecklistProgressTool) Name() string {
 return "get_checklist_progress"
}

func (t *GetChecklistProgressTool) Description() string {
 return "Get checklist progress for a specific task. Returns total, completed, and pending checkboxes."
}

func (t *GetChecklistProgressTool) Parameters() map[string]interface{} {
 return map[string]interface{}{
  "type": "object",
  "properties": map[string]interface{}{
   "task_id": map[string]interface{}{
    "type":        "string",
    "description": "Memos task ID (UID)",
   },
  },
  "required": []string{"task_id"},
 }
}

type GetChecklistProgressInput struct {
 TaskID string `json:"task_id"`
}

type GetChecklistProgressOutput struct {
 TaskID    string                  `json:"task_id"`
 Stats     checklist.ChecklistStats `json:"stats"`
 Checkboxes []checklist.Checkbox    `json:"checkboxes"`
 Summary   string                  `json:"summary"`
}

func (t *GetChecklistProgressTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
 // Parse input
 inputBytes, err := json.Marshal(input)
 if err != nil {
  return nil, fmt.Errorf("failed to marshal input: %w", err)
 }

 var params GetChecklistProgressInput
 if err := json.Unmarshal(inputBytes, &params); err != nil {
  return nil, fmt.Errorf("failed to parse input: %w", err)
 }

 t.l.Infof(ctx, "get_checklist_progress: task_id=%s", params.TaskID)

 // Fetch task
 task, err := t.memosRepo.GetTask(ctx, params.TaskID)
 if err != nil {
  return nil, fmt.Errorf("failed to fetch task: %w", err)
 }

 // Get stats and checkboxes
 stats := t.checklistSvc.GetStats(task.Content)
 checkboxes := t.checklistSvc.ParseCheckboxes(task.Content)

 // Generate summary
 summary := fmt.Sprintf("📊 Tiến độ: %d/%d hoàn thành (%.0f%%)", stats.Completed, stats.Total, stats.Progress)
 if stats.Total == 0 {
  summary = "Task này không có checklist"
 } else if stats.Progress == 100 {
  summary += " ✅ Hoàn thành!"
 }

 return GetChecklistProgressOutput{
  TaskID:     params.TaskID,
  Stats:      stats,
  Checkboxes: checkboxes,
  Summary:    summary,
 }, nil
}

var _ agent.Tool = (*GetChecklistProgressTool)(nil)
```

```go
// internal/agent/tools/update_checklist_item.go
package tools

import (
 "context"
 "encoding/json"
 "fmt"

 "autonomous-task-management/internal/agent"
 "autonomous-task-management/internal/checklist"
 "autonomous-task-management/internal/task/repository"
 pkgLog "autonomous-task-management/pkg/log"
)

type UpdateChecklistItemTool struct {
 memosRepo    repository.MemosRepository
 vectorRepo   repository.VectorRepository
 checklistSvc checklist.Service
 l            pkgLog.Logger
}

func NewUpdateChecklistItemTool(
 memosRepo repository.MemosRepository,
 vectorRepo repository.VectorRepository,
 checklistSvc checklist.Service,
 l pkgLog.Logger,
) *UpdateChecklistItemTool {
 return &UpdateChecklistItemTool{
  memosRepo:    memosRepo,
  vectorRepo:   vectorRepo,
  checklistSvc: checklistSvc,
  l:            l,
 }
}

func (t *UpdateChecklistItemTool) Name() string {
 return "update_checklist_item"
}

func (t *UpdateChecklistItemTool) Description() string {
 return "Update a checklist item in a task. Can mark items as checked or unchecked by matching text."
}

func (t *UpdateChecklistItemTool) Parameters() map[string]interface{} {
 return map[string]interface{}{
  "type": "object",
  "properties": map[string]interface{}{
   "task_id": map[string]interface{}{
    "type":        "string",
    "description": "Memos task ID (UID)",
   },
   "item_text": map[string]interface{}{
    "type":        "string",
    "description": "Text of the checklist item to update (partial match OK)",
   },
   "checked": map[string]interface{}{
    "type":        "boolean",
    "description": "New checked state (true = checked, false = unchecked)",
   },
  },
  "required": []string{"task_id", "item_text", "checked"},
 }
}

type UpdateChecklistItemInput struct {
 TaskID   string `json:"task_id"`
 ItemText string `json:"item_text"`
 Checked  bool   `json:"checked"`
}

type UpdateChecklistItemOutput struct {
 TaskID  string `json:"task_id"`
 Updated bool   `json:"updated"`
 Count   int    `json:"count"`
 Summary string `json:"summary"`
}

func (t *UpdateChecklistItemTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
 // Parse input
 inputBytes, err := json.Marshal(input)
 if err != nil {
  return nil, fmt.Errorf("failed to marshal input: %w", err)
 }

 var params UpdateChecklistItemInput
 if err := json.Unmarshal(inputBytes, &params); err != nil {
  return nil, fmt.Errorf("failed to parse input: %w", err)
 }

 t.l.Infof(ctx, "update_checklist_item: task_id=%s item=%q checked=%v", params.TaskID, params.ItemText, params.Checked)

 // Fetch task
 task, err := t.memosRepo.GetTask(ctx, params.TaskID)
 if err != nil {
  return nil, fmt.Errorf("failed to fetch task: %w", err)
 }

 // Update checkbox
 output, err := t.checklistSvc.UpdateCheckbox(ctx, checklist.UpdateCheckboxInput{
  Content:      task.Content,
  CheckboxText: params.ItemText,
  Checked:      params.Checked,
 })
 if err != nil {
  return nil, fmt.Errorf("failed to update checkbox: %w", err)
 }

 if !output.Updated {
  return UpdateChecklistItemOutput{
   TaskID:  params.TaskID,
   Updated: false,
   Count:   0,
   Summary: fmt.Sprintf("Không tìm thấy checkbox với text: %q", params.ItemText),
  }, nil
 }

 // Update Memos
 if err := t.memosRepo.UpdateTask(ctx, params.TaskID, output.Content); err != nil {
  return nil, fmt.Errorf("failed to update Memos: %w", err)
 }

 // ✅ Phase 3 webhook handles re-embedding
 t.l.Infof(ctx, "Updated checklist for task %s", params.TaskID)

 // Generate summary
 action := "unchecked"
 if params.Checked {
  action = "checked"
 }
 summary := fmt.Sprintf("✅ Đã %s %d checkbox(es) matching %q", action, output.Count, params.ItemText)

 return UpdateChecklistItemOutput{
  TaskID:  params.TaskID,
  Updated: true,
  Count:   output.Count,
  Summary: summary,
 }, nil
}

var _ agent.Tool = (*UpdateChecklistItemTool)(nil)
```

---

### Task 4.8: Telegram Commands for Manual Control

**Mục tiêu:** Add Telegram commands để user có thể manually control checklist

**Files:**

- `internal/task/delivery/telegram/handler.go` (update)

**New Commands:**

- `/progress <task_id>` - Show checklist progress
- `/complete <task_id>` - Mark all checkboxes as complete
- `/check <task_id> <item_text>` - Check specific item
- `/uncheck <task_id> <item_text>` - Uncheck specific item

**Implementation:**

```go
// Add to internal/task/delivery/telegram/handler.go

func (h *handler) processMessage(ctx context.Context, msg *pkgTelegram.Message) error {
 sc := model.Scope{UserID: fmt.Sprintf("telegram_%d", msg.From.ID)}

 // Handle commands
 switch {
 case msg.Text == "/start":
  return h.handleStart(ctx, msg.Chat.ID)

 case msg.Text == "/help":
  return h.handleHelp(ctx, msg.Chat.ID)

 case strings.HasPrefix(msg.Text, "/search "):
  query := strings.TrimSpace(strings.TrimPrefix(msg.Text, "/search"))
  return h.handleSearch(ctx, sc, query, msg.Chat.ID)

 case strings.HasPrefix(msg.Text, "/ask "):
  query := strings.TrimSpace(strings.TrimPrefix(msg.Text, "/ask"))
  return h.handleAgentOrchestrator(ctx, sc, query, msg.Chat.ID)

 case strings.HasPrefix(msg.Text, "/progress "):
  taskID := strings.TrimSpace(strings.TrimPrefix(msg.Text, "/progress"))
  return h.handleProgress(ctx, sc, taskID, msg.Chat.ID)

 case strings.HasPrefix(msg.Text, "/complete "):
  taskID := strings.TrimSpace(strings.TrimPrefix(msg.Text, "/complete"))
  return h.handleComplete(ctx, sc, taskID, msg.Chat.ID)

 case strings.HasPrefix(msg.Text, "/check "):
  return h.handleCheckItem(ctx, sc, msg.Text, msg.Chat.ID, true)

 case strings.HasPrefix(msg.Text, "/uncheck "):
  return h.handleCheckItem(ctx, sc, msg.Text, msg.Chat.ID, false)

 default:
  // Default: Create task
  return h.handleCreateTask(ctx, sc, msg)
 }
}

// handleProgress shows checklist progress
func (h *handler) handleProgress(ctx context.Context, sc model.Scope, taskID string, chatID int64) error {
 if taskID == "" {
  return h.bot.SendMessage(chatID, "❌ Vui lòng nhập task ID.\n\nVí dụ: `/progress abc123`")
 }

 h.bot.SendMessage(chatID, "📊 Đang kiểm tra tiến độ...")

 // Use agent tool to get progress
 result, err := h.checklistSvc.GetProgress(ctx, taskID)
 if err != nil {
  h.l.Errorf(ctx, "Failed to get progress: %v", err)
  return h.bot.SendMessage(chatID, "❌ Không thể lấy tiến độ. Vui lòng kiểm tra task ID.")
 }

 // Format response
 var response strings.Builder
 response.WriteString(fmt.Sprintf("📊 **Tiến độ Task: %s**\n\n", taskID))
 response.WriteString(fmt.Sprintf("✅ Hoàn thành: %d/%d (%.0f%%)\n", result.Completed, result.Total, result.Progress))
 response.WriteString(fmt.Sprintf("⏳ Còn lại: %d\n\n", result.Pending))

 if len(result.Items) > 0 {
  response.WriteString("**Chi tiết:**\n")
  for i, item := range result.Items {
   checkbox := "☐"
   if item.Checked {
    checkbox = "☑"
   }
   response.WriteString(fmt.Sprintf("%d. %s %s\n", i+1, checkbox, item.Text))
  }
 }

 return h.bot.SendMessageWithMode(chatID, response.String(), "Markdown")
}

// handleComplete marks all checkboxes as complete
func (h *handler) handleComplete(ctx context.Context, sc model.Scope, taskID string, chatID int64) error {
 if taskID == "" {
  return h.bot.SendMessage(chatID, "❌ Vui lòng nhập task ID.\n\nVí dụ: `/complete abc123`")
 }

 h.bot.SendMessage(chatID, "✅ Đang đánh dấu hoàn thành...")

 // Use automation usecase to complete task
 if err := h.automationUC.CompleteTask(ctx, sc, taskID); err != nil {
  h.l.Errorf(ctx, "Failed to complete task: %v", err)
  return h.bot.SendMessage(chatID, "❌ Không thể hoàn thành task. Vui lòng thử lại.")
 }

 return h.bot.SendMessage(chatID, fmt.Sprintf("✅ Đã đánh dấu hoàn thành task: %s", taskID))
}

// handleCheckItem checks/unchecks specific checklist item
func (h *handler) handleCheckItem(ctx context.Context, sc model.Scope, text string, chatID int64, checked bool) error {
 // Parse command: /check <task_id> <item_text>
 parts := strings.SplitN(text, " ", 3)
 if len(parts) < 3 {
  action := "check"
  if !checked {
   action = "uncheck"
  }
  return h.bot.SendMessage(chatID, fmt.Sprintf("❌ Vui lòng nhập đầy đủ.\n\nVí dụ: `/%s abc123 Write tests`", action))
 }

 taskID := strings.TrimSpace(parts[1])
 itemText := strings.TrimSpace(parts[2])

 action := "checking"
 if !checked {
  action = "unchecking"
 }
 h.bot.SendMessage(chatID, fmt.Sprintf("⏳ Đang %s...", action))

 // Update checklist item
 output, err := h.checklistSvc.UpdateItem(ctx, taskID, itemText, checked)
 if err != nil {
  h.l.Errorf(ctx, "Failed to update item: %v", err)
  return h.bot.SendMessage(chatID, "❌ Không thể cập nhật. Vui lòng thử lại.")
 }

 if !output.Updated {
  return h.bot.SendMessage(chatID, fmt.Sprintf("❌ Không tìm thấy checkbox với text: %q", itemText))
 }

 emoji := "☑"
 if !checked {
  emoji = "☐"
 }

 // ⚠️ PRO-TIP: Partial match warning
 // If multiple checkboxes matched, inform user to be more specific
 warningMsg := ""
 if output.Count > 1 {
  warningMsg = fmt.Sprintf("\n\n⚠️ Lưu ý: %d checkboxes được cập nhật. Nếu không đúng ý, hãy gõ text cụ thể hơn.", output.Count)
 }

 return h.bot.SendMessage(chatID, fmt.Sprintf("%s Đã cập nhật %d checkbox(es) matching %q%s", emoji, output.Count, itemText, warningMsg))
}

// Update /help message
func (h *handler) handleHelp(ctx context.Context, chatID int64) error {
 message := `📖 **Hướng dẫn sử dụng**

**🆕 Tạo Task**
Gửi tin nhắn bình thường:
• "Họp team lúc 10am ngày mai"
• "Deadline dự án ABC vào 15/3"

**🔍 Tìm kiếm nhanh**
/search [từ khóa]
• /search meeting - Tìm tất cả meeting
• /search deadline march - Tìm deadline tháng 3

**🧠 Trợ lý thông minh**
/ask [câu hỏi]
• /ask Tôi có meeting nào tuần này?
• /ask Deadline nào gần nhất?

**📊 Quản lý Checklist**
/progress [task_id] - Xem tiến độ
/complete [task_id] - Đánh dấu hoàn thành
/check [task_id] [text] - Check item
/uncheck [task_id] [text] - Uncheck item

**💡 Mẹo:**
• Agent mode (/ask) thông minh hơn nhưng chậm hơn
• Search mode (/search) nhanh hơn
• Webhook tự động tick checklist khi PR merged`

 return h.bot.SendMessageWithMode(chatID, message, "Markdown")
}
```

---

### Task 4.9: Wiring in main.go

**Mục tiêu:** Wire all Phase 4 components in main.go

**Files:**

- `cmd/api/main.go` (update)

**Implementation:**

```go
// Add to cmd/api/main.go

import (
 // ... existing imports ...
 "autonomous-task-management/internal/automation"
 "autonomous-task-management/internal/checklist"
 "autonomous-task-management/internal/webhook"
)

func main() {
 // ... existing setup ...

 // Phase 4: Automation & Checklist Management
 var gitWebhookHandler *webhook.Handler

 if cfg.Telegram.BotToken != "" && cfg.Gemini.APIKey != "" && cfg.Memos.AccessToken != "" {
  logger.Info(ctx, "Initializing Phase 4 components...")

  // Checklist Service
  checklistSvc := checklist.New()

  // Automation UseCase
  automationUC := automation.New(taskRepo, vectorRepoInterface, checklistSvc, logger)

  // Webhook Handler
  if cfg.Webhook.Enabled {
   webhookSecurityConfig := webhook.SecurityConfig{
    Secret:          cfg.Webhook.Secret,
    AllowedIPs:      cfg.Webhook.AllowedIPs,
    RateLimitPerMin: cfg.Webhook.RateLimitPerMin,
   }
   gitWebhookHandler = webhook.NewHandler(automationUC, webhookSecurityConfig, logger)
   logger.Info(ctx, "Git webhook handler initialized")
  }

  // Update Telegram handler with automation UC
  telegramHandler = tgDelivery.New(logger, taskUC, telegramBot, agentOrchestrator, automationUC, checklistSvc)

  // Register checklist tools
  toolRegistry.Register(tools.NewGetChecklistProgressTool(taskRepo, checklistSvc, logger))
  if vectorRepoInterface != nil {
   toolRegistry.Register(tools.NewUpdateChecklistItemTool(taskRepo, vectorRepoInterface, checklistSvc, logger))
  }

  logger.Info(ctx, "Phase 4 initialized successfully")
 }

 // HTTP Server with webhook routes
 httpServer, err := httpserver.New(logger, httpserver.Config{
  Logger:             logger,
  Port:               cfg.HTTPServer.Port,
  Mode:               cfg.HTTPServer.Mode,
  Environment:        cfg.Environment.Name,
  TelegramHandler:    telegramHandler,
  WebhookHandler:     webhookHandler,
  GitWebhookHandler:  gitWebhookHandler, // NEW
 })
 if err != nil {
  logger.Error(ctx, "Failed to initialize HTTP server: ", err)
  return
 }

 // ... rest of main.go ...
}
```

**Update httpserver to register webhook routes:**

```go
// internal/httpserver/httpserver.go

type Config struct {
 Logger            pkgLog.Logger
 Port              string
 Mode              string
 Environment       string
 TelegramHandler   interface{}
 WebhookHandler    interface{} // Memos webhook
 GitWebhookHandler interface{} // Git webhook (NEW)
}

func (s *httpServer) setupRoutes() {
 // ... existing routes ...

 // API routes
 apiGroup := s.router.Group("/api/v1")
 {
  // Health check
  apiGroup.GET("/health", s.healthHandler.Check)

  // Telegram webhook
  if s.telegramHandler != nil {
   apiGroup.POST("/webhook/telegram", s.telegramHandler.HandleWebhook)
  }

  // Memos webhook (Phase 3)
  if s.webhookHandler != nil {
   apiGroup.POST("/webhook/memos", s.webhookHandler.HandleMemosWebhook)
  }

  // Git webhooks (Phase 4)
  if s.gitWebhookHandler != nil {
   apiGroup.POST("/webhook/github", s.gitWebhookHandler.HandleGitHubWebhook)
   apiGroup.POST("/webhook/gitlab", s.gitWebhookHandler.HandleGitLabWebhook)
  }
 }
}
```

---

### Task 4.10: Configuration Updates

**Mục tiêu:** Update configuration files for Phase 4

**Files:**

- `config/config.yaml`
- `.env.example`
- `config/config.go`

**config.yaml updates:**

```yaml
# Add webhook configuration
webhook:
  enabled: true
  secret: "${WEBHOOK_SECRET}"
  allowed_ips: [] # Empty = allow all
  rate_limit_per_min: 100

# Add automation configuration
automation:
  auto_complete_on_merge: true
  auto_archive_completed: false
  match_strategy: "hybrid" # tag, keyword, or hybrid
```

**.env.example updates:**

```bash
# Webhook Configuration
WEBHOOK_SECRET=your-webhook-secret-here
WEBHOOK_ENABLED=true
WEBHOOK_RATE_LIMIT=100

# Automation Configuration
AUTO_COMPLETE_ON_MERGE=true
AUTO_ARCHIVE_COMPLETED=false
```

**config.go updates:**

```go
// Add to config/config.go

type Config struct {
 // ... existing fields ...

 Webhook    WebhookConfig    `yaml:"webhook"`
 Automation AutomationConfig `yaml:"automation"`
}

type WebhookConfig struct {
 Enabled         bool     `yaml:"enabled"`
 Secret          string   `yaml:"secret"`
 AllowedIPs      []string `yaml:"allowed_ips"`
 RateLimitPerMin int      `yaml:"rate_limit_per_min"`
}

type AutomationConfig struct {
 AutoCompleteOnMerge  bool   `yaml:"auto_complete_on_merge"`
 AutoArchiveCompleted bool   `yaml:"auto_archive_completed"`
 MatchStrategy        string `yaml:"match_strategy"`
}
```

---

## Testing Strategy

### Unit Tests

**Checklist Service Tests:**

```go
// internal/checklist/service_test.go

func TestParseCheckboxes(t *testing.T) {
 // Test parsing various checkbox formats
 // Test nested checkboxes
 // Test mixed content (text + checkboxes)
}

func TestUpdateCheckbox(t *testing.T) {
 // Test partial text matching
 // Test case-insensitive matching
 // Test multiple matches
 // Test no matches
}

func TestGetStats(t *testing.T) {
 // Test empty content
 // Test all checked
 // Test all unchecked
 // Test mixed state
}

func TestIsFullyCompleted(t *testing.T) {
 // Test 100% completion
 // Test partial completion
 // Test no checkboxes
}
```

**Webhook Security Tests:**

```go
// internal/webhook/security_test.go

func TestValidateGitHubSignature(t *testing.T) {
 // Test valid signature
 // Test invalid signature
 // Test missing secret
 // Test timing attack resistance
}

func TestRateLimiter(t *testing.T) {
 // Test rate limit enforcement
 // Test token refill
 // Test multiple sources
}
```

**Webhook Parser Tests:**

```go
// internal/webhook/github_test.go

func TestParsePushEvent(t *testing.T) {
 // Test valid push event
 // Test branch extraction
 // Test commit info extraction
}

func TestParsePullRequestEvent(t *testing.T) {
 // Test opened PR
 // Test merged PR
 // Test closed PR
}
```

**Automation UseCase Tests:**

```go
// internal/automation/usecase_test.go

func TestProcessWebhook(t *testing.T) {
 // Test successful match and update
 // Test no matches
 // Test multiple matches
 // Test error handling
}

func TestTaskMatcher(t *testing.T) {
 // Test tag-based matching
 // Test keyword-based matching
 // Test hybrid matching
 // Test deduplication
}
```

### Integration Tests

**End-to-End Webhook Flow:**

```go
// test/integration/webhook_test.go

func TestGitHubWebhookFlow(t *testing.T) {
 // 1. Send GitHub webhook
 // 2. Verify signature validation
 // 3. Verify task matching
 // 4. Verify checklist update
 // 5. Verify Qdrant re-embedding
}

func TestGitLabWebhookFlow(t *testing.T) {
 // Similar to GitHub test
}
```

**Telegram Checklist Commands:**

```go
// test/integration/telegram_checklist_test.go

func TestProgressCommand(t *testing.T) {
 // Send /progress command
 // Verify response format
}

func TestCompleteCommand(t *testing.T) {
 // Send /complete command
 // Verify all checkboxes checked
 // Verify Memos updated
}
```

### Manual Testing Checklist

**Webhook Setup:**

- [ ] Configure GitHub webhook with secret
- [ ] Configure GitLab webhook with token
- [ ] Test signature verification
- [ ] Test rate limiting
- [ ] Test IP whitelist (if configured)

**Checklist Operations:**

- [ ] Create task with checkboxes via Telegram
- [ ] Test `/progress` command
- [ ] Test `/complete` command
- [ ] Test `/check` command with partial match
- [ ] Test `/uncheck` command

**Automation Flow:**

- [ ] Create task with `#repo/myproject` tag
- [ ] Create PR in GitHub/GitLab
- [ ] Merge PR
- [ ] Verify webhook received
- [ ] Verify task checkboxes auto-checked
- [ ] Verify Qdrant re-embedded

**Agent Tools:**

- [ ] Test `/ask` with checklist progress query
- [ ] Test agent updating checklist via tool
- [ ] Verify tool execution logging

---

## Deployment Guide

### GitHub Webhook Setup

1. **Go to Repository Settings → Webhooks → Add webhook**

2. **Configure webhook:**
   - Payload URL: `https://your-domain.com/api/v1/webhook/github`
   - Content type: `application/json`
   - Secret: (copy from `.env` WEBHOOK_SECRET)
   - Events: Select `Push`, `Pull requests`, `Issues`

3. **Test webhook:**

   ```bash
   # GitHub will send a ping event
   # Check logs: docker compose logs -f backend
   ```

### GitLab Webhook Setup

1. **Go to Project Settings → Webhooks → Add webhook**

2. **Configure webhook:**
   - URL: `https://your-domain.com/api/v1/webhook/gitlab`
   - Secret token: (copy from `.env` WEBHOOK_SECRET)
   - Trigger: Select `Push events`, `Merge request events`, `Issue events`
   - SSL verification: Enable

3. **Test webhook:**

   ```bash
   # GitLab has a "Test" button
   # Check logs: docker compose logs -f backend
   ```

### Task Tagging Convention

**For webhook automation to work, tasks must be tagged properly:**

```markdown
## Task Example

#repo/myproject #pr/123

- [ ] Implement feature X
- [ ] Write unit tests
- [ ] Update documentation

PR: https://github.com/user/myproject/pull/123
```

**Tag format:**

- Repository: `#repo/<repo-name>`
- Pull Request: `#pr/<number>`
- Issue: `#issue/<number>`
- Branch: `#branch/<branch-name>` (optional)

---

## Performance Considerations

### Webhook Processing

**Target:** <100ms acknowledgment, <2s background processing

**Optimization:**

- Immediate HTTP 200 response
- Background goroutine for processing
- Context timeout (2 minutes)
- Rate limiting (100 req/min)

### Checklist Parsing

**Target:** <10ms for typical task (50 lines)

**Optimization:**

- Compiled regex patterns
- Single-pass parsing
- Minimal allocations

### Task Matching

**Target:** <500ms for search + match

**Optimization:**

- Vector search (fast)
- Tag-based matching (precise)
- Keyword fallback (flexible)
- Result deduplication

### Memory Usage

**Target:** <50MB for Phase 4 components

**Monitoring:**

- Webhook handler goroutines
- Regex pattern cache
- Rate limiter buckets

---

## Troubleshooting Guide

### Webhook Issues

**Problem:** Webhook signature verification fails

```
GitHub signature verification failed
```

**Solution:**

- Verify WEBHOOK_SECRET matches GitHub/GitLab config
- Check signature header name (X-Hub-Signature-256 vs X-Gitlab-Token)
- Ensure payload is read correctly (not double-read)

**Problem:** Webhook rate limit exceeded

```
Rate limit exceeded for github
```

**Solution:**

- Increase WEBHOOK_RATE_LIMIT in config
- Check for webhook spam/loops
- Implement exponential backoff on sender side

**Problem:** No tasks matched

```
No matching tasks found for event
```

**Solution:**

- Verify task has correct tags (#repo/myproject, #pr/123)
- Check task is embedded in Qdrant
- Try keyword-based matching
- Check logs for match criteria

### Checklist Issues

**Problem:** Checkbox not updating

```
Không tìm thấy checkbox với text: "..."
```

**Solution:**

- Use partial text match (not full line)
- Check for typos in item text
- Verify task has checkboxes
- Try `/progress` to see all items

**Problem:** Progress shows 0/0

```
Task này không có checklist
```

**Solution:**

- Verify task content has `- [ ]` format
- Check for correct spacing (space after `]`)
- Ensure checkboxes are not in code blocks

### Automation Issues

**Problem:** PR merged but task not updated

```
Webhook processed: No matching tasks found
```

**Solution:**

- Add `#pr/<number>` tag to task
- Add PR link in task content
- Check webhook logs for event details
- Verify repository name matches

**Problem:** All checkboxes checked but task not archived

```
Task still visible after completion
```

**Solution:**

- Auto-archive is disabled by default
- Enable in config: `AUTO_ARCHIVE_COMPLETED=true`
- Or manually archive via Memos UI

---

## Implementation Checklist

### Task 4.1: Checklist Service

- [ ] Create `internal/checklist/service.go`
- [ ] Create `internal/checklist/parser.go`
- [ ] Create `internal/checklist/types.go`
- [ ] Implement ParseCheckboxes
- [ ] Implement GetStats
- [ ] Implement UpdateCheckbox
- [ ] Implement UpdateAllCheckboxes
- [ ] Implement IsFullyCompleted
- [ ] Write unit tests

### Task 4.2: Webhook Security

- [ ] Create `internal/webhook/security.go`
- [ ] Create `internal/webhook/types.go`
- [ ] Implement GitHub signature verification
- [ ] Implement GitLab token verification
- [ ] Implement IP whitelist
- [ ] Implement rate limiter
- [ ] Write security tests

### Task 4.3: GitHub Webhook Handler

- [ ] Create `internal/webhook/github.go`
- [ ] Implement ParsePushEvent
- [ ] Implement ParsePullRequestEvent
- [ ] Implement ParseIssueEvent
- [ ] Write parser tests

### Task 4.4: GitLab Webhook Handler

- [ ] Create `internal/webhook/gitlab.go`
- [ ] Implement ParsePushEvent
- [ ] Implement ParseMergeRequestEvent
- [ ] Implement ParseIssueEvent
- [ ] Write parser tests

### Task 4.5: Automation UseCase

- [ ] Create `internal/automation/usecase.go`
- [ ] Create `internal/automation/matcher.go`
- [ ] Create `internal/automation/types.go`
- [ ] Create `internal/automation/interface.go`
- [ ] Implement ProcessWebhook
- [ ] Implement CompleteTask
- [ ] Implement TaskMatcher
- [ ] Write usecase tests

### Task 4.6: Webhook HTTP Handler

- [ ] Create `internal/webhook/handler.go`
- [ ] Implement HandleGitHubWebhook
- [ ] Implement HandleGitLabWebhook
- [ ] Implement background processing
- [ ] Write handler tests

### Task 4.7: Agent Tools

- [ ] Create `internal/agent/tools/get_checklist_progress.go`
- [ ] Create `internal/agent/tools/update_checklist_item.go`
- [ ] Register tools in registry
- [ ] Write tool tests

### Task 4.8: Telegram Commands

- [ ] Update `internal/task/delivery/telegram/handler.go`
- [ ] Implement `/progress` command
- [ ] Implement `/complete` command
- [ ] Implement `/check` command
- [ ] Implement `/uncheck` command
- [ ] Update `/help` message
- [ ] Write command tests

### Task 4.9: Wiring

- [ ] Update `cmd/api/main.go`
- [ ] Initialize checklist service
- [ ] Initialize automation usecase
- [ ] Initialize webhook handler
- [ ] Register webhook routes
- [ ] Register agent tools
- [ ] Test complete startup

### Task 4.10: Configuration

- [ ] Update `config/config.yaml`
- [ ] Update `.env.example`
- [ ] Update `config/config.go`
- [ ] Test configuration loading
- [ ] Document environment variables

### Testing & Documentation

- [ ] Write unit tests for all components
- [ ] Write integration tests
- [ ] Manual testing with real webhooks
- [ ] Update README with webhook setup
- [ ] Create troubleshooting guide
- [ ] Performance testing

---

## Success Metrics

### Functionality

- [ ] Checklist parsing works for all formats
- [ ] Webhook signature verification passes
- [ ] GitHub webhooks processed correctly
- [ ] GitLab webhooks processed correctly
- [ ] Task matching achieves >90% accuracy
- [ ] Checklist updates sync to Qdrant
- [ ] Agent tools work correctly
- [ ] Telegram commands respond properly

### Performance

- [ ] Webhook acknowledgment <100ms
- [ ] Background processing <2s
- [ ] Checklist parsing <10ms
- [ ] Task matching <500ms
- [ ] Memory usage <50MB for Phase 4

### Reliability

- [ ] Rate limiting prevents abuse
- [ ] Signature verification prevents spoofing
- [ ] Graceful degradation on failures
- [ ] No goroutine leaks
- [ ] Comprehensive error logging

### User Experience

- [ ] Clear error messages
- [ ] Helpful command examples
- [ ] Progress visualization
- [ ] Webhook setup documentation
- [ ] Troubleshooting guide

---

## Deliverables Summary

### Core Components

1. **Checklist Service** - Parse and manipulate markdown checkboxes
2. **Webhook Security** - Signature verification, rate limiting, IP whitelist
3. **GitHub/GitLab Parsers** - Parse webhook payloads
4. **Automation UseCase** - Match events to tasks, update checklists
5. **Webhook HTTP Handler** - Receive and process webhooks
6. **Agent Tools** - Checklist progress and update tools
7. **Telegram Commands** - Manual checklist control

### Key Features

- **Automatic Checklist Completion** - PR merged → checkboxes checked
- **Manual Control** - Telegram commands for checklist management
- **Agent Integration** - Agent can query and update checklists
- **Security** - Signature verification, rate limiting
- **Flexible Matching** - Tag-based + keyword-based matching

### Documentation

- Webhook setup guide (GitHub/GitLab)
- Task tagging convention
- Telegram command reference
- Troubleshooting guide
- Performance optimization tips

**Estimated Implementation Time:** 5-7 days (40-56 hours)

**Dependencies:** Phase 3 Advanced must be 100% complete

**Risk Level:** Medium (webhook security critical, matching logic complex)

---

## 🎯 Next Steps

1. **Verify Phase 3 Advanced** - Ensure all components working
2. **Start with Checklist Service** - Core parsing logic
3. **Add Webhook Security** - Critical for production
4. **Implement Parsers** - GitHub and GitLab
5. **Build Automation Logic** - Task matching and updates
6. **Add HTTP Handlers** - Webhook endpoints
7. **Extend Agent Tools** - Checklist management
8. **Add Telegram Commands** - User control
9. **Integration Testing** - End-to-end validation
10. **Deploy and Configure** - Setup webhooks on Git platforms

**Ready to begin Phase 4 implementation!** 🚀
