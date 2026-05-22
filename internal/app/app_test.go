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

func TestReadingComfortControlsAndLayout(t *testing.T) {
	v := makeVault(t)
	s := NewServer(v, "", "")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Areas/Daily%20Briefings/2026-05-22-briefing.md", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	for _, want := range []string{
		`data-focus-toggle`,
		`data-font-size-select`,
		`aria-label="Toggle reading focus"`,
		`<article class="note reading-surface">`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing reading control markup %q in:\n%s", want, body)
		}
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_static/style.css", nil)
	s.ServeHTTP(w, r)
	css := w.Body.String()
	for _, want := range []string{
		`.reading-surface{max-width:var(--measure)`,
		`.note .content table`,
		`width:min(100%,var(--measure))`,
		`body.reading-focus .side`,
		`[data-font-size="large"]`,
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("missing reading comfort CSS %q in:\n%s", want, css)
		}
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_static/app.js", nil)
	s.ServeHTTP(w, r)
	js := w.Body.String()
	for _, want := range []string{
		`notes-web:font-size`,
		`notes-web:reading-focus`,
		`initReadingControls`,
		`data-font-size-select`,
		`data-focus-toggle`,
	} {
		if !strings.Contains(js, want) {
			t.Fatalf("missing reading comfort JS %q in:\n%s", want, js)
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
		`<span class="task-meta due-date" title="Due date">📅 2026-05-19</span>`,
		`<span class="task-meta done-date" title="Done date">✅ 2026-05-20</span>`,
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
	for _, want := range []string{`<h1>#daily</h1>`, `Daily Briefing`, `/Areas/Daily%20Briefings/2026-05-22-briefing.md`} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing tag detail markup %q in:\n%s", want, body)
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
		"--measure:78ch",
		".content{font-size:17px;line-height:1.72;overflow-wrap:anywhere}",
		".content>:where(p,ul,ol,blockquote,details,dl){max-width:var(--measure)}",
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
		"/_api/palette",
		"metaKey",
		"ev.key === '/'",
		"data-palette-results",
		"location.href = item.url",
	} {
		if !strings.Contains(js, want) {
			t.Fatalf("missing palette JS %q in:\n%s", want, js)
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
		`<label class="theme-picker">Theme`,
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
		".theme-picker{display:flex",
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
		`class="todo-group overdue"`,
		`Overdue <span class="count">1</span>`,
		`class="task-row"`,
		`class="task-date overdue-date"`,
		`title="Copy task ID"`,
		`class="task-id"`,
		`<details class="todo-group done"`,
		`<summary><h2>Done <span class="count">1</span></h2></summary>`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing TODO polish markup %q in:\n%s", want, body)
		}
	}
}

func TestTagsPageUsesProgressiveDisclosureAndControls(t *testing.T) {
	v := makeVault(t)
	// Create enough one-off tags to prove rare tags are not dumped into the main view.
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
		`class="tag-stats"`,
		`class="tag-controls"`,
		`placeholder="Filter tags…"`,
		`Popular tags`,
		`Alphabetical index`,
		`Rare tags`,
		`data-tag-filter`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing progressive tags markup %q in:\n%s", want, body)
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
