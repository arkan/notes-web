package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Disabled reason types (stable code + message for API responses)
// ---------------------------------------------------------------------------

// DisabledReason describes why an action is unavailable.
type DisabledReason struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// inboxDisabledReason returns a DisabledReason if inbox/capture is disabled,
// or nil if the functionality is available.
func (v *Vault) inboxDisabledReason(cfg Config) *DisabledReason {
	if !cfg.Editing.Enabled {
		return &DisabledReason{
			Code:    "editing_disabled",
			Message: "Editing is not enabled.",
		}
	}
	if v.isConfiguredHidden("Inbox", cfg.Hidden) || v.isConfiguredHidden("Inbox/Archive", cfg.Hidden) {
		return &DisabledReason{
			Code:    "inbox_hidden",
			Message: "Inbox or Inbox/Archive is configured hidden.",
		}
	}
	for _, rel := range []string{"Inbox", "Inbox/Archive"} {
		abs := filepath.Join(v.Root, filepath.FromSlash(rel))
		if err := checkSymlinkAncestor(v.Root, abs, false); err != nil {
			return &DisabledReason{Code: "inbox_unavailable", Message: "Inbox is not editable."}
		}
		fi, err := os.Lstat(abs)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return &DisabledReason{Code: "inbox_unavailable", Message: "Inbox cannot be inspected."}
		}
		if fi.Mode()&os.ModeSymlink != 0 || !fi.IsDir() {
			return &DisabledReason{Code: "inbox_unavailable", Message: "Inbox is not editable."}
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Inbox capture path helpers
// ---------------------------------------------------------------------------

// isInboxCaptureRel returns true if rel is a valid Inbox capture path:
// a direct .md child of Inbox/, not _template.md, not under Archive/,
// not dot/trash, not configured hidden.
func (v *Vault) isInboxCaptureRel(rel string, cfg Config) bool {
	rel = filepath.ToSlash(strings.Trim(rel, "/"))
	parts := strings.Split(rel, "/")
	if len(parts) != 2 || parts[0] != "Inbox" {
		return false
	}
	name := parts[1]

	// Must be .md
	if !strings.HasSuffix(strings.ToLower(name), ".md") {
		return false
	}
	// Not _template.md
	if v.isTemplateRel(rel, cfg.Editing.TemplateName) {
		return false
	}
	// Not under Archive/
	if strings.HasPrefix(rel, "Inbox/Archive/") {
		return false
	}
	// Not dot/trash
	if v.isDotBlocked(rel) || v.isTrashRel(rel, cfg.Editing.TrashPath) {
		return false
	}
	// Not configured hidden
	if v.isConfiguredHidden(rel, cfg.Hidden) {
		return false
	}
	return true
}

// isInboxArchiveRel returns true if rel is under Inbox/Archive/.
func isInboxArchiveRel(rel string) bool {
	rel = filepath.ToSlash(strings.Trim(rel, "/"))
	return strings.HasPrefix(rel, "Inbox/Archive/")
}

// ---------------------------------------------------------------------------
// Inbox capture filename generation
// ---------------------------------------------------------------------------

// captureFilename generates a local-timestamped Inbox capture filename:
// Inbox/YYYY-MM-DD-HHMMSS-<slug>.md
func captureFilename(title string) (string, error) {
	slug := editSlugify(title)
	if slug == "" {
		return "", fmt.Errorf("title produces an empty filename")
	}
	now := time.Now()
	ts := now.Format("2006-01-02-150405")
	return fmt.Sprintf("Inbox/%s-%s.md", ts, slug), nil
}

// resolveCollisionSuffix finds an available filename by appending numeric
// suffixes (-2, -3, ...). Returns the resolved basename (not the full path).
// It tries up to 1000 candidates. Returns error if no free suffix exists.
func resolveCollisionSuffix(absPath string) (string, error) {
	if _, err := os.Lstat(absPath); os.IsNotExist(err) {
		return filepath.Base(absPath), nil
	}
	dir := filepath.Dir(absPath)
	ext := filepath.Ext(absPath)
	stem := strings.TrimSuffix(filepath.Base(absPath), ext)
	for i := 2; i <= 1000; i++ {
		candidate := filepath.Join(dir, fmt.Sprintf("%s-%d%s", stem, i, ext))
		if _, err := os.Lstat(candidate); os.IsNotExist(err) {
			return filepath.Base(candidate), nil
		}
	}
	return "", fmt.Errorf("no available filename after 1000 collision suffixes")
}

// ---------------------------------------------------------------------------
// Fallback capture content
// ---------------------------------------------------------------------------

// buildFallbackCaptureContent returns the exact fallback Markdown for an Inbox
// capture when no template is found. Always ends with exactly one trailing
// newline. Body preserves internal and trailing whitespace; only EOF is
// normalized to one final newline.
func buildFallbackCaptureContent(title, body, capturedAt string) string {
	titleLine := "# " + title
	if body == "" {
		return fmt.Sprintf("---\ntype: capture\nstatus: inbox\ncaptured_at: %s\n---\n%s\n", capturedAt, titleLine)
	}
	// Normalize EOF: ensure exactly one trailing newline.
	normBody := strings.TrimRight(body, "\n") + "\n"
	return fmt.Sprintf("---\ntype: capture\nstatus: inbox\ncaptured_at: %s\n---\n%s\n\n%s", capturedAt, titleLine, normBody)
}

// ---------------------------------------------------------------------------
// Task title normalization for convert-to-task
// ---------------------------------------------------------------------------

// taskTitleFromCapture normalizes a capture title into a single-line task
// title: collapses internal whitespace and strips Markdown heading prefix.
func taskTitleFromCapture(title string) string {
	// Strip Markdown heading prefix (#, ##, etc.)
	re := regexp.MustCompile(`^#{1,6}\s+`)
	title = re.ReplaceAllString(title, "")
	// Collapse internal whitespace (spaces, tabs, newlines) to single spaces.
	title = regexp.MustCompile(`\s+`).ReplaceAllString(title, " ")
	return strings.TrimSpace(title)
}

// ---------------------------------------------------------------------------
// Convert-task destination resolution
// ---------------------------------------------------------------------------

// resolveTaskDestination resolves the destination path for a task conversion.
// Returns the absolute path, relative path, and any error.
// If todo_file is configured but missing, returns a DisabledReason.
// If todo_file is not configured, falls back to today's daily note.
func (v *Vault) resolveTaskDestination(cfg Config) (absPath, relPath string, disabled *DisabledReason, err error) {
	todoFile := strings.TrimSpace(cfg.Todo.TodoFile)

	if todoFile != "" {
		// todo_file is configured — resolve it.
		abs, rel, rErr := v.resolveEditPath(todoFile)
		if rErr != nil {
			return "", "", nil, rErr
		}
		// Check if the path exists as a directory.
		if fi, sErr := os.Stat(abs); sErr == nil {
			if fi.IsDir() {
				return "", "", &DisabledReason{
					Code:    "missing_task_destination",
					Message: "Configured todo.todo_file is a directory, not a file.",
				}, nil
			}
		} else if os.IsNotExist(sErr) {
			return "", "", &DisabledReason{
				Code:    "missing_task_destination",
				Message: "Configured todo.todo_file does not exist. Create it or configure a different path.",
			}, nil
		} else {
			return "", "", &DisabledReason{
				Code:    "missing_task_destination",
				Message: "Configured todo.todo_file cannot be inspected.",
			}, nil
		}
		if sErr := checkSymlinkAncestor(v.Root, abs, true); sErr != nil {
			return "", "", &DisabledReason{
				Code:    "missing_task_destination",
				Message: "Configured todo.todo_file is not accessible (symlink issue).",
			}, nil
		}
		return abs, rel, nil, nil
	}

	// No todo_file configured — fall back to today's daily note.
	todayStr := time.Now().Format("2006-01-02")
	note := v.DailyNoteForDate(todayStr)
	if note == nil {
		return "", "", &DisabledReason{
			Code:    "missing_task_destination",
			Message: "No todo.todo_file configured and today's daily note not found. Create a daily note or configure todo.todo_file.",
		}, nil
	}
	return note.Path, note.RelPath, nil, nil
}

// ---------------------------------------------------------------------------
// Inbox listing data helper (server-side data only, no UI template)
// ---------------------------------------------------------------------------

// InboxEntry represents one Inbox capture for listing.
type InboxEntry struct {
	Path         string    `json:"path"`
	Stem         string    `json:"stem"`
	Title        string    `json:"title"`
	Captured     string    `json:"captured"`
	Excerpt      string    `json:"excerpt"`
	URL          string    `json:"url"`
	capturedTime time.Time `json:"-"`
}

// ListInboxCaptures returns all valid Inbox captures, newest first.
// Excludes Archive, templates, dot/trash, hidden, and symlink files.
// Returns nil if Inbox/ itself is a symlink (safety).
// Used by the GET /_inbox route (data helper, kept backend-only).
func (v *Vault) ListInboxCaptures(cfg Config) []InboxEntry {
	inboxAbs := filepath.Join(v.Root, "Inbox")

	// Check Inbox/ itself is not a symlink.
	if fi, lErr := os.Lstat(inboxAbs); lErr == nil && fi.Mode()&os.ModeSymlink != 0 {
		return nil
	}
	// Check symlink ancestors on Inbox/.
	if cErr := checkSymlinkAncestor(v.Root, inboxAbs, false); cErr != nil {
		return nil
	}

	ents, err := os.ReadDir(inboxAbs)
	if err != nil {
		return nil
	}

	var entries []InboxEntry
	for _, e := range ents {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		rel := filepath.ToSlash(filepath.Join("Inbox", e.Name()))
		if !v.isInboxCaptureRel(rel, cfg) {
			continue
		}
		abs := filepath.Join(v.Root, filepath.FromSlash(rel))
		// Skip symlink files.
		fi, lErr := os.Lstat(abs)
		if lErr != nil || fi.Mode()&os.ModeSymlink != 0 {
			continue
		}
		// Skip symlink ancestors.
		if cErr := checkSymlinkAncestor(v.Root, abs, true); cErr != nil {
			continue
		}

		note, rErr := v.ReadNote(abs)
		if rErr != nil {
			continue
		}

		stem := strings.TrimSuffix(e.Name(), ".md")
		captured, capturedTime := captureTimestampForEntry(note)

		// Excerpt: plain text from body after frontmatter, capped.
		excerpt := extractExcerpt(note.Body, 200)

		entries = append(entries, InboxEntry{
			Path:         rel,
			Stem:         stem,
			Title:        v.Title(note),
			Captured:     captured,
			Excerpt:      excerpt,
			URL:          v.URLForRel(rel),
			capturedTime: capturedTime,
		})
	}

	// Sort newest captured/modtime first, stable path tie-breaker ascending.
	sortInboxEntries(entries)
	return entries
}

func captureTimestampForEntry(note Note) (string, time.Time) {
	switch value := note.Frontmatter["captured_at"].(type) {
	case string:
		return value, parseCaptureTime(value, note.ModTime)
	case time.Time:
		return value.Format(time.RFC3339), value
	default:
		return note.ModTime.Format(time.RFC3339), note.ModTime
	}
}

// parseCaptureTime attempts to parse a captured_at string as RFC3339 or common
// YAML timestamp formats, falling back to modTime.
func parseCaptureTime(capturedAt string, modTime time.Time) time.Time {
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02 15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, capturedAt); err == nil {
			return t
		}
	}
	return modTime
}

func sortInboxEntries(entries []InboxEntry) {
	sort.SliceStable(entries, func(i, j int) bool {
		ti := entries[i].capturedTime
		tj := entries[j].capturedTime
		if !ti.IsZero() && !tj.IsZero() && !ti.Equal(tj) {
			return ti.After(tj)
		}
		// Fallback: stable path ascending.
		return entries[i].Path < entries[j].Path
	})
}

// extractExcerpt returns the first n bytes of plain text from body after
// stripping Markdown syntax. Simple implementation: just strip frontmatter
// and return first n bytes of the remaining text.
func extractExcerpt(body string, maxLen int) string {
	// Remove leading/trailing whitespace.
	body = strings.TrimSpace(body)
	if len(body) > maxLen {
		// Try to break at a word boundary.
		truncated := body[:maxLen]
		if idx := strings.LastIndexAny(truncated, " \n\t"); idx > maxLen/2 {
			truncated = truncated[:idx]
		}
		return truncated + "..."
	}
	return body
}

// editInboxPage renders the server-backed Inbox capture page.
func (s *Server) editInboxPage(w http.ResponseWriter, r *http.Request) {
	cfg := s.vault.LoadConfig()
	if reason := s.vault.inboxDisabledReason(cfg); reason != nil {
		if reason.Code == "editing_disabled" {
			http.NotFound(w, r)
			return
		}
		http.Error(w, reason.Message, http.StatusForbidden)
		return
	}

	entries := s.vault.ListInboxCaptures(cfg)
	_, _, convertDisabled, convertErr := s.vault.resolveTaskDestination(cfg)
	if convertErr != nil {
		convertDisabled = &DisabledReason{Code: "invalid_task_destination", Message: "Task destination is invalid."}
	}

	c := setCurrentAppRoute(s.common("Inbox"), "inbox")
	c["InboxEntries"] = entries
	c["InboxCount"] = len(entries)
	c["ConvertTaskDisabledReason"] = convertDisabled
	s.render(w, "inbox", c)
}

// ---------------------------------------------------------------------------
// API: POST /_api/edit/capture  —  Create a new Inbox capture
// ---------------------------------------------------------------------------

type captureRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type captureResponse struct {
	Path string `json:"path"`
	URL  string `json:"url"`
}

func (s *Server) editCapture(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeEditError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.editAPICheck(w, r, true) {
		return
	}

	cfg := s.vault.LoadConfig()

	// Check inbox not disabled.
	if reason := s.vault.inboxDisabledReason(cfg); reason != nil {
		writeStructuredError(w, reason.Code, reason.Message, http.StatusForbidden)
		return
	}

	var req captureRequest
	if err := decodeEditJSONBody(r, &req); err != nil {
		writeEditError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		writeEditError(w, "title is required", http.StatusBadRequest)
		return
	}

	// Generate filename.
	filename, err := captureFilename(req.Title)
	if err != nil {
		writeEditError(w, err.Error(), http.StatusBadRequest)
		return
	}
	absPath, rel, err := s.vault.resolveEditPath(filename)
	if err != nil {
		writeEditError(w, "invalid path", http.StatusInternalServerError)
		return
	}

	// Symlink ancestor check on Inbox/ (before creating dirs).
	if err := checkSymlinkAncestor(s.vault.Root, filepath.Dir(absPath), false); err != nil {
		writeEditError(w, "Inbox path is not editable (symlink)", http.StatusForbidden)
		return
	}

	// Create Inbox/ directory if missing.
	parentAbs := filepath.Dir(absPath)
	if err := os.MkdirAll(parentAbs, 0o755); err != nil {
		writeEditError(w, "cannot create Inbox directory", http.StatusInternalServerError)
		return
	}

	// Handle collision: auto-suffix.
	resolvedName, cErr := resolveCollisionSuffix(absPath)
	if cErr != nil {
		writeStructuredError(w, "capture_collision_exhausted", "No available filename after collision suffixes.", http.StatusConflict)
		return
	}
	if resolvedName != filepath.Base(absPath) {
		absPath = filepath.Join(parentAbs, resolvedName)
		rel = filepath.ToSlash(filepath.Join("Inbox", resolvedName))
	}

	// Check again after potential path change.
	if s.vault.isDotBlocked(rel) || s.vault.isTrashRel(rel, cfg.Editing.TrashPath) {
		writeEditError(w, "path not allowed", http.StatusForbidden)
		return
	}

	// Resolve content via template or fallback.
	capturedAt := time.Now().Format(time.RFC3339)
	title := strings.TrimSpace(req.Title)
	body := req.Body

	_, _, tmplContent, tErr := s.vault.resolveNearestTemplate(rel, cfg)
	if tErr != nil {
		// Bad template (symlink or unreadable) blocks with stable code/message.
		writeStructuredError(w, "capture_template_error", "Capture template is not usable (symlink or unreadable).", http.StatusForbidden)
		return
	}

	var content string
	if tmplContent != "" {
		content = applyTemplate(tmplContent, templateVars{
			Title:      title,
			Slug:       editSlugify(title),
			Path:       rel,
			Folder:     "Inbox",
			Date:       todayDate(),
			Body:       body,
			CapturedAt: capturedAt,
		})
	} else {
		content = buildFallbackCaptureContent(title, body, capturedAt)
	}

	// Atomic write.
	if err := atomicWriteFile(absPath, []byte(content)); err != nil {
		writeEditError(w, "cannot write capture file", http.StatusInternalServerError)
		return
	}

	s.vault.ClearIndexCache()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(captureResponse{
		Path: rel,
		URL:  s.vault.URLForRel(rel),
	})
}

// ---------------------------------------------------------------------------
// API: POST /_api/edit/inbox/archive  —  Archive an Inbox capture
// ---------------------------------------------------------------------------

type inboxArchiveRequest struct {
	Path string `json:"path"`
}

type inboxArchiveResponse struct {
	Path string `json:"path"`
	URL  string `json:"url"`
}

func (s *Server) editInboxArchive(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeEditError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.editAPICheck(w, r, true) {
		return
	}

	cfg := s.vault.LoadConfig()

	// Check inbox not disabled.
	if reason := s.vault.inboxDisabledReason(cfg); reason != nil {
		writeStructuredError(w, reason.Code, reason.Message, http.StatusForbidden)
		return
	}

	var req inboxArchiveRequest
	if err := decodeEditJSONBody(r, &req); err != nil {
		writeEditError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Path == "" {
		writeEditError(w, "path is required", http.StatusBadRequest)
		return
	}

	srcAbs, srcRel, err := s.vault.resolveEditPath(req.Path)
	if err != nil {
		writeEditError(w, "invalid path", http.StatusBadRequest)
		return
	}

	// Source must be an Inbox capture.
	if !s.vault.isInboxCaptureRel(srcRel, cfg) {
		writeStructuredError(w, "invalid_inbox_source", "Source is not a valid Inbox capture.", http.StatusBadRequest)
		return
	}

	// Source must exist.
	srcFi, err := os.Stat(srcAbs)
	if err != nil {
		writeEditError(w, "source not found", http.StatusNotFound)
		return
	}
	if srcFi.IsDir() {
		writeStructuredError(w, "invalid_inbox_source", "Source is a directory, not a capture.", http.StatusBadRequest)
		return
	}

	// Symlink check on source.
	if err := checkSymlinkAncestor(s.vault.Root, srcAbs, true); err != nil {
		writeEditError(w, "source path is not editable (symlink)", http.StatusForbidden)
		return
	}

	// Build archive target: Inbox/Archive/<same filename>, with collision suffix.
	archiveDir := filepath.Join(s.vault.Root, filepath.FromSlash("Inbox/Archive"))
	archiveRel := "Inbox/Archive/" + filepath.Base(srcAbs)

	// Symlink check on archive path ancestors.
	if err := checkSymlinkAncestor(s.vault.Root, archiveDir, false); err != nil {
		writeEditError(w, "Archive path is not editable (symlink)", http.StatusForbidden)
		return
	}

	// Create Archive/ directory.
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		writeEditError(w, "cannot create Archive directory", http.StatusInternalServerError)
		return
	}

	archiveAbs := filepath.Join(archiveDir, filepath.Base(srcAbs))
	resolvedArchiveName, cErr := resolveCollisionSuffix(archiveAbs)
	if cErr != nil {
		writeStructuredError(w, "archive_collision_exhausted", "No available archive filename after collision suffixes.", http.StatusConflict)
		return
	}
	archiveAbs = filepath.Join(archiveDir, resolvedArchiveName)
	archiveRel = "Inbox/Archive/" + resolvedArchiveName

	// Block archive into dot/trash/template (defense-in-depth).
	archiveRelSlash := filepath.ToSlash(archiveRel)
	if s.vault.isDotBlocked(archiveRelSlash) || s.vault.isTrashRel(archiveRelSlash, cfg.Editing.TrashPath) {
		writeEditError(w, "archive path not allowed", http.StatusForbidden)
		return
	}

	// Move the file.
	if err := os.Rename(srcAbs, archiveAbs); err != nil {
		writeEditError(w, "cannot archive capture", http.StatusInternalServerError)
		return
	}

	s.vault.ClearIndexCache()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(inboxArchiveResponse{
		Path: archiveRelSlash,
		URL:  s.vault.URLForRel(archiveRelSlash),
	})
}

// ---------------------------------------------------------------------------
// API: POST /_api/edit/inbox/move  —  Move an Inbox capture to a new path
// ---------------------------------------------------------------------------

type inboxMoveRequest struct {
	Path               string `json:"path"`
	TargetPath         string `json:"target_path"`
	ConfirmMissingDirs bool   `json:"confirm_missing_dirs"`
	ConfirmHidden      bool   `json:"confirm_hidden"`
}

func (s *Server) editInboxMove(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeEditError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.editAPICheck(w, r, true) {
		return
	}

	cfg := s.vault.LoadConfig()

	// Check inbox not disabled.
	if reason := s.vault.inboxDisabledReason(cfg); reason != nil {
		writeStructuredError(w, reason.Code, reason.Message, http.StatusForbidden)
		return
	}

	var req inboxMoveRequest
	if err := decodeEditJSONBody(r, &req); err != nil {
		writeEditError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Path == "" {
		writeEditError(w, "path is required", http.StatusBadRequest)
		return
	}
	if req.TargetPath == "" {
		writeEditError(w, "target_path is required", http.StatusBadRequest)
		return
	}

	// Resolve source.
	srcAbs, srcRel, err := s.vault.resolveEditPath(req.Path)
	if err != nil {
		writeEditError(w, "invalid source path", http.StatusBadRequest)
		return
	}

	// Source must be an existing Inbox capture.
	if !s.vault.isInboxCaptureRel(srcRel, cfg) {
		writeStructuredError(w, "invalid_inbox_source", "Source is not a valid Inbox capture.", http.StatusBadRequest)
		return
	}
	srcFi, err := os.Stat(srcAbs)
	if err != nil {
		writeEditError(w, "source not found", http.StatusNotFound)
		return
	}
	if srcFi.IsDir() {
		writeStructuredError(w, "invalid_inbox_source", "Source is a directory, not a capture.", http.StatusBadRequest)
		return
	}
	if err := checkSymlinkAncestor(s.vault.Root, srcAbs, true); err != nil {
		writeEditError(w, "source path is not editable (symlink)", http.StatusForbidden)
		return
	}

	// Resolve target.
	targetRel := strings.TrimSpace(req.TargetPath)
	targetRel = strings.TrimPrefix(targetRel, "/")
	if !isMarkdownEditable(targetRel) {
		writeStructuredError(w, "invalid_target", "Target path must end with .md.", http.StatusBadRequest)
		return
	}
	targetAbs, targetRel, err := s.vault.resolveEditPath(targetRel)
	if err != nil {
		writeStructuredError(w, "invalid_target", "Invalid target path.", http.StatusBadRequest)
		return
	}

	// Block target dot/trash/template.
	if s.vault.isDotBlocked(targetRel) || s.vault.isTrashRel(targetRel, cfg.Editing.TrashPath) {
		writeStructuredError(w, "invalid_target", "Target path is not allowed (dot/trash).", http.StatusForbidden)
		return
	}
	if s.vault.isTemplateRel(targetRel, cfg.Editing.TemplateName) {
		writeStructuredError(w, "invalid_target", "Target path is not allowed (template).", http.StatusForbidden)
		return
	}

	// Symlink check on target ancestors.
	if err := checkSymlinkAncestor(s.vault.Root, targetAbs, false); err != nil {
		writeStructuredError(w, "invalid_target", "Target path is not editable (symlink).", http.StatusForbidden)
		return
	}

	// Collision check.
	if _, err := os.Lstat(targetAbs); err == nil {
		writeStructuredError(w, "move_collision", "Target already exists; choose a different path.", http.StatusConflict)
		return
	}

	// Missing parent directories check.
	targetParentAbs := filepath.Dir(targetAbs)
	missingDirs := missingParentDirs(targetParentAbs)
	if len(missingDirs) > 0 && !req.ConfirmMissingDirs {
		relDirs := s.relDirsForResponse(missingDirs)
		writeMoveMissingDirsConfirmation(w, relDirs)
		return
	}

	// Hidden path confirmation.
	if s.vault.isConfiguredHidden(targetRel, cfg.Hidden) && !req.ConfirmHidden {
		writeMoveHiddenConfirmation(w, targetRel)
		return
	}

	// Track created dirs for rollback on failure.
	var createdDirs []string
	if len(missingDirs) > 0 {
		for _, d := range missingDirs {
			if err := os.MkdirAll(d, 0o755); err != nil {
				writeEditError(w, "cannot create directories", http.StatusInternalServerError)
				return
			}
			createdDirs = append(createdDirs, d)
		}
	}

	// Perform the move (preserve content exactly, no link rewrite).
	if err := os.Rename(srcAbs, targetAbs); err != nil {
		// Best-effort clean up created dirs.
		cleanupCreatedDirs(createdDirs, srcAbs)
		writeEditError(w, "cannot move capture", http.StatusInternalServerError)
		return
	}

	s.vault.ClearIndexCache()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(map[string]any{
		"path": targetRel,
		"url":  s.vault.URLForRel(targetRel),
	})
}

// ---------------------------------------------------------------------------
// API: POST /_api/edit/inbox/convert-task  —  Convert capture to task
// ---------------------------------------------------------------------------

type inboxConvertTaskRequest struct {
	Path string `json:"path"`
}

type inboxConvertTaskResponse struct {
	TaskFile    string `json:"task_file"`
	ArchivePath string `json:"archive_path"`
	ArchiveURL  string `json:"archive_url"`
}

type inboxConvertTaskWarning struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	TaskFile string `json:"task_file"`
}

func (s *Server) editInboxConvertTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeEditError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.editAPICheck(w, r, true) {
		return
	}

	cfg := s.vault.LoadConfig()

	// Check inbox not disabled.
	if reason := s.vault.inboxDisabledReason(cfg); reason != nil {
		writeStructuredError(w, reason.Code, reason.Message, http.StatusForbidden)
		return
	}

	var req inboxConvertTaskRequest
	if err := decodeEditJSONBody(r, &req); err != nil {
		writeEditError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Path == "" {
		writeEditError(w, "path is required", http.StatusBadRequest)
		return
	}

	srcAbs, srcRel, err := s.vault.resolveEditPath(req.Path)
	if err != nil {
		writeEditError(w, "invalid path", http.StatusBadRequest)
		return
	}

	// Source must be an existing Inbox capture.
	if !s.vault.isInboxCaptureRel(srcRel, cfg) {
		writeStructuredError(w, "invalid_inbox_source", "Source is not a valid Inbox capture.", http.StatusBadRequest)
		return
	}
	srcFi, err := os.Stat(srcAbs)
	if err != nil {
		writeEditError(w, "source not found", http.StatusNotFound)
		return
	}
	if srcFi.IsDir() {
		writeStructuredError(w, "invalid_inbox_source", "Source is a directory, not a capture.", http.StatusBadRequest)
		return
	}
	if err := checkSymlinkAncestor(s.vault.Root, srcAbs, true); err != nil {
		writeEditError(w, "source path is not editable (symlink)", http.StatusForbidden)
		return
	}

	// Read source capture to extract title.
	srcNote, err := s.vault.ReadNote(srcAbs)
	if err != nil {
		writeEditError(w, "cannot read source capture", http.StatusInternalServerError)
		return
	}
	srcTitle := s.vault.Title(srcNote)
	taskTitle := taskTitleFromCapture(srcTitle)

	// Resolve task destination.
	destAbs, destRel, disabled, rErr := s.vault.resolveTaskDestination(cfg)
	if rErr != nil {
		writeEditError(w, "invalid task destination", http.StatusBadRequest)
		return
	}
	if disabled != nil {
		// Disabled reasons with stable code.
		writeStructuredError(w, disabled.Code, disabled.Message, http.StatusConflict)
		return
	}

	// Validate destination.
	if err := checkSymlinkAncestor(s.vault.Root, destAbs, true); err != nil {
		writeEditError(w, "task destination is not editable (symlink)", http.StatusForbidden)
		return
	}
	destRelSlash := filepath.ToSlash(destRel)
	if s.vault.isDotBlocked(destRelSlash) || s.vault.isTrashRel(destRelSlash, cfg.Editing.TrashPath) || s.vault.isConfiguredHidden(destRelSlash, cfg.Hidden) || s.vault.isTemplateRel(destRelSlash, cfg.Editing.TemplateName) {
		writeStructuredError(w, "invalid_task_destination", "Task destination path is not allowed.", http.StatusForbidden)
		return
	}
	if !isMarkdownEditable(destRelSlash) {
		writeStructuredError(w, "invalid_task_destination", "Task destination must be a .md file.", http.StatusBadRequest)
		return
	}

	// Pre-compute archive path with collision suffix.
	archiveDir := filepath.Join(s.vault.Root, filepath.FromSlash("Inbox/Archive"))
	archiveRel := "Inbox/Archive/" + filepath.Base(srcAbs)
	if err := checkSymlinkAncestor(s.vault.Root, archiveDir, false); err != nil {
		writeEditError(w, "Archive path is not editable (symlink)", http.StatusForbidden)
		return
	}

	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		writeEditError(w, "cannot create Archive directory", http.StatusInternalServerError)
		return
	}

	archiveAbs := filepath.Join(archiveDir, filepath.Base(srcAbs))
	resolvedArchiveName, cErr := resolveCollisionSuffix(archiveAbs)
	if cErr != nil {
		writeStructuredError(w, "archive_collision_exhausted", "No available archive filename after collision suffixes.", http.StatusConflict)
		return
	}
	archiveAbs = filepath.Join(archiveDir, resolvedArchiveName)
	archiveRel = "Inbox/Archive/" + resolvedArchiveName
	archiveRelSlash := filepath.ToSlash(archiveRel)

	// Build task line: - [ ] {task title} 📥 [[Inbox/Archive/{filename_no_ext}]]
	archiveLink := strings.TrimSuffix(archiveRelSlash, ".md")
	taskLine := fmt.Sprintf("- [ ] %s 📥 [[%s]]\n", taskTitle, archiveLink)

	// Append task to destination.
	destData, err := os.ReadFile(destAbs)
	if err != nil {
		writeEditError(w, "cannot read task destination", http.StatusInternalServerError)
		return
	}
	destMode := os.FileMode(0o644)
	if fi, fiErr := os.Stat(destAbs); fiErr == nil {
		destMode = fi.Mode().Perm()
	}
	newDestData := append([]byte{}, destData...)
	if len(newDestData) > 0 && newDestData[len(newDestData)-1] != '\n' {
		newDestData = append(newDestData, '\n')
	}
	newDestData = append(newDestData, []byte(taskLine)...)

	// Write updated destination (atomic).
	if err := writeFileWithMode(destAbs, newDestData, destMode); err != nil {
		writeEditError(w, "cannot append task to destination", http.StatusInternalServerError)
		return
	}

	// Archive the capture.
	if err := os.Rename(srcAbs, archiveAbs); err != nil {
		// Archive failed — best-effort rollback the append.
		if rbErr := writeFileWithMode(destAbs, destData, destMode); rbErr != nil {
			// Rollback also failed — return warning with 200 (task was written).
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-store")
			json.NewEncoder(w).Encode(inboxConvertTaskWarning{
				Code:     "task_written_archive_failed",
				Message:  "Task was written, but the capture could not be archived. Resolve the capture manually.",
				TaskFile: destRelSlash,
			})
			return
		}
		writeEditError(w, "task was reverted because the capture could not be archived", http.StatusInternalServerError)
		return
	}

	s.vault.ClearIndexCache()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(inboxConvertTaskResponse{
		TaskFile:    destRelSlash,
		ArchivePath: archiveRelSlash,
		ArchiveURL:  s.vault.URLForRel(archiveRelSlash),
	})
}

// ---------------------------------------------------------------------------
// Response helpers
// ---------------------------------------------------------------------------

// writeStructuredError writes a structured JSON error with code and message.
func writeStructuredError(w http.ResponseWriter, code, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"code":    code,
		"message": message,
	})
}

// writeMoveMissingDirsConfirmation writes a 409 response for missing directories.
func writeMoveMissingDirsConfirmation(w http.ResponseWriter, missingDirs []string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusConflict)
	json.NewEncoder(w).Encode(map[string]any{
		"code":                  "missing_dirs",
		"message":               "Create missing folders before moving this capture.",
		"requires_confirmation": "missing_dirs",
		"missing_dirs":          missingDirs,
	})
}

// writeMoveHiddenConfirmation writes a 409 response for hidden path confirmation.
func writeMoveHiddenConfirmation(w http.ResponseWriter, targetRel string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusConflict)
	json.NewEncoder(w).Encode(map[string]any{
		"code":                  "hidden",
		"message":               "Target path is configured hidden.",
		"requires_confirmation": "hidden",
		"path":                  targetRel,
	})
}

// cleanupCreatedDirs removes the created directories best-effort, in reverse
// order. The source path is preserved from removal if it happens to match
// a created dir (defense-in-depth).
func cleanupCreatedDirs(dirs []string, preserveAbs string) {
	for i := len(dirs) - 1; i >= 0; i-- {
		if dirs[i] == preserveAbs {
			continue
		}
		os.Remove(dirs[i])
	}
}
