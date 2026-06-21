package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Config: Editing defaults and YAML parsing
// ---------------------------------------------------------------------------

func TestEditingConfigDefaults(t *testing.T) {
	v := makeVault(t)
	cfg := v.LoadConfig()
	if cfg.Editing.Enabled {
		t.Fatal("editing.enabled should default to false")
	}
	if cfg.Editing.TrashPath != "_trash" {
		t.Fatalf("editing.trash_path default = %q, want %q", cfg.Editing.TrashPath, "_trash")
	}
	if cfg.Editing.TemplateName != "_template.md" {
		t.Fatalf("editing.template_name default = %q, want %q", cfg.Editing.TemplateName, "_template.md")
	}
	if !cfg.Editing.HideTemplates {
		t.Fatal("editing.hide_templates should default to true")
	}
	if cfg.Editing.SlugMode != "kebab_lowercase" {
		t.Fatalf("editing.slug default = %q, want %q", cfg.Editing.SlugMode, "kebab_lowercase")
	}
}

func TestEditingConfigYAMLParse(t *testing.T) {
	v := makeVault(t)
	yaml := `editing:
  enabled: true
  trash_path: .my_trash
  template_name: my_template.md
  hide_templates: false
  slug: camelCase
`
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := v.LoadConfig()
	if !cfg.Editing.Enabled {
		t.Fatal("editing.enabled should be true from YAML")
	}
	if cfg.Editing.TrashPath != ".my_trash" {
		t.Fatalf("editing.trash_path = %q, want %q", cfg.Editing.TrashPath, ".my_trash")
	}
	if cfg.Editing.TemplateName != "my_template.md" {
		t.Fatalf("editing.template_name = %q, want %q", cfg.Editing.TemplateName, "my_template.md")
	}
	if cfg.Editing.HideTemplates {
		t.Fatal("editing.hide_templates should be false from YAML")
	}
	if cfg.Editing.SlugMode != "camelCase" {
		t.Fatalf("editing.slug = %q, want %q", cfg.Editing.SlugMode, "camelCase")
	}
}

// ---------------------------------------------------------------------------
// Edit API: editing disabled / enabled gates
// ---------------------------------------------------------------------------

func TestEditAPIDisabledReturns403(t *testing.T) {
	v := makeVault(t)
	// No editing config => defaults => enabled=false
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/ping", nil)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", "any-token")
	s.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("edit API disabled: expected 403, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("expected JSON error response: %v", err)
	}
	if resp["error"] != "editing not enabled" {
		t.Fatalf("expected 'editing not enabled' error, got %q", resp["error"])
	}
}

func TestEditAPIEnabledWithValidCSRFAndJSON(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/ping", nil)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("edit API enabled with valid CSRF: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("expected JSON response: %v", err)
	}
	if resp["status"] != "ok" {
		t.Fatalf("expected status=ok, got %q", resp["status"])
	}
}

func TestEditAPIMissingCSRFReturns403(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/ping", nil)
	r.Header.Set("Content-Type", "application/json")
	// No X-CSRF-Token header
	s.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("missing CSRF: expected 403, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("expected JSON error response: %v", err)
	}
	if resp["error"] != "invalid or missing CSRF token" {
		t.Fatalf("expected CSRF error, got %q", resp["error"])
	}
}

func TestEditAPIInvalidCSRFReturns403(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/ping", nil)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", "this-is-the-wrong-token")
	s.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("invalid CSRF: expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEditAPIRejectsApplicationJSONx(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/ping", nil)
	r.Header.Set("Content-Type", "application/jsonx")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("application/jsonx: expected 415, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEditAPIRejectsTextJSON(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/ping", nil)
	r.Header.Set("Content-Type", "text/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("text/json: expected 415, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEditAPIAcceptsJSONWithCharset(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/ping", nil)
	r.Header.Set("Content-Type", "application/json; charset=iso-8859-1")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	// Valid JSON media type with charset should be accepted by mime.ParseMediaType.
	// This test ensures isJSONMediaType handles params.
	if w.Code != http.StatusOK {
		t.Fatalf("application/json with charset: expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEditAPIRejectsMalformedJSONBody(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/ping", strings.NewReader(`{invalid json}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("malformed JSON body: expected 400, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("expected JSON error response: %v", err)
	}
	if !strings.Contains(resp["error"], "invalid JSON body") {
		t.Fatalf("expected error containing 'invalid JSON body', got %q", resp["error"])
	}
}

func TestEditAPIAcceptsEmptyBody(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/ping", nil)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("empty body: expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEditAPIAcceptsValidJSONBody(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/ping", strings.NewReader(`{}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("valid JSON body: expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDecodeEditJSONBodyRejectsUnknownFields(t *testing.T) {
	var typed struct{ Name string }
	r := httptest.NewRequest("POST", "/_api/edit/ping", strings.NewReader(`{"unknown":1}`))
	if err := decodeEditJSONBody(r, &typed); err == nil {
		t.Fatal("decodeEditJSONBody should reject unknown fields on typed struct")
	}
}

func TestDecodeEditJSONBodyRejectsOversized(t *testing.T) {
	// Create a body exceeding the max size.
	large := make([]byte, editJSONMaxBody+10)
	for i := range large {
		large[i] = 'x'
	}
	bodyStr := string(large)

	// Use a fresh body per call.
	r1 := httptest.NewRequest("POST", "/_api/edit/ping", strings.NewReader(bodyStr))
	if err := decodeEditJSONBody(r1, &map[string]any{}); err == nil {
		t.Fatal("decodeEditJSONBody should reject oversized body")
	}

	r2 := httptest.NewRequest("POST", "/_api/edit/ping", strings.NewReader(bodyStr))
	if err := decodeEditJSONBody(r2, &map[string]any{}); err.Error() != "request body too large" {
		t.Fatalf("expected 'request body too large' error, got %v", err)
	}
}

func TestDecodeEditJSONBodyAcceptsEmptyBody(t *testing.T) {
	r := httptest.NewRequest("POST", "/_api/edit/ping", nil)
	var m map[string]any
	if err := decodeEditJSONBody(r, &m); err != nil {
		t.Fatalf("empty body should be accepted, got: %v", err)
	}
}

func TestDecodeEditJSONBodyRejectsMalformed(t *testing.T) {
	r := httptest.NewRequest("POST", "/_api/edit/ping", strings.NewReader(`{not json}`))
	var m map[string]any
	if err := decodeEditJSONBody(r, &m); err == nil {
		t.Fatal("malformed JSON should be rejected")
	}
}

func TestDecodeEditJSONBodyRejectsTrailingData(t *testing.T) {
	r := httptest.NewRequest("POST", "/_api/edit/ping", strings.NewReader(`{"ok":true} {"extra":true}`))
	var m map[string]any
	if err := decodeEditJSONBody(r, &m); err == nil || !strings.Contains(err.Error(), "trailing data") {
		t.Fatalf("trailing JSON should be rejected, got %v", err)
	}
}

func TestEditAPIRejectsFormContentType(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/ping", strings.NewReader("key=value"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("form content-type: expected 415, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("expected JSON error response: %v", err)
	}
	if resp["error"] != "Content-Type must be application/json" {
		t.Fatalf("expected Content-Type error, got %q", resp["error"])
	}
}

func TestEditAPIRejectsGETMethod(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_api/edit/ping", nil)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET method: expected 405, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Path policy: dot-prefixed, configured hidden, trash, template
// ---------------------------------------------------------------------------

func TestDotPrefixedDirectURLBlocked(t *testing.T) {
	v := makeVault(t)
	dotPath := filepath.Join(v.Root, ".hidden", "Note.md")
	if err := os.MkdirAll(filepath.Dir(dotPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dotPath, []byte("# Dot note\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/.hidden/Note.md", nil)
	s.ServeHTTP(w, r)

	if w.Code != 404 {
		t.Fatalf("dot-prefixed direct URL: expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDotPrefixedExcludedFromEnumeration(t *testing.T) {
	v := makeVault(t)
	dotPath := filepath.Join(v.Root, ".hidden", "Note.md")
	if err := os.MkdirAll(filepath.Dir(dotPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dotPath, []byte("# Dot note\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := v.LoadConfig()
	if !v.isExcludedFromEnumeration(".hidden/Note.md", cfg) {
		t.Fatal("dot-prefixed path should be excluded from enumeration")
	}
}

func TestTrashDirectURLBlocked(t *testing.T) {
	v := makeVault(t)
	trashPath := filepath.Join(v.Root, "_trash", "OldNote.md")
	if err := os.MkdirAll(filepath.Dir(trashPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(trashPath, []byte("# Trashed\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_trash/OldNote.md", nil)
	s.ServeHTTP(w, r)

	if w.Code != 404 {
		t.Fatalf("trash direct URL: expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTrashExcludedFromEnumeration(t *testing.T) {
	v := makeVault(t)
	trashPath := filepath.Join(v.Root, "_trash", "OldNote.md")
	if err := os.MkdirAll(filepath.Dir(trashPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(trashPath, []byte("# Trashed\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := v.LoadConfig()
	if !v.isExcludedFromEnumeration("_trash/OldNote.md", cfg) {
		t.Fatal("trash path should be excluded from enumeration")
	}
}

func TestTemplateDirectURLAllowed(t *testing.T) {
	v := makeVault(t)
	tmplPath := filepath.Join(v.Root, "Areas", "_template.md")
	if err := os.MkdirAll(filepath.Dir(tmplPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tmplPath, []byte("# Template\n\nContent\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Areas/_template.md", nil)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("template direct URL: expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTemplateExcludedFromEnumeration(t *testing.T) {
	v := makeVault(t)
	tmplPath := filepath.Join(v.Root, "_template.md")
	if err := os.WriteFile(tmplPath, []byte("# Root template\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := v.LoadConfig()
	if !v.isExcludedFromEnumeration("_template.md", cfg) {
		t.Fatal("template path should be excluded from enumeration when hide_templates=true")
	}

	// Verify a template deeper in the tree is also excluded
	areaTmpl := filepath.Join(v.Root, "Areas", "_template.md")
	if err := os.MkdirAll(filepath.Dir(areaTmpl), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(areaTmpl, []byte("# Area template\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !v.isExcludedFromEnumeration("Areas/_template.md", cfg) {
		t.Fatal("nested template path should be excluded from enumeration")
	}
}

func TestTemplateEnumerationRespectsHideTemplatesFalse(t *testing.T) {
	v := makeVault(t)
	tmplPath := filepath.Join(v.Root, "_template.md")
	if err := os.WriteFile(tmplPath, []byte("# Root template\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Set hide_templates: false
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("editing:\n  hide_templates: false\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := v.LoadConfig()
	if v.isExcludedFromEnumeration("_template.md", cfg) {
		t.Fatal("template path should NOT be excluded when hide_templates=false")
	}
}

func TestConfiguredHiddenDirectURLAllowed(t *testing.T) {
	v := makeVault(t)
	hiddenPath := filepath.Join(v.Root, "Areas", "Secret.md")
	if err := os.MkdirAll(filepath.Dir(hiddenPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(hiddenPath, []byte("# Secret\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("hidden:\n  - Areas/Secret.md\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Areas/Secret.md", nil)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("configured hidden direct URL: expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestConfiguredHiddenFolderDirectURLShowsChildren(t *testing.T) {
	v := makeVault(t)
	hiddenDir := filepath.Join(v.Root, "HiddenFolder")
	if err := os.MkdirAll(hiddenDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hiddenDir, "Child.md"), []byte("# Child\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hiddenDir, "Sibling.md"), []byte("# Sibling\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("hidden:\n  - HiddenFolder\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewServer(v, "", "")

	// Direct URL to the hidden folder should list its children.
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/HiddenFolder", nil)
	s.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("configured hidden folder direct URL: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Child.md") || !strings.Contains(w.Body.String(), "Sibling.md") {
		t.Fatalf("configured hidden folder should list its children:\n%s", w.Body.String())
	}

	// Tree enumeration must exclude HiddenFolder.
	if treeContainsRel(v.Tree(3), "HiddenFolder") {
		t.Fatalf("HiddenFolder should not appear in Tree enumeration: %+v", v.Tree(3))
	}
}

func TestConfiguredHiddenExcludedFromMarkdownFiles(t *testing.T) {
	v := makeVault(t)
	hiddenPath := filepath.Join(v.Root, "Areas", "Hidden.md")
	if err := os.MkdirAll(filepath.Dir(hiddenPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(hiddenPath, []byte("# Hidden\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("hidden:\n  - Areas/Hidden.md\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	files := v.MarkdownFiles()
	for _, p := range files {
		rel := v.Rel(p)
		if rel == "Areas/Hidden.md" {
			t.Fatalf("configured hidden path should be excluded from MarkdownFiles, found %q", rel)
		}
	}
}

func TestCSRFTokenExposedInTemplateDataWhenEnabled(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)

	s := NewServer(v, "", "")
	data := s.common("Test Title")
	token, ok := data["EditCSRFToken"].(string)
	if !ok || token == "" {
		t.Fatal("EditCSRFToken should be a non-empty string in template data when editing is enabled")
	}
	if token != s.csrfToken {
		t.Fatalf("template data token=%q, want %q", token, s.csrfToken)
	}
}

func TestCSRFTokenNotExposedWhenDisabled(t *testing.T) {
	v := makeVault(t)
	// Editing is disabled by default

	s := NewServer(v, "", "")
	data := s.common("Test Title")
	if _, ok := data["EditCSRFToken"]; ok {
		t.Fatal("EditCSRFToken should NOT be in template data when editing is disabled")
	}
}

func TestTrashFolderDirectURLBlocked(t *testing.T) {
	v := makeVault(t)
	trashDir := filepath.Join(v.Root, "_trash")
	childPath := filepath.Join(trashDir, "TrashedNote.md")
	if err := os.MkdirAll(filepath.Dir(childPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(childPath, []byte("# Trashed\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_trash/TrashedNote.md", nil)
	s.ServeHTTP(w, r)
	if w.Code != 404 {
		t.Fatalf("trash note direct URL: expected 404, got %d: %s", w.Code, w.Body.String())
	}

	// Also block direct folder URL.
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest("GET", "/_trash", nil)
	s.ServeHTTP(w2, r2)
	if w2.Code != 404 {
		t.Fatalf("trash folder direct URL: expected 404, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestDotPrefixedFolderDirectURLBlocked(t *testing.T) {
	v := makeVault(t)
	dotDir := filepath.Join(v.Root, ".hidden")
	childPath := filepath.Join(dotDir, "Note.md")
	if err := os.MkdirAll(filepath.Dir(childPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(childPath, []byte("# Dot folder note\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/.hidden/Note.md", nil)
	s.ServeHTTP(w, r)
	if w.Code != 404 {
		t.Fatalf("dot-folder note direct URL: expected 404, got %d: %s", w.Code, w.Body.String())
	}

	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest("GET", "/.hidden", nil)
	s.ServeHTTP(w2, r2)
	if w2.Code != 404 {
		t.Fatalf("dot-folder direct URL: expected 404, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestConfiguredHiddenFolderListingExcludesDotTrashTemplate(t *testing.T) {
	v := makeVault(t)
	hiddenDir := filepath.Join(v.Root, "SecretArea")
	if err := os.MkdirAll(hiddenDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Normal child that should show.
	if err := os.WriteFile(filepath.Join(hiddenDir, "Normal.md"), []byte("# Normal\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Dot-prefixed child that must be excluded even in hidden folder listing.
	dotChild := filepath.Join(hiddenDir, ".hidden_child")
	if err := os.MkdirAll(dotChild, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dotChild, "DotNote.md"), []byte("# Dot\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Trash child that must be excluded.
	trashChild := filepath.Join(hiddenDir, "_trash")
	if err := os.MkdirAll(trashChild, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(trashChild, "Trashed.md"), []byte("# Trashed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Template child that must be excluded when hide_templates is true.
	if err := os.WriteFile(filepath.Join(hiddenDir, "_template.md"), []byte("# Template\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("hidden:\n  - SecretArea\nediting:\n  enabled: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/SecretArea", nil)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("hidden folder direct URL: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()

	if !strings.Contains(body, "Normal.md") {
		t.Fatalf("hidden folder listing should include Normal.md:\n%s", body)
	}
	if strings.Contains(body, ".hidden_child") || strings.Contains(body, "DotNote.md") {
		t.Fatal("hidden folder listing should exclude dot-prefixed children")
	}
	if strings.Contains(body, "_trash") && strings.Contains(body, "Trashed.md") {
		t.Fatal("hidden folder listing should exclude trash children")
	}
	if strings.Contains(body, "_template.md") {
		t.Fatal("hidden folder listing should exclude template children when hide_templates is true")
	}
}

func TestEditingEnabledPreservesDefaults(t *testing.T) {
	v := makeVault(t)
	// editing.enabled: true with no other editing fields set.
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("editing:\n  enabled: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := v.LoadConfig()
	if !cfg.Editing.Enabled {
		t.Fatal("editing.enabled should be true")
	}
	if cfg.Editing.TrashPath != "_trash" {
		t.Fatalf("editing.trash_path = %q, want %q", cfg.Editing.TrashPath, "_trash")
	}
	if cfg.Editing.TemplateName != "_template.md" {
		t.Fatalf("editing.template_name = %q, want %q", cfg.Editing.TemplateName, "_template.md")
	}
	if !cfg.Editing.HideTemplates {
		t.Fatal("editing.hide_templates should default to true")
	}
	if cfg.Editing.SlugMode != "kebab_lowercase" {
		t.Fatalf("editing.slug = %q, want %q", cfg.Editing.SlugMode, "kebab_lowercase")
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// enableEditing writes a minimal editing.enabled=true config to the vault.
func enableEditing(t *testing.T, v *Vault) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("editing:\n  enabled: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}
