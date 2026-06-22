package app

import (
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
	if !strings.Contains(js, `ev.key === 'Tab' && document.querySelector('[data-note-actions-toggle][aria-expanded="true"]')`) {
		t.Fatalf("note actions menu should close on Tab even before async menu focus settles:\n%s", js)
	}
}

func sidebarVaultHTML(t *testing.T, body string) string {
	t.Helper()
	start := strings.Index(body, `<section><h3>Vault</h3>`)
	if start < 0 {
		t.Fatalf("missing sidebar vault section in:\n%s", body)
	}
	end := strings.Index(body[start:], `</section>`)
	if end < 0 {
		t.Fatalf("missing sidebar vault section end in:\n%s", body)
	}
	return body[start : start+end+len(`</section>`)]
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

func TestModernWorkbenchPhase1ShellAndTokens(t *testing.T) {
	v := makeVault(t)
	s := NewServer(v, "", "")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Areas/Target.md", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	for _, want := range []string{
		`<div class="shell" data-workbench-shell>`,
		`<aside class="side" aria-label="Vault navigation">`,
		`<div class="workbench-center">`,
		`<main class="main" id="main-content" tabindex="-1">`,
		`<aside class="context-pane" id="workbench-context-pane" data-context-pane aria-label="Workbench context">`,
		`Read-only context only. Page actions stay in the main content during this phase.`,
		`<article class="note reading-surface"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing Modern Workbench shell markup %q in:\n%s", want, body)
		}
	}
	for _, unwanted := range []string{`href="/_inbox"`, `href="/_settings"`} {
		if strings.Contains(body, unwanted) {
			t.Fatalf("future route should stay hidden in Phase 1, found %q in:\n%s", unwanted, body)
		}
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_static/style.css", nil)
	s.ServeHTTP(w, r)
	css := w.Body.String()
	for _, want := range []string{
		"--bg:#07090f",
		"--accent:#2f7cff",
		"--pane-left-width:248px",
		"--pane-right-width:318px",
		".shell{display:grid;grid-template-columns:var(--pane-left-width) minmax(0,1fr) var(--pane-right-width);",
		".workbench-center{min-width:0;min-height:0;height:100%;overflow:hidden;background:var(--bg-elevated)}",
		".context-pane{min-width:0;height:100%;min-height:0;overflow:auto;",
		"@media(max-width:1120px){body{overflow:auto}",
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("missing Modern Workbench CSS %q in:\n%s", want, css)
		}
	}
}

func TestModernWorkbenchPhase2NavigationContextAndRightPaneContracts(t *testing.T) {
	v := makeVault(t)
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()

	appNav := extractBetween(t, body, `aria-label="App navigation"`, `</nav>`)
	assertOrdered(t, appNav, []string{`href="/"`, `href="/_todo"`, `href="/_projects"`, `href="/_calendar"`, `href="/_search"`, `href="/_tags"`, `href="/_maintenance"`})
	for _, forbidden := range []string{`href="/_inbox"`, `href="/_settings"`, `href="/_trash"`} {
		if strings.Contains(appNav, forbidden) {
			t.Fatalf("app nav should not expose %q when editing is disabled:\n%s", forbidden, appNav)
		}
	}
	if strings.Contains(body, `data-global-edit-context`) {
		t.Fatalf("palette edit context should not render when editing is disabled:\n%s", body)
	}
	for _, want := range []string{
		`data-right-pane-toggle-primary aria-controls="workbench-context-pane" aria-expanded="true"`,
		`data-right-pane-select aria-label="Context pane"`,
		`notes-web:right-pane-open`,
		`<h3 id="context-current-heading">Current page</h3>`,
		`<h3 id="context-links-heading">Quick links</h3>`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing Phase 2 layout contract %q in:\n%s", want, body)
		}
	}
	contextPane := extractBetween(t, body, `<aside class="context-pane"`, `</aside>`)
	for _, forbidden := range []string{`data-edit-open`, `data-edit-new`, `data-edit-rename`, `data-edit-trash`, `data-trash-restore`, `Restore as`} {
		if strings.Contains(contextPane, forbidden) {
			t.Fatalf("context pane must stay read-only; found %q in:\n%s", forbidden, contextPane)
		}
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_search", nil)
	s.ServeHTTP(w, r)
	body = w.Body.String()
	if !strings.Contains(body, `href="/_search" aria-current="page"`) {
		t.Fatalf("search app nav item should be current:\n%s", body)
	}

	if err := os.WriteFile(filepath.Join(v.Root, "Search.md"), []byte("# Search\n\nUser note named like an app route.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/Search.md", nil)
	s.ServeHTTP(w, r)
	body = w.Body.String()
	appNav = extractBetween(t, body, `aria-label="App navigation"`, `</nav>`)
	if strings.Contains(appNav, `href="/_search" aria-current="page"`) {
		t.Fatalf("app nav active state must use route, not user title:\n%s", appNav)
	}

	enableEditing(t, v)
	s = NewServer(v, "", "")
	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/", nil)
	s.ServeHTTP(w, r)
	body = w.Body.String()
	appNav = extractBetween(t, body, `aria-label="App navigation"`, `</nav>`)
	assertOrdered(t, appNav, []string{`href="/"`, `href="/_inbox"`, `href="/_todo"`, `href="/_calendar"`, `href="/_search"`, `href="/_tags"`, `href="/_maintenance"`, `href="/_trash"`})
	if !strings.Contains(body, `data-global-edit-context data-edit-csrf="`) {
		t.Fatalf("palette should get global edit context only when editing is enabled:\n%s", body)
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_static/app.js", nil)
	s.ServeHTTP(w, r)
	js := w.Body.String()
	for _, want := range []string{
		"const rightPaneStorageKey = 'notes-web:right-pane-open'",
		"function initRightPaneControls()",
		"pane.toggleAttribute('inert', !visible)",
		"focusWasInside",
		"primaryToggle.focus()",
		"function setModalIsolation(active)",
		"el.dataset.modalIsolated = 'true'",
		"setSidebarA11y(document.body.classList.contains('sidebar-open'))",
		"(max-width: 1120px)",
		"function trapSettingsFocus(ev)",
		"'Hide context pane'",
		"'Show context pane'",
		"(min-width: 1121px)",
	} {
		if !strings.Contains(js, want) {
			t.Fatalf("missing right-pane JS contract %q in:\n%s", want, js)
		}
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_static/style.css", nil)
	s.ServeHTTP(w, r)
	css := w.Body.String()
	for _, want := range []string{
		`:root[data-right-pane-open="false"] .shell`,
		`.shell.right-pane-collapsed`,
		`.context-module`,
		`.workbench-topbar`,
		`@media(max-width:1120px){.workbench-center`,
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("missing right-pane/context CSS %q in:\n%s", want, css)
		}
	}
}

func TestModernWorkbenchPhase3ANoteReadingContracts(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Areas/Target.md", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	for _, want := range []string{
		`<section class="note-page note-reading-stack" aria-label="Note">`,
		`<nav class="crumb reading-surface" aria-label="Breadcrumb">`,
		`<article class="note reading-surface"`,
		`<header class="note-header">`,
		`<div class="note-title-block">`,
		`<div class="note-actions" data-note-actions>`,
		`data-note-actions-toggle aria-haspopup="menu" aria-expanded="false" aria-label="Actions">⚙</button>`,
		`data-edit-open>Edit</button>`,
		`data-edit-new>New</button>`,
		`data-edit-rename>Rename</button>`,
		`data-edit-trash data-edit-trash-kind="note">Move to Trash</button>`,
		`<button class="note-actions-item copy-link" role="menuitem" type="button" data-copy-path>Copy path</button>`,
		`<div class="content">`,
		`class="link-panel forward-links"`,
		`class="link-panel backlinks"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing Phase 3A note contract %q in:\n%s", want, body)
		}
	}
	contextPane := extractBetween(t, body, `<aside class="context-pane"`, `</aside>`)
	for _, forbidden := range []string{`data-edit-open`, `data-edit-new`, `data-edit-rename`, `data-edit-trash`, `data-trash-restore`, `Move to Trash`, `Restore as`} {
		if strings.Contains(contextPane, forbidden) {
			t.Fatalf("context pane must stay read-only during Phase 3A; found %q in:\n%s", forbidden, contextPane)
		}
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_static/style.css", nil)
	s.ServeHTTP(w, r)
	css := w.Body.String()
	for _, want := range []string{
		".note-page{width:100%;max-width:840px;margin:0 auto;display:grid;gap:14px;align-content:start}",
		".note-page>.note{display:grid;grid-template-columns:minmax(0,1fr);gap:22px;",
		".note .note-header{display:grid;grid-template-columns:minmax(0,1fr) auto;",
		".note-page .content{color:var(--ink);font-size:17px;line-height:1.74;overflow-wrap:anywhere}",
		".note .frontmatter{margin:1.4rem 0;padding:13px 14px;",
		".note-page>.toc,.note-page>.link-panel{padding:14px 15px;",
		".note-page>.toc ul{columns:2;",
		"@media(max-width:640px){.note-page{width:100%;gap:10px}",
		".note-page .content :where(h1,h2,h3,h4,h5,h6,[id]){scroll-margin-top:32px}",
		"@media(max-width:1120px){.note-page .content :where(h1,h2,h3,h4,h5,h6,[id]){scroll-margin-top:76px}}",
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("missing Phase 3A note CSS %q in:\n%s", want, css)
		}
	}
}

func TestModernWorkbenchPhase3BHomeCockpitContracts(t *testing.T) {
	v := makeVault(t)
	s := NewServer(v, "", "")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	for _, want := range []string{
		`<div class="page-header home-header home-cockpit-header">`,
		`<p class="eyebrow">Daily cockpit</p>`,
		`<p class="home-header-summary">Start with the daily note, clear urgent work, then scan supporting context.</p>`,
		`<div class="home-dashboard" data-home-dashboard>`,
		`class="home-block home-block-today today-card card"`,
		`<p class="eyebrow">Today</p>`,
		`class="home-due-now-summary" aria-label="Urgent task summary"`,
		`>Review tasks</a>`,
		`class="home-block home-block-todos open-todos card"`,
		`<p class="eyebrow">Urgent tasks</p>`,
		`<p class="home-block-intent">Overdue and today tasks stay in the main flow.</p>`,
		`data-home-order=`,
		`--home-block-order:`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing Phase 3B home cockpit markup %q in:\n%s", want, body)
		}
	}
	for _, forbidden := range []string{`Quick capture`, `quick capture`, `data-quick-capture`, `placeholder="Capture`, `data-capture`} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("Phase 3B must not render Quick capture UI; found %q in:\n%s", forbidden, body)
		}
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_static/style.css", nil)
	s.ServeHTTP(w, r)
	css := w.Body.String()
	for _, want := range []string{
		".home-cockpit-header{max-width:1040px",
		".home-dashboard{max-width:900px;grid-template-columns:1fr;gap:16px}",
		".home-main,.home-side{display:grid;gap:16px;align-content:start}",
		".home-block-today{position:relative;overflow:hidden;padding:clamp",
		".home-today-preview{max-height:18rem}",
		".home-due-now-summary{display:flex;flex-wrap:wrap;align-items:center;gap:8px",
		".home-block-today h2{font-size:clamp",
		".home-block-todos{padding:18px",
		".home-todos .todo-section.overdue{border-color:color-mix(in srgb,var(--danger),transparent 58%)}",
		"@media(max-width:1280px){.home-dashboard{max-width:900px;grid-template-columns:1fr}",
		".home-block{order:var(--home-block-order,0)}",
		"@media(max-width:640px){.home-cockpit-header{display:grid;gap:12px",
		"@media(max-width:640px){.home-main,.home-side{display:contents}.home-today-preview{max-height:12rem;overflow:auto;padding:13px}}",
		"@media(max-width:640px){.home-due-now-summary{gap:7px;margin-bottom:12px}",
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("missing Phase 3B home cockpit CSS %q in:\n%s", want, css)
		}
	}
}

func TestModernWorkbenchPhase4BInboxUIContracts(t *testing.T) {
	v := makeVault(t)
	s := NewServer(v, "", "")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_inbox", nil)
	s.ServeHTTP(w, r)
	if w.Code != 404 {
		t.Fatalf("Inbox route should be hidden when editing is disabled, got %d", w.Code)
	}

	vHidden := makeVault(t)
	if err := os.WriteFile(filepath.Join(vHidden.Root, ".notes-web.yaml"), []byte("editing:\n  enabled: true\nhidden:\n  - Inbox\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sHidden := NewServer(vHidden, "", "")
	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/", nil)
	sHidden.ServeHTTP(w, r)
	if strings.Contains(w.Body.String(), `href="/_inbox"`) || strings.Contains(w.Body.String(), `data-quick-capture`) {
		t.Fatalf("Inbox nav and Quick capture should stay hidden when Inbox is hidden:\n%s", w.Body.String())
	}
	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_inbox", nil)
	sHidden.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatalf("hidden Inbox route should be blocked with 403, got %d", w.Code)
	}

	enableEditing(t, v)
	if err := os.MkdirAll(filepath.Join(v.Root, "Tasks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(v.Root, "Tasks", "Inbox.md"), []byte("# Inbox tasks\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("editing:\n  enabled: true\ntodo:\n  todo_file: Tasks/Inbox.md\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeInboxCapture(t, v, "Inbox/static-contract.md", "Static Contract")
	s = NewServer(v, "", "")

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	for _, want := range []string{
		`href="/_inbox"`,
		`<section class="home-quick-capture card" data-quick-capture data-edit-csrf="`,
		`<h2 id="home-quick-capture-title">Quick capture</h2>`,
		`data-quick-capture-input`,
		`data-quick-capture-submit>Save to Inbox</button>`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing Phase 4B Home/Inbox contract %q in:\n%s", want, body)
		}
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_inbox", nil)
	s.ServeHTTP(w, r)
	body = w.Body.String()
	for _, want := range []string{
		`<section class="inbox-view app-page" data-inbox-context data-edit-csrf="`,
		`<h1>Inbox</h1>`,
		`data-inbox-count`,
		`data-inbox-entry data-inbox-path="Inbox/static-contract.md"`,
		`data-inbox-archive`,
		`data-inbox-move`,
		`data-inbox-convert`,
		`Static Contract`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing Phase 4B Inbox contract %q in:\n%s", want, body)
		}
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_static/app.js", nil)
	s.ServeHTTP(w, r)
	js := w.Body.String()
	for _, want := range []string{
		"function initInboxUI()",
		"function initQuickCapture()",
		"/_api/edit/capture",
		"/_api/edit/inbox/archive",
		"/_api/edit/inbox/move",
		"/_api/edit/inbox/convert-task",
		"confirm_missing_dirs",
		"confirm_hidden",
		"form.requestSubmit",
	} {
		if !strings.Contains(js, want) {
			t.Fatalf("missing Phase 4B JS contract %q in:\n%s", want, js)
		}
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_static/style.css", nil)
	s.ServeHTTP(w, r)
	css := w.Body.String()
	for _, want := range []string{
		".home-quick-capture{display:grid;grid-template-columns:minmax(0,.8fr) minmax(0,1.2fr)",
		".home-quick-capture textarea{width:100%;min-height:92px",
		".app-page{width:100%;max-width:1120px;margin:0 auto;display:grid;gap:18px",
		".trash-view.app-page,.inbox-view.app-page{max-width:1120px}",
		".inbox-card{display:grid;grid-template-columns:minmax(0,1fr) auto",
		"@media(max-width:1360px){.home-quick-capture,.inbox-card{grid-template-columns:1fr}",
		"@media(max-width:760px){.home-quick-capture,.inbox-card{grid-template-columns:1fr}",
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("missing Phase 4B CSS contract %q in:\n%s", want, css)
		}
	}
}

func TestModernWorkbenchPhase5aSurfaceContracts(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_maintenance", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	for _, want := range []string{`<h1>Maintenance</h1>`, `href="/_broken-links"`, `href="/_orphans"`, `href="/_dataview"`, `href="/_trash"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing maintenance contract %q in:\n%s", want, body)
		}
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_static/style.css", nil)
	s.ServeHTTP(w, r)
	css := w.Body.String()
	for _, want := range []string{
		".search-page-form{display:grid;grid-template-columns:minmax(0,1fr) auto",
		".rich-results{max-width:900px;margin:16px auto 0",
		".maintenance-grid{display:grid;gap:16px;max-width:920px",
		".maintenance-cards{display:grid;grid-template-columns:repeat(2,minmax(0,1fr))",
		".trash-card{display:grid;grid-template-columns:minmax(0,1fr) auto",
		"@media(max-width:760px){.search-page-form{grid-template-columns:1fr",
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("missing Phase 5a CSS contract %q in:\n%s", want, css)
		}
	}
}

func TestInboxDisabledForSymlinkInbox(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	if err := os.RemoveAll(filepath.Join(v.Root, "Inbox")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/tmp/nonexistent-inbox", filepath.Join(v.Root, "Inbox")); err != nil {
		t.Fatal(err)
	}
	s := NewServer(v, "", "")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	if strings.Contains(body, `href="/_inbox"`) || strings.Contains(body, `data-quick-capture`) {
		t.Fatalf("symlink Inbox should disable nav and Quick capture:\n%s", body)
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_inbox", nil)
	s.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatalf("symlink Inbox route should be forbidden, got %d: %s", w.Code, w.Body.String())
	}
}

func extractBetween(t *testing.T, body, startToken, endToken string) string {
	t.Helper()
	start := strings.Index(body, startToken)
	if start < 0 {
		t.Fatalf("missing start token %q in:\n%s", startToken, body)
	}
	end := strings.Index(body[start:], endToken)
	if end < 0 {
		t.Fatalf("missing end token %q after %q in:\n%s", endToken, startToken, body[start:])
	}
	return body[start : start+end]
}

func assertOrdered(t *testing.T, body string, tokens []string) {
	t.Helper()
	last := -1
	for _, token := range tokens {
		idx := strings.Index(body, token)
		if idx < 0 {
			t.Fatalf("missing ordered token %q in:\n%s", token, body)
		}
		if idx <= last {
			t.Fatalf("token %q was not after previous token in:\n%s", token, body)
		}
		last = idx
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
		".note-page .content img,.edit-preview-panel .content img{max-width:100%;height:auto;border:0;border-radius:12px;background:transparent;box-shadow:none}",
		".media-placeholder{display:grid;grid-template-columns:34px minmax(0,1fr)",
		".media-placeholder-icon{display:grid;place-items:center;inline-size:34px;block-size:34px",
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
		`<button class="palette-button btn" data-palette-open aria-haspopup="dialog" aria-expanded="false" aria-label="Open command palette">⌘K</button>`,
		`<button class="settings-fab btn icon-btn" data-settings-open aria-haspopup="dialog" aria-label="Settings">`,
		`<div class="palette" data-palette hidden>`,
		`<div class="palette-search-row">`,
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
		"function trapPaletteFocus(ev)",
		"/_api/palette",
		"metaKey",
		"ev.key === '/'",
		"aria-activedescendant",
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
	if strings.Contains(js, "button.addEventListener('mouseenter'") {
		t.Fatalf("palette result hover interactions should be delegated; per-button listeners are unstable when hover rerenders:\n%s", js)
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
		`class="folder-view folder-surface" data-edit-context data-edit-kind="folder" data-edit-directory="Areas" data-edit-path="Areas" data-edit-csrf="`,
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
		`class="trash-view app-page" data-trash-context data-edit-csrf="`,
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
		`<select data-density-select aria-label="Density">`,
		`<option value="compact">Compact</option>`,
		`<option value="comfortable">Comfortable</option>`,
		`<select data-reading-focus-select aria-label="Reading focus">`,
		`<option value="off">Off</option>`,
		`<option value="on">On</option>`,
		`data-palette-recent-clear`,
		`data-palette-recent-status`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing theme/prefs control markup %q", want)
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
		"[data-density=\"comfortable\"]",
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
		`notes-web:right-pane-open`,
		`notes-web:density`,
		`document.documentElement.dataset.readingFocus`,
		`data-palette-open aria-haspopup="dialog" aria-expanded="false" aria-label="Open command palette">⌘K</button>`,
		`<dt>⌘/Ctrl B</dt><dd>Toggle reading focus</dd>`,
		`aria-label="App navigation"`,
		`<span class="nav-icon nav-icon-search" aria-hidden="true">⌕</span>Search</a>`,
		`<div class="side-header">`,
		`<span class="nav-icon nav-icon-favorite" aria-hidden="true">★</span>Daily Briefings`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing no-flash/sidebar markup %q in:\n%s", want, body)
		}
	}
	for _, unwanted := range []string{`class="search sidebar-search"`, `<dt>⌘/Ctrl F</dt><dd>Toggle reading focus</dd>`, `<dt>⌘/Ctrl B</dt><dd>Toggle sidebar</dd>`, `class="sidebar-footer"`, `<span>Settings</span>`} {
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
		"function applyDensity(density)",
		"function initDensityPicker()",
		"densityStorageKey",
		"paletteRecentStorageKey",
		"const nonRecentMatches = matches.filter",
		"function readPaletteRecents",
		"function readCurrentPaletteRecents",
		"currentServerPaletteByURL",
		"function recordPaletteRecent",
		"function clearPaletteRecents",
		"function renderPaletteItem",
		"palette-recent-header",
		"data-palette-recent-clear",
		"data-reading-focus-select",
		"removeAttribute('data-reading-focus')",
		"function syncReadingFocusSelect",
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
		".settings-fab{position:fixed;left:18px;bottom:18px",
		":root[data-reading-focus=\"true\"] .palette-button,body.reading-focus .palette-button{display:inline-flex!important}",
		":root[data-reading-focus=\"true\"] .mobile-menu,body.reading-focus .mobile-menu{display:none!important}",
		".palette-recent-header{display:flex;align-items:center;gap:8px",
		".setting-row-end{display:inline-flex;align-items:center;justify-content:flex-end;gap:10px",
		"[data-density=\"comfortable\"] .card,[data-density=\"comfortable\"] .note-card",
		"[data-density=\"comfortable\"] .home-block-today{padding:clamp(30px,3.2vw,34px)}",
		"[data-density=\"comfortable\"] .home-side .home-block,[data-density=\"comfortable\"] .recent-notes{padding:24px!important}",
		"[data-density=\"comfortable\"] .calendar-page .calendar-day{min-height:76px}",
		".dataview-diagnostic pre,.dataview-error pre{max-width:100%;overflow:auto;white-space:pre-wrap",
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

// ---------------------------------------------------------------------------
// Phase 5a: Maintenance page and nav
// ---------------------------------------------------------------------------

func TestMaintenancePageRenders(t *testing.T) {
	v := makeVault(t)
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_maintenance", nil)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("maintenance page: expected 200, got %d", w.Code)
	}
	body := w.Body.String()

	for _, want := range []string{
		`<h1>Maintenance</h1>`,
		`card maintenance-section`,
		`/_maintenance`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("maintenance page missing %q in:\n%s", want, body)
		}
	}
	// Links section must exist and link to diagnostic detail pages.
	for _, link := range []string{`href="/_broken-links"`, `href="/_orphans"`, `href="/_dataview"`} {
		if !strings.Contains(body, link) {
			t.Fatalf("maintenance page missing link %q in:\n%s", link, body)
		}
	}
}

func TestMaintenancePageIncludesDiagnosticCounts(t *testing.T) {
	v := makeVault(t)
	// Create a note with a broken wikilink.
	createNote := func(rel, content string) {
		p := filepath.Join(v.Root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	createNote("Areas/WithLink.md", "# With Link\n\nSee [[MissingTarget]] and [[AnotherMissing]].\n")
	createNote("Areas/Orphan.md", "# Orphan\n\nNo incoming links.\n")

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_maintenance", nil)
	s.ServeHTTP(w, r)

	body := w.Body.String()
	data := s.buildMaintenanceData()

	if !strings.Contains(body, `href="/_broken-links"`) {
		t.Fatal("maintenance should link to broken links")
	}
	if !strings.Contains(body, `href="/_orphans"`) {
		t.Fatal("maintenance should link to orphans")
	}
	for _, want := range []string{
		fmt.Sprintf("%d occurrence", data.BrokenTotal),
		fmt.Sprintf("%d note", data.BrokenAffectedNotes),
		fmt.Sprintf("%d distinct target", data.BrokenDistinctTargets),
		fmt.Sprintf("%d orphan", data.OrphanTotal),
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("maintenance count parity missing %q in:\n%s", want, body)
		}
	}
}

func TestMaintenancePageDoesNotExposePaths(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)

	// Create hidden, template, and trash entries.
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("editing:\n  enabled: true\nhidden:\n  - Hidden\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	hiddenDir := filepath.Join(v.Root, "Hidden")
	if err := os.MkdirAll(hiddenDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hiddenDir, "Secret.md"), []byte("# Secret\n\n[[HiddenMissing]]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(v.Root, "_template.md"), []byte("# Root template\n\n[[TemplateMissing]]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	trashRoot := filepath.Join(v.Root, "_trash")
	if err := os.MkdirAll(trashRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(trashRoot, "TrashedNote.md"), []byte("# Trashed\n\n[[TrashMissing]]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_maintenance", nil)
	s.ServeHTTP(w, r)

	body := w.Body.String()
	for _, unwanted := range []string{
		"Secret.md",
		"HiddenMissing",
		"_template.md",
		"TemplateMissing",
		"TrashedNote.md",
		"TrashMissing",
		"OriginalPath",
		"original_path",
		".notes-web-trash.json",
	} {
		if strings.Contains(body, unwanted) {
			t.Fatalf("maintenance must not expose %q in:\n%s", unwanted, body)
		}
	}
}

func TestMaintenanceNavAppearsWhenEditingDisabled(t *testing.T) {
	v := makeVault(t)
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()

	appNav := extractBetween(t, body, `aria-label="App navigation"`, `</nav>`)
	assertOrdered(t, appNav, []string{`href="/"`, `href="/_todo"`, `href="/_projects"`, `href="/_calendar"`, `href="/_search"`, `href="/_tags"`, `href="/_maintenance"`})

	for _, forbid := range []string{`href="/_inbox"`, `href="/_trash"`} {
		if strings.Contains(appNav, forbid) {
			t.Fatalf("nav should not contain %q when editing disabled", forbid)
		}
	}
}

func TestMaintenanceNavAppearsWhenEditingEnabled(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()

	appNav := extractBetween(t, body, `aria-label="App navigation"`, `</nav>`)
	assertOrdered(t, appNav, []string{`href="/"`, `href="/_inbox"`, `href="/_todo"`, `href="/_projects"`, `href="/_calendar"`, `href="/_search"`, `href="/_tags"`, `href="/_maintenance"`, `href="/_trash"`})
}

func TestContextPaneIncludesMaintenance(t *testing.T) {
	v := makeVault(t)
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()

	contextLinks := extractBetween(t, body, `aria-label="Context quick links"`, `</nav>`)
	if !strings.Contains(contextLinks, `href="/_maintenance"`) {
		t.Fatalf("context pane quick links should include Maintenance:\n%s", contextLinks)
	}
}

// ---------------------------------------------------------------------------
// Phase 5b: Calendar page
// ---------------------------------------------------------------------------

func TestCalendarPageRendersCurrentMonth(t *testing.T) {
	v := makeVault(t)
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_calendar", nil)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("calendar page: expected 200, got %d", w.Code)
	}
	body := w.Body.String()

	for _, want := range []string{
		`<h1>Daily notes</h1>`,
		`class="calendar-month-grid"`,
		`class="calendar-day"`,
		`Previous month`,
		`Next month`,
		`calendar-month-label`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("calendar page missing %q in:\n%s", want, body)
		}
	}

	// Should link to month navigation.
	if !strings.Contains(body, `/_calendar?month=`) && !strings.Contains(body, `/_calendar?date=`) {
		t.Fatal("calendar page should contain calendar links with /_calendar prefix")
	}
}

func TestCalendarPageMonthQuery(t *testing.T) {
	v := makeVault(t)
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_calendar?month=2026-06", nil)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("calendar page with month param: expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "June 2026") {
		t.Fatalf("calendar should show June 2026 month label, got:\n%s", body)
	}
}

func TestCalendarPageDateQuerySelectsMonthAndDailyNote(t *testing.T) {
	v := makeVault(t)
	dailyRel := "Daily Notes/2026/2026-06/2026-06-15.md"
	dailyAbs := filepath.Join(v.Root, filepath.FromSlash(dailyRel))
	if err := os.MkdirAll(filepath.Dir(dailyAbs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dailyAbs, []byte("# Daily Fifteen\n\nSelected day content.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_calendar?date=2026-06-15", nil)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("calendar page with date param: expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{
		"June 2026",
		`calendar-day has-note selected`,
		"Daily Fifteen",
		`href="/Daily%20Notes/2026/2026-06/2026-06-15.md"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("calendar date selection missing %q in:\n%s", want, body)
		}
	}
}

func TestCalendarPageDateQueryDoesNotShiftInNegativeTimezone(t *testing.T) {
	originalLocal := time.Local
	time.Local = time.FixedZone("Negative", -8*60*60)
	t.Cleanup(func() { time.Local = originalLocal })

	v := makeVault(t)
	dailyRel := "Daily Notes/2026/2026-06/2026-06-01.md"
	dailyAbs := filepath.Join(v.Root, filepath.FromSlash(dailyRel))
	if err := os.MkdirAll(filepath.Dir(dailyAbs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dailyAbs, []byte("# June First\n\nNo timezone shift.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_calendar?date=2026-06-01", nil)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("calendar page with date param: expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{"June 2026", "Jun 1, 2026", "June First"} {
		if !strings.Contains(body, want) {
			t.Fatalf("calendar date should not shift in negative timezone; missing %q in:\n%s", want, body)
		}
	}
}

func TestCalendarPageInvalidMonthFallsBack(t *testing.T) {
	v := makeVault(t)
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_calendar?month=not-a-month", nil)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("calendar page with invalid month: expected 200, got %d", w.Code)
	}
	// Should render the current month (not crash).
	body := w.Body.String()
	if !strings.Contains(body, `class="calendar-month-grid"`) {
		t.Fatalf("calendar should render even with invalid month:\n%s", body)
	}
}

func TestCalendarPageInvalidDateFallsBack(t *testing.T) {
	v := makeVault(t)
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_calendar?date=bad-date", nil)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("calendar page with invalid date: expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `class="calendar-month-grid"`) {
		t.Fatalf("calendar should render even with invalid date:\n%s", body)
	}
}

func TestCalendarPageExcludesHiddenAndTemplateAndTrash(t *testing.T) {
	v := makeVault(t)
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("editing:\n  enabled: true\n  hide_templates: false\nhidden:\n  - Daily Notes/Hidden\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a legitimate daily note for today.
	now := now()
	todayStr := now.Format("2006-01-02")
	year, month, _ := now.Date()
	dailyRel := filepath.Join("Daily Notes", fmt.Sprintf("%d", year), fmt.Sprintf("%d-%02d", year, month), todayStr+".md")
	dailyAbs := filepath.Join(v.Root, filepath.FromSlash(dailyRel))
	if err := os.MkdirAll(filepath.Dir(dailyAbs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dailyAbs, []byte("# Today\nDaily note content.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a date-stamped note under hidden, template, trash, and dot paths.
	must := func(rel, body string) {
		p := filepath.Join(v.Root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	must("Daily Notes/Hidden/2026-06/2026-06-15.md", "# Hidden daily\n")
	must("Daily Notes/2026/2026-06-16/_template.md", "# Template daily\n")
	must("_trash/Daily Notes/2026/2026-06/2026-06-15.md", "# Trashed daily\n")
	must("Daily Notes/.dot/2026-06/2026-06-15.md", "# Dot daily\n")

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_calendar", nil)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("calendar page: expected 200, got %d", w.Code)
	}
	body := w.Body.String()

	// Must not include hidden, template, trash, or dot paths.
	for _, unwanted := range []string{
		"Daily Notes/Hidden",
		"Hidden daily",
		"_template.md",
		"Template daily",
		"Trashed daily",
		"Dot daily",
	} {
		if strings.Contains(body, unwanted) {
			t.Fatalf("calendar must not expose path %q in:\n%s", unwanted, body)
		}
	}
}

func TestModernWorkbenchPhase5bPolishContracts(t *testing.T) {
	v := makeVault(t)
	s := NewServer(v, "", "")

	for _, tc := range []struct {
		path string
		want []string
	}{
		{path: "/_calendar?date=2026-06-15", want: []string{`class="calendar-page"`, `class="calendar-workbench"`, `calendar-selected-panel`, `class="calendar-grid-note"`}},
		{path: "/_todo?today=2026-05-20", want: []string{`class="todo-overview"`, `href="#todo-overdue"`, `data-todo-search`, `data-task-id=`, `data-task-menu`}},
		{path: "/_tags", want: []string{`class="tags-page"`, `class="tag-controls"`, `data-tag-filter`, `data-hide-rare`, `data-tag-chip`, `class="tag-index-grid"`}},
		{path: "/_tags/demo", want: []string{`class="tag-detail-page"`, `href="/_tags"`, `class="note-card-grid tag-note-list"`}},
	} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", tc.path, nil)
		s.ServeHTTP(w, r)
		body := w.Body.String()
		for _, want := range tc.want {
			if !strings.Contains(body, want) {
				t.Fatalf("%s missing Phase 5b markup %q in:\n%s", tc.path, want, body)
			}
		}
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_static/style.css", nil)
	s.ServeHTTP(w, r)
	css := w.Body.String()
	for _, want := range []string{
		"Modern Workbench Phase 5b: Tasks, Calendar, Tags polish",
		".calendar-workbench{display:grid;grid-template-columns:minmax(0,1fr) minmax(280px,360px)",
		".calendar-month-grid{display:grid;grid-template-columns:repeat(7,minmax(0,1fr))",
		".todo-overview{display:grid;grid-template-columns:repeat(5,minmax(0,1fr))",
		".todo-toolbar label:not(.todo-search):not(.todo-filter-tag):not(.toggle){display:grid}",
		".tag-index-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(220px,1fr))",
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("missing Phase 5b CSS %q in:\n%s", want, css)
		}
	}
}

func TestCalendarNavAppearsWhenEditingDisabled(t *testing.T) {
	v := makeVault(t)
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()

	appNav := extractBetween(t, body, `aria-label="App navigation"`, `</nav>`)
	if !strings.Contains(appNav, `href="/_calendar"`) {
		t.Fatalf("Calendar nav link should appear in app nav:\n%s", appNav)
	}
	assertOrdered(t, appNav, []string{`href="/"`, `href="/_todo"`, `href="/_projects"`, `href="/_calendar"`, `href="/_search"`})
}

func TestCalendarNavAppearsWhenEditingEnabled(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()

	appNav := extractBetween(t, body, `aria-label="App navigation"`, `</nav>`)
	assertOrdered(t, appNav, []string{`href="/"`, `href="/_inbox"`, `href="/_todo"`, `href="/_projects"`, `href="/_calendar"`, `href="/_search"`, `href="/_tags"`, `href="/_maintenance"`, `href="/_trash"`})
}

func TestContextPaneIncludesCalendar(t *testing.T) {
	v := makeVault(t)
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()

	contextLinks := extractBetween(t, body, `aria-label="Context quick links"`, `</nav>`)
	if !strings.Contains(contextLinks, `href="/_calendar"`) {
		t.Fatalf("context pane quick links should include Calendar:\n%s", contextLinks)
	}
}

// ---------------------------------------------------------------------------
// Phase 5c: Projects page
// ---------------------------------------------------------------------------

func TestProjectsPageRendersActiveProjects(t *testing.T) {
	v := makeVault(t)
	writeProject := func(rel, body, mod string) {
		p := filepath.Join(v.Root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		mt, err := time.Parse(time.RFC3339, mod)
		if err != nil {
			t.Fatal(err)
		}
		os.Chtimes(p, mt, mt)
	}
	writeProject("Projects/Alpha.md", "---\nstatus: active\n---\n# Alpha\n", "2026-06-10T10:00:00Z")
	writeProject("Projects/Beta/Plan.md", "---\nstatus: active\n---\n# Beta Plan\n", "2026-06-11T10:00:00Z")
	writeProject("Projects/Done.md", "---\nstatus: done\n---\n# Done\n", "2026-06-14T10:00:00Z")

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_projects", nil)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("projects page: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()

	// Should include active projects.
	if !strings.Contains(body, "Alpha") || !strings.Contains(body, "Beta") {
		t.Fatalf("projects page should include active projects Alpha and Beta:\n%s", body)
	}
	// Should exclude inactive/done.
	if strings.Contains(body, "Done") {
		t.Fatalf("projects page must not include Done project:\n%s", body)
	}
	// Should link to project pages.
	if !strings.Contains(body, `href="/Projects/Alpha"`) && !strings.Contains(body, `href="/Projects/Alpha.md"`) {
		t.Fatalf("projects page should link to Alpha project:\n%s", body)
	}
}

func TestProjectsPageExcludesInactiveAndNoStatus(t *testing.T) {
	v := makeVault(t)
	writeProject := func(rel, body, mod string) {
		p := filepath.Join(v.Root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		mt, err := time.Parse(time.RFC3339, mod)
		if err != nil {
			t.Fatal(err)
		}
		os.Chtimes(p, mt, mt)
	}
	writeProject("Projects/Active.md", "---\nstatus: active\n---\n# Active\n", "2026-06-10T10:00:00Z")
	writeProject("Projects/Inactive.md", "---\nstatus: inactive\n---\n# Inactive\n", "2026-06-11T10:00:00Z")
	writeProject("Projects/NoStatus.md", "# No Status\n", "2026-06-12T10:00:00Z")

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_projects", nil)
	s.ServeHTTP(w, r)

	body := w.Body.String()
	if !strings.Contains(body, "Active") {
		t.Fatalf("projects page should include Active project:\n%s", body)
	}
	if strings.Contains(body, "Inactive") || strings.Contains(body, "NoStatus") {
		t.Fatalf("projects page must exclude inactive/no-status projects:\n%s", body)
	}
}

func TestProjectsPageIncludesNestedProjectFolder(t *testing.T) {
	v := makeVault(t)
	// Direct project file.
	direct := filepath.Join(v.Root, "Projects", "Direct.md")
	if err := os.MkdirAll(filepath.Dir(direct), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(direct, []byte("---\nstatus: active\n---\n# Direct\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Nested project folder.
	nestedDir := filepath.Join(v.Root, "Projects", "Nested")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nestedDir, "Plan.md"), []byte("---\nstatus: active\n---\n# Nested Plan\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_projects", nil)
	s.ServeHTTP(w, r)

	body := w.Body.String()
	if !strings.Contains(body, "Direct") || !strings.Contains(body, "Nested") {
		t.Fatalf("projects page should include both direct and nested projects:\n%s", body)
	}
}

func TestProjectsPageExcludesTemplate(t *testing.T) {
	v := makeVault(t)
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("editing:\n  enabled: true\n  hide_templates: false\nhidden:\n  - Projects/Hidden\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Active project using non-template paths only.
	activeDir := filepath.Join(v.Root, "Projects", "ActiveProj")
	if err := os.MkdirAll(activeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(activeDir, "Plan.md"), []byte("---\nstatus: active\n---\n# Plan\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Template and non-enumerable projects should be excluded even with hide_templates=false.
	for rel, body := range map[string]string{
		"Projects/TemplateOnly/_template.md":       "---\nstatus: active\n---\n# Template Only\n",
		"Projects/Hidden/Plan.md":                  "---\nstatus: active\n---\n# Hidden Project\n",
		"Projects/.dot/Plan.md":                    "---\nstatus: active\n---\n# Dot Project\n",
		"_trash/2026-06-21/Projects/Trash/Plan.md": "---\nstatus: active\n---\n# Trash Project\n",
	} {
		path := filepath.Join(v.Root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_projects", nil)
	s.ServeHTTP(w, r)

	body := w.Body.String()
	// Extract only the main content area for project-specific checks.
	mainContent := extractBetween(t, body, `main class="main"`, `</main>`)
	if !strings.Contains(mainContent, "ActiveProj") {
		t.Fatalf("projects page should include ActiveProj:\n%s", mainContent)
	}
	for _, unwanted := range []string{"_template", "Template Only", "Hidden Project", "Dot Project", "Trash Project"} {
		if strings.Contains(mainContent, unwanted) {
			t.Fatalf("projects page must not expose %q:\n%s", unwanted, mainContent)
		}
	}
}

func TestProjectsPageShowsCountAndLatest(t *testing.T) {
	v := makeVault(t)
	writeProject := func(rel, body, mod string) {
		p := filepath.Join(v.Root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		mt, err := time.Parse(time.RFC3339, mod)
		if err != nil {
			t.Fatal(err)
		}
		os.Chtimes(p, mt, mt)
	}
	// Multi-note project: once a project has at least one active note, existing
	// semantics count and consider other notes in that project for latest note.
	writeProject("Projects/Multi/Note1.md", "---\nstatus: active\n---\n# Note One\n", "2026-06-10T10:00:00Z")
	writeProject("Projects/Multi/Note2.md", "---\nstatus: done\n---\n# Note Two (latest non-active)\n", "2026-06-12T10:00:00Z")

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_projects", nil)
	s.ServeHTTP(w, r)

	body := w.Body.String()
	if !strings.Contains(body, "Multi") {
		t.Fatalf("projects page should include Multi project:\n%s", body)
	}
	// Should show note count.
	if !strings.Contains(body, "Note Two (latest non-active)") {
		t.Fatalf("projects page should show latest note title:\n%s", body)
	}
}

func TestHomepageActiveProjectsLimitUnchanged(t *testing.T) {
	v := makeVault(t)
	writeProject := func(rel, body, mod string) {
		p := filepath.Join(v.Root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		mt, err := time.Parse(time.RFC3339, mod)
		if err != nil {
			t.Fatal(err)
		}
		os.Chtimes(p, mt, mt)
	}
	for i := 0; i < 10; i++ {
		label := fmt.Sprintf("Project%02d", i)
		writeProject(fmt.Sprintf("Projects/%s.md", label), fmt.Sprintf("---\nstatus: active\n---\n# %s\n", label), fmt.Sprintf("2026-06-%02dT10:00:00Z", 10+i))
	}

	idx, err := v.BuildIndex()
	if err != nil {
		t.Fatal(err)
	}

	// Homepage ActiveProjects with limit=5 should return at most 5.
	limited := v.ActiveProjects(idx, 5)
	if len(limited) > 5 {
		t.Fatalf("ActiveProjects with limit=5 should return at most 5, got %d", len(limited))
	}

	// ActiveProjectsAll returns all.
	all := v.ActiveProjectsAll(idx)
	if len(all) < 9 {
		t.Fatalf("ActiveProjectsAll should return all active projects, got %d", len(all))
	}
}

func TestProjectsNavAppearsWhenEditingDisabled(t *testing.T) {
	v := makeVault(t)
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()

	appNav := extractBetween(t, body, `aria-label="App navigation"`, `</nav>`)
	if !strings.Contains(appNav, `href="/_projects"`) {
		t.Fatalf("Projects nav link should appear in app nav:\n%s", appNav)
	}
	assertOrdered(t, appNav, []string{`href="/"`, `href="/_todo"`, `href="/_projects"`, `href="/_calendar"`})
}

func TestProjectsNavAppearsWhenEditingEnabled(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()

	appNav := extractBetween(t, body, `aria-label="App navigation"`, `</nav>`)
	assertOrdered(t, appNav, []string{`href="/"`, `href="/_inbox"`, `href="/_todo"`, `href="/_projects"`, `href="/_calendar"`, `href="/_search"`, `href="/_tags"`, `href="/_maintenance"`, `href="/_trash"`})
}

func TestContextPaneIncludesProjects(t *testing.T) {
	v := makeVault(t)
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()

	contextLinks := extractBetween(t, body, `aria-label="Context quick links"`, `</nav>`)
	if !strings.Contains(contextLinks, `href="/_projects"`) {
		t.Fatalf("context pane quick links should include Projects:\n%s", contextLinks)
	}
}

func TestProjectsPageEmptyState(t *testing.T) {
	v := makeVault(t)
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_projects", nil)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("projects page: expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "No active projects found") {
		t.Fatalf("projects page should show empty state when no projects:\n%s", body)
	}
}
