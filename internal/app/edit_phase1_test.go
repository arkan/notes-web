package app

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Source fetch
// ---------------------------------------------------------------------------

func TestEditSourceSuccess(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_api/edit/source/Areas/Target.md", nil)
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("source fetch: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("expected JSON: %v", err)
	}
	if resp["path"] != "Areas/Target.md" {
		t.Fatalf("path = %q, want %q", resp["path"], "Areas/Target.md")
	}
	content, ok := resp["content"].(string)
	if !ok || !strings.Contains(content, "# Target") {
		t.Fatal("content should contain # Target")
	}
	hash, ok := resp["hash"].(string)
	if !ok || hash == "" {
		t.Fatal("hash should be non-empty")
	}

	// no-store header.
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("source fetch should set Cache-Control: no-store")
	}
}

func TestEditSourceConfiguredHidden(t *testing.T) {
	v := makeVault(t)
	// Create a configured hidden note.
	hiddenPath := filepath.Join(v.Root, "Areas", "Hidden.md")
	if err := os.WriteFile(hiddenPath, []byte("# Hidden Content\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("hidden:\n  - Areas/Hidden.md\nediting:\n  enabled: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_api/edit/source/Areas/Hidden.md", nil)
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("configured hidden source: expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEditSourceTemplate(t *testing.T) {
	v := makeVault(t)
	tmplPath := filepath.Join(v.Root, "Areas", "_template.md")
	if err := os.MkdirAll(filepath.Dir(tmplPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tmplPath, []byte("# Template\nContent\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	enableEditing(t, v)

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_api/edit/source/Areas/_template.md", nil)
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("template source: expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEditSourceDotPathBlocked(t *testing.T) {
	v := makeVault(t)
	dotPath := filepath.Join(v.Root, ".hidden", "Note.md")
	if err := os.MkdirAll(filepath.Dir(dotPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dotPath, []byte("# Dot\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	enableEditing(t, v)

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_api/edit/source/.hidden/Note.md", nil)
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 403 && w.Code != 404 {
		t.Fatalf("dot path source: expected 403/404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEditSourceTrashBlocked(t *testing.T) {
	v := makeVault(t)
	trashPath := filepath.Join(v.Root, "_trash", "Old.md")
	if err := os.MkdirAll(filepath.Dir(trashPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(trashPath, []byte("# Old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	enableEditing(t, v)

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_api/edit/source/_trash/Old.md", nil)
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 403 && w.Code != 404 {
		t.Fatalf("trash source: expected 403/404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEditSourceEditingDisabled(t *testing.T) {
	v := makeVault(t)
	// Editing is disabled by default.
	s := NewServer(v, "", "")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_api/edit/source/Areas/Target.md", nil)
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 403 {
		t.Fatalf("editing disabled source: expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEditSourceMissingCSRF(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_api/edit/source/Areas/Target.md", nil)
	// No X-CSRF-Token header.
	s.ServeHTTP(w, r)

	if w.Code != 403 {
		t.Fatalf("missing CSRF: expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEditSourceNonExistentNote(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_api/edit/source/Areas/Nonexistent.md", nil)
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 404 {
		t.Fatalf("non-existent source: expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEditSourceSymlinkTargetBlocked(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	realPath := filepath.Join(v.Root, "Areas", "RealNote.md")
	if err := os.WriteFile(realPath, []byte("# Real\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	symPath := filepath.Join(v.Root, "Areas", "SymSrcLink.md")
	if err := os.Symlink("RealNote.md", symPath); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(symPath)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_api/edit/source/Areas/SymSrcLink.md", nil)
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 403 {
		t.Fatalf("source symlink target: expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEditSourceSymlinkAncestorBlocked(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	// Create a symlink directory as ancestor.
	realDir := filepath.Join(v.Root, "Areas", "RealNotes")
	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatal(err)
	}
	symDir := filepath.Join(v.Root, "Areas", "LinkNotes")
	if err := os.Symlink("RealNotes", symDir); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(symDir)
	realNote := filepath.Join(realDir, "Note.md")
	if err := os.WriteFile(realNote, []byte("# Under Symlink\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_api/edit/source/Areas/LinkNotes/Note.md", nil)
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 403 {
		t.Fatalf("source symlink ancestor: expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEditPreviewSymlinkAncestorBlocked(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	// Create a symlink directory ancestor.
	realDir := filepath.Join(v.Root, "Areas", "RealNotesPreview")
	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatal(err)
	}
	symDir := filepath.Join(v.Root, "Areas", "LinkNotesPreview")
	if err := os.Symlink("RealNotesPreview", symDir); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(symDir)

	body := `{"path":"Areas/LinkNotesPreview/Note.md","content":"# Preview Under Symlink"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/preview", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 403 {
		t.Fatalf("preview symlink ancestor: expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEditPreviewNonExistingPathNoSymlinkBlock(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	// Preview of a non-existing path should be allowed as long as ancestors
	// that exist are not symlinks.
	body := `{"path":"Areas/NewNote.md","content":"# New Note Preview"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/preview", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("preview non-existing path: expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEditSourceNonMarkdown(t *testing.T) {
	v := makeVault(t)
	nonMD := filepath.Join(v.Root, "file.txt")
	if err := os.WriteFile(nonMD, []byte("text"), 0o644); err != nil {
		t.Fatal(err)
	}
	enableEditing(t, v)
	s := NewServer(v, "", "")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_api/edit/source/file.txt", nil)
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 400 {
		t.Fatalf("non-markdown source: expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEditSourceEncodedPathCharacters(t *testing.T) {
	v := makeVault(t)
	encodedPath := filepath.Join(v.Root, "Areas", "Question? Mark.md")
	if err := os.WriteFile(encodedPath, []byte("# Question Mark\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	enableEditing(t, v)
	s := NewServer(v, "", "")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_api/edit/source/Areas/Question%3F%20Mark.md", nil)
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("encoded source path: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("expected JSON: %v", err)
	}
	if resp["path"] != "Areas/Question? Mark.md" {
		t.Fatalf("path = %q, want encoded path decoded", resp["path"])
	}
}

// ---------------------------------------------------------------------------
// Preview
// ---------------------------------------------------------------------------

func TestEditPreviewRendersMarkdown(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	body := `{"path":"Areas/Test.md","content":"# Hello\n\n**bold** text"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/preview", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("preview: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("expected JSON: %v", err)
	}
	html, ok := resp["html"].(string)
	if !ok || html == "" {
		t.Fatal("html should be non-empty")
	}
	if !strings.Contains(html, "<h1") || !strings.Contains(html, "Hello") {
		t.Fatalf("preview HTML should contain rendered heading:\n%s", html)
	}
	if !strings.Contains(html, "<strong>") || !strings.Contains(html, "bold") {
		t.Fatalf("preview HTML should contain rendered bold text:\n%s", html)
	}

	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("preview should set Cache-Control: no-store")
	}
}

func TestEditPreviewRendersWikilink(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	// Preview with a wikilink that resolves to an existing note.
	body := `{"path":"Areas/Test.md","content":"# Preview\n\nSee [[Target]] for details."}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/preview", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("preview wikilink: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("expected JSON: %v", err)
	}
	html, _ := resp["html"].(string)
	if !strings.Contains(html, "/Areas/Target.md") && !strings.Contains(html, "Target") {
		t.Fatalf("preview should render wikilink to Target:\n%s", html)
	}
}

func TestEditPreviewRejectsNonJSON(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/preview", strings.NewReader("not json"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 415 {
		t.Fatalf("non-JSON preview: expected 415, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEditPreviewMissingCSRF(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/preview", strings.NewReader(`{"path":"x","content":"y"}`))
	r.Header.Set("Content-Type", "application/json")
	s.ServeHTTP(w, r)

	if w.Code != 403 {
		t.Fatalf("preview missing CSRF: expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEditPreviewDotPathBlocked(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	body := `{"path":".hidden/Note.md","content":"# Test"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/preview", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 403 {
		t.Fatalf("dot path preview: expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Save
// ---------------------------------------------------------------------------

func TestEditSaveSuccess(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	// Get current hash from source.
	wSrc := httptest.NewRecorder()
	rSrc := httptest.NewRequest("GET", "/_api/edit/source/Areas/Target.md", nil)
	rSrc.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(wSrc, rSrc)

	var srcResp map[string]any
	json.Unmarshal(wSrc.Body.Bytes(), &srcResp)
	baseHash, _ := srcResp["hash"].(string)

	// Save new content.
	newContent := "# Updated Target\n\nModified content.\n"
	saveBody := `{"path":"Areas/Target.md","content":"` + jsonEscape(newContent) + `","base_hash":"` + baseHash + `"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("PUT", "/_api/edit/save", strings.NewReader(saveBody))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("save: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("expected JSON: %v", err)
	}
	if resp["status"] != "saved" {
		t.Fatalf("status = %q, want 'saved'", resp["status"])
	}
	newHash, _ := resp["hash"].(string)
	if newHash == "" || newHash == baseHash {
		t.Fatalf("hash should be non-empty and different from base: base=%q new=%q", baseHash, newHash)
	}

	// Verify file content on disk.
	note, err := v.ReadNote("Areas/Target.md")
	if err != nil {
		t.Fatal(err)
	}
	if note.Text != newContent {
		t.Fatalf("file content = %q, want %q", note.Text, newContent)
	}

	// no-store header.
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("save should set Cache-Control: no-store")
	}
}

func TestEditSavePreservesMode(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)

	targetPath := filepath.Join(v.Root, "Areas", "Target.md")
	fi, err := os.Stat(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	originalMode := fi.Mode().Perm()

	s := NewServer(v, "", "")

	// Get hash.
	wSrc := httptest.NewRecorder()
	rSrc := httptest.NewRequest("GET", "/_api/edit/source/Areas/Target.md", nil)
	rSrc.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(wSrc, rSrc)
	var srcResp map[string]any
	json.Unmarshal(wSrc.Body.Bytes(), &srcResp)
	baseHash, _ := srcResp["hash"].(string)

	saveBody := `{"path":"Areas/Target.md","content":"# Mode Test\n","base_hash":"` + baseHash + `"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("PUT", "/_api/edit/save", strings.NewReader(saveBody))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("save mode: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	newFi, err := os.Stat(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if newFi.Mode().Perm() != originalMode {
		t.Fatalf("file mode changed: was %o, now %o", originalMode, newFi.Mode().Perm())
	}
}

func TestEditSaveIndexInvalidated(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)

	// Pre-build index.
	idxBefore, err := v.BuildIndex()
	if err != nil {
		t.Fatal(err)
	}
	if len(idxBefore.Notes) == 0 {
		t.Fatal("index should have notes before save")
	}

	s := NewServer(v, "", "")

	// Get hash.
	wSrc := httptest.NewRecorder()
	rSrc := httptest.NewRequest("GET", "/_api/edit/source/Areas/Target.md", nil)
	rSrc.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(wSrc, rSrc)
	var srcResp map[string]any
	json.Unmarshal(wSrc.Body.Bytes(), &srcResp)
	baseHash, _ := srcResp["hash"].(string)

	// Save new content.
	saveBody := `{"path":"Areas/Target.md","content":"---\ntitle: Updated Title\n---\n# Updated\n","base_hash":"` + baseHash + `"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("PUT", "/_api/edit/save", strings.NewReader(saveBody))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("save index: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Rebuild index and verify the updated title.
	idxAfter, err := v.BuildIndex()
	if err != nil {
		t.Fatal(err)
	}
	meta, ok := idxAfter.ByRel["Areas/Target.md"]
	if !ok {
		t.Fatal("target note should be in index after save")
	}
	if meta.Title != "Updated Title" {
		t.Fatalf("title after save = %q, want 'Updated Title'", meta.Title)
	}
}

func TestEditSaveSymlinkTargetBlocked(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	// Create a real note and a symlink pointing to it.
	realPath := filepath.Join(v.Root, "Areas", "RealNote.md")
	if err := os.WriteFile(realPath, []byte("# Real\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	symPath := filepath.Join(v.Root, "Areas", "SymLink.md")
	if err := os.Symlink("RealNote.md", symPath); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(symPath)

	saveBody := `{"path":"Areas/SymLink.md","content":"# Overwrite\n","base_hash":"000000000000000000000000000000000000000000000000000"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("PUT", "/_api/edit/save", strings.NewReader(saveBody))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 403 {
		t.Fatalf("symlink save: expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEditSaveDotPathBlocked(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	saveBody := `{"path":".hidden/file.md","content":"# Bad\n","base_hash":"000000000000000000000000000000000000000000000000000"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("PUT", "/_api/edit/save", strings.NewReader(saveBody))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 403 {
		t.Fatalf("dot path save: expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEditSaveTrashBlocked(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	saveBody := `{"path":"_trash/Old.md","content":"# Bad\n","base_hash":"000000000000000000000000000000000000000000000000000"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("PUT", "/_api/edit/save", strings.NewReader(saveBody))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 403 {
		t.Fatalf("trash save: expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEditSaveConfiguredHiddenAllowed(t *testing.T) {
	v := makeVault(t)
	hiddenPath := filepath.Join(v.Root, "Areas", "Hidden.md")
	if err := os.WriteFile(hiddenPath, []byte("# Hidden Original\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("hidden:\n  - Areas/Hidden.md\nediting:\n  enabled: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewServer(v, "", "")

	// Get hash.
	wSrc := httptest.NewRecorder()
	rSrc := httptest.NewRequest("GET", "/_api/edit/source/Areas/Hidden.md", nil)
	rSrc.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(wSrc, rSrc)
	var srcResp map[string]any
	json.Unmarshal(wSrc.Body.Bytes(), &srcResp)
	baseHash, _ := srcResp["hash"].(string)

	newContent := "# Hidden Updated\n"
	saveBody := `{"path":"Areas/Hidden.md","content":"` + jsonEscape(newContent) + `","base_hash":"` + baseHash + `"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("PUT", "/_api/edit/save", strings.NewReader(saveBody))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("configured hidden save: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	note, err := v.ReadNote(hiddenPath)
	if err != nil {
		t.Fatal(err)
	}
	if note.Text != newContent {
		t.Fatalf("saved content = %q, want %q", note.Text, newContent)
	}
}

func TestEditSaveTemplateAllowed(t *testing.T) {
	v := makeVault(t)
	tmplPath := filepath.Join(v.Root, "Areas", "_template.md")
	if err := os.MkdirAll(filepath.Dir(tmplPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tmplPath, []byte("# Template Original\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	enableEditing(t, v)
	s := NewServer(v, "", "")

	// Get hash.
	wSrc := httptest.NewRecorder()
	rSrc := httptest.NewRequest("GET", "/_api/edit/source/Areas/_template.md", nil)
	rSrc.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(wSrc, rSrc)
	var srcResp map[string]any
	json.Unmarshal(wSrc.Body.Bytes(), &srcResp)
	baseHash, _ := srcResp["hash"].(string)

	newContent := "# Template Updated\n"
	saveBody := `{"path":"Areas/_template.md","content":"` + jsonEscape(newContent) + `","base_hash":"` + baseHash + `"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("PUT", "/_api/edit/save", strings.NewReader(saveBody))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("template save: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	note, err := v.ReadNote(tmplPath)
	if err != nil {
		t.Fatal(err)
	}
	if note.Text != newContent {
		t.Fatalf("saved template content = %q, want %q", note.Text, newContent)
	}
}

func TestEditSaveEditingDisabled(t *testing.T) {
	v := makeVault(t)
	s := NewServer(v, "", "")

	saveBody := `{"path":"Areas/Target.md","content":"# Test\n"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("PUT", "/_api/edit/save", strings.NewReader(saveBody))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 403 {
		t.Fatalf("editing disabled save: expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEditSaveMalformedJSON(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("PUT", "/_api/edit/save", strings.NewReader(`{bad json}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 400 {
		t.Fatalf("malformed JSON save: expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEditSaveUnknownFieldsRejected(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("PUT", "/_api/edit/save", strings.NewReader(`{"path":"x","content":"y","base_hash":"z","unknown":"field"}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	// With DisallowUnknownFields, unknown fields are rejected.
	if w.Code != 400 {
		t.Fatalf("unknown fields save: expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEditSaveOversizedBody(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	// Create a body larger than max size.
	large := make([]byte, editJSONMaxBody+100)
	for i := range large {
		large[i] = 'x'
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("PUT", "/_api/edit/save", strings.NewReader(string(large)))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 400 {
		t.Fatalf("oversized body: expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEditSaveNonMarkdownExtension(t *testing.T) {
	v := makeVault(t)
	txtPath := filepath.Join(v.Root, "file.txt")
	if err := os.WriteFile(txtPath, []byte("text"), 0o644); err != nil {
		t.Fatal(err)
	}
	enableEditing(t, v)
	s := NewServer(v, "", "")

	saveBody := `{"path":"file.txt","content":"new text"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("PUT", "/_api/edit/save", strings.NewReader(saveBody))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 400 {
		t.Fatalf("non-md extension: expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEditSourceDotMarkdownRejected(t *testing.T) {
	v := makeVault(t)
	// Create a .markdown file (should be rejected, only .md is editable).
	mdPath := filepath.Join(v.Root, "Areas", "Test.markdown")
	if err := os.WriteFile(mdPath, []byte("# Markdown ext\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	enableEditing(t, v)
	s := NewServer(v, "", "")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_api/edit/source/Areas/Test.markdown", nil)
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 400 {
		t.Fatalf(".markdown source: expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEditPreviewDotMarkdownRejected(t *testing.T) {
	v := makeVault(t)
	mdPath := filepath.Join(v.Root, "Areas", "Test.markdown")
	if err := os.WriteFile(mdPath, []byte("# Markdown ext\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	enableEditing(t, v)
	s := NewServer(v, "", "")

	body := `{"path":"Areas/Test.markdown","content":"# Preview"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/preview", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 400 {
		t.Fatalf(".markdown preview: expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEditSaveDotMarkdownRejected(t *testing.T) {
	v := makeVault(t)
	mdPath := filepath.Join(v.Root, "Areas", "Test.markdown")
	if err := os.WriteFile(mdPath, []byte("# Markdown ext\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	enableEditing(t, v)
	s := NewServer(v, "", "")

	saveBody := `{"path":"Areas/Test.markdown","content":"# Save","base_hash":"000000000000000000000000000000000000000000000000000"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("PUT", "/_api/edit/save", strings.NewReader(saveBody))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 400 {
		t.Fatalf(".markdown save: expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEditPreviewEmptyContent(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	// Preview with empty content should be accepted.
	body := `{"path":"Areas/Target.md","content":""}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/preview", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("preview empty content: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	// Response should be valid JSON with an html key (may be empty string
	// since empty Markdown content produces no rendered HTML beyond layout).
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("expected JSON response: %v", err)
	}
	if _, ok := resp["html"]; !ok {
		t.Fatal("preview response should contain 'html' key")
	}
}

func TestEditSaveEmptyContent(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	// Get hash.
	wSrc := httptest.NewRecorder()
	rSrc := httptest.NewRequest("GET", "/_api/edit/source/Areas/Target.md", nil)
	rSrc.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(wSrc, rSrc)
	var srcResp map[string]any
	json.Unmarshal(wSrc.Body.Bytes(), &srcResp)
	baseHash, _ := srcResp["hash"].(string)

	// Save with empty content should be accepted.
	saveBody := `{"path":"Areas/Target.md","content":"","base_hash":"` + baseHash + `"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("PUT", "/_api/edit/save", strings.NewReader(saveBody))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("save empty content: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify disk content is now empty.
	note, err := v.ReadNote("Areas/Target.md")
	if err != nil {
		t.Fatal(err)
	}
	if note.Text != "" {
		t.Fatalf("note content should be empty after save, got %q", note.Text)
	}
}

func TestEditSaveMissingBaseHash(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	saveBody := `{"path":"Areas/Target.md","content":"# No hash"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("PUT", "/_api/edit/save", strings.NewReader(saveBody))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 400 {
		t.Fatalf("missing base_hash: expected 400, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["error"] != "base_hash is required" {
		t.Fatalf("expected 'base_hash is required', got %q", resp["error"])
	}
}

func TestEditSaveConflictReturns409(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	// Use a non-matching base_hash.
	saveBody := `{"path":"Areas/Target.md","content":"# Conflict Test\n","base_hash":"0000000000000000000000000000000000000000000000000000"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("PUT", "/_api/edit/save", strings.NewReader(saveBody))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 409 {
		t.Fatalf("conflict save: expected 409, got %d: %s", w.Code, w.Body.String())
	}

	// Verify disk content unchanged.
	note, err := v.ReadNote("Areas/Target.md")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(note.Text, "Conflict Test") {
		t.Fatal("conflict save should NOT modify disk content")
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// jsonEscape escapes a string for embedding in a JSON literal.
func jsonEscape(s string) string {
	b := strings.Builder{}
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
