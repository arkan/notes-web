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
	for _, want := range []string{`<details class="tree-folder" data-tree-path="Areas">`, `<summary><span aria-hidden="true">📁</span> Areas</summary>`, `data-copy-link`} {
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
	for _, want := range []string{"copyText", "navigator.clipboard", "execCommand", "data-copy", "data-copy-code", "closest('pre')", "notes-web:sidebar-open", "data-tree-path", "localStorage"} {
		if !strings.Contains(js, want) {
			t.Fatalf("missing %q in app.js:\n%s", want, js)
		}
	}
}

func TestSidebarShowsDailyNoteFilesNestedByYearAndMonth(t *testing.T) {
	v := makeVault(t)
	dailyRel := "Daily Notes/2026/2026-05/2026-05-23.md"
	dailyPath := filepath.Join(v.Root, filepath.FromSlash(dailyRel))
	if err := os.MkdirAll(filepath.Dir(dailyPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dailyPath, []byte("# Daily note\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()

	for _, want := range []string{
		`data-tree-path="Daily Notes"`,
		`data-tree-path="Daily Notes/2026"`,
		`data-tree-path="Daily Notes/2026/2026-05"`,
		`href="/Daily%20Notes/2026/2026-05/2026-05-23.md"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("sidebar should include nested daily note entry %q in:\n%s", want, body)
		}
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
