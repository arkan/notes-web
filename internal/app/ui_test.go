package app

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSidebarFoldersClosedAndCopyScriptAvailable(t *testing.T) {
	v := makeVault(t)
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	for _, want := range []string{`<details class="tree-folder" data-tree-path="Areas">`, `<summary><span aria-hidden="true">📁</span> Areas</summary>`, `data-copy-path`} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in home HTML", want)
		}
	}
	if strings.Contains(body, `<details class="tree-folder" data-tree-path="Areas" open>`) {
		t.Fatalf("sidebar folders should be closed by default")
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_static/app.js", nil)
	s.ServeHTTP(w, r)
	js := w.Body.String()
	for _, want := range []string{"copyText", "navigator.clipboard", "execCommand", "data-copy", "data-copy-code", "data-copy-path", "currentCopyPath", "location.pathname", "closest('pre')", "notes-web:sidebar-open", "data-tree-path", "localStorage"} {
		if !strings.Contains(js, want) {
			t.Fatalf("missing %q in app.js:\n%s", want, js)
		}
	}
	for _, forbidden := range []string{"location.href", "data-copy-link", "Link copied"} {
		if strings.Contains(js, forbidden) {
			t.Fatalf("app.js should not use %q for page path copying", forbidden)
		}
	}
}

func sidebarVaultHTML(t *testing.T, body string) string {
	t.Helper()
	start := strings.Index(body, `<section><h3>Vault</h3>`)
	if start < 0 {
		t.Fatalf("missing sidebar vault section in:\n%s", body)
	}
	end := strings.Index(body[start:], `</section></aside>`)
	if end < 0 {
		t.Fatalf("missing sidebar vault section end in:\n%s", body)
	}
	return body[start : start+end]
}

func TestSidebarHomePrunesNestedDailyNotesUntilActive(t *testing.T) {
	v := makeVault(t)
	dailyRel := "Daily Notes/2026/2026-05/2026-05-23.md"
	dailyPath := filepath.Join(v.Root, filepath.FromSlash(dailyRel))
	if err := os.MkdirAll(filepath.Dir(dailyPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dailyPath, []byte("# Daily note\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	siblingRel := "Daily Notes/2026/2026-05/2026-05-22.md"
	if err := os.WriteFile(filepath.Join(v.Root, filepath.FromSlash(siblingRel)), []byte("# Sibling daily note\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	sidebar := sidebarVaultHTML(t, body)

	if !strings.Contains(sidebar, `data-tree-path="Daily Notes"`) {
		t.Fatalf("sidebar should include top-level Daily Notes folder in:\n%s", body)
	}
	for _, unwanted := range []string{
		`data-tree-path="Daily Notes/2026"`,
		`data-tree-path="Daily Notes/2026/2026-05"`,
		`href="/Daily%20Notes/2026/2026-05/2026-05-23.md"`,
	} {
		if strings.Contains(sidebar, unwanted) {
			t.Fatalf("homepage sidebar should not include nested daily note entry %q in:\n%s", unwanted, body)
		}
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/Daily%20Notes/2026/2026-05/2026-05-23.md", nil)
	s.ServeHTTP(w, r)
	body = w.Body.String()
	sidebar = sidebarVaultHTML(t, body)
	for _, want := range []string{
		`<details class="tree-folder active-branch" data-tree-path="Daily Notes" open>`,
		`<details class="tree-folder active-branch" data-tree-path="Daily Notes/2026" open>`,
		`<details class="tree-folder active-branch" data-tree-path="Daily Notes/2026/2026-05" open>`,
		`<a class="active" aria-current="page" href="/Daily%20Notes/2026/2026-05/2026-05-23.md"><span aria-hidden="true">📄</span> 2026-05-23.md</a>`,
	} {
		if !strings.Contains(sidebar, want) {
			t.Fatalf("active note sidebar should include branch entry %q in:\n%s", want, body)
		}
	}
	if strings.Contains(sidebar, `href="/Daily%20Notes/2026/2026-05/2026-05-22.md"`) {
		t.Fatalf("active note sidebar should not include sibling daily files in:\n%s", body)
	}
}

func TestMainContainerUsesAvailableWidthWithoutHorizontalPageOverflow(t *testing.T) {
	v := makeVault(t)
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_static/style.css", nil)
	s.ServeHTTP(w, r)
	css := w.Body.String()
	if strings.Contains(css, ".main{max-width:") {
		t.Fatalf("main container should not have a max-width cap:\n%s", css)
	}
	for _, want := range []string{
		"grid-template-columns:300px minmax(0,1fr)",
		".main{min-width:0;width:100%;",
		".note header{display:flex;align-items:start;justify-content:space-between;gap:20px;min-width:0}",
		".content{font-size:17px;line-height:1.72;overflow-wrap:anywhere}",
		".content a{overflow-wrap:anywhere}",
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("missing overflow-safe CSS %q in:\n%s", want, css)
		}
	}
}

func TestReadableTypographyKeepsTextComfortableAndWideBlocksUseful(t *testing.T) {
	v := makeVault(t)
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_static/style.css", nil)
	s.ServeHTTP(w, r)
	css := w.Body.String()
	for _, want := range []string{
		"--measure:100%",
		".content{font-size:17px;line-height:1.72;overflow-wrap:anywhere}",
		".content>:where(p,ul,ol,blockquote,details,dl){max-width:100%}",
		".content>:where(pre,.markdown-table-wrap,.mermaid,img){max-width:100%}",
		".content p{margin:0 0 1.05rem}",
		".content h2{margin:2.2rem 0 1rem}",
		".content pre{overflow:auto;max-width:100%;",
		".markdown-table-wrap{width:100%;max-width:100%;overflow-x:auto;",
		".content .markdown-table-wrap table{display:table;width:100%;min-width:680px;",
		".content .markdown-table-wrap td:last-child{min-width:18rem;max-width:34rem}",
		"@media(max-width:640px){.content .markdown-table-wrap table{min-width:580px;font-size:.82rem;",
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("missing readable typography CSS %q in:\n%s", want, css)
		}
	}
}

func TestCommandPaletteMarkupAPIAndClientBehavior(t *testing.T) {
	v := makeVault(t)
	s := NewServer(v, "", "")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	for _, want := range []string{
		`<button class="palette-button btn" data-palette-open aria-label="Open command palette">⌘K</button>`,
		`<div class="palette" data-palette hidden>`,
		`<input data-palette-input`,
		`<div class="palette-results" data-palette-results role="listbox">`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing palette markup %q in:\n%s", want, body)
		}
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_api/palette", nil)
	s.ServeHTTP(w, r)
	json := w.Body.String()
	for _, want := range []string{`"title":"Target"`, `"url":"/Areas/Target.md"`, `"kind":"note"`} {
		if !strings.Contains(json, want) {
			t.Fatalf("missing palette API data %q in:\n%s", want, json)
		}
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_static/app.js", nil)
	s.ServeHTTP(w, r)
	js := w.Body.String()
	for _, want := range []string{
		"function initCommandPalette()",
		"function loadPaletteItems()",
		"/_api/palette",
		"metaKey",
		"ev.key === '/'",
		"data-palette-results",
		"results?.addEventListener('click'",
		"results?.addEventListener('mousemove'",
		"location.assign(item.url)",
		"Loading…",
		"Unable to load search results.",
	} {
		if !strings.Contains(js, want) {
			t.Fatalf("missing palette JS %q in:\n%s", want, js)
		}
	}
	if strings.Contains(js, "button.addEventListener('mouseenter'") || strings.Contains(js, "button.addEventListener('click'") {
		t.Fatalf("palette result interactions should be delegated; per-button listeners are unstable when hover rerenders:\n%s", js)
	}
}

func TestEditModeHooksRenderOnlyWhenEditingEnabled(t *testing.T) {
	v := makeVault(t)
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Areas/Target.md", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	for _, unwanted := range []string{`data-edit-open`, `data-edit-new`, `data-edit-rename`, `data-edit-trash`, `data-edit-csrf=`} {
		if strings.Contains(body, unwanted) {
			t.Fatalf("edit hook %q should not render when editing is disabled:\n%s", unwanted, body)
		}
	}

	v = makeVault(t)
	enableEditing(t, v)
	s = NewServer(v, "", "")
	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/Areas/Target.md", nil)
	s.ServeHTTP(w, r)
	body = w.Body.String()
	for _, want := range []string{
		`data-edit-surface data-edit-kind="note" data-edit-path="Areas/Target.md" data-edit-csrf="`,
		`<div class="note-actions" data-note-actions>`,
		`data-note-actions-toggle aria-haspopup="menu" aria-expanded="false" aria-label="Actions">⚙</button>`,
		`<div class="note-actions-menu" data-note-actions-menu role="menu" aria-label="Note actions" hidden>`,
		`<button class="note-actions-item" role="menuitem" type="button" data-edit-new>New</button>`,
		`<button class="note-actions-item" role="menuitem" type="button" data-edit-open>Edit</button>`,
		`<button class="note-actions-item" role="menuitem" type="button" data-edit-rename>Rename</button>`,
		`<button class="note-actions-item danger" role="menuitem" type="button" data-edit-trash data-edit-trash-kind="note">Move to Trash</button>`,
		`<button class="note-actions-item copy-link" role="menuitem" type="button" data-copy-path>Copy path</button>`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing edit-mode markup %q in:\n%s", want, body)
		}
	}
	if strings.Count(body, `data-note-actions-toggle`) != 1 {
		t.Fatalf("note header should render one compact actions trigger:\n%s", body)
	}
}

func TestEditModeHooksDoNotRenderForDotMarkdownNotes(t *testing.T) {
	v := makeVault(t)
	legacyPath := filepath.Join(v.Root, "Areas", "Legacy.markdown")
	if err := os.WriteFile(legacyPath, []byte("# Legacy\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	enableEditing(t, v)
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Areas/Legacy.markdown", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	if strings.Contains(body, `data-edit-open`) || strings.Contains(body, `data-edit-surface`) || strings.Contains(body, `data-edit-new`) || strings.Contains(body, `data-edit-rename`) || strings.Contains(body, `data-edit-trash`) {
		t.Fatalf("edit hooks should not render for .markdown files:\n%s", body)
	}
}

func TestPhase2EditCRUDHooksRenderInFolderAndMissingContexts(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Areas", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	for _, want := range []string{
		`class="folder-view reading-surface" data-edit-context data-edit-kind="folder" data-edit-directory="Areas" data-edit-path="Areas" data-edit-csrf="`,
		`<div class="note-actions" data-note-actions>`,
		`data-note-actions-toggle aria-haspopup="menu" aria-expanded="false" aria-label="Actions">⚙</button>`,
		`<div class="note-actions-menu" data-note-actions-menu role="menu" aria-label="Folder actions" hidden>`,
		`<button class="note-actions-item" role="menuitem" type="button" data-edit-new>New</button>`,
		`<button class="note-actions-item danger" role="menuitem" type="button" data-edit-trash data-edit-trash-kind="folder">Move to Trash</button>`,
		`<button class="note-actions-item copy-link" role="menuitem" type="button" data-copy-path>Copy path</button>`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing folder edit CRUD markup %q in:\n%s", want, body)
		}
	}
	if strings.Contains(body, `data-edit-rename`) {
		t.Fatalf("non-empty folder UI should not expose Rename action:\n%s", body)
	}
	if err := os.Mkdir(filepath.Join(v.Root, "Empty Folder"), 0o755); err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/Empty%20Folder", nil)
	s.ServeHTTP(w, r)
	emptyFolderBody := w.Body.String()
	if !strings.Contains(emptyFolderBody, `data-edit-rename`) || !strings.Contains(emptyFolderBody, `data-can-rename="true"`) {
		t.Fatalf("empty folder UI should expose Rename action:\n%s", emptyFolderBody)
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_missing?name=Missing%20Note&source=Areas/Target.md", nil)
	s.ServeHTTP(w, r)
	body = w.Body.String()
	for _, want := range []string{
		`data-missing-create-context data-edit-csrf="`,
		`data-missing-target="Missing Note"`,
		`data-missing-source="Areas/Target.md"`,
		`data-edit-missing-create>Create this note</button>`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing missing-link create markup %q in:\n%s", want, body)
		}
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_missing?name=Missing%20Note", nil)
	s.ServeHTTP(w, r)
	body = w.Body.String()
	if strings.Contains(body, `data-edit-missing-create`) || strings.Contains(body, `data-missing-create-context`) {
		t.Fatalf("missing-link create should require source context:\n%s", body)
	}
}

func TestPhase3TrashUIHooksRender(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	snapshot := filepath.Join(v.Root, "_trash", "2026-06-21T120000-abcdef")
	if err := os.MkdirAll(snapshot, 0o755); err != nil {
		t.Fatal(err)
	}
	meta := `{"original_path":"Areas/Old.md","trashed_at":"2026-06-21T12:00:00Z","kind":"note"}`
	if err := os.WriteFile(filepath.Join(snapshot, ".notes-web-trash.json"), []byte(meta), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewServer(v, "", "")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Areas/Target.md", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	if !strings.Contains(body, `data-edit-trash data-edit-trash-kind="note">Move to Trash</button>`) {
		t.Fatalf("note page should expose Move to Trash action:\n%s", body)
	}
	if strings.Contains(body, `Delete`) || strings.Contains(body, `Purge`) {
		t.Fatalf("note page should not expose Delete/Purge copy:\n%s", body)
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/Areas", nil)
	s.ServeHTTP(w, r)
	body = w.Body.String()
	if !strings.Contains(body, `data-edit-trash data-edit-trash-kind="folder">Move to Trash</button>`) {
		t.Fatalf("folder page should expose Move to Trash action:\n%s", body)
	}
	if strings.Contains(body, `Delete`) || strings.Contains(body, `Purge`) {
		t.Fatalf("folder page should not expose Delete/Purge copy:\n%s", body)
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_trash", nil)
	s.ServeHTTP(w, r)
	body = w.Body.String()
	for _, want := range []string{
		`class="trash-view reading-surface" data-trash-context data-edit-csrf="`,
		`class="trash-card" data-trash-entry data-trash-snapshot="2026-06-21T120000-abcdef" data-trash-original-path="Areas/Old.md"`,
		`<span class="trash-kind">note</span>`,
		`<h2>Areas/Old.md</h2>`,
		`<dt>Trashed at</dt><dd>2026-06-21T12:00:00Z</dd>`,
		`<dt>Snapshot</dt><dd><small>2026-06-21T120000-abcdef</small></dd>`,
		`data-trash-restore>Restore</button>`,
		`data-trash-restore-as>Restore as…</button>`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("trash page missing %q in:\n%s", want, body)
		}
	}
	if strings.Contains(body, `Delete`) || strings.Contains(body, `Purge`) {
		t.Fatalf("trash page should not expose Delete/Purge copy:\n%s", body)
	}
}

func TestHiddenBadgeOnConfiguredHiddenNote(t *testing.T) {
	v := makeVault(t)
	hiddenPath := filepath.Join(v.Root, "Areas", "HiddenBadge.md")
	if err := os.WriteFile(hiddenPath, []byte("# Hidden Badge\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("hidden:\n  - Areas/HiddenBadge.md\nediting:\n  enabled: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Areas/HiddenBadge.md", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	if !strings.Contains(body, `<span class="chip badge-hidden">Hidden</span>`) {
		t.Fatalf("configured hidden note should show Hidden badge:\n%s", body)
	}
}

func TestTemplateBadgeOnTemplateNote(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	tmplPath := filepath.Join(v.Root, "Areas", "_template.md")
	if err := os.MkdirAll(filepath.Dir(tmplPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tmplPath, []byte("# Template Note\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Areas/_template.md", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	if !strings.Contains(body, `<span class="chip badge-template">Template</span>`) {
		t.Fatalf("template note should show Template badge:\n%s", body)
	}
	for _, want := range []string{`data-edit-new`, `data-edit-open`} {
		if !strings.Contains(body, want) {
			t.Fatalf("template note should keep edit action %q:\n%s", want, body)
		}
	}
	for _, unwanted := range []string{`data-edit-rename`, `data-edit-trash`, `data-can-trash="true"`, `data-can-rename="true"`} {
		if strings.Contains(body, unwanted) {
			t.Fatalf("template note should not expose management action %q:\n%s", unwanted, body)
		}
	}
}

func TestHiddenBadgeOnConfiguredHiddenFolder(t *testing.T) {
	v := makeVault(t)
	hiddenDir := filepath.Join(v.Root, "SecretFolder")
	if err := os.MkdirAll(hiddenDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("hidden:\n  - SecretFolder\nediting:\n  enabled: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/SecretFolder", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	if !strings.Contains(body, `<span class="chip badge-hidden">Hidden</span>`) {
		t.Fatalf("configured hidden folder should show Hidden badge:\n%s", body)
	}
}

func TestPaletteActionStringsInJS(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_static/app.js", nil)
	s.ServeHTTP(w, r)
	js := w.Body.String()

	for _, action := range []string{
		"Edit current",
		"New here",
		"Rename current",
		"Move to Trash",
		"Open Trash",
	} {
		if !strings.Contains(js, action) {
			t.Fatalf("palette action %q should be defined in JS:\n%s", action, js)
		}
	}
	// Verify actions are dispatched, not navigated.
	if !strings.Contains(js, `item.kind === 'action'`) {
		t.Fatal("palette should dispatch action kind, not navigate")
	}
}

func TestEditModeClientAssetsExposePhase1Behavior(t *testing.T) {
	v := makeVault(t)
	s := NewServer(v, "", "")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_static/app.js", nil)
	s.ServeHTTP(w, r)
	js := w.Body.String()
	for _, want := range []string{
		"function initEditMode()",
		"function initNoteActionMenus()",
		"function positionNoteActionsMenu",
		"function closeNoteActionMenus",
		"function openEditMode(surface)",
		"function openCreateDialog(trigger)",
		"function openRenameDialog(trigger)",
		"function openMissingCreateDialog(trigger)",
		"/_api/edit/source/",
		"/_api/edit/preview",
		"/_api/edit/save",
		"/_api/edit/create",
		"/_api/edit/rename",
		"/_api/edit/missing-link-create",
		"Preview stale",
		"Unsaved changes",
		"Save conflict",
		"Copy draft",
		"Reload disk",
		"Create note",
		"Create folder",
		"Impact preview",
		"Hidden paths remain accessible by direct URL",
		"data-edit-new",
		"data-edit-rename",
		"data-edit-missing-create",
		"data-note-actions-toggle",
		"data-note-actions-menu",
		"beforeunload",
		"ev.key.toLowerCase() !== 'e'",
		"ev.key.toLowerCase() === 's'",
		"ev.key === 'Enter'",
	} {
		if !strings.Contains(js, want) {
			t.Fatalf("missing edit-mode JS %q in:\n%s", want, js)
		}
	}
	// Verify enhanceEditPreview does NOT call initDataviewTables.
	// Find the enhanceEditPreview function body and check it doesn't contain
	// initDataviewTables.
	funcStart := strings.Index(js, "function enhanceEditPreview")
	if funcStart < 0 {
		t.Fatal("enhanceEditPreview function not found in JS")
	}
	// Find the next function definition after enhanceEditPreview
	funcEnd := strings.Index(js[funcStart+len("function enhanceEditPreview"):], "\nfunction ")
	funcBody := js[funcStart : funcStart+len("function enhanceEditPreview")+funcEnd]
	if strings.Contains(funcBody, "initDataviewTables") {
		t.Fatal("enhanceEditPreview must NOT call initDataviewTables (static preview)")
	}
	// Verify modified-click guard mentions modifier keys.
	for _, want := range []string{"ev.button !== 0", "ev.metaKey", "ev.ctrlKey", "ev.shiftKey", "ev.altKey"} {
		if !strings.Contains(js, want) {
			t.Fatalf("modified-click guard missing %q in:\n%s", want, js)
		}
	}
	for _, want := range []string{"let editNavigationConfirmed = false", "editNavigationConfirmed = true", "if (editNavigationConfirmed) return"} {
		if !strings.Contains(js, want) {
			t.Fatalf("confirmed navigation guard missing %q in:\n%s", want, js)
		}
	}
	for _, want := range []string{"function trapEditModalFocus", "ev.key !== 'Tab'", "returnFocus.focus", "openEditModal('New', 'edit-create-modal', trigger)"} {
		if !strings.Contains(js, want) {
			t.Fatalf("modal focus trap missing %q in:\n%s", want, js)
		}
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_static/style.css", nil)
	s.ServeHTTP(w, r)
	css := w.Body.String()
	for _, want := range []string{
		".note.is-editing",
		".edit-workbench",
		".edit-tabs",
		".edit-stale-badge",
		".edit-source-panel textarea",
		".edit-message-conflict",
		".edit-modal",
		".edit-impact-group",
		".edit-template-snippet",
		".edit-modal-actions",
		".note-actions-toggle",
		".note-actions-menu",
		".note-actions-item",
		"@media(max-width:850px){body.edit-mode-active{overflow:hidden}",
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("missing edit-mode CSS %q in:\n%s", want, css)
		}
	}
}

func TestPhase3TrashClientAssetsExposeUIBehavior(t *testing.T) {
	v := makeVault(t)
	s := NewServer(v, "", "")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_static/app.js", nil)
	s.ServeHTTP(w, r)
	js := w.Body.String()
	for _, want := range []string{
		"function moveEditPathToTrash",
		"function restoreTrashEntry",
		"function restoreTrashEntryAs",
		"data-edit-trash",
		"data-trash-restore",
		"data-trash-restore-as",
		"/_api/edit/trash",
		"/_api/edit/trash/restore",
		"Move to Trash failed",
		"This folder is not empty. Empty it before moving it to Trash.",
		"requires_confirmation === 'restore_as'",
		"Restore as relative path:",
	} {
		if !strings.Contains(js, want) {
			t.Fatalf("missing Phase 3 trash JS %q in:\n%s", want, js)
		}
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_static/style.css", nil)
	s.ServeHTTP(w, r)
	css := w.Body.String()
	for _, want := range []string{
		".btn.danger",
		".trash-view",
		".trash-list",
		".trash-card",
		".trash-actions",
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("missing Phase 3 trash CSS %q in:\n%s", want, css)
		}
	}
}

func TestMobileSidebarOverlayBehavior(t *testing.T) {
	v := makeVault(t)
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	for _, want := range []string{
		`<button class="mobile-menu btn icon-btn" data-sidebar-toggle aria-label="Open sidebar" aria-expanded="false">☰</button>`,
		`<div class="sidebar-backdrop" data-sidebar-close></div>`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing mobile sidebar markup %q", want)
		}
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_static/style.css", nil)
	s.ServeHTTP(w, r)
	css := w.Body.String()
	for _, want := range []string{
		".mobile-menu{display:none}",
		"@media(max-width:850px){.shell{display:block}",
		".mobile-menu{display:inline-flex",
		".side{position:fixed;left:0;top:0;width:min(86vw,340px);max-width:340px;transform:translateX(-100%);z-index:30;height:100vh;",
		"body.sidebar-open .side{transform:translateX(0)}",
		"body.sidebar-open .sidebar-backdrop{display:block}",
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("missing mobile sidebar CSS %q in:\n%s", want, css)
		}
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_static/app.js", nil)
	s.ServeHTTP(w, r)
	js := w.Body.String()
	for _, want := range []string{
		"function openSidebar()",
		"function closeSidebar()",
		"data-sidebar-toggle",
		"data-sidebar-close",
		"side?.addEventListener('click'",
		"target.closest('a')",
		"closeSidebar();",
	} {
		if !strings.Contains(js, want) {
			t.Fatalf("missing mobile sidebar JS %q in:\n%s", want, js)
		}
	}
}

func TestThemeControlsSupportLightDarkSepiaAndAuto(t *testing.T) {
	v := makeVault(t)
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	for _, want := range []string{
		`<section class="settings-section">`,
		`<select data-theme-select aria-label="Theme">`,
		`<option value="auto">Auto</option>`,
		`<option value="light">Light</option>`,
		`<option value="dark">Dark</option>`,
		`<option value="sepia">Sepia</option>`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing theme control markup %q", want)
		}
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_static/style.css", nil)
	s.ServeHTTP(w, r)
	css := w.Body.String()
	for _, want := range []string{
		"[data-theme=dark]",
		"[data-theme=sepia]",
		"@media(prefers-color-scheme:dark)",
		".setting-row{display:grid",
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("missing theme CSS %q in:\n%s", want, css)
		}
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_static/app.js", nil)
	s.ServeHTTP(w, r)
	js := w.Body.String()
	for _, want := range []string{
		"const themeStorageKey = 'notes-web:theme'",
		"function applyTheme(theme)",
		"function initThemePicker()",
		"document.documentElement.dataset.theme",
		"data-theme-select",
	} {
		if !strings.Contains(js, want) {
			t.Fatalf("missing theme JS %q in:\n%s", want, js)
		}
	}
}

func TestVisualPolishFoundationAndPaletteStates(t *testing.T) {
	v := makeVault(t)
	s := NewServer(v, "", "")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_static/style.css", nil)
	s.ServeHTTP(w, r)
	css := w.Body.String()
	for _, want := range []string{
		"--surface:",
		"--surface-raised:",
		"--space-4:",
		"--radius-lg:",
		":focus-visible",
		".btn",
		".chip",
		".empty-state",
		".palette-item.is-selected",
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("missing visual foundation CSS %q in:\n%s", want, css)
		}
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_static/app.js", nil)
	s.ServeHTTP(w, r)
	js := w.Body.String()
	for _, want := range []string{
		"paletteSelectedIndex",
		"aria-selected",
		"Enter",
		"ArrowDown",
		"ArrowUp",
		"palette-shortcuts",
	} {
		if !strings.Contains(js, want) {
			t.Fatalf("missing polished palette JS %q in:\n%s", want, js)
		}
	}
}

func TestHomepageProjectFilterClientContract(t *testing.T) {
	v := makeVault(t)
	projectPath := filepath.Join(v.Root, filepath.FromSlash("Projects/Filterable.md"))
	if err := os.MkdirAll(filepath.Dir(projectPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(projectPath, []byte("---\nstatus: active\n---\n# Filterable\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewServer(v, "", "")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	for _, want := range []string{
		`data-home-block="active_projects"`,
		`data-home-project-filter`,
		`data-home-project-row`,
		`data-home-project-search-text`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing homepage project filter markup %q in:\n%s", want, body)
		}
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_static/app.js", nil)
	s.ServeHTTP(w, r)
	js := w.Body.String()
	for _, want := range []string{
		"function initHomepageProjectFilter()",
		"data-home-project-filter",
		"data-home-project-row",
		"dataset.homeProjectSearchText",
		"row.hidden = !visible",
		"initHomepageProjectFilter();",
	} {
		if !strings.Contains(js, want) {
			t.Fatalf("missing homepage project filter JS %q in:\n%s", want, js)
		}
	}
}

func TestSearchResultsAreRichAndEmptyStateIsHelpful(t *testing.T) {
	v := makeVault(t)
	s := NewServer(v, "", "")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_search?q=tag%3Adaily+target", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	for _, want := range []string{
		`class="search-result-card"`,
		`class="result-title"`,
		`class="result-path"`,
		`class="result-snippet"`,
		`Search syntax`,
		`Examples:`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing rich search result markup %q in:\n%s", want, body)
		}
	}
	if strings.Contains(body, `Daily Briefing.md:0</a>`) || strings.Contains(body, `:0</a>`) {
		t.Fatalf("search results should not expose raw :0 line suffixes:\n%s", body)
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_search?q=not-a-real-query", nil)
	s.ServeHTTP(w, r)
	body = w.Body.String()
	for _, want := range []string{`class="empty-state"`, `No results for`, `Try a broader search`} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing helpful empty state %q in:\n%s", want, body)
		}
	}
}

func TestTODOPageUsesCountersStructuredRowsAndCollapsedDone(t *testing.T) {
	v := makeVault(t)
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_todo?today=2026-05-20", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	for _, want := range []string{
		`class="todo-shell"`,
		`class="todo-toolbar"`,
		`class="todo-section overdue"`,
		`<h2 id="todo-overdue">Overdue</h2><span class="count">1</span>`,
		`class="task-row"`,
		`class="task-checkbox"`,
		`class="task-due overdue-date"`,
		`class="task-menu" data-task-menu`,
		`data-copy="todo done `,
		`Mark as done`,
		`Copy todo ID`,
		`data-todo-hide-done`,
		`<section class="todo-section done"`,
		`<h2 id="todo-done">Done</h2><span class="count">1</span>`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing TODO polish markup %q in:\n%s", want, body)
		}
	}
	for _, forbidden := range []string{`td promote `, `td demote `, `td reschedule `, `Promote`, `Demote`, `Re-schedule`} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("TODO dropdown should only expose mark-done and copy-id actions, found %q in:\n%s", forbidden, body)
		}
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_static/app.js", nil)
	s.ServeHTTP(w, r)
	js := w.Body.String()
	for _, want := range []string{
		"function positionTodoDropdown(menu, dropdown)",
		"dropdown.style.position = 'fixed'",
		"window.innerHeight",
		"Math.min(window.innerHeight - dropdownRect.height - 8, Math.max(8, buttonRect.top - dropdownRect.height - 6))",
		"positionTodoDropdown(menu, dropdown)",
	} {
		if !strings.Contains(js, want) {
			t.Fatalf("TODO dropdown must be viewport-positioned to avoid clipping, missing %q in:\n%s", want, js)
		}
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_static/style.css", nil)
	s.ServeHTTP(w, r)
	css := w.Body.String()
	for _, want := range []string{
		".task-menu-dropdown{position:fixed;",
		"z-index:40",
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("TODO dropdown must escape clipped task containers, missing CSS %q in:\n%s", want, css)
		}
	}
}

func TestTagsPageHasWorkingFilterMarkupWithoutMetricCards(t *testing.T) {
	v := makeVault(t)
	// Create enough one-off tags to prove rare tags are still available without metric cards.
	for i := 0; i < 8; i++ {
		p := filepath.Join(v.Root, "Areas", "Tags", "Rare"+string(rune('A'+i))+".md")
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("---\ntags: [rare"+string(rune('a'+i))+"]\n---\n# Rare\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_tags", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	for _, want := range []string{
		`class="tag-controls"`,
		`placeholder="Filter tags…"`,
		`aria-label="Filter tags"`,
		`Popular tags`,
		`Alphabetical index`,
		`Rare tags`,
		`data-tag-filter`,
		`data-tag-name=`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing tag filter markup %q in:\n%s", want, body)
		}
	}
	for _, unwanted := range []string{`class="tag-stats"`, `total tags`, `one-off tags`} {
		if strings.Contains(body, unwanted) {
			t.Fatalf("tags page should not contain metric card %q:\n%s", unwanted, body)
		}
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_static/app.js", nil)
	s.ServeHTTP(w, r)
	js := w.Body.String()
	for _, want := range []string{
		"function initTagFilter()",
		"data-tag-name",
		"closest('.tag-letter')",
		"tagFilter.addEventListener('input'",
	} {
		if !strings.Contains(js, want) {
			t.Fatalf("missing robust tag filter JS %q in:\n%s", want, js)
		}
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_static/style.css", nil)
	s.ServeHTTP(w, r)
	css := w.Body.String()
	if !strings.Contains(css, "[hidden]{display:none!important}") {
		t.Fatalf("hidden filtered tags must beat chip display styles, missing [hidden] override in CSS:\n%s", css)
	}
}

func TestSidebarPaletteAndNoFlashContracts(t *testing.T) {
	v := makeVault(t)
	s := NewServer(v, "", "")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	for _, want := range []string{
		`<script>try{`,
		`notes-web:reading-focus`,
		`document.documentElement.dataset.readingFocus`,
		`data-palette-open aria-label="Open command palette">⌘K</button>`,
		`<dt>⌘/Ctrl B</dt><dd>Toggle sidebar</dd>`,
		`<a class="nav" href="/_search"><span class="nav-icon nav-icon-search">⌕</span>Search</a>`,
		`<div class="side-header"><div><a class="brand" href="/">Notes Web</a><span class="side-subtitle">Personal knowledge base</span></div><button class="settings-button btn ghost icon-btn" data-settings-open aria-haspopup="dialog" aria-label="Settings"><span aria-hidden="true">⚙</span></button></div>`,
		`<span class="nav-icon nav-icon-favorite">★</span>Daily Briefings`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing no-flash/sidebar markup %q in:\n%s", want, body)
		}
	}
	for _, unwanted := range []string{`class="search sidebar-search"`, `<dt>⌘/Ctrl F</dt><dd>Toggle reading focus</dd>`, `class="sidebar-footer"`, `<span>Settings</span>`} {
		if strings.Contains(body, unwanted) {
			t.Fatalf("layout should not contain obsolete sidebar/search markup %q:\n%s", unwanted, body)
		}
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_static/app.js", nil)
	s.ServeHTTP(w, r)
	js := w.Body.String()
	for _, want := range []string{
		"ev.key.toLowerCase() === 'b'",
		"toggleReadingFocus()",
		"applyInitialPreferences()",
	} {
		if !strings.Contains(js, want) {
			t.Fatalf("missing shortcut/no-flash JS %q in:\n%s", want, js)
		}
	}
	if strings.Contains(js, "ev.key.toLowerCase() === 'f'") {
		t.Fatalf("Cmd/Ctrl+F must remain available for browser find, not sidebar toggling:\n%s", js)
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_static/style.css", nil)
	s.ServeHTTP(w, r)
	css := w.Body.String()
	for _, want := range []string{
		"font-family:ui-sans-serif,system-ui,-apple-system,Segoe UI,Roboto,sans-serif",
		".palette-button{position:fixed;right:18px;bottom:18px",
		".side{position:sticky;top:0;height:100vh;overflow:auto",
		".nav-icon{width:32px;font-size:24px",
		".nav-icon-search{font-size:29px",
		".nav-icon-favorite{font-size:21px",
		".settings-button{inline-size:52px;block-size:52px;padding:0;border-radius:999px;flex:0 0 auto;font-size:28px",
		":root[data-reading-focus=\"true\"] .side{display:none}",
		":root[data-reading-focus=\"true\"] .shell{grid-template-columns:minmax(0,1fr)}",
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("missing sidebar/no-flash CSS %q in:\n%s", want, css)
		}
	}
	for _, unwanted := range []string{
		".side{display:flex;flex-direction:column;overflow:hidden",
		".side>section:last-of-type{flex:1;min-height:0;overflow:auto",
		".palette-button{top:18px;right:18px;bottom:auto}",
	} {
		if strings.Contains(css, unwanted) {
			t.Fatalf("CSS should not contain obsolete override %q:\n%s", unwanted, css)
		}
	}
}

func TestDiagnosticsPagesGroupAndLimitMassiveLists(t *testing.T) {
	v := makeVault(t)
	p := filepath.Join(v.Root, "Areas", "ManyBroken.md")
	if err := os.WriteFile(p, []byte("# Many\n[[Missing]]\n[[OtherMissing]]\n[[Missing]]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_broken-links", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	for _, want := range []string{
		`class="diagnostic-summary"`,
		`distinct targets`,
		`class="broken-link-group"`,
		`<summary><strong>Missing</strong>`,
		`occurrences`,
		`Show top`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing grouped broken-link markup %q in:\n%s", want, body)
		}
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_orphans", nil)
	s.ServeHTTP(w, r)
	body = w.Body.String()
	for _, want := range []string{`class="diagnostic-summary"`, `class="filter-bar"`, `class="note-card-grid"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing orphan diagnostics markup %q in:\n%s", want, body)
		}
	}
}

func TestUISmokeScriptExists(t *testing.T) {
	if _, err := os.Stat(filepath.Join("..", "..", "scripts", "ui-smoke.sh")); err != nil {
		t.Fatalf("expected UI smoke script: %v", err)
	}
}

func TestSidebarHighlightsActiveNoteAndOpensContainingFolders(t *testing.T) {
	v := makeVault(t)
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Areas/Target.md", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	for _, want := range []string{
		`<details class="tree-folder active-branch" data-tree-path="Areas" open>`,
		`<a class="active" aria-current="page" href="/Areas/Target.md"><span aria-hidden="true">📄</span> Target.md</a>`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing active sidebar markup %q in:\n%s", want, body)
		}
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_static/style.css", nil)
	s.ServeHTTP(w, r)
	css := w.Body.String()
	for _, want := range []string{
		".tree a.active",
		".tree-folder.active-branch>summary",
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("missing active sidebar CSS %q in:\n%s", want, css)
		}
	}
}
