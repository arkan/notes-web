package app

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

func TestBacklinksAndSearch(t *testing.T) {
	v := makeVault(t)
	backs := v.BacklinksTo("Areas/Target.md")
	found := false
	for _, n := range backs {
		if n.RelPath == "Areas/Linker.md" {
			found = true
		}
	}
	if !found {
		t.Fatalf("Linker backlink not found: %+v", backs)
	}
	results, err := NewSearcher(v).Search("target")
	if err != nil {
		t.Fatal(err)
	}
	found = false
	for _, r := range results {
		if r.RelPath == "Areas/Linker.md" {
			found = true
		}
	}
	if !found {
		t.Fatalf("Linker search result not found: %+v", results)
	}
}

func TestSearchRanksTitleMatchesAndHighlightsSnippets(t *testing.T) {
	v := makeVault(t)
	results, err := NewSearcher(v).Search("target")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) < 2 {
		t.Fatalf("expected multiple search results, got %+v", results)
	}
	if results[0].RelPath != "Areas/Target.md" {
		t.Fatalf("title match should rank first, got %+v", results)
	}
	foundHighlightedSnippet := false
	for _, result := range results {
		if strings.Contains(result.SnippetHTML, `<mark>target</mark>`) || strings.Contains(result.SnippetHTML, `<mark>Target</mark>`) {
			foundHighlightedSnippet = true
		}
		if strings.Contains(result.SnippetHTML, "<script") {
			t.Fatalf("snippet HTML must be escaped: %+v", result)
		}
	}
	if !foundHighlightedSnippet {
		t.Fatalf("expected highlighted snippet in results: %+v", results)
	}
}

func TestSearchQuerySyntaxFiltersTagsPathsTitlesFrontmatterAndPhrases(t *testing.T) {
	v := makeVault(t)
	searcher := NewSearcher(v)

	cases := []struct {
		name    string
		query   string
		want    []string
		notWant []string
	}{
		{name: "tag filter", query: "tag:daily", want: []string{"Areas/Daily Briefings/2026-05-22-briefing.md"}, notWant: []string{"Areas/Target.md"}},
		{name: "path filter", query: "path:\"Daily Briefings\"", want: []string{"Areas/Daily Briefings/2026-05-22-briefing.md"}, notWant: []string{"Areas/Target.md"}},
		{name: "title filter", query: "title:Target", want: []string{"Areas/Target.md"}, notWant: []string{"Areas/Linker.md"}},
		{name: "frontmatter filter", query: "frontmatter:title=\"Daily Briefing\"", want: []string{"Areas/Daily Briefings/2026-05-22-briefing.md"}, notWant: []string{"Areas/Target.md"}},
		{name: "quoted exact phrase", query: "\"Hello [[Target|the target]]\"", want: []string{"Areas/Daily Briefings/2026-05-22-briefing.md"}, notWant: []string{"Areas/Linker.md"}},
		{name: "combined filter and term", query: "tag:daily target", want: []string{"Areas/Daily Briefings/2026-05-22-briefing.md"}, notWant: []string{"Areas/Target.md", "Areas/Linker.md"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			results, err := searcher.Search(tc.query)
			if err != nil {
				t.Fatal(err)
			}
			for _, rel := range tc.want {
				if !searchResultsContainRel(results, rel) {
					t.Fatalf("query %q missing %s in %+v", tc.query, rel, results)
				}
			}
			for _, rel := range tc.notWant {
				if searchResultsContainRel(results, rel) {
					t.Fatalf("query %q unexpectedly included %s in %+v", tc.query, rel, results)
				}
			}
		})
	}

	results, err := searcher.Search("tag:daily target")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || !strings.Contains(results[0].SnippetHTML, "<mark>") {
		t.Fatalf("combined query should highlight matched content, got %+v", results)
	}
}

func searchResultsContainRel(results []SearchResult, rel string) bool {
	for _, result := range results {
		if result.RelPath == rel {
			return true
		}
	}
	return false
}

func TestReadingSettingsLiveInSidebarModalAndFocusUsesShortcut(t *testing.T) {
	v := makeVault(t)
	s := NewServer(v, "", "")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Areas/Daily%20Briefings/2026-05-22-briefing.md", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	for _, want := range []string{
		`<article class="note reading-surface">`,
		`class="settings-button btn ghost icon-btn" data-settings-open aria-haspopup="dialog" aria-label="Settings"`,
		`class="settings-modal"`,
		`data-settings-modal hidden`,
		`data-theme-select`,
		`data-font-size-select`,
		`Keyboard shortcuts`,
		`⌘/Ctrl B`,
		`Toggle sidebar`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing settings/modal markup %q in:\n%s", want, body)
		}
	}
	for _, unwanted := range []string{
		`data-focus-toggle`,
		`class="reading-controls"`,
		`<label class="theme-picker">Theme`,
	} {
		if strings.Contains(body, unwanted) {
			t.Fatalf("old inline reading/sidebar control should be removed %q in:\n%s", unwanted, body)
		}
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_static/style.css", nil)
	s.ServeHTTP(w, r)
	css := w.Body.String()
	for _, want := range []string{
		`.reading-surface{max-width:none`,
		`.frontmatter{width:100%;max-width:none`,
		`.side-header{display:flex;align-items:flex-start;justify-content:space-between`,
		`.settings-button{inline-size:52px;block-size:52px;padding:0`,
		`.settings-modal[hidden]{display:none}`,
		`.settings-dialog`,
		`body.reading-focus .side`,
		`[data-font-size="large"]`,
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("missing settings/reading CSS %q in:\n%s", want, css)
		}
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_static/app.js", nil)
	s.ServeHTTP(w, r)
	js := w.Body.String()
	for _, want := range []string{
		`notes-web:font-size`,
		`notes-web:reading-focus`,
		`initSettingsModal`,
		`data-settings-open`,
		`data-settings-close`,
		`toggleReadingFocus()`,
		`ev.key.toLowerCase() === 'b'`,
	} {
		if !strings.Contains(js, want) {
			t.Fatalf("missing settings/focus JS %q in:\n%s", want, js)
		}
	}
}

func TestNoteBreadcrumbSegmentsAreClickableAndShareReadingWidth(t *testing.T) {
	v := makeVault(t)
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Areas/Daily%20Briefings/2026-05-22-briefing.md", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	for _, want := range []string{
		`<nav class="crumb reading-surface" aria-label="Breadcrumb">`,
		`<a href="/">Home</a>`,
		`<a href="/Areas">Areas</a>`,
		`<a href="/Areas/Daily%20Briefings">Daily Briefings</a>`,
		`<a href="/Areas/Daily%20Briefings/2026-05-22-briefing.md" aria-current="page">2026-05-22-briefing.md</a>`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing clickable breadcrumb markup %q in:\n%s", want, body)
		}
	}
	if strings.Contains(body, `Home</a> / Areas/Daily Briefings/2026-05-22-briefing.md`) {
		t.Fatalf("breadcrumb should not render raw non-clickable path in:\n%s", body)
	}
}

func TestFolderViewUsesNoteLayoutAndClickableBreadcrumbs(t *testing.T) {
	v := makeVault(t)
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Areas/Daily%20Briefings", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	for _, want := range []string{
		`<nav class="crumb reading-surface" aria-label="Breadcrumb">`,
		`<a href="/">Home</a>`,
		`<a href="/Areas">Areas</a>`,
		`<a href="/Areas/Daily%20Briefings" aria-current="page">Daily Briefings</a>`,
		`<article class="folder-view reading-surface">`,
		`<header><div><p class="eyebrow">Folder</p><h1>Daily Briefings</h1></div><div class="note-actions"><button class="copy-link btn ghost" data-copy-link>Copy link</button></div></header>`,
		`<ul class="list folder-list">`,
		`<a href="/Areas/Daily%20Briefings/2026-05-22-briefing.md">📄 2026-05-22-briefing.md</a>`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing folder layout markup %q in:\n%s", want, body)
		}
	}
	if strings.Contains(body, `<div class="page-header"><h1>📁 Areas/Daily Briefings</h1>`) {
		t.Fatalf("folder view should not use the old full-width page-header layout:\n%s", body)
	}
	if strings.Contains(body, `<small>Areas/Daily Briefings</small>`) {
		t.Fatalf("folder header should not repeat the breadcrumb path:\n%s", body)
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_static/style.css", nil)
	s.ServeHTTP(w, r)
	css := w.Body.String()
	if strings.Contains(css, `.crumb a[aria-current="page"]{color:var(--muted);text-decoration:none}`) {
		t.Fatalf("current breadcrumb segment should remain visibly clickable, got CSS:\n%s", css)
	}
}

func TestPrimaryPagesUseFluidFullWidthSurface(t *testing.T) {
	v := makeVault(t)
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_static/style.css", nil)
	s.ServeHTTP(w, r)
	css := w.Body.String()
	for _, want := range []string{
		`--measure:100%`,
		`.reading-surface{max-width:none`,
		`.note .content>p,.note .content>ul,.note .content>ol,.note .content>blockquote,.note .content>details,.note .content>h1,.note .content>h2,.note .content>h3,.note .content>h4,.note .content>h5,.note .content>h6{width:100%;margin-left:0;margin-right:0}`,
		`.content .contains-task-list{padding-left:0;list-style:none;display:grid;gap:12px;max-width:none;margin-left:0;margin-right:0}`,
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("missing fluid full-width layout CSS %q in:\n%s", want, css)
		}
	}
	for _, unwanted := range []string{
		`--measure:1180px`,
		`.reading-surface{max-width:var(--measure);margin-inline:auto}`,
		`width:min(100%,var(--measure))`,
		`max-width:var(--measure);margin-left:auto;margin-right:auto`,
	} {
		if strings.Contains(css, unwanted) {
			t.Fatalf("layout still contains restrictive width CSS %q in:\n%s", unwanted, css)
		}
	}
}

func TestSearchPageShowsQuerySyntaxHelp(t *testing.T) {
	v := makeVault(t)
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_search?q=tag%3Adaily+target", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	for _, want := range []string{
		`value="tag:daily target"`,
		`Search syntax`,
		`tag:daily`,
		`path:&quot;Areas/Daily Briefings&quot;`,
		`frontmatter:title=&quot;Daily Briefing&quot;`,
		`&quot;exact phrase&quot;`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing search UI help %q in:\n%s", want, body)
		}
	}
}

func TestTODOShowsCopyableTaskIDs(t *testing.T) {
	v := makeVault(t)
	note, err := v.ReadNote("Areas/TODO.md")
	if err != nil {
		t.Fatal(err)
	}
	doc := NewRenderer(v).Render(note)
	for _, want := range []string{`class="task-id"`, `data-copy="1c496356"`, `tid:1c496356`, `data-copy="149d256b"`} {
		if !strings.Contains(doc.HTML, want) {
			t.Fatalf("missing %q in TODO HTML:\n%s", want, doc.HTML)
		}
	}
}

func TestTaskMetadataRendersAsReadableBadges(t *testing.T) {
	v := makeVault(t)
	note, err := v.ReadNote("Areas/TODO.md")
	if err != nil {
		t.Fatal(err)
	}
	doc := NewRenderer(v).Render(note)
	for _, want := range []string{
		`<span class="task-meta due-date" title="Due date">Due 2026-05-19</span>`,
		`<span class="task-meta done-date" title="Done date">Done 2026-05-20</span>`,
	} {
		if !strings.Contains(doc.HTML, want) {
			t.Fatalf("missing task metadata badge %q in TODO HTML:\n%s", want, doc.HTML)
		}
	}

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_static/style.css", nil)
	s.ServeHTTP(w, r)
	css := w.Body.String()
	for _, want := range []string{
		".contains-task-list{padding-left:0;list-style:none}",
		".task-list-item{display:flex;align-items:baseline;gap:10px;",
		".task-meta{display:inline-flex",
		".done-date",
		".due-date",
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("missing task CSS %q in:\n%s", want, css)
		}
	}
}

func TestFrontendAssetsAreEmbeddedFilesWithoutNewFrameworkLayers(t *testing.T) {
	for _, path := range []string{
		"templates/layout.html",
		"templates/tag.html",
		"static/style.css",
		"static/app.js",
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected frontend asset file %s: %v", path, err)
		}
	}

	uiSource, err := os.ReadFile("ui.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, unwanted := range []string{"const templates = `", "const css = `", "const js = `"} {
		if strings.Contains(string(uiSource), unwanted) {
			t.Fatalf("ui.go should embed extracted assets, not keep raw string %q", unwanted)
		}
	}

	for _, path := range []string{"templates/layout.html", "static/style.css", "static/app.js"} {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		content := string(b)
		lower := strings.ToLower(content)
		for _, unwanted := range []string{"alpine", "htmx"} {
			if strings.Contains(lower, unwanted) {
				t.Fatalf("%s should not introduce frontend framework layer %q", path, unwanted)
			}
		}
		for _, leakedGoRawString := range []string{"const templates = `", "const css = `", "const js = `"} {
			if strings.Contains(content, leakedGoRawString) {
				t.Fatalf("%s should not contain leaked ui.go raw string marker %q", path, leakedGoRawString)
			}
		}
	}
}

func TestTagsPagesAndBadges(t *testing.T) {
	v := makeVault(t)
	s := NewServer(v, "", "")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_tags", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	for _, want := range []string{`<h1>Tags</h1>`, `href="/_tags/daily"`, `#daily`} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing tags index markup %q in:\n%s", want, body)
		}
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_tags/daily", nil)
	s.ServeHTTP(w, r)
	body = w.Body.String()
	for _, want := range []string{
		`<div class="page-header tag-detail-header">`,
		`<p class="eyebrow">Tag</p><h1>#daily</h1>`,
		`<span class="count">1 note</span>`,
		`<a class="btn ghost" href="/_tags">All tags</a>`,
		`<button class="copy-link btn ghost" data-copy-link>Copy link</button>`,
		`<ul class="note-card-grid tag-note-list">`,
		`<li class="note-card tag-note-card">`,
		`Daily Briefing`,
		`/Areas/Daily%20Briefings/2026-05-22-briefing.md`,
		`Areas/Daily Briefings/2026-05-22-briefing.md`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing polished tag detail markup %q in:\n%s", want, body)
		}
	}
	for _, unwanted := range []string{`{{define "tag"}}{{template "layout-start" .}}<h1>#`, `<ul class="list"><li><a href="/Areas/Daily%20Briefings/2026-05-22-briefing.md">Daily Briefing</a>`} {
		if strings.Contains(body, unwanted) {
			t.Fatalf("tag detail should not use bare legacy list markup %q in:\n%s", unwanted, body)
		}
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/Areas/Daily%20Briefings/2026-05-22-briefing.md", nil)
	s.ServeHTTP(w, r)
	body = w.Body.String()
	if !strings.Contains(body, `class="tag-badge chip" href="/_tags/daily"`) {
		t.Fatalf("missing note tag badge in:\n%s", body)
	}
}

func TestDashboardSummarizesDailyTodosAndLinkHealth(t *testing.T) {
	v := makeVault(t)
	dashboard, err := v.BuildDashboard()
	if err != nil {
		t.Fatal(err)
	}
	if dashboard.LatestDaily == nil || dashboard.LatestDaily.RelPath != "Areas/Daily Briefings/2026-05-22-briefing.md" {
		t.Fatalf("unexpected latest daily: %+v", dashboard.LatestDaily)
	}
	if len(dashboard.OpenTasks) != 1 || dashboard.OpenTasks[0].ID != "1c496356" || dashboard.OpenTasks[0].Due != "2026-05-19" {
		t.Fatalf("unexpected open tasks: %+v", dashboard.OpenTasks)
	}
	if dashboard.BrokenLinkCount != 1 {
		t.Fatalf("broken link count=%d want 1", dashboard.BrokenLinkCount)
	}
	if dashboard.OrphanNoteCount == 0 {
		t.Fatalf("expected at least one orphan note")
	}

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	for _, want := range []string{
		`<h2>Open TODOs</h2>`,
		`Change Captur tires`,
		`href="/_todo"`,
		`Broken links`,
		`Orphan notes`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing dashboard markup %q in:\n%s", want, body)
		}
	}
}

func TestDashboardLinkHealthUsesIndexResolver(t *testing.T) {
	v := makeVault(t)
	idx, err := v.BuildIndex()
	if err != nil {
		t.Fatal(err)
	}
	resolver := NewIndexResolver(idx)
	if got := resolver.Resolve("Target"); got.Kind != "unique" || got.RelPath != "Areas/Target.md" {
		t.Fatalf("unexpected Target resolution: %+v", got)
	}
	if got := resolver.Resolve("Meeting Notes"); got.Kind != "ambiguous" {
		t.Fatalf("unexpected Meeting Notes resolution: %+v", got)
	}
	if got := resolver.Resolve("Missing"); got.Kind != "missing" {
		t.Fatalf("unexpected Missing resolution: %+v", got)
	}
	if broken := CountBrokenWikiLinks(idx, resolver); broken != 1 {
		t.Fatalf("broken links=%d want 1", broken)
	}
	if orphans := CountOrphanNotes(idx, resolver); orphans == 0 {
		t.Fatalf("expected orphan notes")
	}
}

func TestMarkdownFilesSkipHiddenDirectories(t *testing.T) {
	v := makeVault(t)
	hidden := filepath.Join(v.Root, ".claude", "Hidden.md")
	if err := os.MkdirAll(filepath.Dir(hidden), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(hidden, []byte("# Hidden\n[[MissingHidden]]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	configuredHidden := filepath.Join(v.Root, "Private", "Hidden.md")
	if err := os.MkdirAll(filepath.Dir(configuredHidden), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configuredHidden, []byte("# Config Hidden\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("favorites:\n  - Areas/Daily Briefings\nhidden:\n  - Private\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, p := range v.MarkdownFiles() {
		rel := v.Rel(p)
		if strings.HasPrefix(rel, ".claude/") || strings.HasPrefix(rel, "Private/") {
			t.Fatalf("hidden markdown file should be skipped, got %s in %+v", rel, v.MarkdownFiles())
		}
	}
	idx, err := v.BuildIndex()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := idx.ByRel[".claude/Hidden.md"]; ok {
		t.Fatalf("hidden dotdir note should not be indexed")
	}
	if _, ok := idx.ByRel["Private/Hidden.md"]; ok {
		t.Fatalf("configured hidden note should not be indexed")
	}
}

func TestBrokenLinksAndOrphanNotesPages(t *testing.T) {
	v := makeVault(t)
	idx, err := v.BuildIndex()
	if err != nil {
		t.Fatal(err)
	}
	resolver := NewIndexResolver(idx)
	broken := BrokenWikiLinks(idx, resolver)
	if len(broken) != 1 || broken[0].Target != "Missing" || broken[0].Source.RelPath != "Areas/Daily Briefings/2026-05-22-briefing.md" {
		t.Fatalf("unexpected broken links: %+v", broken)
	}
	orphans := OrphanNotes(idx, resolver)
	if len(orphans) == 0 || !noteMetaSliceContainsRel(orphans, "Areas/TODO.md") {
		t.Fatalf("expected TODO orphan note in: %+v", orphans)
	}

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_broken-links", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	for _, want := range []string{
		`<h1>Broken links</h1>`,
		`Missing`,
		`Daily Briefing`,
		`Hello [[Target|the target]] and [[Missing]].`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing broken links markup %q in:\n%s", want, body)
		}
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_orphans", nil)
	s.ServeHTTP(w, r)
	body = w.Body.String()
	for _, want := range []string{
		`<h1>Orphan notes</h1>`,
		`Areas/TODO.md`,
		`href="/Areas/TODO.md"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing orphan notes markup %q in:\n%s", want, body)
		}
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/", nil)
	s.ServeHTTP(w, r)
	body = w.Body.String()
	for _, want := range []string{`href="/_broken-links"`, `href="/_orphans"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("dashboard metric should link to detail page %q in:\n%s", want, body)
		}
	}
}

func noteMetaSliceContainsRel(items []NoteMeta, rel string) bool {
	for _, item := range items {
		if item.RelPath == rel {
			return true
		}
	}
	return false
}

func TestTODOViewGroupsTasksByDueDateAndStatus(t *testing.T) {
	v := makeVault(t)
	path := filepath.Join(v.Root, "Areas", "TODO.md")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.WriteString("- [ ] Pay invoice 📅 2026-05-20 <!-- tid:today123 -->\n- [ ] Plan trip 📅 2026-06-01 <!-- tid:future123 -->\n- [ ] Read later <!-- tid:nodate123 -->\n")
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		t.Fatal(err)
	}

	board, err := v.BuildTaskBoard("2026-05-20")
	if err != nil {
		t.Fatal(err)
	}
	if len(board.Overdue) != 1 || board.Overdue[0].ID != "1c496356" {
		t.Fatalf("unexpected overdue tasks: %+v", board.Overdue)
	}
	if len(board.Today) != 1 || board.Today[0].ID != "today123" {
		t.Fatalf("unexpected today tasks: %+v", board.Today)
	}
	if len(board.Upcoming) != 1 || board.Upcoming[0].ID != "future123" {
		t.Fatalf("unexpected upcoming tasks: %+v", board.Upcoming)
	}
	if len(board.NoDate) != 1 || board.NoDate[0].ID != "nodate123" {
		t.Fatalf("unexpected no-date tasks: %+v", board.NoDate)
	}
	if len(board.Done) != 1 || board.Done[0].ID != "149d256b" {
		t.Fatalf("unexpected done tasks: %+v", board.Done)
	}

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/_todo?today=2026-05-20", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	for _, want := range []string{
		`<h1>TODOs</h1>`,
		`Overdue <span class="count">`,
		`Change Captur tires`,
		`Today <span class="count">`,
		`Pay invoice`,
		`Upcoming <span class="count">`,
		`Plan trip`,
		`No due date <span class="count">`,
		`Read later`,
		`Done <span class="count">`,
		`Buy dog food`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing TODO view markup %q in:\n%s", want, body)
		}
	}
}

func TestNoteLinksShowForwardLinksAndBacklinkContext(t *testing.T) {
	v := makeVault(t)
	outgoing := v.ForwardLinksFrom(Note{RelPath: "Areas/Daily Briefings/2026-05-22-briefing.md", Body: "See [[Target|alias]] and [[Missing]]."})
	if len(outgoing) != 2 || outgoing[0].Target != "Target" || outgoing[0].Kind != "unique" || outgoing[1].Kind != "missing" {
		t.Fatalf("unexpected outgoing links: %+v", outgoing)
	}
	backlinks := v.BacklinksWithContext("Areas/Target.md")
	foundLinkerContext := false
	for _, backlink := range backlinks {
		if backlink.Source.RelPath == "Areas/Linker.md" && strings.Contains(backlink.Context, "See [[Target]]") {
			foundLinkerContext = true
		}
	}
	if !foundLinkerContext {
		t.Fatalf("expected Linker backlink context in: %+v", backlinks)
	}

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Areas/Target.md", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	for _, want := range []string{
		`Forward links <span class="count">`,
		`No forward links.`,
		`Backlinks <span class="count">`,
		`Areas/Linker.md`,
		`See [[Target]]`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing link markup %q in:\n%s", want, body)
		}
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/Areas/Daily%20Briefings/2026-05-22-briefing.md", nil)
	s.ServeHTTP(w, r)
	body = w.Body.String()
	for _, want := range []string{
		`Forward links <span class="count">`,
		`href="/Areas/Target.md"`,
		`class="missing-link"`,
		`Missing`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing forward link markup %q in:\n%s", want, body)
		}
	}
}

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
	for _, want := range []string{"copyText", "navigator.clipboard", "execCommand", "data-copy", "notes-web:sidebar-open", "data-tree-path", "localStorage"} {
		if !strings.Contains(js, want) {
			t.Fatalf("missing %q in app.js:\n%s", want, js)
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
		".content>:where(pre,table,.mermaid,img){max-width:100%}",
		".content p{margin:0 0 1.05rem}",
		".content h2{margin:2.2rem 0 1rem}",
		".content pre{overflow:auto;max-width:100%;",
		".content table{display:block;overflow-x:auto;max-width:100%;",
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
		`class="todo-board"`,
		`class="todo-column overdue"`,
		`Overdue <span class="count">1</span>`,
		`class="task-card"`,
		`class="task-checkbox"`,
		`class="task-date overdue-date"`,
		`title="Copy task ID"`,
		`class="task-id"`,
		`<details class="todo-column done"`,
		`<summary><h2>Done <span class="count">1</span></h2></summary>`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing TODO polish markup %q in:\n%s", want, body)
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
