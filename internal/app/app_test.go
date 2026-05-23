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
	must(".notes-web.yaml", "favorites:\n  - Areas/Daily Briefings\ndaily_glob: Areas/Daily Briefings/*-briefing.md\n")
	must("Areas/Daily Briefings/2026-05-22-briefing.md", "---\ntitle: Daily Briefing\ntags: [daily]\n---\n# Heading One\n\nHello [[Target|the target]] and [[Missing]].\n\n| A | B |\n|---|---|\n| 1 | 2 |\n\n- [x] done\n- [ ] todo\n\n> [!note] A callout\n> body\n\n```mermaid\ngraph TD; A-->B;\n```\n")
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
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("favorites:\n  - Areas/Secret\n  - Areas/Hidden.md\nhidden:\n  - Areas/Secret\n  - Areas/Hidden.md\n"), 0o644); err != nil {
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
	for _, want := range []string{"Daily Briefing", "frontmatter", "<table>", "type=\"checkbox\" checked", "class=\"callout note callout-note\"", "class=\"mermaid\"", "/Areas/Target.md", "/_missing?name=Missing"} {
		if !strings.Contains(doc.HTML, want) {
			t.Fatalf("missing %q in html:\n%s", want, doc.HTML)
		}
	}
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
	for _, want := range []string{
		`const panelStateStorageKey = 'notes-web:panel-open';`,
		`function restorePanelState()`,
		`[data-panel-state]`,
		`localStorage.setItem(panelStateStorageKey`,
		`restorePanelState();`,
	} {
		if !strings.Contains(js, want) {
			t.Fatalf("missing panel localStorage JS %q in:\n%s", want, js)
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
