package app

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func makeVault(t *testing.T) *Vault {
	t.Helper()
	root := t.TempDir()
	must := func(rel, body string) {
		t.Helper()
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	must(".notes-web.yaml", "favorites:\n  - path: Areas/Daily Briefings\n    label: Daily Briefings\ndaily_glob: Areas/Daily Briefings/*-briefing.md\n")
	must("Areas/Daily Briefings/2026-05-22-briefing.md", "---\ntitle: Daily Briefing\ntags: [daily]\n---\n# Heading One\n\nHello [[Target|the target]] and [[Missing]].\n\n| A | B |\n|---|---|\n| 1 | 2 |\n\n- [x] done\n- [ ] todo\n\n```go\nfmt.Println(\"copy me\")\n```\n\n> [!note] A callout\n> body\n\n```mermaid\ngraph TD; A-->B;\n```\n")
	must("Areas/Target.md", "# Target\n")
	must("Areas/Work/Meeting Notes.md", "# Work\n")
	must("Areas/Personal/Meeting Notes.md", "# Personal\n")
	must("Areas/Linker.md", "See [[Target]] and [brief](Daily%20Briefings/2026-05-22-briefing.md)\n")
	must("Areas/TODO.md", "# TODO\n\n- [ ] Change Captur tires 📅 2026-05-19 <!-- tid:1c496356 -->\n- [x] Buy dog food ✅ 2026-05-20 <!-- tid:149d256b -->\n")
	v, err := NewVault(root)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestSafePathRejectsTraversal(t *testing.T) {
	v := makeVault(t)
	p, err := v.ResolveURLPath("/Areas/Target.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(filepath.ToSlash(p), "/Areas/Target.md") {
		t.Fatalf("unexpected path %s", p)
	}
	if _, err := v.ResolveURLPath("/../secret"); err == nil {
		t.Fatal("expected traversal rejection")
	}
}

func TestConfiguredHiddenPathsAreNotNavigableOrServed(t *testing.T) {
	v := makeVault(t)
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("favorites:\n  - path: Areas/Secret\n    label: Secret\n  - path: Areas/Hidden.md\n    label: Hidden\nhidden:\n  - Areas/Secret\n  - Areas/Hidden.md\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	secretDir := filepath.Join(v.Root, "Areas", "Secret")
	if err := os.MkdirAll(secretDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(secretDir, "Note.md"), []byte("# Secret\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(v.Root, "Areas", "Hidden.md"), []byte("# Hidden\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if treeContainsRel(v.Tree(3), "Areas/Secret") || treeContainsRel(v.Tree(3), "Areas/Hidden.md") {
		t.Fatalf("hidden path leaked into tree: %+v", v.Tree(3))
	}

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Areas", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	if strings.Contains(body, "Secret") || strings.Contains(body, "Hidden.md") {
		t.Fatalf("hidden path leaked into folder listing:\n%s", body)
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/Areas/Secret/Note.md", nil)
	s.ServeHTTP(w, r)
	if w.Code != 404 {
		t.Fatalf("hidden note direct URL status=%d want 404, body=%s", w.Code, w.Body.String())
	}
}

func TestFavoritesUseConfiguredPathAndLabel(t *testing.T) {
	v := makeVault(t)
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("favorites:\n  - path: Areas/Daily Briefings\n    label: Briefings\n  - path: _todo\n    label: Todos\n  - path: Projects/\n    label: Projects\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	favorites := v.Favorites()
	if len(favorites) != 3 {
		t.Fatalf("favorites len=%d want 3: %+v", len(favorites), favorites)
	}
	if favorites[0].Path != "Areas/Daily Briefings" || favorites[0].Label != "Briefings" || favorites[0].URL != "/Areas/Daily%20Briefings" {
		t.Fatalf("first favorite mismatch: %+v", favorites[0])
	}
	if favorites[1].Path != "_todo" || favorites[1].Label != "Todos" || favorites[1].URL != "/_todo" {
		t.Fatalf("todo favorite mismatch: %+v", favorites[1])
	}
	if favorites[2].Path != "Projects" || favorites[2].Label != "Projects" || favorites[2].URL != "/Projects" {
		t.Fatalf("projects favorite mismatch: %+v", favorites[2])
	}
}

func TestFavoriteRequiresConfiguredLabel(t *testing.T) {
	v := makeVault(t)
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("favorites:\n  - path: _todo\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if favorites := v.Favorites(); len(favorites) != 0 {
		t.Fatalf("favorite without label should be ignored, got %+v", favorites)
	}
}

func treeContainsRel(nodes []TreeNode, rel string) bool {
	for _, node := range nodes {
		if node.Rel == rel || treeContainsRel(node.Children, rel) {
			return true
		}
	}
	return false
}

func TestFolderPageSortsByNameAscendingByDefaultAndOffersSortLinks(t *testing.T) {
	v := makeVault(t)
	writeNoteForFolderSortTest(t, v, "Areas/SortTest/Beta.md", "2026-05-21T08:00:00Z")
	writeNoteForFolderSortTest(t, v, "Areas/SortTest/Alpha.md", "2026-05-22T08:00:00Z")

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Areas/SortTest", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()

	assertInOrder(t, body, "Alpha.md", "Beta.md")
	for _, want := range []string{
		`href="/Areas/SortTest?sort=name&amp;dir=asc" aria-current="true"`,
		`href="/Areas/SortTest?sort=name&amp;dir=desc"`,
		`href="/Areas/SortTest?sort=modified&amp;dir=desc"`,
		`href="/Areas/SortTest?sort=modified&amp;dir=asc"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("folder page missing sort control %q in:\n%s", want, body)
		}
	}
}

func TestFolderPageSortCanUseQueryAndConfigDefault(t *testing.T) {
	v := makeVault(t)
	writeNoteForFolderSortTest(t, v, "Areas/SortTest/A-Older.md", "2026-05-21T08:00:00Z")
	writeNoteForFolderSortTest(t, v, "Areas/SortTest/Z-Newer.md", "2026-05-22T08:00:00Z")

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Areas/SortTest?sort=modified&dir=desc", nil)
	s.ServeHTTP(w, r)
	assertInOrder(t, w.Body.String(), "Z-Newer.md", "A-Older.md")

	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("folder_sort: modified_desc\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/Areas/SortTest", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	assertInOrder(t, body, "Z-Newer.md", "A-Older.md")
	if !strings.Contains(body, `href="/Areas/SortTest?sort=modified&amp;dir=desc" aria-current="true"`) {
		t.Fatalf("configured default sort should be marked current in:\n%s", body)
	}
}

func writeNoteForFolderSortTest(t *testing.T, v *Vault, rel, mod string) {
	t.Helper()
	p := filepath.Join(v.Root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("# "+strings.TrimSuffix(filepath.Base(rel), filepath.Ext(rel))+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mt, err := time.Parse(time.RFC3339, mod)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(p, mt, mt); err != nil {
		t.Fatal(err)
	}
}

func assertInOrder(t *testing.T, body string, first string, second string) {
	t.Helper()
	if listStart := strings.Index(body, `<ul class="list folder-list">`); listStart >= 0 {
		body = body[listStart:]
		if listEnd := strings.Index(body, `</ul>`); listEnd >= 0 {
			body = body[:listEnd]
		}
	}
	firstIndex := strings.Index(body, first)
	secondIndex := strings.Index(body, second)
	if firstIndex < 0 || secondIndex < 0 || firstIndex > secondIndex {
		t.Fatalf("expected %q before %q in:\n%s", first, second, body)
	}
}

func TestWikilinkResolution(t *testing.T) {
	v := makeVault(t)
	if r := v.ResolveWikiLink("Target"); r.Kind != "unique" {
		t.Fatalf("Target kind=%s", r.Kind)
	}
	if r := v.ResolveWikiLink("Meeting Notes"); r.Kind != "ambiguous" || len(r.Matches) != 2 {
		t.Fatalf("Meeting Notes=%+v", r)
	}
	if r := v.ResolveWikiLink("Does Not Exist"); r.Kind != "missing" {
		t.Fatalf("missing kind=%s", r.Kind)
	}
}

func TestMarkdownRenderingFeatures(t *testing.T) {
	v := makeVault(t)
	note, err := v.ReadNote("Areas/Daily Briefings/2026-05-22-briefing.md")
	if err != nil {
		t.Fatal(err)
	}
	r := NewRenderer(v)
	doc := r.Render(note)
	for _, want := range []string{"Daily Briefing", "frontmatter", "<table>", "type=\"checkbox\" checked", "class=\"callout note callout-note\"", "class=\"mermaid\"", "class=\"code-block\"", "class=\"code-copy\"", "data-copy-code", "aria-label=\"Copy code block\"", "/Areas/Target.md", "/_missing?name=Missing"} {
		if !strings.Contains(doc.HTML, want) {
			t.Fatalf("missing %q in html:\n%s", want, doc.HTML)
		}
	}
}

func TestTaskParserExtractsOperationalMetadataAndKeepsTitleClean(t *testing.T) {
	task, ok := parseTaskLine("- [ ] ⏫ Phase 1 Silex — Core KB #project/silex #admin ➕ 2026-05-24 📅 2026-07-01 🔁 every month on the 1st <!-- tid:abc123 -->")
	if !ok {
		t.Fatal("expected task line to parse")
	}
	if task.Text != "Phase 1 Silex — Core KB" {
		t.Fatalf("title should hide task metadata, got %q", task.Text)
	}
	if task.Priority != "P1" || task.PriorityRank != 1 {
		t.Fatalf("priority=%q rank=%d, want P1/1", task.Priority, task.PriorityRank)
	}
	if task.Added != "2026-05-24" || task.Due != "2026-07-01" || task.Repeat != "every month on the 1st" {
		t.Fatalf("metadata not extracted: added=%q due=%q repeat=%q", task.Added, task.Due, task.Repeat)
	}
	if strings.Join(task.Tags, ",") != "project/silex,admin" {
		t.Fatalf("tags=%v", task.Tags)
	}
	if task.ID != "abc123" {
		t.Fatalf("id=%q", task.ID)
	}
}

func TestTaskTextHTMLLinkifiesURLsWithoutTrustingRawHTML(t *testing.T) {
	cases := []struct {
		name string
		text string
		want string
	}{
		{
			name: "raw URL",
			text: `Read <script>alert(1)</script> https://example.com/a?x=1&y=2.`,
			want: `Read &lt;script&gt;alert(1)&lt;/script&gt; <a href="https://example.com/a?x=1&amp;y=2" target="_blank" rel="noopener noreferrer">https://example.com/a?x=1&amp;y=2</a>.`,
		},
		{
			name: "markdown URL",
			text: `Read [the doc](https://example.com/doc?x=1&y=2) today`,
			want: `Read <a href="https://example.com/doc?x=1&amp;y=2" target="_blank" rel="noopener noreferrer">the doc</a> today`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			task := TaskItem{Text: tc.text}
			if got := string(task.TextHTML()); got != tc.want {
				t.Fatalf("TextHTML()=%q, want %q", got, tc.want)
			}
		})
	}
}

func TestTodoPageUsesDenseReadOnlyRowsAndHidesTaskIDs(t *testing.T) {
	v := makeVault(t)
	todoPath := filepath.Join(v.Root, "Areas", "TODO.md")
	body := strings.Join([]string{
		"# TODO",
		"",
		"- [ ] ⏫ Overdue task [[Target|the target]] https://example.com/path?x=1&y=2 #admin ➕ 2026-05-01 📅 2026-05-19 <!-- tid:overdue123 -->",
		"- [ ] 🔼 Today task #project/silex ➕ 2026-05-20 📅 2026-05-22 🔁 every week <!-- tid:today123 -->",
		"- [ ] Upcoming task #book 📅 2026-07-01 <!-- tid:upcoming123 -->",
		"- [ ] No date task #inbox <!-- tid:nodate123 -->",
		"- [x] Done middle task #admin ✅ 2026-05-21 <!-- tid:done123 -->",
		"- [x] Done newest task #admin ✅ 2026-05-23 <!-- tid:done-newest123 -->",
		"- [x] Done oldest task #admin ✅ 2026-05-20 <!-- tid:done-oldest123 -->",
	}, "\n")
	if err := os.WriteFile(todoPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_todo?today=2026-05-22", nil)
	s.ServeHTTP(w, r)
	html := w.Body.String()

	for _, want := range []string{
		`class="todo-shell"`,
		`4 open · 1 overdue · 1 today · 1 upcoming · 1 no date hidden`,
		`placeholder="Search tasks"`,
		`data-todo-filter="tag"`,
		`data-todo-filter="priority"`,
		`data-todo-hide-nodate`,
		`data-todo-hide-done`,
		`<section class="todo-section overdue"`,
		`<section class="todo-section today"`,
		`<section class="todo-section upcoming"`,
		`<section class="todo-section no-date"`,
		`<section class="todo-section done"`,
		`data-task-id="overdue123"`,
		`data-tags=" admin"`,
		`data-priority="P1"`,
		`class="task-priority p1">P1</span>`,
		`<span class="task-title">Overdue task <a href="/Areas/Target.md" target="_self">the target</a> <a href="https://example.com/path?x=1&amp;y=2" target="_blank" rel="noopener noreferrer">https://example.com/path?x=1&amp;y=2</a></span>`,
		`<span class="task-tag">#admin</span>`,
		`Added 2026-05-01`,
		`Repeats every week`,
		`<div class="task-actions"><button type="button" class="task-menu" data-task-menu aria-haspopup="menu" aria-expanded="false" aria-label="Task actions" title="Task actions">⋯</button><div class="task-menu-dropdown" role="menu" hidden><button type="button" role="menuitem" data-copy="todo done overdue123">Mark as done</button><button type="button" role="menuitem" data-copy="overdue123">Copy todo ID</button></div></div>`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("missing dense TODO markup %q in:\n%s", want, html)
		}
	}
	if strings.Contains(html, `Copy task command`) || strings.Contains(html, `class="task-copy"`) || strings.Contains(html, `data-copy="td done `) || strings.Contains(html, `td promote `) || strings.Contains(html, `td demote `) || strings.Contains(html, `td reschedule `) || strings.Contains(html, `Promote`) || strings.Contains(html, `Demote`) || strings.Contains(html, `Re-schedule`) {
		t.Fatalf("legacy inline copy task command should be removed; actions belong in the dropdown:\n%s", html)
	}
	if strings.Contains(html, `class="task-title" href=`) || strings.Contains(html, `href="/Areas/TODO.md#line-3">Overdue task`) {
		t.Fatalf("task title text should not link to the source line; only links inside the task text should be clickable:\n%s", html)
	}
	if strings.Contains(html, `class="task-project"`) || strings.Contains(html, `>Inbox</a>`) || strings.Contains(html, `>Admin</a>`) {
		t.Fatalf("task project text/link should not be rendered in TODO rows:\n%s", html)
	}
	if strings.Contains(html, "tid:overdue123") || strings.Contains(html, "tid:today123") {
		t.Fatalf("technical tid labels should stay hidden from the visible TODO UI:\n%s", html)
	}
	if strings.Contains(html, `<details class="todo-section no-date"`) || strings.Contains(html, `<details class="todo-section done"`) {
		t.Fatalf("TODO sections must not be collapsible details/summary panels:\n%s", html)
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_static/style.css", nil)
	s.ServeHTTP(w, r)
	css := w.Body.String()
	for _, wantCSS := range []string{
		`.task-primary{display:grid;grid-template-columns:minmax(0,1fr) auto 28px;gap:10px;align-items:center}`,
	} {
		if !strings.Contains(css, wantCSS) {
			t.Fatalf("TODO due date and action menu should align to the right with a compact trailing grid %q in:\n%s", wantCSS, css)
		}
	}
	for _, forbidden := range []string{
		`.todo-section.no-date:not([open]) .task-list`,
		`.todo-section.done:not([open]) .task-list`,
		`.todo-section>summary{cursor:pointer`,
	} {
		if strings.Contains(css, forbidden) {
			t.Fatalf("TODO sections are static sections; CSS must not hide task lists behind details/open state %q in:\n%s", forbidden, css)
		}
	}

	assertInOrder(t, html, `todo-section overdue`, `todo-section today`)
	assertInOrder(t, html, `todo-section today`, `todo-section upcoming`)
	assertInOrder(t, html, `Done newest task`, `Done middle task`)
	assertInOrder(t, html, `Done middle task`, `Done oldest task`)
}

func TestNotePanelsAreCollapsibleAndPersistTheirOpenState(t *testing.T) {
	v := makeVault(t)
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Areas/Daily%20Briefings/2026-05-22-briefing.md", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	for _, want := range []string{
		`<details class="toc" data-panel-state="toc" open><summary>Table of contents`,
		`<details class="frontmatter" data-panel-state="frontmatter" open><summary>Frontmatter`,
		`<details class="link-panel forward-links" data-panel-state="forward-links" open><summary><span class="panel-title">Forward links</span>`,
		`<details class="link-panel backlinks" data-panel-state="backlinks" open><summary><span class="panel-title">Backlinks</span>`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing persisted collapsible panel markup %q in:\n%s", want, body)
		}
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_static/app.js", nil)
	s.ServeHTTP(w, r)
	js := w.Body.String()
	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_static/style.css", nil)
	s.ServeHTTP(w, r)
	css := w.Body.String()
	for _, want := range []string{
		`.task-row[hidden]{display:none!important}`,
		`.todo-toolbar label:not(.todo-search):not(.todo-filter-tag):not(.toggle){display:none}`,
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("missing TODO hidden-row CSS %q in:\n%s", want, css)
		}
	}

	for _, want := range []string{
		`const panelStateStorageKey = 'notes-web:panel-open';`,
		`function restorePanelState()`,
		`[data-panel-state]`,
		`localStorage.setItem(panelStateStorageKey`,
		`restorePanelState();`,
		`function initTodoFilters()`,
		`const todoFilterStorageKey = 'notes-web:todo-filters';`,
		`readTodoFilterState()`,
		`writeTodoFilterState({ tag: tag?.value || '', priority: priority?.value || '', date: date?.value || '', group: group?.value || 'Due date', hideNoDate: Boolean(hideNoDate?.checked), hideDone: Boolean(hideDone?.checked) })`,
		`restoreTodoFilterState({ tag, priority, date, group, hideNoDate, hideDone })`,
		`updateTodoTagCounts(rows, { q, selectedPriority, selectedDate, hideNoDate: Boolean(hideNoDate?.checked), hideDone: Boolean(hideDone?.checked), today })`,
		`countTodoTags(rows, filters)`,
		`option.hidden = count === 0`,
		`option.disabled = count === 0`,
		`option.textContent = '#' + option.value + ' (' + String(counts.get(option.value) || 0) + ')'`,
		`updateTodoSectionCounts(sections)`,
		`section.querySelectorAll('.task-row:not([hidden])').length`,
		`data-todo-search`,
		`data-todo-filter="tag"`,
		`data-todo-hide-nodate`,
		`data-todo-hide-done`,
		`matchesDone`,
		`todoDateGroup`,
		`sortTodoRows`,
		`renderTodoGroupedView(shell, rows, group?.value || 'Due date')`,
		`todo-dynamic-groups`,
		`function initTodoActions()`,
		`data-task-menu`,
		`closeTodoMenus`,
		`initTodoActions();`,
		`initTodoFilters();`,
	} {
		if !strings.Contains(js, want) {
			t.Fatalf("missing panel localStorage JS %q in:\n%s", want, js)
		}
	}
	for _, forbidden := range []string{
		`function populateTodoSelect`,
		`function uniqueTodoValues`,
		`select.options.length > 1`,
		`select.dataset.populated`,
	} {
		if strings.Contains(js, forbidden) {
			t.Fatalf("TODO tag filters must not keep legacy client-side select population compatibility %q in:\n%s", forbidden, js)
		}
	}
}

func TestLinkPanelSummariesUseCompactInlineTypography(t *testing.T) {
	v := makeVault(t)
	s := NewServer(v, "", "")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Areas/Daily%20Briefings/2026-05-22-briefing.md", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	if strings.Contains(body, `<summary><h2>Forward links`) || strings.Contains(body, `<summary><h2>Backlinks`) {
		t.Fatalf("link panel summaries must not use block h2 headings inside summary:\n%s", body)
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_static/style.css", nil)
	s.ServeHTTP(w, r)
	css := w.Body.String()
	for _, want := range []string{
		`.link-panel summary{cursor:pointer;display:flex;align-items:center;gap:8px;font-size:1rem;font-weight:650;line-height:1.35}`,
		`.link-panel .panel-title{display:inline;font:inherit}`,
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("missing compact inline link-panel CSS %q in:\n%s", want, css)
		}
	}
}

func TestCalloutsRenderWithTypesIconsTitlesAndClosedMarkup(t *testing.T) {
	v := makeVault(t)
	note := Note{RelPath: "Areas/Callouts.md", Body: "> [!warning] Check this\n> Important body\n\n> [!tip]\n> Useful body\n"}
	doc := NewRenderer(v).Render(note)
	for _, want := range []string{
		`<div class="callout warning callout-warning" data-callout="warning">`,
		`<span class="callout-icon" aria-hidden="true">⚠️</span>`,
		`<span class="callout-title-text">Check this</span>`,
		`<div class="callout-body"><p>Important body</p></div>`,
		`<div class="callout tip callout-tip" data-callout="tip">`,
		`<span class="callout-icon" aria-hidden="true">💡</span>`,
		`<span class="callout-title-text">Tip</span>`,
		`</div>`,
	} {
		if !strings.Contains(doc.HTML, want) {
			t.Fatalf("missing callout HTML %q in:\n%s", want, doc.HTML)
		}
	}

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_static/style.css", nil)
	s.ServeHTTP(w, r)
	css := w.Body.String()
	for _, want := range []string{
		".callout-warning",
		".callout-tip",
		".callout-icon",
		".callout-body",
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("missing callout CSS %q in:\n%s", want, css)
		}
	}
}

func TestVaultIndexBuildsTypedMetadata(t *testing.T) {
	v := makeVault(t)
	idx, err := v.BuildIndex()
	if err != nil {
		t.Fatal(err)
	}
	if len(idx.Notes) != len(v.MarkdownFiles()) {
		t.Fatalf("index note count=%d want %d", len(idx.Notes), len(v.MarkdownFiles()))
	}
	daily, ok := idx.ByRel["Areas/Daily Briefings/2026-05-22-briefing.md"]
	if !ok {
		t.Fatalf("daily note missing from index: %#v", idx.ByRel)
	}
	if daily.Title != "Daily Briefing" || daily.URL != "/Areas/Daily%20Briefings/2026-05-22-briefing.md" {
		t.Fatalf("bad daily metadata: %+v", daily)
	}
	if !stringSliceContains(daily.Tags, "daily") {
		t.Fatalf("daily tags missing frontmatter tag: %+v", daily.Tags)
	}
	if !stringSliceContains(daily.OutgoingWikiLinks, "Target") || !stringSliceContains(daily.OutgoingWikiLinks, "Missing") {
		t.Fatalf("daily outgoing wikilinks missing: %+v", daily.OutgoingWikiLinks)
	}
	if _, ok := idx.Tags["daily"]; !ok {
		t.Fatalf("tag index missing daily: %+v", idx.Tags)
	}
}

func TestNumericTagsAreFilteredExceptYearsFrom1900To2100(t *testing.T) {
	v := makeVault(t)
	p := filepath.Join(v.Root, "Areas", "Numeric Tags.md")
	if err := os.WriteFile(p, []byte("---\ntitle: Numeric Tags\ntags: [123, 1899, 1900, 2026, 2100, 2101, project]\n---\n# Numeric Tags\n\nInline tags: #456 #1899 #1900 #2026 #2100 #2101 #topic/123 #abc123\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	idx, err := v.BuildIndex()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"1900", "2026", "2100", "project", "topic/123", "abc123"} {
		if _, ok := idx.Tags[want]; !ok {
			t.Fatalf("expected tag %q in index tags: %+v", want, idx.Tags)
		}
	}
	for _, unwanted := range []string{"123", "456", "1899", "2101"} {
		if _, ok := idx.Tags[unwanted]; ok {
			t.Fatalf("numeric non-year tag %q should be filtered from index tags: %+v", unwanted, idx.Tags)
		}
	}

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_tags", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	for _, want := range []string{`href="/_tags/1900"`, `href="/_tags/2026"`, `href="/_tags/2100"`, `href="/_tags/project"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected allowed tag link %q in tags page:\n%s", want, body)
		}
	}
	for _, unwanted := range []string{`href="/_tags/123"`, `href="/_tags/456"`, `href="/_tags/1899"`, `href="/_tags/2101"`} {
		if strings.Contains(body, unwanted) {
			t.Fatalf("unexpected numeric non-year tag link %q in tags page:\n%s", unwanted, body)
		}
	}
}

func stringSliceContains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func TestHomeDashboardUsesCalendarActiveProjectsAndRecentNotes(t *testing.T) {
	oldNow := now
	now = func() time.Time { return time.Date(2026, 5, 23, 12, 0, 0, 0, time.Local) }
	defer func() { now = oldNow }()

	v := makeVault(t)
	old, err := time.Parse(time.RFC3339, "2026-05-01T08:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range v.MarkdownFiles() {
		if err := os.Chtimes(p, old, old); err != nil {
			t.Fatal(err)
		}
	}
	writeNote := func(rel, body string, mod string) {
		t.Helper()
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
		if err := os.Chtimes(p, mt, mt); err != nil {
			t.Fatal(err)
		}
	}
	writeNote("Areas/Daily Briefings/2026-05-22-briefing.md", "---\ntitle: Daily Briefing\ntags: [daily]\n---\n# Heading One\n", "2026-05-22T16:00:00Z")
	writeNote("Areas/France-Publications/Hub2.md", "# Hub2\n", "2026-05-22T15:00:00Z")
	writeNote("Areas/Notes-web/Homepage.md", "# Homepage\n", "2026-05-22T14:00:00Z")
	writeNote("Areas/Amsterdam/Move.md", "# Move\n", "2026-05-20T12:00:00Z")
	writeNote("Areas/Santé/2026-05-21.md", "# Santé\n", "2026-05-21T09:00:00Z")
	writeNote("Areas/Daily Briefings/2026-05-21-briefing.md", "# Older Briefing\n", "2026-05-21T08:00:00Z")

	dashboard, err := v.BuildDashboard()
	if err != nil {
		t.Fatal(err)
	}
	if dashboard.TodayLabel != "Friday, May 22" {
		t.Fatalf("selected dashboard day should come from latest daily note date, got %q", dashboard.TodayLabel)
	}
	if dashboard.Calendar.MonthLabel != "May 2026" {
		t.Fatalf("calendar month label=%q", dashboard.Calendar.MonthLabel)
	}
	if !calendarHasDay(dashboard.Calendar.Weeks, "22", true, true) || !calendarHasDay(dashboard.Calendar.Weeks, "21", false, true) {
		t.Fatalf("calendar should mark selected day and days with daily notes: %+v", dashboard.Calendar.Weeks)
	}
	if len(dashboard.ActiveProjects) == 0 || dashboard.ActiveProjects[0].Label != "France-Publications" {
		t.Fatalf("active projects should group by project and sort by update time: %+v", dashboard.ActiveProjects)
	}
	if !activeProjectContains(dashboard.ActiveProjects, "Notes-web") || !activeProjectContains(dashboard.ActiveProjects, "Amsterdam") {
		t.Fatalf("active projects missing expected project groups: %+v", dashboard.ActiveProjects)
	}

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	for _, want := range []string{
		`<div class="home-dashboard">`,
		`<section class="today-card card">`,
		`Friday, May 22`,
		`<section class="active-projects card">`,
		`France-Publications`,
		`Notes-web`,
		`<aside class="home-calendar card">`,
		`May 2026`,
		`class="calendar-day has-note selected" href="/?date=2026-05-22" aria-label="2026-05-22">22</a>`,
		`class="calendar-day today" href="/?date=2026-05-23" aria-label="2026-05-23">23</a>`,
		`<section class="selected-day card">`,
		`<section class="quick-jump card">`,
		`<section class="recent-notes card">`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing dashboard homepage markup %q in:\n%s", want, body)
		}
	}
}

func calendarHasDay(weeks [][]CalendarDay, label string, selected, hasNote bool) bool {
	for _, week := range weeks {
		for _, day := range week {
			if day.Label == label && day.Selected == selected && day.HasNote == hasNote {
				return true
			}
		}
	}
	return false
}

func activeProjectContains(projects []ActiveProject, label string) bool {
	for _, project := range projects {
		if project.Label == label {
			return true
		}
	}
	return false
}

func TestHomeDashboardDateQuerySelectsCalendarDay(t *testing.T) {
	v := makeVault(t)
	p := filepath.Join(v.Root, filepath.FromSlash("Areas/Daily Briefings/2026-05-21-briefing.md"))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("# Older Briefing\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	old, err := time.Parse(time.RFC3339, "2026-05-21T08:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(p, old, old); err != nil {
		t.Fatal(err)
	}
	latest, err := time.Parse(time.RFC3339, "2026-05-22T09:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	latestPath := filepath.Join(v.Root, filepath.FromSlash("Areas/Daily Briefings/2026-05-22-briefing.md"))
	if err := os.Chtimes(latestPath, latest, latest); err != nil {
		t.Fatal(err)
	}
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/?date=2026-05-21", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	if !strings.Contains(body, `Thursday, May 21`) || !strings.Contains(body, `aria-label="2026-05-21">21</a>`) {
		t.Fatalf("date query should select requested day in homepage:\n%s", body)
	}
}

func TestHomeDashboardCSSDefinesTwoColumnCalendarLayout(t *testing.T) {
	v := makeVault(t)
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_static/style.css", nil)
	s.ServeHTTP(w, r)
	css := w.Body.String()
	for _, want := range []string{
		`.home-dashboard{display:grid;grid-template-columns:minmax(0,1fr) 280px`,
		`.home-main{min-width:0;display:grid;gap:18px}`,
		`.home-calendar`,
		`.calendar-grid{display:grid;grid-template-columns:repeat(7,minmax(0,1fr))`,
		`.calendar-day.selected`,
		`.active-project-row`,
		`@media (max-width: 900px)`,
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("missing homepage calendar/project CSS %q in:\n%s", want, css)
		}
	}
}
