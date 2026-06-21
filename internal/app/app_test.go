package app

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
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
	must(".notes-web.yaml", "sidebar:\n  favorites:\n    items:\n      - path: Areas/Daily Briefings\n        label: Daily Briefings\ndaily_glob: Areas/Daily Briefings/*-briefing.md\n")
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

func TestConfiguredHiddenPathsAreNonEnumeratedButDirectURLAddressable(t *testing.T) {
	v := makeVault(t)
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("sidebar:\n  favorites:\n    items:\n      - path: Areas/Secret\n        label: Secret\n      - path: Areas/Hidden.md\n        label: Hidden\nhidden:\n  - Areas/Secret\n  - Areas/Hidden.md\n"), 0o644); err != nil {
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

	// Configured hidden paths must be excluded from enumeration (tree).
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

	// Configured hidden paths MUST be accessible by direct URL (new semantics).
	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/Areas/Secret/Note.md", nil)
	s.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("configured hidden note direct URL status=%d want 200, body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "# Secret") && !strings.Contains(w.Body.String(), "Secret") {
		t.Fatalf("configured hidden note content should be served, got:\n%s", w.Body.String())
	}
}

func TestFavoritesUseConfiguredPathAndLabel(t *testing.T) {
	v := makeVault(t)
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("sidebar:\n  favorites:\n    items:\n      - path: Areas/Daily Briefings\n        label: Briefings\n      - path: _todo\n        label: Todos\n      - path: Projects/\n        label: Projects\n"), 0o644); err != nil {
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

func TestTopLevelFavoritesConfigIsIgnored(t *testing.T) {
	v := makeVault(t)
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("favorites:\n  - path: Areas/Daily Briefings\n    label: Legacy Briefings\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, fav := range v.Favorites() {
		if fav.Label == "Legacy Briefings" {
			t.Fatalf("top-level favorites should be ignored, got %+v", v.Favorites())
		}
	}
}

func TestFavoriteRequiresConfiguredLabel(t *testing.T) {
	v := makeVault(t)
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("sidebar:\n  favorites:\n    items:\n      - path: _todo\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if favorites := v.Favorites(); len(favorites) != 0 {
		t.Fatalf("favorite without label should be ignored, got %+v", favorites)
	}
}

func TestStructuredUIVisibilityConfigHidesUIOnly(t *testing.T) {
	v := makeVault(t)
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("sidebar:\n  explore:\n    visible: false\n  favorites:\n    items:\n      - path: Areas/Daily Briefings\n        label: Briefings\nhomepage:\n  blocks:\n    calendar:\n      visible: false\n    todos:\n      visible: false\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := v.LoadConfig()
	ui := cfg.UI()
	if !ui.HideHomepageCalendar || !ui.HideHomepageTodos || !ui.HideSidebarExplore {
		t.Fatalf("structured visibility config did not populate UI flags: %+v", ui)
	}

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	for _, unwanted := range []string{
		`<h3>Explore</h3>`,
		`data-home-block="calendar"`,
		`data-home-block="todos"`,
		`Open TODO dashboard`,
	} {
		if strings.Contains(body, unwanted) {
			t.Fatalf("hidden UI block leaked %q in:\n%s", unwanted, body)
		}
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_todo?today=2026-05-20", nil)
	s.ServeHTTP(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `class="todo-shell"`) {
		t.Fatalf("hidden TODO should remain directly accessible, status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestSidebarFavoritesVisibleFalseHidesFavoritesFromSidebarHomeAndPalette(t *testing.T) {
	v := makeVault(t)
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("sidebar:\n  favorites:\n    visible: false\n    items:\n      - path: Areas/Daily Briefings\n        label: My Briefings\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if favs := v.Favorites(); len(favs) != 0 {
		t.Fatalf("Favorites() should return empty when visible=false, got %+v", favs)
	}

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()

	// Sidebar: Favorites section heading and star icon should be absent
	for _, unwanted := range []string{
		`<h3>Favorites</h3>`,
		`nav-icon-favorite`,
	} {
		if strings.Contains(body, unwanted) {
			t.Fatalf("hidden favorites leaked %q in:\n%s", unwanted, body)
		}
	}

	// Quick-jump: the hardcoded "Briefings" link is not a favorite;
	// verify the configured favorite label "My Briefings" does not appear
	if strings.Contains(body, `My Briefings`) {
		t.Fatalf("quick-jump should not contain My Briefings favorite link:\n%s", body)
	}

	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/_api/palette", nil)
	s.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("palette API status = %d, want %d", w.Code, http.StatusOK)
	}
	json := w.Body.String()
	if strings.Contains(json, `"kind":"favorite"`) {
		t.Fatalf("palette API should not include favorites when hidden, got:\n%s", json)
	}
}

func TestHiddenBlocksFavoritesHidesFavorites(t *testing.T) {
	v := makeVault(t)
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("hidden_blocks:\n  - favorites\nsidebar:\n  favorites:\n    items:\n      - path: Areas/Daily Briefings\n        label: Briefings\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if favs := v.Favorites(); len(favs) != 0 {
		t.Fatalf("Favorites() should return empty when hidden_blocks includes favorites, got %+v", favs)
	}

	cfg := v.LoadConfig()
	if !cfg.UI().HideSidebarFavorites {
		t.Fatal("HideSidebarFavorites should be true when hidden_blocks includes favorites")
	}
}

func TestHomepageTodoVisibilityDoesNotHideSidebarTodo(t *testing.T) {
	v := makeVault(t)
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("sidebar:\n  favorites:\n    items:\n      - path: Areas/Daily Briefings\n        label: Briefings\nhomepage:\n  blocks:\n    todos:\n      visible: false\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	if strings.Contains(body, `data-home-block="todos"`) {
		t.Fatalf("homepage TODO block should be hidden:\n%s", body)
	}
	if !strings.Contains(body, `<h3>Explore</h3>`) || !strings.Contains(body, `href="/_todo"`) {
		t.Fatalf("sidebar Explore TODO link should remain visible when only homepage todos are hidden:\n%s", body)
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

func TestFolderPageCapsAt100Entries(t *testing.T) {
	v := makeVault(t)
	// Create 250 notes in a test folder.
	dir := "LargeFolder"
	for i := 0; i < 250; i++ {
		rel := fmt.Sprintf("%s/Note%03d.md", dir, i)
		p := filepath.Join(v.Root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("# Note\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/"+dir, nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	if !strings.Contains(body, "Showing first 100 of 250") {
		t.Fatalf("expected cap message for 250 entries, got:\n%s", body)
	}
	// Should render roughly 100 list items.
	itemCount := strings.Count(body, `<li><a`)
	if itemCount < 90 || itemCount > 110 {
		t.Fatalf("expected about 100 <li><a items, got %d", itemCount)
	}
	// Sort links should still be present.
	if !strings.Contains(body, `sort=name&amp;dir=asc`) {
		t.Fatalf("folder sort links should remain when capped:\n%s", body)
	}

	// Small folder (under 100) should not be capped.
	smallDir := "SmallFolder"
	for i := 0; i < 5; i++ {
		rel := fmt.Sprintf("%s/File%03d.md", smallDir, i)
		p := filepath.Join(v.Root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("# Small\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest("GET", "/"+smallDir, nil)
	s.ServeHTTP(w2, r2)
	body2 := w2.Body.String()
	if strings.Contains(body2, "Showing first 100") {
		t.Fatalf("small folder should not show cap message:\n%s", body2)
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

func TestMarkdownTablesAreWrappedForMobileHorizontalScroll(t *testing.T) {
	v := makeVault(t)
	note := Note{Path: "table.md", Body: strings.Join([]string{
		"# Table",
		"",
		"| Categorie | Date | Status | Valeur |",
		"|-----------|------|--------|--------|",
		"| sante | 2026-05-30 | active | Une valeur longue |",
	}, "\n")}
	doc := NewRenderer(v).Render(note)
	for _, want := range []string{`<div class="markdown-table-wrap"><table>`, `<th>Categorie</th>`, `</table></div>`} {
		if !strings.Contains(doc.HTML, want) {
			t.Fatalf("markdown table missing mobile wrapper %q in:\n%s", want, doc.HTML)
		}
	}
}

func TestLargeCodeFenceIsShortened(t *testing.T) {
	v := makeVault(t)
	// 300 lines of code should trigger the 200-line cap.
	var lines []string
	for i := 0; i < 300; i++ {
		lines = append(lines, fmt.Sprintf("line %d", i))
	}
	fence := "```go\n" + strings.Join(lines, "\n") + "\n```"
	body := "# Large code\n\n" + fence + "\n"
	doc := NewRenderer(v).Render(Note{RelPath: "Large.md", Body: body})
	html := doc.HTML
	if !strings.Contains(html, "Large code block shortened") {
		t.Fatalf("expected code fence shortening note, got:\n%s", html)
	}
	if !strings.Contains(html, "Showing first 200 of 300") {
		t.Fatalf("expected 'Showing first 200 of 300' note, got:\n%s", html)
	}
	// Short code fence (50 lines) should not be truncated.
	var shortLines []string
	for i := 0; i < 50; i++ {
		shortLines = append(shortLines, fmt.Sprintf("short line %d", i))
	}
	shortFence := "```py\n" + strings.Join(shortLines, "\n") + "\n```"
	shortBody := "# Small code\n\n" + shortFence + "\n"
	shortDoc := NewRenderer(v).Render(Note{RelPath: "Small.md", Body: shortBody})
	if strings.Contains(shortDoc.HTML, "Large code block shortened") {
		t.Fatalf("short code fence should not be shortened:\n%s", shortDoc.HTML)
	}

	longLineBody := "# Long line\n\n```txt\n" + strings.Repeat("x", 50*1024) + "\n```\n"
	longLineDoc := NewRenderer(v).Render(Note{RelPath: "LongLine.md", Body: longLineBody})
	if !strings.Contains(longLineDoc.HTML, "Showing first 40 KiB of a very long code block") {
		t.Fatalf("single-line oversized code fence should be shortened without panic:\n%s", longLineDoc.HTML)
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
	writeNote("Projects/France-Publications.md", "---\nstatus: active\n---\n# France-Publications\n", "2026-05-22T15:30:00Z")
	writeNote("Projects/Notes-web.md", "---\nstatus: active\n---\n# Notes-web\n", "2026-05-22T14:30:00Z")
	writeNote("Projects/Amsterdam/Move.md", "---\nstatus: active\n---\n# Move\n", "2026-05-20T12:30:00Z")
	writeNote("Projects/Inactive.md", "---\nstatus: inactive\n---\n# Inactive\n", "2026-05-23T10:00:00Z")
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
	if activeProjectContains(dashboard.ActiveProjects, "Santé") {
		t.Fatalf("active projects should exclude non-Projects folders: %+v", dashboard.ActiveProjects)
	}
	if activeProjectContains(dashboard.ActiveProjects, "Inactive") {
		t.Fatalf("active projects should exclude inactive projects: %+v", dashboard.ActiveProjects)
	}

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	for _, want := range []string{
		`<div class="home-dashboard" data-home-dashboard>`,
		`<div class="home-main" data-home-column-stack="main">`,
		`<aside class="home-side" data-home-column-stack="side">`,
		`data-home-block="today" data-home-column="main"`,
		`data-home-block="active_projects" data-home-column="main"`,
		`data-home-project-filter`,
		`data-home-project-row`,
		`class="active-project-link"`,
		`France-Publications`,
		`Notes-web`,
		`data-home-block="calendar" data-home-column="side"`,
		`May 2026`,
		`class="calendar-day has-note selected" href="/?date=2026-05-22" aria-label="2026-05-22">22</a>`,
		`class="calendar-day today" href="/?date=2026-05-23" aria-label="2026-05-23">23</a>`,
		`data-home-block="selected_day" data-home-column="side"`,
		`Friday, May 22`,
		`data-home-block="quick_jump" data-home-column="side"`,
		`data-home-block="recent_notes" data-home-column="main"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing dashboard homepage markup %q in:\n%s", want, body)
		}
	}
}

func TestActiveProjectsOnlyUseProjectsFolder(t *testing.T) {
	v := makeVault(t)
	writeProjectFixture := func(rel, body, mod string) {
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
	writeProjectFixture("Projects/Alpha.md", "---\nstatus: active\n---\n# Alpha\n", "2026-06-10T10:00:00Z")
	writeProjectFixture("Projects/Beta/Plan.md", "---\nstatus: active\n---\n# Beta Plan\n", "2026-06-11T10:00:00Z")
	writeProjectFixture("Projects/Done.md", "---\nstatus: done\n---\n# Done\n", "2026-06-14T10:00:00Z")
	writeProjectFixture("Projects/NoStatus.md", "# No Status\n", "2026-06-15T10:00:00Z")
	writeProjectFixture("Areas/Gamma/Plan.md", "---\nstatus: active\n---\n# Gamma Plan\n", "2026-06-12T10:00:00Z")
	writeProjectFixture("Root.md", "# Root\n", "2026-06-13T10:00:00Z")

	idx, err := v.BuildIndex()
	if err != nil {
		t.Fatal(err)
	}
	projects := v.ActiveProjects(idx, 20)
	if !activeProjectContains(projects, "Alpha") || !activeProjectContains(projects, "Beta") {
		t.Fatalf("active projects should include direct project files and project folders: %+v", projects)
	}
	for _, project := range projects {
		if project.Label == "Gamma" || project.Label == "Root" || project.Label == "Areas" || project.Label == "Done" || project.Label == "NoStatus" {
			t.Fatalf("active projects should only include active Projects/ children: %+v", projects)
		}
		if !strings.HasPrefix(project.RelPath, "Projects/") {
			t.Fatalf("active project rel path should stay under Projects/: %+v", project)
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

func TestHomepageBlockRenderingUsesConfiguredBlocksAndQuickJumpItems(t *testing.T) {
	v := makeVault(t)
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("homepage:\n  blocks:\n    quick_jump:\n      items:\n        - label: Custom Jump\n          path: Areas/Target.md\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()

	for _, id := range []string{"today", "quick_jump", "todos", "active_projects", "calendar", "selected_day", "recent_notes", "diagnostics"} {
		marker := `data-home-block="` + id + `"`
		if count := strings.Count(body, marker); count != 1 {
			t.Fatalf("homepage block %q should render exactly once, got %d in:\n%s", id, count, body)
		}
	}
	if count := strings.Count(body, `data-home-order="`); count != 8 {
		t.Fatalf("each homepage block should expose a global mobile order, got %d markers in:\n%s", count, body)
	}
	if count := strings.Count(body, `--home-block-order:`); count != 8 {
		t.Fatalf("each homepage block should expose a CSS order variable, got %d markers in:\n%s", count, body)
	}
	if !strings.Contains(body, `class="home-shortcut" href="/Areas/Target.md"`) || !strings.Contains(body, `Custom Jump`) {
		t.Fatalf("configured quick-jump link should render in homepage block:\n%s", body)
	}
	if count := strings.Count(body, `class="home-shortcut"`); count != 1 {
		t.Fatalf("quick-jump should render only configured homepage links, got %d shortcuts:\n%s", count, body)
	}
	if strings.Contains(body, `<a class="active-project-row"`) {
		t.Fatalf("active project rows must not be anchors wrapping nested links:\n%s", body)
	}
}

func TestHomepageQuickJumpEmptyState(t *testing.T) {
	v := makeVault(t)
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("homepage:\n  blocks:\n    quick_jump:\n      items: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	if !strings.Contains(body, `data-home-block="quick_jump"`) || !strings.Contains(body, `No shortcuts configured.`) {
		t.Fatalf("empty quick-jump should render an empty state:\n%s", body)
	}
	if strings.Contains(body, `class="home-shortcut"`) {
		t.Fatalf("explicit empty quick-jump config should not render shortcuts:\n%s", body)
	}
}

func TestHomepageTodayPreviewAndEmptyState(t *testing.T) {
	oldNow := now
	now = func() time.Time { return time.Date(2026, 5, 22, 12, 0, 0, 0, time.Local) }
	defer func() { now = oldNow }()

	v := makeVault(t)
	// Create a daily note matching the default daily_notes_glob pattern
	dailyRel := "Daily Notes/2026/2026-05/2026-05-22.md"
	p := filepath.Join(v.Root, filepath.FromSlash(dailyRel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("# May 22 Daily Note\nToday content with Heading One.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	for _, want := range []string{`data-home-block="today"`, `2026-05-22`, `class="home-today-preview content"`, `May 22 Daily Note`, `class="btn" href="/Daily%20Notes/2026/2026-05/2026-05-22.md"`, `Open full note`} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing today preview markup %q in:\n%s", want, body)
		}
	}
	if strings.Contains(body, `class="btn primary" href="/Daily%20Notes/2026/2026-05/2026-05-22.md"`) {
		t.Fatalf("open full note should use the standard card action button, not primary:\n%s", body)
	}

	// Empty state: use a date query for a date without any daily note or briefing
	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/?date=2099-01-01", nil)
	s.ServeHTTP(w, r)
	body = w.Body.String()
	if !strings.Contains(body, `2099-01-01`) || !strings.Contains(body, `No daily note for this date.`) {
		t.Fatalf("today block should show empty state when no daily note exists for selected date:\n%s", body)
	}
}

func TestHomepageTodosRenderOnlyOverdueAndTodaySections(t *testing.T) {
	oldNow := now
	now = func() time.Time { return time.Date(2026, 5, 20, 12, 0, 0, 0, time.Local) }
	defer func() { now = oldNow }()

	v := makeVault(t)
	todoPath := filepath.Join(v.Root, "Areas", "TODO.md")
	f, err := os.OpenFile(todoPath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.WriteString("- [ ] Pay invoice #admin 📅 2026-05-20 <!-- tid:today123 -->\n- [ ] Plan trip #travel 📅 2026-06-01 <!-- tid:future123 -->\n- [ ] Read later #inbox <!-- tid:nodate123 -->\n")
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		t.Fatal(err)
	}

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	for _, want := range []string{`data-home-block="todos"`, `class="todo-section overdue"`, `id="home-todo-overdue">Late`, `class="todo-section today"`, `id="home-todo-today">Today`, `today123`, `Change Captur tires`} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing homepage TODO markup %q in:\n%s", want, body)
		}
	}
	for _, unwanted := range []string{`todo-section upcoming`, `todo-section no-date`, `todo-section done`, `future123`, `nodate123`, `149d256b`} {
		if strings.Contains(body, unwanted) {
			t.Fatalf("homepage TODO block should not include %q:\n%s", unwanted, body)
		}
	}
}

func TestHomepageOrderDefaultIncludesAllBlocks(t *testing.T) {
	v := makeVault(t)
	cfg := v.LoadConfig()
	blocks := cfg.OrderedVisibleBlocks()
	ids := make([]string, len(blocks))
	for i, b := range blocks {
		ids[i] = b.ID
	}
	want := []string{"today", "calendar", "todos", "active_projects", "selected_day", "quick_jump", "recent_notes", "diagnostics"}
	if len(ids) != len(want) {
		t.Fatalf("got %d blocks %v, want %d %v", len(ids), ids, len(want), want)
	}
	for i, id := range ids {
		if id != want[i] {
			t.Fatalf("block[%d] = %q, want %q", i, id, want[i])
		}
	}
}

func TestHomepageOrderSkipsUnknownIDsAndAppendsMissing(t *testing.T) {
	v := makeVault(t)
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("homepage:\n  order:\n    - today\n    - nonexistent_block\n    - diagnostics\n    - todos\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := v.LoadConfig()
	blocks := cfg.OrderedVisibleBlocks()
	ids := make([]string, len(blocks))
	for i, b := range blocks {
		ids[i] = b.ID
	}
	// expected: today, diagnostics, todos, then remaining defaults in order
	expected := []string{"today", "diagnostics", "todos", "calendar", "active_projects", "selected_day", "quick_jump", "recent_notes"}
	if len(ids) != len(expected) {
		t.Fatalf("got %d blocks %v, want %d %v", len(ids), ids, len(expected), expected)
	}
	for i, id := range ids {
		if id != expected[i] {
			t.Fatalf("block[%d] = %q, want %q", i, id, expected[i])
		}
	}
}

func TestHomepageBlockColumns(t *testing.T) {
	v := makeVault(t)
	cfg := v.LoadConfig()
	blocks := cfg.OrderedVisibleBlocks()
	mainIDs := map[string]bool{"today": true, "todos": true, "active_projects": true, "recent_notes": true}
	for _, b := range blocks {
		if mainIDs[b.ID] && b.Column != "main" {
			t.Fatalf("block %q should be in main column, got %q", b.ID, b.Column)
		}
		if !mainIDs[b.ID] && b.Column != "side" {
			t.Fatalf("block %q should be in side column, got %q", b.ID, b.Column)
		}
	}
}

func TestHomepageViewDesktopColumnStacks(t *testing.T) {
	v := makeVault(t)
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("homepage:\n  order:\n    - quick_jump\n    - today\n    - calendar\n    - todos\n    - selected_day\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewServer(v, "", "")
	hv := s.buildHomepageView(v.LoadConfig(), Dashboard{})
	for i, b := range hv.Blocks {
		if b.Order != i {
			t.Fatalf("block %q order = %d, want %d in %+v", b.ID, b.Order, i, hv.Blocks)
		}
	}
	mainIDs := make([]string, len(hv.MainBlocks))
	for i, b := range hv.MainBlocks {
		mainIDs[i] = b.ID
		if b.Column != "main" {
			t.Fatalf("main stack contains side block: %+v", hv.MainBlocks)
		}
	}
	sideIDs := make([]string, len(hv.SideBlocks))
	for i, b := range hv.SideBlocks {
		sideIDs[i] = b.ID
		if b.Column != "side" {
			t.Fatalf("side stack contains main block: %+v", hv.SideBlocks)
		}
	}
	if got, want := strings.Join(mainIDs, ","), "today,todos,active_projects,recent_notes"; got != want {
		t.Fatalf("main stack order = %q, want %q", got, want)
	}
	if got, want := strings.Join(sideIDs, ","), "quick_jump,calendar,selected_day,diagnostics"; got != want {
		t.Fatalf("side stack order = %q, want %q", got, want)
	}
	if hv.SideBlocks[0].Order != 0 || hv.MainBlocks[0].Order != 1 || hv.SideBlocks[1].Order != 2 || hv.MainBlocks[1].Order != 3 {
		t.Fatalf("column stacks should keep original global order indices, main=%+v side=%+v", hv.MainBlocks, hv.SideBlocks)
	}
}

func TestHomepageBlockVisibility(t *testing.T) {
	v := makeVault(t)
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("homepage:\n  blocks:\n    today:\n      visible: false\n    todos:\n      visible: false\n    calendar:\n      visible: false\n    active_projects:\n      visible: false\n    recent_notes:\n      visible: false\n    diagnostics:\n      visible: false\n    selected_day:\n      visible: false\n    quick_jump:\n      visible: false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := v.LoadConfig()
	blocks := cfg.OrderedVisibleBlocks()
	if len(blocks) != 0 {
		t.Fatalf("expected 0 blocks when all are hidden, got %+v", blocks)
	}
}

func TestHomepageBlockTodayVisibility(t *testing.T) {
	v := makeVault(t)
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("homepage:\n  blocks:\n    today:\n      visible: false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := v.LoadConfig()
	blocks := cfg.OrderedVisibleBlocks()
	for _, b := range blocks {
		if b.ID == "today" {
			t.Fatalf("today block should be hidden when visible=false, got %+v", blocks)
		}
	}
	// All other blocks should still be present
	found := map[string]bool{}
	for _, b := range blocks {
		found[b.ID] = true
	}
	for _, id := range []string{"quick_jump", "todos", "active_projects", "calendar", "selected_day", "recent_notes", "diagnostics"} {
		if !found[id] {
			t.Fatalf("block %q should still be visible when only today is hidden: %+v", id, blocks)
		}
	}
}

func TestHomepageOrderFiltersHiddenBlocks(t *testing.T) {
	v := makeVault(t)
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("homepage:\n  order:\n    - todos\n    - calendar\n    - today\n  blocks:\n    todos:\n      visible: false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := v.LoadConfig()
	blocks := cfg.OrderedVisibleBlocks()
	ids := make([]string, len(blocks))
	for i, b := range blocks {
		ids[i] = b.ID
	}
	// todos is hidden, so we should get: calendar, today, then the rest in default order
	expected := []string{"calendar", "today", "active_projects", "selected_day", "quick_jump", "recent_notes", "diagnostics"}
	if len(ids) != len(expected) {
		t.Fatalf("got %d blocks %v, want %d %v", len(ids), ids, len(expected), expected)
	}
	for i, id := range ids {
		if id != expected[i] {
			t.Fatalf("block[%d] = %q, want %q", i, id, expected[i])
		}
	}
}

func TestLegacyHomepageBlockTodoAliasIgnored(t *testing.T) {
	v := makeVault(t)
	// The old "todo" key should NOT be parsed by Config — only "todos" is canonical.
	// Write a config with both legacy todo and canonical todos visible=false.
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("homepage:\n  blocks:\n    todo:\n      visible: false\n    todos:\n      visible: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := v.LoadConfig()
	// canonical todos should be visible (not hidden)
	if cfg.Homepage.Blocks.Todos.Hidden() {
		t.Fatal("canonical todos should NOT be hidden when only legacy todo is set to false")
	}
	if cfg.UI().HideHomepageTodos {
		t.Fatal("HideHomepageTodos should be false when only legacy todo is hidden")
	}
}

func TestQuickJumpDefaults(t *testing.T) {
	v := makeVault(t)
	items := v.QuickJumpItems()
	if len(items) != 4 {
		t.Fatalf("default quick-jump should have 4 items, got %d: %+v", len(items), items)
	}
	expected := []struct{ label, url string }{
		{"Today", "/"},
		{"TODO", "/_todo"},
		{"Search", "/_search"},
		{"Daily Briefings", "/Areas/Daily%20Briefings"},
	}
	for i, want := range expected {
		if items[i].Label != want.label || items[i].URL != want.url {
			t.Fatalf("item[%d] = %+v, want label=%q url=%q", i, items[i], want.label, want.url)
		}
	}
}

func TestQuickJumpExplicitEmpty(t *testing.T) {
	v := makeVault(t)
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("homepage:\n  blocks:\n    quick_jump:\n      items: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	items := v.QuickJumpItems()
	if len(items) != 0 {
		t.Fatalf("explicit empty quick_jump should return 0 items, got %d: %+v", len(items), items)
	}
}

func TestQuickJumpCustomItems(t *testing.T) {
	v := makeVault(t)
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("homepage:\n  blocks:\n    quick_jump:\n      items:\n        - label: My Label\n          path: /custom\n        - label: Vault Path\n          path: Areas/Target.md\n        - label: TODO\n          path: _todo\n        - label: \"\"\n          path: /empty-label\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	items := v.QuickJumpItems()
	// Should skip empty label, keep the rest (3 items)
	if len(items) != 3 {
		t.Fatalf("expected 3 quick-jump items (empty label skipped), got %d: %+v", len(items), items)
	}
	if items[0].Label != "My Label" || items[0].URL != "/custom" {
		t.Fatalf("item[0] = %+v, want label=My Label url=/custom", items[0])
	}
	if items[1].Label != "Vault Path" || items[1].URL != "/Areas/Target.md" {
		t.Fatalf("item[1] = %+v, want url=/Areas/Target.md", items[1])
	}
	if items[2].Label != "TODO" || items[2].URL != "/_todo" {
		t.Fatalf("item[2] = %+v, want url=/_todo", items[2])
	}
}

func TestQuickJumpSkipsHiddenVaultPath(t *testing.T) {
	v := makeVault(t)
	// Create a hidden file and reference it in quick_jump
	if err := os.WriteFile(filepath.Join(v.Root, "Areas", "Secret.md"), []byte("# Secret\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("hidden:\n  - Areas/Secret.md\nhomepage:\n  blocks:\n    quick_jump:\n      items:\n        - label: Secret\n          path: Areas/Secret.md\n        - label: Visible\n          path: Areas/Target.md\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	items := v.QuickJumpItems()
	if len(items) != 1 || items[0].Label != "Visible" {
		t.Fatalf("expected only visible item, got %+v", items)
	}
}

func TestQuickJumpSkipsTODOWhenHidden(t *testing.T) {
	v := makeVault(t)
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("hidden_blocks:\n  - todo\nhomepage:\n  blocks:\n    quick_jump:\n      items:\n        - label: TODO\n          path: _todo\n        - label: Search\n          path: _search\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	items := v.QuickJumpItems()
	if len(items) != 1 || items[0].Label != "Search" {
		t.Fatalf("expected only Search item when todo hidden, got %+v", items)
	}
}

func TestHiddenBlocksCalendarHidesOnlyCalendar(t *testing.T) {
	v := makeVault(t)
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("hidden_blocks:\n  - calendar\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := v.LoadConfig()
	if !cfg.UI().HideHomepageCalendar {
		t.Fatal("HideHomepageCalendar should be true when hidden_blocks includes calendar")
	}
	// selected_day should still be visible
	if cfg.Homepage.Blocks.SelectedDay.Hidden() {
		t.Fatal("selected_day should NOT be hidden when only calendar is hidden")
	}
	blocks := cfg.OrderedVisibleBlocks()
	for _, b := range blocks {
		if b.ID == "selected_day" {
			return // found it
		}
	}
	t.Fatalf("selected_day block should appear in OrderedVisibleBlocks when only calendar is hidden: %+v", blocks)
}

func TestHiddenBlocksTodosHidesHomepageAndSidebarTodo(t *testing.T) {
	v := makeVault(t)
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("hidden_blocks:\n  - todos\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := v.LoadConfig()
	// hidden_blocks: [todos] normalizes to "todo" and hides both homepage and sidebar TODO
	if !cfg.UI().HideHomepageTodos {
		t.Fatal("HideHomepageTodos should be true when hidden_blocks includes todos")
	}
	if !cfg.UI().HideSidebarTodo {
		t.Fatal("HideSidebarTodo should ALSO be true when hidden_blocks includes todos (normalizes to 'todo')")
	}
}

func TestHiddenBlocksTodoSingularAlsoHidesBoth(t *testing.T) {
	v := makeVault(t)
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("hidden_blocks:\n  - todo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := v.LoadConfig()
	if !cfg.UI().HideHomepageTodos {
		t.Fatal("HideHomepageTodos should be true when hidden_blocks includes todo")
	}
	if !cfg.UI().HideSidebarTodo {
		t.Fatal("HideSidebarTodo should be true when hidden_blocks includes todo")
	}
}

func TestHiddenBlocksFavoritesDoesNotHideQuickJump(t *testing.T) {
	v := makeVault(t)
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("hidden_blocks:\n  - favorites\nhomepage:\n  blocks:\n    quick_jump:\n      items:\n        - label: Custom\n          path: /custom\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Quick-jump items should still be accessible (not driven by sidebar favorites)
	items := v.QuickJumpItems()
	if len(items) != 1 || items[0].Label != "Custom" {
		t.Fatalf("quick-jump should still work when favorites are hidden, got %+v", items)
	}
}

func TestHomepageViewTodoBucketsOverdueAndTodayOnly(t *testing.T) {
	oldNow := now
	now = func() time.Time { return time.Date(2026, 5, 20, 12, 0, 0, 0, time.Local) }
	defer func() { now = oldNow }()

	v := makeVault(t)
	// Append more tasks to test
	todoPath := filepath.Join(v.Root, "Areas", "TODO.md")
	f, err := os.OpenFile(todoPath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.WriteString("- [ ] Pay invoice #admin 📅 2026-05-20 <!-- tid:today123 -->\n- [ ] Plan trip #travel 📅 2026-06-01 <!-- tid:future123 -->\n- [ ] Read later #inbox <!-- tid:nodate123 -->\n")
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		t.Fatal(err)
	}

	s := NewServer(v, "", "")
	hv := s.buildHomepageView(v.LoadConfig(), Dashboard{})

	if len(hv.TodoOverdue) != 1 || hv.TodoOverdue[0].ID != "1c496356" {
		t.Fatalf("expected 1 overdue task (Change Captur tires), got %+v", hv.TodoOverdue)
	}
	if len(hv.TodoToday) != 1 || hv.TodoToday[0].ID != "today123" {
		t.Fatalf("expected 1 today task (Pay invoice), got %+v", hv.TodoToday)
	}
	// No upcoming, no-date, or done should appear
	taskIDs := map[string]bool{}
	for _, t := range hv.TodoOverdue {
		taskIDs[t.ID] = true
	}
	for _, t := range hv.TodoToday {
		taskIDs[t.ID] = true
	}
	if taskIDs["future123"] {
		t.Fatal("upcoming task should not appear in homepage view")
	}
	if taskIDs["nodate123"] {
		t.Fatal("no-date task should not appear in homepage view")
	}
	if taskIDs["149d256b"] {
		t.Fatal("done task should not appear in homepage view")
	}
}

func TestDailyForDateSelectsByNowDateNotLatestModified(t *testing.T) {
	v := makeVault(t)
	// Create a daily note for today and one for tomorrow (latest modified)
	todayStr := time.Now().Format("2006-01-02")
	tomorrowStr := time.Now().AddDate(0, 0, 1).Format("2006-01-02")

	writeDaily := func(dateStr string, body string) {
		p := filepath.Join(v.Root, "Areas", "Daily Briefings", dateStr+"-briefing.md")
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		// Make tomorrow's note newer
		if dateStr == tomorrowStr {
			mt := time.Now().Add(1 * time.Hour)
			if err := os.Chtimes(p, mt, mt); err != nil {
				t.Fatal(err)
			}
		}
	}
	writeDaily(todayStr, "# Today Note\n")
	writeDaily(tomorrowStr, "# Tomorrow Note\n")

	// DailyForDate should find today's note even though tomorrow's is newer
	note := v.DailyForDate(todayStr)
	if note == nil {
		t.Fatalf("DailyForDate(%q) should find today's note", todayStr)
	}
	if !strings.Contains(note.Body, "# Today Note") {
		t.Fatalf("DailyForDate returned wrong note: %+v", note)
	}

	// Should return nil for dates without a daily
	missing := v.DailyForDate("1999-01-01")
	if missing != nil {
		t.Fatalf("DailyForDate for non-existent date should return nil, got %+v", missing)
	}
}

func TestActiveProjectsLimitDefaultAndConfigured(t *testing.T) {
	v := makeVault(t)
	// Default limit should be 20
	cfg := v.LoadConfig()
	if limit := cfg.Homepage.Blocks.ActiveProjects.LimitOrDefault(20); limit != 20 {
		t.Fatalf("default active projects limit should be 20, got %d", limit)
	}
	// Configured limit
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("homepage:\n  blocks:\n    active_projects:\n      limit: 3\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg = v.LoadConfig()
	if limit := cfg.Homepage.Blocks.ActiveProjects.LimitOrDefault(20); limit != 3 {
		t.Fatalf("configured active projects limit should be 3, got %d", limit)
	}
	// Non-positive limit uses default
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("homepage:\n  blocks:\n    active_projects:\n      limit: 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg = v.LoadConfig()
	if limit := cfg.Homepage.Blocks.ActiveProjects.LimitOrDefault(20); limit != 20 {
		t.Fatalf("zero active projects limit should use default 20, got %d", limit)
	}
	// Negative limit uses default
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("homepage:\n  blocks:\n    active_projects:\n      limit: -5\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg = v.LoadConfig()
	if limit := cfg.Homepage.Blocks.ActiveProjects.LimitOrDefault(20); limit != 20 {
		t.Fatalf("negative active projects limit should use default 20, got %d", limit)
	}
}

func TestRecentNotesLimitDefaultAndConfigured(t *testing.T) {
	v := makeVault(t)
	cfg := v.LoadConfig()
	if limit := cfg.Homepage.Blocks.RecentNotes.LimitOrDefault(10); limit != 10 {
		t.Fatalf("default recent notes limit should be 10, got %d", limit)
	}
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("homepage:\n  blocks:\n    recent_notes:\n      limit: 5\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg = v.LoadConfig()
	if limit := cfg.Homepage.Blocks.RecentNotes.LimitOrDefault(10); limit != 5 {
		t.Fatalf("configured recent notes limit should be 5, got %d", limit)
	}
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("homepage:\n  blocks:\n    recent_notes:\n      limit: 0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg = v.LoadConfig()
	if limit := cfg.Homepage.Blocks.RecentNotes.LimitOrDefault(10); limit != 10 {
		t.Fatalf("zero recent notes limit should use default 10, got %d", limit)
	}
}

func TestHomepageViewBlocksPresentInTemplateData(t *testing.T) {
	v := makeVault(t)
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	s.ServeHTTP(w, r)
	// The HomepageView should drive the rendered homepage and keep legacy template keys available.
	body := w.Body.String()
	if !strings.Contains(body, `class="home-dashboard"`) {
		t.Fatalf("homepage dashboard should still render with existing template:\n%s", body)
	}
}

func TestHomepageQuickJumpItemsResolvedInServerHome(t *testing.T) {
	v := makeVault(t)
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("homepage:\n  blocks:\n    quick_jump:\n      items:\n        - label: Custom Jump\n          path: Areas/Target.md\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	items := v.QuickJumpItems()
	if len(items) != 1 || items[0].URL != "/Areas/Target.md" {
		t.Fatalf("custom quick-jump item not resolved correctly: %+v", items)
	}
}

func TestHomepageBlockCalendarSelectedDayIndependence(t *testing.T) {
	v := makeVault(t)
	// Hide calendar only
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("homepage:\n  blocks:\n    calendar:\n      visible: false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := v.LoadConfig()
	if !cfg.Homepage.Blocks.Calendar.Hidden() {
		t.Fatal("calendar should be hidden")
	}
	if cfg.Homepage.Blocks.SelectedDay.Hidden() {
		t.Fatal("selected_day should NOT be hidden when only calendar is hidden")
	}
	blocks := cfg.OrderedVisibleBlocks()
	foundSelectedDay := false
	for _, b := range blocks {
		if b.ID == "selected_day" {
			foundSelectedDay = true
		}
		if b.ID == "calendar" {
			t.Fatal("calendar block should be absent from OrderedVisibleBlocks")
		}
	}
	if !foundSelectedDay {
		t.Fatal("selected_day should still be in OrderedVisibleBlocks")
	}

	// Hide selected_day only
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("homepage:\n  blocks:\n    selected_day:\n      visible: false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg = v.LoadConfig()
	if cfg.Homepage.Blocks.Calendar.Hidden() {
		t.Fatal("calendar should NOT be hidden when only selected_day is hidden")
	}
	if !cfg.Homepage.Blocks.SelectedDay.Hidden() {
		t.Fatal("selected_day should be hidden")
	}
	blocks = cfg.OrderedVisibleBlocks()
	foundCalendar := false
	for _, b := range blocks {
		if b.ID == "selected_day" {
			t.Fatal("selected_day block should be absent")
		}
		if b.ID == "calendar" {
			foundCalendar = true
		}
	}
	if !foundCalendar {
		t.Fatal("calendar should still be in OrderedVisibleBlocks")
	}
}

func TestQuickJumpDefaultsRespectHiddenBlocksTodo(t *testing.T) {
	v := makeVault(t)
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("hidden_blocks:\n  - todo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	items := v.QuickJumpItems()
	// TODO item should be filtered out, default items: Today, Search, Daily Briefings
	if len(items) != 3 {
		t.Fatalf("default quick-jump should have 3 items when todo is hidden, got %d: %+v", len(items), items)
	}
	for _, item := range items {
		if item.Label == "TODO" || item.URL == "/_todo" {
			t.Fatalf("TODO should not appear in quick-jump when hidden_blocks hides todo: %+v", items)
		}
	}
}

func TestQuickJumpDefaultsRespectHiddenBlocksTodos(t *testing.T) {
	v := makeVault(t)
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("hidden_blocks:\n  - todos\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	items := v.QuickJumpItems()
	if len(items) != 3 {
		t.Fatalf("default quick-jump should have 3 items when todos is hidden, got %d: %+v", len(items), items)
	}
	for _, item := range items {
		if item.Label == "TODO" || item.URL == "/_todo" {
			t.Fatalf("TODO should not appear when hidden_blocks hides todos: %+v", items)
		}
	}
}

func TestQuickJumpDefaultDailyBriefingsHidden(t *testing.T) {
	v := makeVault(t)
	// Hide Areas/Daily Briefings path
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("hidden:\n  - Areas/Daily Briefings\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	items := v.QuickJumpItems()
	// Default items: Today, TODO, Search (Daily Briefings path is hidden)
	if len(items) != 3 {
		t.Fatalf("expected 3 default items when Daily Briefings is hidden, got %d: %+v", len(items), items)
	}
	for _, item := range items {
		if item.Label == "Daily Briefings" {
			t.Fatalf("Daily Briefings should be skipped when its vault path is hidden: %+v", items)
		}
	}
}

func TestQuickJumpSlashPrefixedVaultPathRespectsHidden(t *testing.T) {
	v := makeVault(t)
	// Create a hidden file
	if err := os.WriteFile(filepath.Join(v.Root, "Areas", "Secret.md"), []byte("# Secret\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("hidden:\n  - Areas/Secret.md\nhomepage:\n  blocks:\n    quick_jump:\n      items:\n        - label: Secret\n          path: /Areas/Secret.md\n        - label: Visible\n          path: /Areas/Target.md\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	items := v.QuickJumpItems()
	if len(items) != 1 || items[0].Label != "Visible" {
		t.Fatalf("expected only Visible item when /Areas/Secret.md is hidden, got %+v", items)
	}
	if items[0].URL != "/Areas/Target.md" {
		t.Fatalf("Visible item URL should be /Areas/Target.md, got %q", items[0].URL)
	}
}

func TestQuickJumpUnderscorePrefixedVaultPathRespectsHidden(t *testing.T) {
	v := makeVault(t)
	if err := os.WriteFile(filepath.Join(v.Root, "_secret.md"), []byte("# Secret\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("hidden:\n  - _secret.md\nhomepage:\n  blocks:\n    quick_jump:\n      items:\n        - label: Secret\n          path: /_secret.md\n        - label: Todo\n          path: /_todo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	items := v.QuickJumpItems()
	if len(items) != 1 || items[0].URL != "/_todo" {
		t.Fatalf("expected hidden /_secret.md to be filtered while /_todo stays internal, got %+v", items)
	}
}

func TestDailyNoteForDateUsesDailyNotesGlob(t *testing.T) {
	v := makeVault(t)
	// Create a real daily note matching default daily_notes_glob pattern
	noteRel := "Daily Notes/2026/2026-06/2026-06-15.md"
	p := filepath.Join(v.Root, filepath.FromSlash(noteRel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("# Real Daily Note\nContent for 2026-06-15\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	note := v.DailyNoteForDate("2026-06-15")
	if note == nil {
		t.Fatalf("DailyNoteForDate should find note at %s", noteRel)
	}
	if !strings.Contains(note.Body, "Real Daily Note") {
		t.Fatalf("DailyNoteForDate returned wrong note: %+v", note)
	}
	if note.RelPath != noteRel {
		t.Fatalf("DailyNoteForDate relpath=%q, want %q", note.RelPath, noteRel)
	}
}

func TestDailyNoteForDateMissingReturnsNil(t *testing.T) {
	v := makeVault(t)
	note := v.DailyNoteForDate("1999-01-01")
	if note != nil {
		t.Fatalf("DailyNoteForDate for non-existent date should return nil, got %+v", note)
	}
}

func TestHomepageDateQueryRendersSelectedDateDailyNote(t *testing.T) {
	oldNow := now
	now = func() time.Time { return time.Date(2026, 6, 18, 12, 0, 0, 0, time.Local) }
	defer func() { now = oldNow }()

	v := makeVault(t)
	// Create a daily note for June 15 matching daily_notes_glob
	dailyRel := "Daily Notes/2026/2026-06/2026-06-15.md"
	p := filepath.Join(v.Root, filepath.FromSlash(dailyRel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("# June 15 Note\nThis is the selected date daily note.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Also create a briefing for June 15 to prove the daily note wins
	briefRel := "Areas/Daily Briefings/2026-06-15-briefing.md"
	p2 := filepath.Join(v.Root, filepath.FromSlash(briefRel))
	if err := os.MkdirAll(filepath.Dir(p2), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p2, []byte("# June 15 Briefing\nThis is the briefing and should NOT appear.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/?date=2026-06-15", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()

	// The daily note content should appear; the briefing content should NOT
	if !strings.Contains(body, "This is the selected date daily note.") {
		t.Fatalf("homepage should render the selected date daily note, got:\n%s", body)
	}
	if strings.Contains(body, "This is the briefing and should NOT appear") {
		t.Fatalf("homepage should NOT render the briefing when a daily note exists:\n%s", body)
	}
	if !strings.Contains(body, `href="/Daily%20Notes/2026/2026-06/2026-06-15.md"`) {
		t.Fatalf("open full note should point to the selected daily note:\n%s", body)
	}
}

func TestHomepageDateQueryMissingDailyNoteShowsEmptyState(t *testing.T) {
	oldNow := now
	now = func() time.Time { return time.Date(2026, 6, 18, 12, 0, 0, 0, time.Local) }
	defer func() { now = oldNow }()

	v := makeVault(t)
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/?date=1999-01-01", nil)
	s.ServeHTTP(w, r)
	// Should not crash; should render homepage with empty daily note state
	body := w.Body.String()
	if !strings.Contains(body, `class="home-dashboard"`) {
		t.Fatalf("homepage should render even when selected date has no daily note:\n%s", body)
	}
	if !strings.Contains(body, `1999-01-01`) || !strings.Contains(body, `No daily note for this date.`) {
		t.Fatalf("homepage should show selected-date empty state when no daily note exists:\n%s", body)
	}
}

func TestHomepageViewDashboardConsistentWithDateQuery(t *testing.T) {
	v := makeVault(t)
	// Patch now() for deterministic date
	oldNow := now
	now = func() time.Time { return time.Date(2026, 5, 23, 12, 0, 0, 0, time.Local) }
	defer func() { now = oldNow }()

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/?date=2026-05-21", nil)
	s.ServeHTTP(w, r)
	body := w.Body.String()
	// The old .Dashboard key should show the selected date
	if !strings.Contains(body, `Thursday, May 21`) {
		t.Fatalf("dashboard should show selected date Thursday, May 21:\n%s", body)
	}
	// The quick-jump block should still be present (structural marker)
	if !strings.Contains(body, `data-home-block="quick_jump"`) {
		t.Fatalf("quick-jump card should still be present:\n%s", body)
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
		`.home-dashboard{display:grid;grid-template-columns:minmax(0,1fr) minmax(260px,320px)`,
		`.home-main,.home-side{min-width:0;display:grid;gap:18px;align-content:start}`,
		`.home-side{position:sticky;top:18px}`,
		`@media(max-width:900px){.home-dashboard{grid-template-columns:1fr}`,
		`.home-main,.home-side{display:contents}`,
		`.home-block{order:var(--home-block-order,0)}`,
		`.home-side{position:static}`,
		`.home-calendar`,
		`.calendar-grid{display:grid;grid-template-columns:repeat(7,minmax(0,1fr))`,
		`.calendar-day.selected`,
		`.active-project-row`,
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("missing homepage calendar/project CSS %q in:\n%s", want, css)
		}
	}
	if strings.Contains(css, `grid-auto-flow:dense`) || strings.Contains(css, `grid-auto-flow: dense`) {
		t.Fatalf("homepage grid must not use dense auto-placement:\n%s", css)
	}
	if strings.Contains(css, `.home-block[data-home-column=main]{grid-column`) || strings.Contains(css, `.home-block[data-home-column=side]{grid-column`) {
		t.Fatalf("homepage desktop packing should use column stacks, not per-block grid-column rules:\n%s", css)
	}
}

// ---------------------------------------------------------------------------
// Dataview table action HTTP handler tests
// ---------------------------------------------------------------------------

func TestDataviewTableActionSuccess(t *testing.T) {
	v := makeDataviewVault(t)
	writeDataviewFixture(t, v, "Dashboards/Filtered.md",
		"# Filtered\n\n"+
			"```dataview\n"+
			`TABLE status, file.link FROM "Projects" FILTER status DEFAULT "active" CLEARABLE SORT file.name`+"\n"+
			"```\n")
	defer func() {
		os.RemoveAll(filepath.Join(v.Root, "Dashboards", "Filtered.md"))
	}()

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Dashboards/Filtered.md?action=renderDataviewTable&table=1&filter.status=done", nil)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	if !strings.Contains(body, `data-dataview-action="renderDataviewTable"`) {
		t.Fatalf("response missing action data attribute:\n%s", body)
	}
	if !strings.Contains(body, `data-dataview-table="1"`) {
		t.Fatalf("response missing table index:\n%s", body)
	}
	// Should filter to "done" status — only Beta.
	if !strings.Contains(body, `href="/Projects/Beta.md">Beta</a>`) {
		t.Fatalf("response should include Beta (done):\n%s", body)
	}
	if strings.Contains(body, `href="/Projects/Alpha.md">Alpha</a>`) {
		t.Fatalf("response should NOT include Alpha (not done):\n%s", body)
	}
}

func TestDataviewTableActionImplicitCapMatchesStaticRender(t *testing.T) {
	v := makeDataviewVault(t)
	for i := 0; i < 20; i++ {
		writeDataviewFixture(t, v, fmt.Sprintf("CapAction/Note%02d.md", i), "---\ntitle: Note "+fmt.Sprint(i)+"\n---\n# Note\n")
	}
	writeDataviewFixture(t, v, "Dashboards/StaticCap.md",
		"# StaticCap\n\n"+
			"```dataview\n"+
			`TABLE file.link FROM "CapAction" SORT file.name`+"\n"+
			"```\n")
	defer func() {
		os.RemoveAll(filepath.Join(v.Root, "CapAction"))
		os.Remove(filepath.Join(v.Root, "Dashboards", "StaticCap.md"))
	}()

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Dashboards/StaticCap.md?action=renderDataviewTable&table=1", nil)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "Showing first 10 of 20") {
		t.Fatalf("static AJAX render should include the same cap note as full render:\n%s", body)
	}
	if got := strings.Count(body, `/CapAction/Note`); got != 10 {
		t.Fatalf("expected 10 capped rows, got %d in:\n%s", got, body)
	}
}

func TestDataviewTableActionFilterTablesAreNotImplicitlyCapped(t *testing.T) {
	v := makeDataviewVault(t)
	for i := 0; i < 12; i++ {
		writeDataviewFixture(t, v, fmt.Sprintf("CapActionFilter/Active%02d.md", i), "---\ntitle: Active "+fmt.Sprint(i)+"\nstatus: active\n---\n# Active\n")
	}
	for i := 0; i < 2; i++ {
		writeDataviewFixture(t, v, fmt.Sprintf("CapActionFilter/Done%02d.md", i), "---\ntitle: Done "+fmt.Sprint(i)+"\nstatus: done\n---\n# Done\n")
	}
	writeDataviewFixture(t, v, "Dashboards/FilterCapAction.md",
		"# FilterCapAction\n\n"+
			"```dataview\n"+
			`TABLE status, file.link FROM "CapActionFilter" FILTER status DEFAULT "active" CLEARABLE SORT file.name`+"\n"+
			"```\n")
	defer func() {
		os.RemoveAll(filepath.Join(v.Root, "CapActionFilter"))
		os.Remove(filepath.Join(v.Root, "Dashboards", "FilterCapAction.md"))
	}()

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Dashboards/FilterCapAction.md?action=renderDataviewTable&table=1&filter.status=active", nil)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if strings.Contains(body, "dataview-cap-note") {
		t.Fatalf("interactive FILTER AJAX render should not be implicitly capped:\n%s", body)
	}
	if got := strings.Count(body, `/CapActionFilter/Active`); got != 12 {
		t.Fatalf("expected all 12 active rows, got %d in:\n%s", got, body)
	}
	if strings.Contains(body, `/CapActionFilter/Done`) {
		t.Fatalf("active filter should exclude done rows:\n%s", body)
	}
}

func TestDataviewTableActionMethodNotAllowed(t *testing.T) {
	v := makeDataviewVault(t)
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()

	// POST is not allowed for the action.
	r := httptest.NewRequest("POST", "/Dashboards/Dataview.md?action=renderDataviewTable&table=1", nil)
	s.ServeHTTP(w, r)

	if w.Code != 405 {
		t.Fatalf("expected 405 for POST, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "dataview-error") {
		t.Fatalf("response should contain dataview-error:\n%s", w.Body.String())
	}
}

func TestDataviewTableActionUnknownAction(t *testing.T) {
	v := makeDataviewVault(t)
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Dashboards/Dataview.md?action=unknown", nil)
	s.ServeHTTP(w, r)

	if w.Code != 400 {
		t.Fatalf("expected 400 for unknown action, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Unknown action") {
		t.Fatalf("response should mention unknown action:\n%s", w.Body.String())
	}
}

func TestDataviewTableActionMissingTable(t *testing.T) {
	v := makeDataviewVault(t)
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Dashboards/Dataview.md?action=renderDataviewTable", nil)
	s.ServeHTTP(w, r)

	if w.Code != 400 {
		t.Fatalf("expected 400 for missing table, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDataviewTableActionInvalidTable(t *testing.T) {
	v := makeDataviewVault(t)
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Dashboards/Dataview.md?action=renderDataviewTable&table=abc", nil)
	s.ServeHTTP(w, r)

	if w.Code != 400 {
		t.Fatalf("expected 400 for invalid table, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDataviewTableActionOutOfRangeTable(t *testing.T) {
	v := makeDataviewVault(t)
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Dashboards/Dataview.md?action=renderDataviewTable&table=99", nil)
	s.ServeHTTP(w, r)

	if w.Code != 400 {
		t.Fatalf("expected 400 for out-of-range table, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "out of range") {
		t.Fatalf("response should mention out of range:\n%s", w.Body.String())
	}
}

func TestDataviewTableActionNonMarkdown(t *testing.T) {
	v := makeDataviewVault(t)
	writeDataviewFixture(t, v, "Assets/img.png", "fake image\n")
	defer func() {
		os.RemoveAll(filepath.Join(v.Root, "Assets"))
	}()

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Assets/img.png?action=renderDataviewTable&table=1", nil)
	s.ServeHTTP(w, r)

	if w.Code != 400 {
		t.Fatalf("expected 400 for non-markdown, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Markdown note") {
		t.Fatalf("response should mention Markdown:\n%s", w.Body.String())
	}
}

func TestDataviewTableActionMissingNote(t *testing.T) {
	v := makeDataviewVault(t)
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Nonexistent.md?action=renderDataviewTable&table=1", nil)
	s.ServeHTTP(w, r)

	// Non-existent path: should get 404 with dataview-error fragment.
	if w.Code != 404 {
		t.Fatalf("expected 404 for missing note, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "dataview-error") {
		t.Fatalf("response should contain dataview-error fragment:\n%s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Note not found") {
		t.Fatalf("response should mention note not found:\n%s", w.Body.String())
	}
}

func TestDataviewTableActionConfiguredHiddenPath(t *testing.T) {
	v := makeDataviewVault(t)
	// Create a hidden note with a dataview table.
	hiddenPath := filepath.Join(v.Root, "Areas", "Secret.md")
	if err := os.MkdirAll(filepath.Dir(hiddenPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(hiddenPath, []byte("# Secret\n\n```dataview\nTABLE status, file.link FROM \"Projects\" SORT file.name\n```\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("hidden:\n  - Areas/Secret.md\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewServer(v, "", "")
	// Configured hidden path + dataview action is now allowed (direct-read semantics).
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Areas/Secret.md?action=renderDataviewTable&table=1", nil)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("expected 200 for configured hidden note dataview action, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `href="/Projects/Alpha.md"`) {
		t.Fatalf("dataview table on configured hidden note should render project rows:\n%s", w.Body.String())
	}
}

func TestDataviewTableActionAuthProtected(t *testing.T) {
	v := makeDataviewVault(t)
	s := NewServer(v, "user", "pass")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Dashboards/Dataview.md?action=renderDataviewTable&table=1", nil)
	// No auth header.
	s.ServeHTTP(w, r)

	if w.Code != 401 {
		t.Fatalf("expected 401 without auth, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDataviewTableActionUndeclaredFilter(t *testing.T) {
	v := makeDataviewVault(t)
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Dashboards/Dataview.md?action=renderDataviewTable&table=1&filter.undeclared=value", nil)
	s.ServeHTTP(w, r)

	if w.Code != 400 {
		t.Fatalf("expected 400 for undeclared filter, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "undeclared") {
		t.Fatalf("response should mention undeclared:\n%s", w.Body.String())
	}
}

func TestDataviewTableActionSortWithoutDir(t *testing.T) {
	v := makeDataviewVault(t)
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Dashboards/Dataview.md?action=renderDataviewTable&table=1&sort=status", nil)
	s.ServeHTTP(w, r)

	if w.Code != 400 {
		t.Fatalf("expected 400 for sort without dir, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "sort") || !strings.Contains(w.Body.String(), "dir") {
		t.Fatalf("response should mention sort requires dir:\n%s", w.Body.String())
	}
}

func TestDataviewTableActionDirWithoutSort(t *testing.T) {
	v := makeDataviewVault(t)
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Dashboards/Dataview.md?action=renderDataviewTable&table=1&dir=desc", nil)
	s.ServeHTTP(w, r)

	if w.Code != 400 {
		t.Fatalf("expected 400 for dir without sort, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDataviewTableActionPathParamRejected(t *testing.T) {
	v := makeDataviewVault(t)
	s := NewServer(v, "", "")
	// Include path param — should be rejected as security measure.
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Dashboards/Dataview.md?action=renderDataviewTable&table=1&path=/evil", nil)
	s.ServeHTTP(w, r)

	if w.Code != 400 {
		t.Fatalf("expected 400 for path param, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Path parameter") {
		t.Fatalf("response should mention path parameter rejection:\n%s", w.Body.String())
	}
}

func TestDataviewTableActionEmptyPathParamRejected(t *testing.T) {
	v := makeDataviewVault(t)
	s := NewServer(v, "", "")
	// Empty path= param — should also be rejected (key presence check).
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Dashboards/Dataview.md?action=renderDataviewTable&table=1&path=", nil)
	s.ServeHTTP(w, r)

	if w.Code != 400 {
		t.Fatalf("expected 400 for empty path param, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Path parameter") {
		t.Fatalf("response should mention path parameter rejection:\n%s", w.Body.String())
	}
}

func TestDataviewTableActionRepeatedSingleFilter(t *testing.T) {
	v := makeDataviewVault(t)
	writeDataviewFixture(t, v, "Dashboards/FilterSingle.md",
		"# FilterSingle\n\n"+
			"```dataview\n"+
			`TABLE status, file.link FROM "Projects" FILTER status DEFAULT "active" CLEARABLE SORT file.name`+"\n"+
			"```\n")
	defer func() {
		os.RemoveAll(filepath.Join(v.Root, "Dashboards", "FilterSingle.md"))
	}()

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	// Repeated values for single-mode filter should be rejected, even when identical.
	r := httptest.NewRequest("GET", "/Dashboards/FilterSingle.md?action=renderDataviewTable&table=1&filter.status=active&filter.status=active", nil)
	s.ServeHTTP(w, r)

	if w.Code != 400 {
		t.Fatalf("expected 400 for repeated single filter, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "at most one value") {
		t.Fatalf("response should mention at most one value:\n%s", w.Body.String())
	}
}

func TestDataviewTableActionSortNonVisibleColumn(t *testing.T) {
	v := makeDataviewVault(t)
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	// sort fact_count is valid in Dataview.md but query has no fact_count as column.
	// Actually it does have fact_count as "Faits" — use nonexistent "score".
	r := httptest.NewRequest("GET", "/Dashboards/Dataview.md?action=renderDataviewTable&table=1&sort=score&dir=desc", nil)
	s.ServeHTTP(w, r)

	if w.Code != 400 {
		t.Fatalf("expected 400 for sort field not matching visible column, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "does not match any visible column") {
		t.Fatalf("response should mention non-matching sort field:\n%s", w.Body.String())
	}
}

func TestDataviewTableActionSortComplexExpression(t *testing.T) {
	v := makeDataviewVault(t)
	writeDataviewFixture(t, v, "Dashboards/ComplexSort.md",
		"# ComplexSort\n\n"+
			"```dataview\n"+
			`TABLE dateformat(file.mtime, "yyyy") as "Year", file.link FROM "Projects" SORT file.name`+"\n"+
			"```\n")
	defer func() {
		os.RemoveAll(filepath.Join(v.Root, "Dashboards", "ComplexSort.md"))
	}()

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	// Try sorting by the complex expression column (should be rejected).
	q := url.Values{
		"action": {"renderDataviewTable"},
		"table":  {"1"},
		"sort":   {"dateformat(file.mtime, \"yyyy\")"},
		"dir":    {"desc"},
	}
	r := httptest.NewRequest("GET", "/Dashboards/ComplexSort.md?"+q.Encode(), nil)
	s.ServeHTTP(w, r)

	if w.Code != 400 {
		t.Fatalf("expected 400 for complex sort expression, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "not a valid simple field") {
		t.Fatalf("response should mention invalid sort field:\n%s", w.Body.String())
	}
}

func TestDataviewTableActionEscapedLabels(t *testing.T) {
	v := makeDataviewVault(t)
	writeDataviewFixture(t, v, "Dashboards/Escaped.md",
		"# Escaped\n\n"+
			"```dataview\n"+
			`TABLE status as "<script>", file.link FROM "Projects" FILTER status DEFAULT "active" CLEARABLE SORT file.name`+"\n"+
			"```\n")
	defer func() {
		os.RemoveAll(filepath.Join(v.Root, "Dashboards", "Escaped.md"))
	}()

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Dashboards/Escaped.md?action=renderDataviewTable&table=1", nil)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	// The <script> label should be HTML-escaped.
	if strings.Contains(body, `> <script> <`) {
		t.Fatalf("dynamic label should be HTML-escaped:\n%s", body)
	}
	if !strings.Contains(body, `&lt;script&gt;`) {
		t.Fatalf("expected escaped &lt;script&gt; in:\n%s", body)
	}
}

func TestDataviewTableActionMultiFilterWithRepeatedParams(t *testing.T) {
	v := makeDataviewVault(t)
	writeDataviewFixture(t, v, "Dashboards/MultiFilter.md",
		"# MultiFilter\n\n"+
			"```dataview\n"+
			`TABLE tags, file.link FROM "Projects" FILTER tags MODE multi CLEARABLE SORT file.name`+"\n"+
			"```\n")
	defer func() {
		os.RemoveAll(filepath.Join(v.Root, "Dashboards", "MultiFilter.md"))
	}()

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	// Filter by multiple tag values via repeated params.
	r := httptest.NewRequest("GET", "/Dashboards/MultiFilter.md?action=renderDataviewTable&table=1&filter.tags=%23project&filter.tags=%23active", nil)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	// Should filter to notes with project+active tags.
	if !strings.Contains(body, `href="/Projects/Alpha.md">Alpha</a>`) {
		t.Fatalf("response should include Alpha:\n%s", body)
	}
	if !strings.Contains(body, `href="/Projects/Gamma.md">Gamma</a>`) {
		t.Fatalf("response should include Gamma:\n%s", body)
	}
}

func TestDataviewTableActionNonTableBlockError(t *testing.T) {
	v := makeDataviewVault(t)
	writeDataviewFixture(t, v, "Dashboards/NonTable.md",
		"# NonTable\n\n"+
			"```dataview\n"+
			`LIST status FROM "Projects" FILTER status DEFAULT "active"`+"\n"+
			"```\n")
	defer func() {
		os.RemoveAll(filepath.Join(v.Root, "Dashboards", "NonTable.md"))
	}()

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Dashboards/NonTable.md?action=renderDataviewTable&table=1", nil)
	s.ServeHTTP(w, r)

	if w.Code != 400 {
		t.Fatalf("expected 400 for non-TABLE query action, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "dataview-error") {
		t.Fatalf("response should contain dataview-error:\n%s", w.Body.String())
	}
}

func TestDataviewTableActionSortAndDir(t *testing.T) {
	v := makeDataviewVault(t)
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	// Use a column expression as sort field (fact_count is a column).
	r := httptest.NewRequest("GET", "/Dashboards/Dataview.md?action=renderDataviewTable&table=1&sort=fact_count&dir=desc", nil)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	// With sort=fact_count&dir=desc, Gamma (12) should come first, then Alpha (7), then Beta (2).
	gammaIdx := strings.Index(body, "Gamma")
	alphaIdx := strings.Index(body, "Alpha")
	betaIdx := strings.Index(body, "Beta")
	if gammaIdx < 0 || alphaIdx < 0 || betaIdx < 0 {
		t.Fatalf("expected Gamma, Alpha, Beta in response:\n%s", body)
	}
	if gammaIdx > alphaIdx || alphaIdx > betaIdx {
		t.Fatalf("expected desc sort by fact_count: Gamma > Alpha > Beta:\n%s", body)
	}
	// fact_count header should have aria-sort="descending".
	if !strings.Contains(body, `aria-sort="descending"`) {
		t.Fatalf("expected aria-sort=descending for fact_count column in:\n%s", body)
	}
}

func TestDataviewTableActionInvalidDirValue(t *testing.T) {
	v := makeDataviewVault(t)
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Dashboards/Dataview.md?action=renderDataviewTable&table=1&sort=status&dir=invalid", nil)
	s.ServeHTTP(w, r)

	if w.Code != 400 {
		t.Fatalf("expected 400 for invalid dir, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "invalid dir") {
		t.Fatalf("response should mention invalid dir:\n%s", w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Source URL frontmatter validation and rendering
// ---------------------------------------------------------------------------

func TestValidSourceURL_ValidHTTPS(t *testing.T) {
	got := validSourceURL(map[string]any{"source_url": "https://example.com/article"})
	want := "https://example.com/article"
	if got != want {
		t.Fatalf("validSourceURL = %q, want %q", got, want)
	}
}

func TestValidSourceURL_ValidHTTP(t *testing.T) {
	got := validSourceURL(map[string]any{"source_url": "http://example.com"})
	want := "http://example.com"
	if got != want {
		t.Fatalf("validSourceURL = %q, want %q", got, want)
	}
}

func TestValidSourceURL_Missing(t *testing.T) {
	got := validSourceURL(map[string]any{})
	if got != "" {
		t.Fatalf("validSourceURL = %q, want empty for missing key", got)
	}
}

func TestValidSourceURL_EmptyString(t *testing.T) {
	got := validSourceURL(map[string]any{"source_url": ""})
	if got != "" {
		t.Fatalf("validSourceURL = %q, want empty for empty string", got)
	}
}

func TestValidSourceURL_NonString(t *testing.T) {
	got := validSourceURL(map[string]any{"source_url": 42})
	if got != "" {
		t.Fatalf("validSourceURL = %q, want empty for non-string", got)
	}
}

func TestValidSourceURL_RelativePath(t *testing.T) {
	got := validSourceURL(map[string]any{"source_url": "/relative/path"})
	if got != "" {
		t.Fatalf("validSourceURL = %q, want empty for relative path", got)
	}
}

func TestValidSourceURL_NoHost(t *testing.T) {
	got := validSourceURL(map[string]any{"source_url": "https://"})
	if got != "" {
		t.Fatalf("validSourceURL = %q, want empty for scheme-only URL", got)
	}
}

func TestValidSourceURL_JavascriptScheme(t *testing.T) {
	got := validSourceURL(map[string]any{"source_url": "javascript:alert(1)"})
	if got != "" {
		t.Fatalf("validSourceURL = %q, want empty for javascript: URL", got)
	}
}

func TestValidSourceURL_WhitespaceOnly(t *testing.T) {
	got := validSourceURL(map[string]any{"source_url": "   "})
	if got != "" {
		t.Fatalf("validSourceURL = %q, want empty for whitespace", got)
	}
}

func TestValidSourceURL_FTPRejected(t *testing.T) {
	got := validSourceURL(map[string]any{"source_url": "ftp://files.example.com"})
	if got != "" {
		t.Fatalf("validSourceURL = %q, want empty for ftp scheme", got)
	}
}

func TestRenderedDocSourceURL_Populated(t *testing.T) {
	v := makeVault(t)
	note := Note{
		Body: "# Test\n",
		Frontmatter: map[string]any{
			"source_url": "https://example.com/article",
		},
	}
	doc := NewRenderer(v).Render(note)
	want := "https://example.com/article"
	if doc.SourceURL != want {
		t.Fatalf("RenderedDoc.SourceURL = %q, want %q", doc.SourceURL, want)
	}
}

func TestRenderedDocSourceURL_EmptyWhenMissing(t *testing.T) {
	v := makeVault(t)
	note := Note{Body: "# Test\n"}
	doc := NewRenderer(v).Render(note)
	if doc.SourceURL != "" {
		t.Fatalf("RenderedDoc.SourceURL = %q, want empty when missing", doc.SourceURL)
	}
}

func TestRenderedDocSourceURL_EmptyWhenInvalid(t *testing.T) {
	v := makeVault(t)
	note := Note{
		Body: "# Test\n",
		Frontmatter: map[string]any{
			"source_url": "javascript:alert(1)",
		},
	}
	doc := NewRenderer(v).Render(note)
	if doc.SourceURL != "" {
		t.Fatalf("RenderedDoc.SourceURL = %q, want empty for invalid URL", doc.SourceURL)
	}
}

func TestNotePageRendersOpenURL_WhenSourceURLPresent(t *testing.T) {
	v := makeVault(t)
	rel := "Areas/ExternalRef.md"
	p := filepath.Join(v.Root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nsource_url: https://example.com/article\n---\n# External\n\nContent.\n"
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Areas/ExternalRef.md", nil)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}

	bodyHTML := w.Body.String()
	if !strings.Contains(bodyHTML, `Open URL`) {
		t.Fatalf("expected Open URL link in:\n%s", bodyHTML)
	}
	if !strings.Contains(bodyHTML, `href="https://example.com/article"`) {
		t.Fatalf("expected correct href in Open URL link:\n%s", bodyHTML)
	}
}

func TestNotePageDoesNotRenderOpenURL_WhenSourceURLMissing(t *testing.T) {
	v := makeVault(t)
	rel := "Areas/NoSource.md"
	p := filepath.Join(v.Root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "# No Source\n\nNo frontmatter.\n"
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Areas/NoSource.md", nil)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}

	bodyHTML := w.Body.String()
	if strings.Contains(bodyHTML, `Open URL`) {
		t.Fatalf("expected NO Open URL link when source_url missing:\n%s", bodyHTML)
	}
}

func TestNotePageDoesNotRenderOpenURL_WhenSourceURLInvalid(t *testing.T) {
	v := makeVault(t)
	rel := "Areas/BadSource.md"
	p := filepath.Join(v.Root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nsource_url: javascript:alert(1)\n---\n# Bad Source\n\nContent.\n"
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Areas/BadSource.md", nil)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}

	bodyHTML := w.Body.String()
	if strings.Contains(bodyHTML, `Open URL`) {
		t.Fatalf("expected NO Open URL link when source_url is javascript:\n%s", bodyHTML)
	}
}

// ---------------------------------------------------------------------------
// Reading list prompt validation and rendering
// ---------------------------------------------------------------------------

func TestReadingListPrompt_Valid(t *testing.T) {
	fm := map[string]any{"type": "reading_list"}
	got := readingListPrompt(fm, "Areas/Articles/my-article.md")
	want := "Reading list item: Areas/Articles/my-article.md\nChoose one action and update the note frontmatter accordingly:\n- Mark as read\n- Mark as unread\n- Archive"
	if got != want {
		t.Fatalf("readingListPrompt =\n%q\nwant:\n%q", got, want)
	}
}

func TestReadingListPrompt_MissingType(t *testing.T) {
	got := readingListPrompt(map[string]any{}, "some/path.md")
	if got != "" {
		t.Fatalf("readingListPrompt = %q, want empty when type missing", got)
	}
}

func TestReadingListPrompt_WrongType(t *testing.T) {
	got := readingListPrompt(map[string]any{"type": "project"}, "some/path.md")
	if got != "" {
		t.Fatalf("readingListPrompt = %q, want empty when type is not reading_list", got)
	}
}

func TestReadingListPrompt_NonStringType(t *testing.T) {
	got := readingListPrompt(map[string]any{"type": 42}, "some/path.md")
	if got != "" {
		t.Fatalf("readingListPrompt = %q, want empty when type is not a string", got)
	}
}

func TestRenderedDocReadingListPrompt_Populated(t *testing.T) {
	v := makeVault(t)
	note := Note{
		RelPath: "Areas/Articles/article.md",
		Body:    "# Article\n",
		Frontmatter: map[string]any{
			"type": "reading_list",
		},
	}
	doc := NewRenderer(v).Render(note)
	if doc.ReadingListPrompt == "" {
		t.Fatal("ReadingListPrompt should be non-empty for reading_list type")
	}
	if !strings.Contains(doc.ReadingListPrompt, "Areas/Articles/article.md") {
		t.Fatalf("ReadingListPrompt should contain the relpath, got:\n%s", doc.ReadingListPrompt)
	}
	if !strings.Contains(doc.ReadingListPrompt, "Mark as read") {
		t.Fatalf("ReadingListPrompt should contain 'Mark as read', got:\n%s", doc.ReadingListPrompt)
	}
	if !strings.Contains(doc.ReadingListPrompt, "Mark as unread") {
		t.Fatalf("ReadingListPrompt should contain 'Mark as unread', got:\n%s", doc.ReadingListPrompt)
	}
	if !strings.Contains(doc.ReadingListPrompt, "Archive") {
		t.Fatalf("ReadingListPrompt should contain 'Archive', got:\n%s", doc.ReadingListPrompt)
	}
}

func TestRenderedDocReadingListPrompt_EmptyForOtherType(t *testing.T) {
	v := makeVault(t)
	note := Note{
		RelPath: "Areas/Article.md",
		Body:    "# Article\n",
		Frontmatter: map[string]any{
			"type": "project",
		},
	}
	doc := NewRenderer(v).Render(note)
	if doc.ReadingListPrompt != "" {
		t.Fatalf("ReadingListPrompt = %q, want empty for type=project", doc.ReadingListPrompt)
	}
}

func TestRenderedDocReadingListPrompt_EmptyForNoType(t *testing.T) {
	v := makeVault(t)
	note := Note{RelPath: "Areas/Note.md", Body: "# Note\n"}
	doc := NewRenderer(v).Render(note)
	if doc.ReadingListPrompt != "" {
		t.Fatalf("ReadingListPrompt = %q, want empty when no type frontmatter", doc.ReadingListPrompt)
	}
}

func TestNotePageRendersCopyReadingPrompt_WhenTypeReadingList(t *testing.T) {
	v := makeVault(t)
	rel := "Areas/Articles/readme.md"
	p := filepath.Join(v.Root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\ntype: reading_list\n---\n# Readme\n\nContent.\n"
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Areas/Articles/readme.md", nil)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}

	bodyHTML := w.Body.String()
	if !strings.Contains(bodyHTML, `Copy reading prompt`) {
		t.Fatalf("expected 'Copy reading prompt' button in:\n%s", bodyHTML)
	}
	// Verify the data-copy attribute contains the relpath and actions
	if !strings.Contains(bodyHTML, `Areas/Articles/readme.md`) {
		t.Fatalf("expected relpath in data-copy attribute:\n%s", bodyHTML)
	}
	if !strings.Contains(bodyHTML, `Mark as read`) {
		t.Fatalf("expected 'Mark as read' in data-copy:\n%s", bodyHTML)
	}
}

func TestNotePageDoesNotRenderCopyReadingPrompt_WhenTypeIsNotReadingList(t *testing.T) {
	v := makeVault(t)
	rel := "Areas/Articles/other.md"
	p := filepath.Join(v.Root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\ntype: project\n---\n# Other\n\nContent.\n"
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Areas/Articles/other.md", nil)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}

	bodyHTML := w.Body.String()
	if strings.Contains(bodyHTML, `Copy reading prompt`) {
		t.Fatalf("expected NO 'Copy reading prompt' button when type=project:\n%s", bodyHTML)
	}
}

func TestNotePageDoesNotRenderCopyReadingPrompt_WhenNoType(t *testing.T) {
	v := makeVault(t)
	rel := "Areas/Articles/notype.md"
	p := filepath.Join(v.Root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "# No Type\n\nNo frontmatter.\n"
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Areas/Articles/notype.md", nil)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body.String())
	}

	bodyHTML := w.Body.String()
	if strings.Contains(bodyHTML, `Copy reading prompt`) {
		t.Fatalf("expected NO 'Copy reading prompt' button when no type frontmatter:\n%s", bodyHTML)
	}
}

func TestDataviewTableActionRenderAllTables(t *testing.T) {
	v := makeDataviewVault(t)
	// Use existing dashboards: has one TABLE in Dashboards/Dataview.md
	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Dashboards/Dataview.md", nil)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("expected 200 for full page, got %d: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	// Full page should have the data-dataview attributes.
	if !strings.Contains(body, `data-dataview-action="renderDataviewTable"`) {
		t.Fatalf("full page render missing action data attribute:\n%s", body)
	}
	if !strings.Contains(body, `data-dataview-table="1"`) {
		t.Fatalf("full page render missing table index:\n%s", body)
	}
}

func TestQuestionMarkInFilename(t *testing.T) {
	v := makeVault(t)
	// Create a note with a literal "?" in the filename, simulating
	// files like "11 - Pourquoi une VM ?.md".
	noteRel := "Areas/Learning/11 - Pourquoi une VM ?.md"
	p := filepath.Join(v.Root, filepath.FromSlash(noteRel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("# Pourquoi une VM ?\n\nQuestion mark test.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	// The URL must encode "?" as %3F so it is not treated as a query separator.
	r := httptest.NewRequest("GET", "/Areas/Learning/11%20-%20Pourquoi%20une%20VM%20%3F.md", nil)
	s.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("question-mark note status=%d, want 200: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Pourquoi une VM ?") {
		t.Fatalf("question-mark note body missing expected text:\n%s", w.Body.String())
	}

	// Verify that an actual query parameter still works on the path.
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest("GET", "/Areas/Learning/11%20-%20Pourquoi%20une%20VM%20%3F.md?action=unknown", nil)
	s.ServeHTTP(w2, r2)
	// The action is unknown but the path should still resolve to the note.
	if w2.Code != 400 {
		// unknown action returns 400 but path should resolve (no 404)
		t.Fatalf("question-mark note with query status=%d, want 400: %s", w2.Code, w2.Body.String())
	}
}

func TestGzipCompressesHTMLAndCSS(t *testing.T) {
	v := makeVault(t)
	s := NewServer(v, "", "")

	tests := []struct {
		name string
		path string
	}{
		{"homepage", "/"},
		{"todo", "/_todo?today=2026-05-20"},
		{"dataview", "/_dataview"},
		{"style css", "/_static/style.css"},
		{"app js", "/_static/app.js"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", tc.path, nil)
			r.Header.Set("Accept-Encoding", "gzip")
			s.ServeHTTP(w, r)

			if w.Code != 200 {
				t.Fatalf("status=%d, want 200: %s", w.Code, w.Body.String())
			}
			ce := w.Header().Get("Content-Encoding")
			if ce != "gzip" {
				t.Fatalf("Content-Encoding = %q, want gzip", ce)
			}
			if w.Header().Get("Vary") != "Accept-Encoding" {
				t.Fatalf("Vary header = %q, want Accept-Encoding", w.Header().Get("Vary"))
			}
			// Verify body is valid gzip and decompresses.
			zr, err := gzip.NewReader(w.Body)
			if err != nil {
				t.Fatalf("gzip.NewReader: %v", err)
			}
			decompressed, err := io.ReadAll(zr)
			if err != nil {
				t.Fatalf("decompress: %v", err)
			}
			zr.Close()
			if len(decompressed) == 0 {
				t.Fatal("decompressed body is empty")
			}
		})
	}
}

func TestGzipWithoutAcceptEncoding(t *testing.T) {
	v := makeVault(t)
	s := NewServer(v, "", "")

	// Without Accept-Encoding, response must not be gzip compressed.
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("status=%d, want 200: %s", w.Code, w.Body.String())
	}
	if w.Header().Get("Content-Encoding") == "gzip" {
		t.Fatal("response without Accept-Encoding must not be gzip compressed")
	}
	if !strings.Contains(w.Body.String(), "Home") {
		t.Fatalf("uncompressed body missing expected content:\n%s", w.Body.String())
	}
}

func TestGzipAcceptEncodingQuality(t *testing.T) {
	if !acceptsGzip("br, gzip;q=0.5") {
		t.Fatal("gzip with positive q value should be accepted")
	}
	if acceptsGzip("br, gzip;q=0") {
		t.Fatal("gzip with q=0 should be rejected")
	}
}

func TestGzipSkipsBinaryServeFile(t *testing.T) {
	v := makeVault(t)
	// Create a binary file (non-Markdown) to test that ServeFile is not compressed.
	p := filepath.Join(v.Root, "Assets", "test.png")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("fake png content\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewServer(v, "", "")
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/Assets/test.png", nil)
	r.Header.Set("Accept-Encoding", "gzip")
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("status=%d, want 200: %s", w.Code, w.Body.String())
	}
	// Binary files should not be gzip compressed.
	if w.Header().Get("Content-Encoding") == "gzip" {
		t.Fatal("binary file must not be gzip compressed")
	}
	if got, want := w.Body.String(), "fake png content\n"; got != want {
		t.Fatalf("binary response body was modified by gzip wrapper: got %q, want %q", got, want)
	}
}
