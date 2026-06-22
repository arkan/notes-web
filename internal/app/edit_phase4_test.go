package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Config: todo.todo_file
// ---------------------------------------------------------------------------

func TestTodoConfigDefaults(t *testing.T) {
	v := makeVault(t)
	cfg := v.LoadConfig()
	if cfg.Todo.TodoFile != "" {
		t.Fatalf("todo.todo_file default = %q, want empty", cfg.Todo.TodoFile)
	}
}

func TestTodoConfigYAMLParse(t *testing.T) {
	v := makeVault(t)
	yaml := `todo:
  todo_file: "Tasks/Inbox.md"
`
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := v.LoadConfig()
	if cfg.Todo.TodoFile != "Tasks/Inbox.md" {
		t.Fatalf("todo.todo_file = %q, want %q", cfg.Todo.TodoFile, "Tasks/Inbox.md")
	}
}

// ---------------------------------------------------------------------------
// Helpers for Inbox tests
// ---------------------------------------------------------------------------

// writeInboxCapture creates a minimal inbox capture file at the given path.
func writeInboxCapture(t *testing.T, v *Vault, rel, title string) {
	t.Helper()
	p := filepath.Join(v.Root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\ntitle: " + title + "\ntype: capture\nstatus: inbox\ncaptured_at: " + time.Now().Format(time.RFC3339) + "\n---\n# " + title + "\n\nBody text.\n"
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func enableEditingWithConfig(t *testing.T, v *Vault, extraConfig string) {
	t.Helper()
	yaml := "editing:\n  enabled: true\n" + extraConfig
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
}

// ---------------------------------------------------------------------------
// Capture filename generation
// ---------------------------------------------------------------------------

func TestCaptureFilenameGeneratesSlug(t *testing.T) {
	filename, err := captureFilename("Call tax office")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(filename, "Inbox/") {
		t.Fatalf("filename = %q, want Inbox/ prefix", filename)
	}
	if !strings.HasSuffix(filename, ".md") {
		t.Fatalf("filename = %q, want .md suffix", filename)
	}
	// Should contain the slugified title
	if !strings.Contains(filename, "call-tax-office") {
		t.Fatalf("filename = %q, should contain 'call-tax-office'", filename)
	}
	// Should contain timestamp segments
	parts := strings.Split(filename, "/")
	if len(parts) != 2 {
		t.Fatalf("filename = %q, expected 2 path segments", filename)
	}
	if !strings.Contains(parts[1], "-") {
		t.Fatalf("filename basename = %q, should contain timestamp", parts[1])
	}
}

func TestCaptureFilenameEmptyTitle(t *testing.T) {
	_, err := captureFilename("   ")
	if err == nil {
		t.Fatal("expected error for empty title")
	}
}

// ---------------------------------------------------------------------------
// Collision suffix
// ---------------------------------------------------------------------------

func TestResolveCollisionSuffix(t *testing.T) {
	v := makeVault(t)
	basePath := filepath.Join(v.Root, "Inbox", "2026-06-21-143022-test.md")
	if err := os.MkdirAll(filepath.Dir(basePath), 0o755); err != nil {
		t.Fatal(err)
	}
	name, err := resolveCollisionSuffix(basePath)
	if err != nil {
		t.Fatal(err)
	}
	if name != "2026-06-21-143022-test.md" {
		t.Fatalf("no collision: expected basename %q, got %q", "2026-06-21-143022-test.md", name)
	}

	// Create the file and check suffix generation.
	if err := os.WriteFile(basePath, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	name2, err := resolveCollisionSuffix(basePath)
	if err != nil {
		t.Fatal(err)
	}
	if name2 != "2026-06-21-143022-test-2.md" {
		t.Fatalf("collision: expected %q, got %q", "2026-06-21-143022-test-2.md", name2)
	}

	// Create -2 and check -3.
	path2 := filepath.Join(filepath.Dir(basePath), "2026-06-21-143022-test-2.md")
	if err := os.WriteFile(path2, []byte("collision"), 0o644); err != nil {
		t.Fatal(err)
	}
	name3, err := resolveCollisionSuffix(basePath)
	if err != nil {
		t.Fatal(err)
	}
	if name3 != "2026-06-21-143022-test-3.md" {
		t.Fatalf("collision after -2: expected %q, got %q", "2026-06-21-143022-test-3.md", name3)
	}
}

func TestResolveCollisionSuffixExhausted(t *testing.T) {
	v := makeVault(t)
	basePath := filepath.Join(v.Root, "Inbox", "2026-06-21-143022-test.md")
	if err := os.MkdirAll(filepath.Dir(basePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(basePath, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Fill -2 through -1000.
	ext := ".md"
	stem := "2026-06-21-143022-test"
	dir := filepath.Dir(basePath)
	for i := 2; i <= 1000; i++ {
		p := filepath.Join(dir, fmt.Sprintf("%s-%d%s", stem, i, ext))
		if err := os.WriteFile(p, []byte("placeholder"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	_, err := resolveCollisionSuffix(basePath)
	if err == nil {
		t.Fatal("expected error after exhausting collision suffixes")
	}
}

func TestResolveCollisionSuffixTreatsBrokenSymlinkAsCollision(t *testing.T) {
	v := makeVault(t)
	basePath := filepath.Join(v.Root, "Inbox", "2026-06-21-143022-test.md")
	if err := os.MkdirAll(filepath.Dir(basePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/definitely/missing/target.md", basePath); err != nil {
		t.Fatal(err)
	}
	name, err := resolveCollisionSuffix(basePath)
	if err != nil {
		t.Fatal(err)
	}
	if name != "2026-06-21-143022-test-2.md" {
		t.Fatalf("broken symlink should be treated as existing collision, got %q", name)
	}
}

// ---------------------------------------------------------------------------
// isInboxCaptureRel
// ---------------------------------------------------------------------------

func TestIsInboxCaptureRel(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	cfg := v.LoadConfig()

	cases := []struct {
		rel  string
		want bool
	}{
		{"Inbox/hello.md", true},
		{"Inbox/sub/hello.md", false},     // not direct child
		{"Inbox/_template.md", false},     // template
		{"Inbox/Archive/hello.md", false}, // archive
		{"Inbox/.hidden.md", false},       // dot
		{"Inbox/hello.txt", false},        // not .md
		{"Other/hello.md", false},         // not Inbox
		{"Inbox/hello.md", true},          // direct child .md
	}
	for _, tc := range cases {
		got := v.isInboxCaptureRel(tc.rel, cfg)
		if got != tc.want {
			t.Errorf("isInboxCaptureRel(%q) = %v, want %v", tc.rel, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Fallback capture content
// ---------------------------------------------------------------------------

func TestBuildFallbackCaptureContent(t *testing.T) {
	capturedAt := "2026-06-21T14:30:22+02:00"

	// With body
	content := buildFallbackCaptureContent("Call tax office", "Ask about the missing document.", capturedAt)
	want := "---\ntype: capture\nstatus: inbox\ncaptured_at: 2026-06-21T14:30:22+02:00\n---\n# Call tax office\n\nAsk about the missing document.\n"
	if content != want {
		t.Fatalf("fallback content mismatch:\ngot:  %q\nwant: %q", content, want)
	}

	// Without body
	content2 := buildFallbackCaptureContent("Hello", "", capturedAt)
	want2 := "---\ntype: capture\nstatus: inbox\ncaptured_at: 2026-06-21T14:30:22+02:00\n---\n# Hello\n"
	if content2 != want2 {
		t.Fatalf("fallback content (no body) mismatch:\ngot:  %q\nwant: %q", content2, want2)
	}

	// Body with trailing newlines
	content3 := buildFallbackCaptureContent("Test", "Line1\n\nLine2\n\n\n", capturedAt)
	want3 := "---\ntype: capture\nstatus: inbox\ncaptured_at: 2026-06-21T14:30:22+02:00\n---\n# Test\n\nLine1\n\nLine2\n"
	if content3 != want3 {
		t.Fatalf("fallback content (trailing NL) mismatch:\ngot:  %q\nwant: %q", content3, want3)
	}

	content4 := buildFallbackCaptureContent("Spaces", "Line with trailing spaces   ", capturedAt)
	want4 := "---\ntype: capture\nstatus: inbox\ncaptured_at: 2026-06-21T14:30:22+02:00\n---\n# Spaces\n\nLine with trailing spaces   \n"
	if content4 != want4 {
		t.Fatalf("fallback content should preserve trailing spaces before EOF normalization:\ngot:  %q\nwant: %q", content4, want4)
	}
}

// ---------------------------------------------------------------------------
// Template vars: body and captured_at
// ---------------------------------------------------------------------------

func TestApplyTemplateIncludesBodyAndCapturedAt(t *testing.T) {
	content := applyTemplate("{{body}}\n---\n{{captured_at}}", templateVars{
		Title:      "Test",
		Body:       "My body",
		CapturedAt: "2026-06-21T14:30:22+02:00",
	})
	want := "My body\n---\n2026-06-21T14:30:22+02:00"
	if content != want {
		t.Fatalf("applyTemplate body/captured_at: got %q, want %q", content, want)
	}
}

// ---------------------------------------------------------------------------
// Task title normalization
// ---------------------------------------------------------------------------

func TestTaskTitleFromCapture(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"# Call tax office", "Call tax office"},
		{"## Heading two", "Heading two"},
		{"Plain title", "Plain title"},
		{"  spaced  title  ", "spaced title"},
		{"Multi\nline\ntitle", "Multi line title"},
	}
	for _, tc := range cases {
		got := taskTitleFromCapture(tc.input)
		if got != tc.want {
			t.Errorf("taskTitleFromCapture(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Inbox disabled reason
// ---------------------------------------------------------------------------

func TestInboxDisabledReasonEditingDisabled(t *testing.T) {
	v := makeVault(t)
	cfg := v.LoadConfig()
	reason := v.inboxDisabledReason(cfg)
	if reason == nil {
		t.Fatal("expected disabled reason when editing is disabled")
	}
	if reason.Code != "editing_disabled" {
		t.Fatalf("code = %q, want %q", reason.Code, "editing_disabled")
	}
}

func TestInboxDisabledReasonInboxHidden(t *testing.T) {
	v := makeVault(t)
	writeYAML(t, v, "editing:\n  enabled: true\nhidden:\n  - Inbox\n")
	cfg := v.LoadConfig()
	reason := v.inboxDisabledReason(cfg)
	if reason == nil {
		t.Fatal("expected disabled reason when Inbox is hidden")
	}
	if reason.Code != "inbox_hidden" {
		t.Fatalf("code = %q, want %q", reason.Code, "inbox_hidden")
	}
}

func TestInboxDisabledReasonArchiveHidden(t *testing.T) {
	v := makeVault(t)
	writeYAML(t, v, "editing:\n  enabled: true\nhidden:\n  - Inbox/Archive\n")
	cfg := v.LoadConfig()
	reason := v.inboxDisabledReason(cfg)
	if reason == nil {
		t.Fatal("expected disabled reason when Inbox/Archive is hidden")
	}
	if reason.Code != "inbox_hidden" {
		t.Fatalf("code = %q, want %q", reason.Code, "inbox_hidden")
	}
}

func TestInboxEnabled(t *testing.T) {
	v := makeVault(t)
	writeYAML(t, v, "editing:\n  enabled: true\n")
	cfg := v.LoadConfig()
	reason := v.inboxDisabledReason(cfg)
	if reason != nil {
		t.Fatalf("unexpected disabled reason: %+v", reason)
	}
}

// ---------------------------------------------------------------------------
// Resolve task destination
// ---------------------------------------------------------------------------

func TestResolveTaskDestinationConfiguredFile(t *testing.T) {
	v := makeVault(t)
	taskDir := filepath.Join(v.Root, "Tasks")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatal(err)
	}
	taskPath := filepath.Join(taskDir, "Inbox.md")
	if err := os.WriteFile(taskPath, []byte("# Tasks\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeYAML(t, v, "editing:\n  enabled: true\ntodo:\n  todo_file: Tasks/Inbox.md\n")

	cfg := v.LoadConfig()
	abs, rel, disabled, err := v.resolveTaskDestination(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if disabled != nil {
		t.Fatalf("unexpected disabled reason: %+v", disabled)
	}
	if rel != "Tasks/Inbox.md" {
		t.Fatalf("rel = %q, want %q", rel, "Tasks/Inbox.md")
	}
	if abs != taskPath {
		t.Fatalf("abs = %q, want %q", abs, taskPath)
	}
}

func TestResolveTaskDestinationConfiguredMissing(t *testing.T) {
	v := makeVault(t)
	writeYAML(t, v, "editing:\n  enabled: true\ntodo:\n  todo_file: Tasks/Missing.md\n")

	cfg := v.LoadConfig()
	_, _, disabled, err := v.resolveTaskDestination(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if disabled == nil {
		t.Fatal("expected disabled reason when todo_file is missing")
	}
	if disabled.Code != "missing_task_destination" {
		t.Fatalf("code = %q, want %q", disabled.Code, "missing_task_destination")
	}
}

func TestResolveTaskDestinationConfiguredIsDirectory(t *testing.T) {
	v := makeVault(t)
	// Create a directory instead of a file at the todo_file path.
	if err := os.MkdirAll(filepath.Join(v.Root, "Tasks", "Inbox.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeYAML(t, v, "editing:\n  enabled: true\ntodo:\n  todo_file: Tasks/Inbox.md\n")

	cfg := v.LoadConfig()
	_, _, disabled, err := v.resolveTaskDestination(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if disabled == nil {
		t.Fatal("expected disabled reason when todo_file is a directory")
	}
	if disabled.Code != "missing_task_destination" {
		t.Fatalf("code = %q, want %q", disabled.Code, "missing_task_destination")
	}
}

func TestResolveTaskDestinationFallbackDailyNote(t *testing.T) {
	v := makeVault(t)
	// Create today's daily note matching daily_notes_glob.
	todayStr := time.Now().Format("2006-01-02")
	year, month, _ := time.Now().Date()
	dailyRel := filepath.Join("Daily Notes", fmt.Sprintf("%d", year), fmt.Sprintf("%d-%02d", year, month), todayStr+".md")
	dailyAbs := filepath.Join(v.Root, filepath.FromSlash(dailyRel))
	if err := os.MkdirAll(filepath.Dir(dailyAbs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dailyAbs, []byte("# Today\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// No todo.todo_file configured.
	writeYAML(t, v, "editing:\n  enabled: true\n")

	cfg := v.LoadConfig()
	abs, rel, disabled, err := v.resolveTaskDestination(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if disabled != nil {
		t.Fatalf("unexpected disabled reason: %+v", disabled)
	}
	if rel != filepath.ToSlash(dailyRel) {
		t.Fatalf("rel = %q, want %q", rel, filepath.ToSlash(dailyRel))
	}
	if abs != dailyAbs {
		t.Fatalf("abs = %q, want %q", abs, dailyAbs)
	}
}

func TestResolveTaskDestinationFallbackDailyNoteMissing(t *testing.T) {
	v := makeVault(t)
	writeYAML(t, v, "editing:\n  enabled: true\n")

	cfg := v.LoadConfig()
	_, _, disabled, err := v.resolveTaskDestination(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if disabled == nil {
		t.Fatal("expected disabled reason when no todo_file and no daily note")
	}
	if disabled.Code != "missing_task_destination" {
		t.Fatalf("code = %q, want %q", disabled.Code, "missing_task_destination")
	}
}

// ---------------------------------------------------------------------------
// ListInboxCaptures
// ---------------------------------------------------------------------------

func TestListInboxCaptures(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)

	// Create some inbox captures with controlled timestamps.
	now := time.Date(2026, 6, 21, 10, 0, 0, 0, time.Local)
	createCapture := func(rel, title string, modTime time.Time) {
		p := filepath.Join(v.Root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		content := "---\ntitle: " + title + "\ntype: capture\nstatus: inbox\ncaptured_at: " + modTime.Format(time.RFC3339) + "\n---\n# " + title + "\n\nBody.\n"
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	createCapture("Inbox/Second.md", "Second", now.Add(-1*time.Hour))
	createCapture("Inbox/First.md", "First", now)
	// Archive capture — should be excluded.
	createCapture("Inbox/Archive/Archived.md", "Archived", now)
	// Template — should be excluded.
	createCapture("Inbox/_template.md", "Template", now)

	cfg := v.LoadConfig()
	entries := v.ListInboxCaptures(cfg)

	if len(entries) != 2 {
		t.Fatalf("expected 2 inbox entries, got %d: %+v", len(entries), entries)
	}
	// Should be newest first: First (newer) then Second (older).
	if entries[0].Stem != "First" || entries[1].Stem != "Second" {
		t.Fatalf("entries order: got %+v, want First then Second", entries)
	}
}

func TestListInboxCapturesSymlinkInboxReturnsNil(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)

	// Replace Inbox/ with a symlink.
	realInbox := filepath.Join(v.Root, "Inbox")
	if err := os.RemoveAll(realInbox); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/tmp/nonexistent-inbox", realInbox); err != nil {
		t.Fatal(err)
	}

	cfg := v.LoadConfig()
	entries := v.ListInboxCaptures(cfg)
	if entries != nil {
		t.Fatalf("expected nil when Inbox is a symlink, got %+v", entries)
	}
}

func TestListInboxCapturesSortsByParsedCapturedAt(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)

	// Use explicit captured_at values in frontmatter to verify date sorting.
	createCapture := func(rel, title, capturedAt string) {
		p := filepath.Join(v.Root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		// Quote captured_at to prevent YAML from parsing as timestamp.
		content := "---\ntitle: " + title + "\ntype: capture\nstatus: inbox\ncaptured_at: \"" + capturedAt + "\"\n---\n# " + title + "\n\nBody.\n"
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	createCapture("Inbox/Old.md", "Old", "2026-06-20T10:00:00Z")
	createCapture("Inbox/New.md", "New", "2026-06-22T10:00:00Z")
	createCapture("Inbox/Mid.md", "Mid", "2026-06-21T10:00:00Z")

	cfg := v.LoadConfig()
	entries := v.ListInboxCaptures(cfg)
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	// Must be newest first: New (22nd), Mid (21st), Old (20th).
	if entries[0].Stem != "New" || entries[1].Stem != "Mid" || entries[2].Stem != "Old" {
		t.Fatalf("entries order: got %+v, want New > Mid > Old", entries)
	}
}

func TestListInboxCapturesFallsBackToModTimeForInvalidCapturedAt(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)

	older := time.Date(2026, 6, 21, 9, 0, 0, 0, time.Local)
	newer := older.Add(2 * time.Hour)
	for _, item := range []struct {
		rel     string
		title   string
		modTime time.Time
	}{
		{"Inbox/Older.md", "Older", older},
		{"Inbox/Newer.md", "Newer", newer},
	} {
		p := filepath.Join(v.Root, filepath.FromSlash(item.rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		content := "---\ntype: capture\nstatus: inbox\ncaptured_at: not-a-date\n---\n# " + item.title + "\n"
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, item.modTime, item.modTime); err != nil {
			t.Fatal(err)
		}
	}

	entries := v.ListInboxCaptures(v.LoadConfig())
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %+v", entries)
	}
	if entries[0].Stem != "Newer" || entries[1].Stem != "Older" {
		t.Fatalf("invalid captured_at should fall back to modtime sort, got %+v", entries)
	}
}

// ---------------------------------------------------------------------------
// POST /_api/edit/capture
// ---------------------------------------------------------------------------

func TestCaptureCreateSuccess(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	body := `{"title":"Call tax office","body":"Ask about the missing document."}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/capture", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("capture create: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp captureResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("expected JSON response: %v", err)
	}
	if !strings.HasPrefix(resp.Path, "Inbox/") {
		t.Fatalf("path = %q, want Inbox/ prefix", resp.Path)
	}
	if !strings.HasSuffix(resp.Path, ".md") {
		t.Fatalf("path = %q, want .md suffix", resp.Path)
	}
	if !strings.Contains(resp.Path, "call-tax-office") {
		t.Fatalf("path = %q, should contain 'call-tax-office'", resp.Path)
	}
	if resp.URL == "" {
		t.Fatal("url should be non-empty")
	}

	// Verify the file was created.
	abs, _, err := v.resolveEditPath(resp.Path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(abs); err != nil {
		t.Fatalf("capture file not found: %v", err)
	}

	// Verify content.
	data, err := os.ReadFile(abs)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "type: capture") {
		t.Fatalf("content missing frontmatter:\n%s", content)
	}
	if !strings.Contains(content, "# Call tax office") {
		t.Fatalf("content missing title:\n%s", content)
	}
	if !strings.Contains(content, "Ask about the missing document.") {
		t.Fatalf("content missing body:\n%s", content)
	}
	if !strings.Contains(content, "captured_at:") {
		t.Fatalf("content missing captured_at:\n%s", content)
	}
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "captured_at: ") {
			if _, err := time.Parse(time.RFC3339, strings.TrimPrefix(line, "captured_at: ")); err != nil {
				t.Fatalf("captured_at should be RFC3339 with offset, got %q: %v", line, err)
			}
		}
	}
	// Must end with exactly one newline.
	if content[len(content)-1] != '\n' {
		t.Fatal("content must end with newline")
	}
}

func TestCaptureCreateEditingDisabled(t *testing.T) {
	v := makeVault(t)
	// Editing disabled by default.
	s := NewServer(v, "", "")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/capture", strings.NewReader(`{"title":"Test"}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", "any")
	s.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestCaptureCreateNoCSRF(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/capture", strings.NewReader(`{"title":"Test"}`))
	r.Header.Set("Content-Type", "application/json")
	// No CSRF token.
	s.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for missing CSRF, got %d", w.Code)
	}
}

func TestCaptureCreateEmptyTitle(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/capture", strings.NewReader(`{"title":"","body":"body"}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty title, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCaptureCreateInboxHidden(t *testing.T) {
	v := makeVault(t)
	writeYAML(t, v, "editing:\n  enabled: true\nhidden:\n  - Inbox\n")
	s := NewServer(v, "", "")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/capture", strings.NewReader(`{"title":"Test"}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for hidden Inbox, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["code"] != "inbox_hidden" {
		t.Fatalf("code = %q, want %q", resp["code"], "inbox_hidden")
	}
}

func TestCaptureCreateCollision(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	// Create first capture.
	w1 := httptest.NewRecorder()
	r1 := httptest.NewRequest("POST", "/_api/edit/capture", strings.NewReader(`{"title":"Same Title"}`))
	r1.Header.Set("Content-Type", "application/json")
	r1.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w1, r1)
	if w1.Code != http.StatusOK {
		t.Fatalf("first capture: expected 200, got %d: %s", w1.Code, w1.Body.String())
	}
	var resp1 captureResponse
	json.Unmarshal(w1.Body.Bytes(), &resp1)

	// Second capture with same title should auto-suffix.
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest("POST", "/_api/edit/capture", strings.NewReader(`{"title":"Same Title"}`))
	r2.Header.Set("Content-Type", "application/json")
	r2.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w2, r2)
	if w2.Code != http.StatusOK {
		t.Fatalf("second capture (collision): expected 200, got %d: %s", w2.Code, w2.Body.String())
	}
	var resp2 captureResponse
	json.Unmarshal(w2.Body.Bytes(), &resp2)
	if resp2.Path == resp1.Path {
		t.Fatalf("collision: expected different paths, got both %q", resp2.Path)
	}
}

func TestCaptureCreateWithTemplate(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)

	// Create a _template.md in Inbox/ with {{body}} and {{captured_at}}.
	tmplPath := filepath.Join(v.Root, "Inbox", "_template.md")
	if err := os.MkdirAll(filepath.Dir(tmplPath), 0o755); err != nil {
		t.Fatal(err)
	}
	tmplContent := "---\ntitle: {{title}}\ncaptured_at: {{captured_at}}\n---\n# {{title}}\n\nBody:\n\n{{body}}\n"
	if err := os.WriteFile(tmplPath, []byte(tmplContent), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewServer(v, "", "")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/capture", strings.NewReader(`{"title":"Templated Capture","body":"Custom body text."}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("capture with template: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp captureResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	abs, _, err := v.resolveEditPath(resp.Path)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	if !strings.Contains(content, "title: Templated Capture") {
		t.Fatalf("content missing template-substituted title:\n%s", content)
	}
	if !strings.Contains(content, "captured_at: ") {
		t.Fatalf("content missing captured_at from template:\n%s", content)
	}
	if !strings.Contains(content, "Custom body text.") {
		t.Fatalf("content missing template-substituted body:\n%s", content)
	}
}

func TestCaptureCreateBadTemplateSymlink(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	// Create a symlink _template.md in Inbox/.
	tmplPath := filepath.Join(v.Root, "Inbox", "_template.md")
	if err := os.MkdirAll(filepath.Dir(tmplPath), 0o755); err != nil {
		t.Fatal(err)
	}
	// Point to a non-existent target.
	if err := os.Symlink("/nonexistent/template.md", tmplPath); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/capture", strings.NewReader(`{"title":"Test"}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for bad template symlink, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["code"] != "capture_template_error" {
		t.Fatalf("code = %q, want %q", resp["code"], "capture_template_error")
	}
	// Must not leak vault root in response.
	bodyStr := w.Body.String()
	if strings.Contains(bodyStr, v.Root) {
		t.Fatal("response must not leak vault root path")
	}
}

// ---------------------------------------------------------------------------
// POST /_api/edit/inbox/archive
// ---------------------------------------------------------------------------

func TestInboxArchiveSuccess(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	// Create a capture.
	writeInboxCapture(t, v, "Inbox/test-note.md", "Test Note")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/inbox/archive", strings.NewReader(`{"path":"Inbox/test-note.md"}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("archive: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp inboxArchiveResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("expected JSON: %v", err)
	}
	if resp.Path != "Inbox/Archive/test-note.md" {
		t.Fatalf("archive path = %q, want %q", resp.Path, "Inbox/Archive/test-note.md")
	}

	// Verify source is gone.
	if _, err := os.Stat(filepath.Join(v.Root, "Inbox", "test-note.md")); !os.IsNotExist(err) {
		t.Fatal("source capture should be removed after archive")
	}
	// Verify archive exists.
	if _, err := os.Stat(filepath.Join(v.Root, "Inbox/Archive", "test-note.md")); err != nil {
		t.Fatalf("archive file not found: %v", err)
	}
}

func TestInboxArchiveNotInboxCapture(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	// Create a non-capture file (not direct child of Inbox/).
	nonCapture := filepath.Join(v.Root, "Areas", "note.md")
	if err := os.MkdirAll(filepath.Dir(nonCapture), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(nonCapture, []byte("# Test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/inbox/archive", strings.NewReader(`{"path":"Areas/note.md"}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for non-capture source, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["code"] != "invalid_inbox_source" {
		t.Fatalf("code = %q, want %q", resp["code"], "invalid_inbox_source")
	}
}

func TestInboxArchiveCollision(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	writeInboxCapture(t, v, "Inbox/collision-test.md", "Collision Test")
	// Create an archive file with the same name to force collision.
	if err := os.MkdirAll(filepath.Join(v.Root, "Inbox/Archive"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(v.Root, "Inbox/Archive", "collision-test.md"), []byte("# Existing\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/inbox/archive", strings.NewReader(`{"path":"Inbox/collision-test.md"}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("archive collision: expected 200 (auto-suffix), got %d: %s", w.Code, w.Body.String())
	}
	var resp inboxArchiveResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Path == "Inbox/Archive/collision-test.md" {
		t.Fatalf("expected suffixed path, got %q", resp.Path)
	}
	if !strings.Contains(resp.Path, "collision-test-2") {
		t.Fatalf("expected collision suffix -2, got %q", resp.Path)
	}
}

func TestInboxArchiveEditingDisabled(t *testing.T) {
	v := makeVault(t)
	// Editing disabled by default.
	s := NewServer(v, "", "")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/inbox/archive", strings.NewReader(`{"path":"Inbox/test.md"}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", "any")
	s.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// POST /_api/edit/inbox/move
// ---------------------------------------------------------------------------

func TestInboxMoveSuccess(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	writeInboxCapture(t, v, "Inbox/move-test.md", "Move Test")

	// Ensure target parent exists.
	if err := os.MkdirAll(filepath.Join(v.Root, "Projects", "Test"), 0o755); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/inbox/move", strings.NewReader(`{"path":"Inbox/move-test.md","target_path":"Projects/Test/moved.md","confirm_missing_dirs":true}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("move: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("expected JSON: %v", err)
	}
	if resp["path"] != "Projects/Test/moved.md" {
		t.Fatalf("path = %q, want %q", resp["path"], "Projects/Test/moved.md")
	}

	// Source should be gone.
	if _, err := os.Stat(filepath.Join(v.Root, "Inbox", "move-test.md")); !os.IsNotExist(err) {
		t.Fatal("source should be removed after move")
	}
	// Target should exist.
	if _, err := os.Stat(filepath.Join(v.Root, "Projects/Test/moved.md")); err != nil {
		t.Fatalf("target not found: %v", err)
	}
}

func TestInboxMoveMissingDirsConfirmation(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	writeInboxCapture(t, v, "Inbox/move-missing.md", "Missing Dirs")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/inbox/move", strings.NewReader(`{"path":"Inbox/move-missing.md","target_path":"NewFolder/sub/note.md","confirm_missing_dirs":false}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("move missing dirs: expected 409, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["requires_confirmation"] != "missing_dirs" {
		t.Fatalf("requires_confirmation = %q, want %q", resp["requires_confirmation"], "missing_dirs")
	}
}

func TestInboxMoveCollision(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	writeInboxCapture(t, v, "Inbox/collision-source.md", "Collision Source")
	// Create existing target.
	if err := os.MkdirAll(filepath.Join(v.Root, "Existing"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(v.Root, "Existing", "target.md"), []byte("# Existing\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/inbox/move", strings.NewReader(`{"path":"Inbox/collision-source.md","target_path":"Existing/target.md"}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("move collision: expected 409, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["code"] != "move_collision" {
		t.Fatalf("code = %q, want %q", resp["code"], "move_collision")
	}
}

func TestInboxMoveTreatsBrokenSymlinkTargetAsCollision(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	writeInboxCapture(t, v, "Inbox/symlink-source.md", "Symlink Source")
	targetDir := filepath.Join(v.Root, "Existing")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/definitely/missing/target.md", filepath.Join(targetDir, "target.md")); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/inbox/move", strings.NewReader(`{"path":"Inbox/symlink-source.md","target_path":"Existing/target.md"}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("move to broken symlink target should be policy 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestInboxMoveHiddenConfirmation(t *testing.T) {
	v := makeVault(t)
	writeYAML(t, v, "editing:\n  enabled: true\nhidden:\n  - Secret\n")
	s := NewServer(v, "", "")

	writeInboxCapture(t, v, "Inbox/hidden-target.md", "Hidden Target")
	if err := os.MkdirAll(filepath.Join(v.Root, "Secret"), 0o755); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/inbox/move", strings.NewReader(`{"path":"Inbox/hidden-target.md","target_path":"Secret/note.md","confirm_hidden":false}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("move hidden: expected 409, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["requires_confirmation"] != "hidden" {
		t.Fatalf("requires_confirmation = %q, want %q", resp["requires_confirmation"], "hidden")
	}
}

// ---------------------------------------------------------------------------
// POST /_api/edit/inbox/convert-task
// ---------------------------------------------------------------------------

func TestInboxConvertTaskWithTodoFile(t *testing.T) {
	v := makeVault(t)
	writeInboxCapture(t, v, "Inbox/convert-test.md", "Convert Test")

	// Create todo_file.
	taskDir := filepath.Join(v.Root, "Tasks")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "Inbox.md"), []byte("# Tasks\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	writeYAML(t, v, "editing:\n  enabled: true\ntodo:\n  todo_file: Tasks/Inbox.md\n")
	s := NewServer(v, "", "")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/inbox/convert-task", strings.NewReader(`{"path":"Inbox/convert-test.md"}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("convert-task: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp inboxConvertTaskResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("expected JSON: %v", err)
	}
	if resp.TaskFile != "Tasks/Inbox.md" {
		t.Fatalf("task_file = %q, want %q", resp.TaskFile, "Tasks/Inbox.md")
	}
	if !strings.HasPrefix(resp.ArchivePath, "Inbox/Archive/") {
		t.Fatalf("archive_path = %q, want Inbox/Archive/ prefix", resp.ArchivePath)
	}

	// Verify task was appended.
	data, err := os.ReadFile(filepath.Join(taskDir, "Inbox.md"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "- [ ] Convert Test") {
		t.Fatalf("task file missing appended task:\n%s", content)
	}
	if !strings.Contains(content, "📥 [[Inbox/Archive/convert-test]]") {
		t.Fatalf("task file missing archive wikilink:\n%s", content)
	}

	// Verify source is archived.
	archivePath := filepath.Join(v.Root, filepath.FromSlash(resp.ArchivePath))
	if _, err := os.Stat(archivePath); err != nil {
		t.Fatalf("archived capture not found: %v", err)
	}
	if _, err := os.Stat(filepath.Join(v.Root, "Inbox", "convert-test.md")); !os.IsNotExist(err) {
		t.Fatal("source capture should be removed after convert")
	}
}

func TestInboxConvertTaskAppendsWithSeparator(t *testing.T) {
	v := makeVault(t)
	writeInboxCapture(t, v, "Inbox/no-newline.md", "No Newline")
	if err := os.MkdirAll(filepath.Join(v.Root, "Tasks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(v.Root, "Tasks", "Inbox.md"), []byte("# Tasks"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeYAML(t, v, "editing:\n  enabled: true\ntodo:\n  todo_file: Tasks/Inbox.md\n")
	s := NewServer(v, "", "")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/inbox/convert-task", strings.NewReader(`{"path":"Inbox/no-newline.md"}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("convert-task: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	data, err := os.ReadFile(filepath.Join(v.Root, "Tasks", "Inbox.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "# Tasks\n- [ ] No Newline") {
		t.Fatalf("task append should insert separator newline, got:\n%s", string(data))
	}
}

func TestInboxConvertTaskBlocksHiddenAndTemplateDestinations(t *testing.T) {
	for _, tc := range []struct {
		name       string
		configPath string
		config     string
	}{
		{
			name:       "hidden",
			configPath: "Tasks/Inbox.md",
			config:     "editing:\n  enabled: true\nhidden:\n  - Tasks\ntodo:\n  todo_file: Tasks/Inbox.md\n",
		},
		{
			name:       "template",
			configPath: "Tasks/_template.md",
			config:     "editing:\n  enabled: true\ntodo:\n  todo_file: Tasks/_template.md\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := makeVault(t)
			writeInboxCapture(t, v, "Inbox/blocked-destination.md", "Blocked Destination")
			dest := filepath.Join(v.Root, filepath.FromSlash(tc.configPath))
			if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(dest, []byte("# Tasks\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			writeYAML(t, v, tc.config)
			s := NewServer(v, "", "")

			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/_api/edit/inbox/convert-task", strings.NewReader(`{"path":"Inbox/blocked-destination.md"}`))
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("X-CSRF-Token", s.csrfToken)
			s.ServeHTTP(w, r)
			if w.Code != http.StatusForbidden {
				t.Fatalf("expected 403 for %s destination, got %d: %s", tc.name, w.Code, w.Body.String())
			}
		})
	}
}

func TestInboxConvertTaskConfiguredTodoFileMissing(t *testing.T) {
	v := makeVault(t)
	writeInboxCapture(t, v, "Inbox/missing-todo.md", "Missing TODO")
	writeYAML(t, v, "editing:\n  enabled: true\ntodo:\n  todo_file: Tasks/Inbox.md\n")
	s := NewServer(v, "", "")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/inbox/convert-task", strings.NewReader(`{"path":"Inbox/missing-todo.md"}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("convert-task with missing todo_file: expected 409, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["code"] != "missing_task_destination" {
		t.Fatalf("code = %q, want %q", resp["code"], "missing_task_destination")
	}
}

func TestInboxConvertTaskFallbackDailyNote(t *testing.T) {
	v := makeVault(t)
	writeInboxCapture(t, v, "Inbox/daily-convert.md", "Daily Convert")

	// Create today's daily note.
	todayStr := time.Now().Format("2006-01-02")
	year, month, _ := time.Now().Date()
	dailyRel := filepath.Join("Daily Notes", fmt.Sprintf("%d", year), fmt.Sprintf("%d-%02d", year, month), todayStr+".md")
	dailyAbs := filepath.Join(v.Root, filepath.FromSlash(dailyRel))
	if err := os.MkdirAll(filepath.Dir(dailyAbs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dailyAbs, []byte("# Today\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	writeYAML(t, v, "editing:\n  enabled: true\n")
	s := NewServer(v, "", "")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/inbox/convert-task", strings.NewReader(`{"path":"Inbox/daily-convert.md"}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("convert-task with daily note: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp inboxConvertTaskResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.TaskFile != filepath.ToSlash(dailyRel) {
		t.Fatalf("task_file = %q, want %q", resp.TaskFile, filepath.ToSlash(dailyRel))
	}
}

func TestInboxConvertTaskNoDestination(t *testing.T) {
	v := makeVault(t)
	writeInboxCapture(t, v, "Inbox/no-dest.md", "No Dest")
	writeYAML(t, v, "editing:\n  enabled: true\n")
	s := NewServer(v, "", "")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/inbox/convert-task", strings.NewReader(`{"path":"Inbox/no-dest.md"}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("convert-task without destination: expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestInboxConvertTaskSourceNotInboxCapture(t *testing.T) {
	v := makeVault(t)
	writeYAML(t, v, "editing:\n  enabled: true\ntodo:\n  todo_file: Tasks/Inbox.md\n")
	// Create task file.
	if err := os.MkdirAll(filepath.Join(v.Root, "Tasks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(v.Root, "Tasks", "Inbox.md"), []byte("# Tasks\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewServer(v, "", "")

	// Try to convert a non-Inbox file.
	writeInboxCapture(t, v, "Areas/not-inbox.md", "Not Inbox")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/inbox/convert-task", strings.NewReader(`{"path":"Areas/not-inbox.md"}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for non-inbox source, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["code"] != "invalid_inbox_source" {
		t.Fatalf("code = %q, want %q", resp["code"], "invalid_inbox_source")
	}
}

// ---------------------------------------------------------------------------
// Test helper: writeYAML
// ---------------------------------------------------------------------------

func writeYAML(t *testing.T, v *Vault, yaml string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
}
