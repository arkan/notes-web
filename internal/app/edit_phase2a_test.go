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
// Slug helper
// ---------------------------------------------------------------------------

func TestEditSlugifyLowercaseAndHyphenated(t *testing.T) {
	if got := editSlugify("Hello World"); got != "hello-world" {
		t.Fatalf("editSlugify(%q) = %q, want %q", "Hello World", got, "hello-world")
	}
}

func TestEditSlugifyTransliteratesAccents(t *testing.T) {
	cases := []struct{ input, want string }{
		{"Éléphant", "elephant"},
		{"Français", "francais"},
		{"München", "munchen"},
		{"Straße", "strasse"},
		{"São Paulo", "sao-paulo"},
		{"Café Crème", "cafe-creme"},
		{"Ærøskøbing", "aeroskobing"},
		{"naïve", "naive"},
		{"über cool", "uber-cool"},
		{"déjà vu", "deja-vu"},
		{"föhn", "fohn"},
	}
	for _, tc := range cases {
		if got := editSlugify(tc.input); got != tc.want {
			t.Fatalf("editSlugify(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestEditSlugifyTrimsAndCollapsesHyphens(t *testing.T) {
	if got := editSlugify("  special   chars!!  "); got != "special-chars" {
		t.Fatalf("editSlugify = %q, want %q", got, "special-chars")
	}
}

func TestEditSlugifyNumeric(t *testing.T) {
	if got := editSlugify("2026-05-22"); got != "2026-05-22" {
		t.Fatalf("editSlugify = %q, want %q", got, "2026-05-22")
	}
}

// ---------------------------------------------------------------------------
// Template helpers
// ---------------------------------------------------------------------------

func TestApplyTemplateSubstitutesAllVariables(t *testing.T) {
	vars := templateVars{
		Title:  "My Title",
		Slug:   "my-title",
		Path:   "Areas/My Title.md",
		Folder: "Areas",
		Date:   "2026-06-21",
	}
	content := "---\ntitle: {{title}}\n---\n# {{title}}\n\nPath: {{path}}\nFolder: {{folder}}\nDate: {{date}}\nSlug: {{slug}}\n"
	want := "---\ntitle: My Title\n---\n# My Title\n\nPath: Areas/My Title.md\nFolder: Areas\nDate: 2026-06-21\nSlug: my-title\n"
	if got := applyTemplate(content, vars); got != want {
		t.Fatalf("applyTemplate =\n%q\nwant\n%q", got, want)
	}
}

func TestApplyTemplateLeavesUnknownVariables(t *testing.T) {
	vars := templateVars{Title: "Test"}
	content := "{{title}} {{unknown}}"
	want := "Test {{unknown}}"
	if got := applyTemplate(content, vars); got != want {
		t.Fatalf("applyTemplate = %q, want %q", got, want)
	}
}

func TestResolveNearestTemplateFindsParentFirst(t *testing.T) {
	v := makeVault(t)
	// Create a root-level template.
	rootTmpl := filepath.Join(v.Root, "_template.md")
	if err := os.WriteFile(rootTmpl, []byte("root: {{title}}"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a deeper template that should win.
	areaDir := filepath.Join(v.Root, "Areas")
	if err := os.MkdirAll(areaDir, 0o755); err != nil {
		t.Fatal(err)
	}
	areaTmpl := filepath.Join(areaDir, "_template.md")
	if err := os.WriteFile(areaTmpl, []byte("area: {{title}}"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := v.LoadConfig()
	_, _, content, err := v.resolveNearestTemplate("Areas/NewNote.md", cfg)
	if err != nil {
		t.Fatalf("resolveNearestTemplate: %v", err)
	}
	if content != "area: {{title}}" {
		t.Fatalf("expected nearest template (Areas) to win over root, got %q", content)
	}
}

func TestResolveNearestTemplateBlankFallback(t *testing.T) {
	v := makeVault(t)
	cfg := v.LoadConfig()
	_, _, content, err := v.resolveNearestTemplate("Areas/NewNote.md", cfg)
	if err != nil {
		t.Fatalf("resolveNearestTemplate: %v", err)
	}
	if content != "" {
		t.Fatalf("expected empty content when no template exists, got %q", content)
	}
}

func TestResolveNearestTemplateRootTemplate(t *testing.T) {
	v := makeVault(t)
	rootTmpl := filepath.Join(v.Root, "_template.md")
	if err := os.WriteFile(rootTmpl, []byte("root: {{title}}"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := v.LoadConfig()
	_, _, content, err := v.resolveNearestTemplate("Deep/Path/NewNote.md", cfg)
	if err != nil {
		t.Fatalf("resolveNearestTemplate: %v", err)
	}
	if content != "root: {{title}}" {
		t.Fatalf("expected root template, got %q", content)
	}
}

func TestResolveNearestTemplateRootRelPathIsClean(t *testing.T) {
	v := makeVault(t)
	if err := os.WriteFile(filepath.Join(v.Root, "_template.md"), []byte("root"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := v.LoadConfig()
	_, rel, _, err := v.resolveNearestTemplate("NewNote.md", cfg)
	if err != nil {
		t.Fatalf("resolveNearestTemplate: %v", err)
	}
	if rel != "_template.md" {
		t.Fatalf("root template rel = %q, want _template.md", rel)
	}
}

func TestResolveNearestTemplateSymlinkBlocked(t *testing.T) {
	v := makeVault(t)
	// Create a real template and a symlink pointing to it.
	realTmpl := filepath.Join(v.Root, "_real_template.md")
	if err := os.WriteFile(realTmpl, []byte("real"), 0o644); err != nil {
		t.Fatal(err)
	}
	symTmpl := filepath.Join(v.Root, "_template.md")
	if err := os.Symlink("_real_template.md", symTmpl); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(symTmpl)

	cfg := v.LoadConfig()
	_, _, _, err := v.resolveNearestTemplate("Areas/NewNote.md", cfg)
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink error, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Note create
// ---------------------------------------------------------------------------

func TestCreateNoteFromTitleSlugifiesAndWritesFile(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	body := `{"kind":"note","title":"Hello World","directory":"Areas"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/create", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("create note: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp editCreateResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("expected JSON: %v", err)
	}
	if resp.Status != "created" || resp.Kind != "note" {
		t.Fatalf("status=%q kind=%q", resp.Status, resp.Kind)
	}
	if resp.Path != "Areas/hello-world.md" {
		t.Fatalf("path = %q, want %q", resp.Path, "Areas/hello-world.md")
	}

	// Verify file on disk.
	createdPath := filepath.Join(v.Root, "Areas", "hello-world.md")
	if _, err := os.Stat(createdPath); err != nil {
		t.Fatalf("created file should exist: %v", err)
	}
}

func TestCreateNoteFromTitlePreservesDirectorySegments(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	body := `{"kind":"note","title":"My Note","directory":"Areas/Sub","confirm_missing_dirs":true}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/create", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("create note deep dir: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp editCreateResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Path != "Areas/Sub/my-note.md" {
		t.Fatalf("path = %q, want %q", resp.Path, "Areas/Sub/my-note.md")
	}
}

func TestCreateNoteFromTitlePreservesDirectorySegmentsNoConfirmation(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	body := `{"kind":"note","title":"My Note","directory":"Areas/Sub"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/create", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 409 {
		t.Fatalf("missing dirs: expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateNoteExplicitPathOverridesSlug(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	body := `{"kind":"note","path":"Areas/Explicit-Path.md","title":"Should Be Ignored"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/create", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("explicit path: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp editCreateResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Path != "Areas/Explicit-Path.md" {
		t.Fatalf("path = %q, want %q", resp.Path, "Areas/Explicit-Path.md")
	}
}

func TestCreateNoteExplicitPathRejectsNonMD(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	body := `{"kind":"note","path":"Areas/file.txt"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/create", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 400 {
		t.Fatalf("non-md path: expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateNoteCollision(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	// Try to create a note that already exists.
	body := `{"kind":"note","path":"Areas/Target.md"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/create", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 409 {
		t.Fatalf("collision: expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateNoteDotPathBlocked(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	body := `{"kind":"note","path":".git/config.md"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/create", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 403 {
		t.Fatalf("dot path: expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateNoteTrashBlocked(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	body := `{"kind":"note","path":"_trash/NewNote.md"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/create", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 403 {
		t.Fatalf("trash path: expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateNoteTemplateBlocked(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	body := `{"kind":"note","path":"Areas/_template.md"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/create", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 403 {
		t.Fatalf("template creation: expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateNoteMissingParentDirsRequiresConfirmation(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	body := `{"kind":"note","title":"Deep","directory":"NewFolder/Sub"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/create", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 409 {
		t.Fatalf("missing dirs: expected 409, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["requires_confirmation"] != "missing_dirs" {
		t.Fatalf("expected missing_dirs confirmation, got %v", resp)
	}
	missing, ok := resp["missing_dirs"].([]any)
	if !ok || len(missing) != 2 || missing[0] != "NewFolder" || missing[1] != "NewFolder/Sub" {
		t.Fatalf("missing_dirs should be vault-relative and ordered, got %#v", resp["missing_dirs"])
	}
}

func TestCreateNoteTemplateBlockedEvenWhenTemplatesAreNotHidden(t *testing.T) {
	v := makeVault(t)
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("editing:\n  enabled: true\n  hide_templates: false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewServer(v, "", "")

	body := `{"kind":"note","path":"Areas/_template.md"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/create", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 403 {
		t.Fatalf("template creation with hide_templates=false: expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateNoteMissingParentDirsConfirmed(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	body := `{"kind":"note","title":"Deep","directory":"NewFolder/Sub","confirm_missing_dirs":true}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/create", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("confirmed dirs: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	// Verify directories were created.
	if _, err := os.Stat(filepath.Join(v.Root, "NewFolder", "Sub")); err != nil {
		t.Fatal("missing parent directories should have been created")
	}
}

func TestCreateNoteConfiguredHiddenRequiresConfirmation(t *testing.T) {
	v := makeVault(t)
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("hidden:\n  - SecretArea\nediting:\n  enabled: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Pre-create directory so missing-dirs check doesn't interfere.
	if err := os.MkdirAll(filepath.Join(v.Root, "SecretArea"), 0o755); err != nil {
		t.Fatal(err)
	}
	s := NewServer(v, "", "")

	body := `{"kind":"note","title":"Hidden Note","directory":"SecretArea"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/create", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 409 {
		t.Fatalf("hidden: expected 409, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["requires_confirmation"] != "hidden" {
		t.Fatalf("expected hidden confirmation, got %v", resp)
	}
}

func TestCreateNoteConfiguredHiddenConfirmed(t *testing.T) {
	v := makeVault(t)
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("hidden:\n  - SecretArea\nediting:\n  enabled: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Pre-create directory so missing-dirs check doesn't interfere.
	if err := os.MkdirAll(filepath.Join(v.Root, "SecretArea"), 0o755); err != nil {
		t.Fatal(err)
	}
	s := NewServer(v, "", "")

	body := `{"kind":"note","title":"Hidden Note","directory":"SecretArea","confirm_hidden":true,"confirm_missing_dirs":true}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/create", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("confirmed hidden: expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateNoteUsesExplicitContent(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	body := `{"kind":"note","path":"Areas/ExplicitContent.md","content":"# Custom Content\n\nWritten directly.\n"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/create", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("explicit content: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	data, err := os.ReadFile(filepath.Join(v.Root, "Areas", "ExplicitContent.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "# Custom Content\n\nWritten directly.\n" {
		t.Fatalf("explicit content = %q", string(data))
	}
}

func TestCreateNoteAppliesTemplate(t *testing.T) {
	v := makeVault(t)
	// Create a root template.
	if err := os.WriteFile(filepath.Join(v.Root, "_template.md"), []byte("---\ntitle: {{title}}\n---\n# {{title}}\n\nCreated: {{date}}\nFolder: {{folder}}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	enableEditing(t, v)
	s := NewServer(v, "", "")

	body := `{"kind":"note","title":"Template Test","directory":"Areas"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/create", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("template create: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp editCreateResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.TemplatePath == "" {
		t.Fatal("template_path should be non-empty when template is used")
	}
	if !strings.Contains(resp.Content, "title: Template Test") {
		t.Fatalf("content should contain substituted title:\n%s", resp.Content)
	}
	if !strings.Contains(resp.Content, "Folder: Areas") {
		t.Fatalf("content should contain substituted folder:\n%s", resp.Content)
	}
}

func TestCreateNoteBlankFallbackWhenNoTemplate(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	body := `{"kind":"note","title":"Blank","directory":"Areas"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/create", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("blank fallback: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp editCreateResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Content != "" {
		t.Fatalf("content should be empty when no template: %q", resp.Content)
	}
}

func TestCreateNoteSymlinkAncestorBlocked(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	// Create a symlink directory.
	realDir := filepath.Join(v.Root, "RealDir")
	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatal(err)
	}
	symDir := filepath.Join(v.Root, "LinkDir")
	if err := os.Symlink("RealDir", symDir); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(symDir)

	body := `{"kind":"note","path":"LinkDir/NewNote.md"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/create", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 403 {
		t.Fatalf("symlink ancestor: expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateNoteIndexInvalidated(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	// Pre-build index.
	idxBefore, err := v.BuildIndex()
	if err != nil {
		t.Fatal(err)
	}
	countBefore := len(idxBefore.Notes)

	body := `{"kind":"note","title":"New Indexed Note","directory":"Areas"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/create", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("create for index: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// After create + ClearIndexCache, BuildIndex should see the new note.
	idxAfter, err := v.BuildIndex()
	if err != nil {
		t.Fatal(err)
	}
	if len(idxAfter.Notes) <= countBefore {
		t.Fatalf("index should have more notes after create: before=%d after=%d", countBefore, len(idxAfter.Notes))
	}
}

// ---------------------------------------------------------------------------
// Folder create
// ---------------------------------------------------------------------------

func TestCreateFolderPreservesName(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	body := `{"kind":"folder","path":"My Folder With Spaces"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/create", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("create folder: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp editCreateResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Kind != "folder" || resp.Status != "created" {
		t.Fatalf("status=%q kind=%q", resp.Status, resp.Kind)
	}
	if resp.Path != "My Folder With Spaces" {
		t.Fatalf("path = %q, want %q", resp.Path, "My Folder With Spaces")
	}

	// Verify on disk.
	fi, err := os.Stat(filepath.Join(v.Root, "My Folder With Spaces"))
	if err != nil || !fi.IsDir() {
		t.Fatal("folder should exist")
	}
}

func TestCreateFolderCollision(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	// Areas already exists.
	body := `{"kind":"folder","path":"Areas"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/create", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 409 {
		t.Fatalf("folder collision: expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateFolderDotPathBlocked(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	body := `{"kind":"folder","path":".hidden"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/create", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 403 {
		t.Fatalf("dot folder: expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateFolderMissingParentDirsConfirmed(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	body := `{"kind":"folder","path":"NewParent/ChildFolder","confirm_missing_dirs":true}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/create", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("folder with dirs: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if _, err := os.Stat(filepath.Join(v.Root, "NewParent", "ChildFolder")); err != nil {
		t.Fatal("created folder should exist")
	}
}

func TestCreateFolderEmptyPathRejected(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	body := `{"kind":"folder","path":""}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/create", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 400 {
		t.Fatalf("empty folder path: expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateFolderTrashBlocked(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	body := `{"kind":"folder","path":"_trash"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/create", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 403 {
		t.Fatalf("trash folder: expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateFolderInvalidKind(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	body := `{"kind":"invalid"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/create", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 400 {
		t.Fatalf("invalid kind: expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateNoteAccentTransliteration(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	body := `{"kind":"note","title":"Été à Paris","directory":"Areas"}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/create", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("accent create: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp editCreateResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Path != "Areas/ete-a-paris.md" {
		t.Fatalf("accent slug = %q, want %q", resp.Path, "Areas/ete-a-paris.md")
	}
}

// ---------------------------------------------------------------------------
// todayDate override safety valve (restore global after tests)
// ---------------------------------------------------------------------------

func TestTodayDateFormat(t *testing.T) {
	// Just verify format: YYYY-MM-DD
	date := todayDate()
	if len(date) != 10 || date[4] != '-' || date[7] != '-' {
		t.Fatalf("todayDate() = %q, want YYYY-MM-DD format", date)
	}
}
