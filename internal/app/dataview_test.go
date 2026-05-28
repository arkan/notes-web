package app

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeDataviewFixture(t *testing.T, v *Vault, rel, body string) {
	t.Helper()
	p := filepath.Join(v.Root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func makeDataviewVault(t *testing.T) *Vault {
	t.Helper()
	v := makeVault(t)
	writeDataviewFixture(t, v, "Projects/Alpha.md", "---\ntitle: Alpha Project\ntags: [project, active]\nstatus: active\nfact_count: 7\narea: Core\ncreated: 2026-05-01\nscore: 10\naliases:\n  - Alpha\n  - A1\n---\n# Alpha\n\n- [ ] Open alpha task 📅 2026-06-01 ⏫ #project <!-- tid:alpha -->\n- [x] Closed alpha task ✅ 2026-05-20\n")
	writeDataviewFixture(t, v, "Projects/Beta.md", "---\ntitle: Beta Project\ntags: [project]\nstatus: done\nfact_count: 2\narea: Core\ncreated: 2026-04-01\nscore: 4\n---\n# Beta\n\n- [ ] Beta task 📅 2026-07-01 🔽\n")
	writeDataviewFixture(t, v, "Projects/Gamma.md", "---\ntitle: Gamma Project\ntags: [project, active]\nstatus: active\nfact_count: 12\narea: Ops\ncreated: 2026-06-01\nscore: 8\n---\n# Gamma\n")
	writeDataviewFixture(t, v, "Areas/Tagged.md", "---\ntitle: Tagged Area\ntags:\n  - project\n  - reference\nstatus: active\n---\n# Tagged\n")
	writeDataviewFixture(t, v, "Dashboards/Dataview.md", "# Dataview Dashboard\n\n```dataview\nTABLE WITHOUT ID file.link as \"Projet\", status, fact_count as \"Faits\"\nFROM \"Projects\"\nSORT status, file.name\n```\n")
	writeDataviewFixture(t, v, "Dashboards/Multiline.md", "# Multiline\n\n```dataview\nTABLE\n  status AS \"Statut\",\n  rating AS \"Note\",\n  file.link AS \"Projet\"\nFROM \"Projects\"\nWHERE status = \"active\"\nSORT rating DESC, file.name ASC\n```\n")
	return v
}

func TestDataviewTableFromFolderSortsAndRendersFrontmatter(t *testing.T) {
	v := makeDataviewVault(t)
	n, err := v.ReadNote("Dashboards/Dataview.md")
	if err != nil {
		t.Fatal(err)
	}
	html := NewRenderer(v).Render(n).HTML
	for _, want := range []string{
		`class="dataview dataview-table-wrap"`,
		`>Projet</th>`,
		`>status</th>`,
		`>Faits</th>`,
		`href="/Projects/Alpha.md">Alpha</a>`,
		`href="/Projects/Gamma.md">Gamma</a>`,
		`<td>active</td>`,
		`<td class="number">7</td>`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("rendered dataview missing %q in:\n%s", want, html)
		}
	}
	if strings.Contains(html, `href="/Projects/Alpha.md">Alpha Project</a>`) {
		t.Fatalf("file.link should display file name, not frontmatter title:\n%s", html)
	}
	if strings.Contains(html, `href=\\\"`) || strings.Contains(html, `%22`) {
		t.Fatalf("dataview links must not include escaped quote characters that browsers encode as %%22:\n%s", html)
	}
	assertInOrder(t, html, ">Alpha</a>", ">Gamma</a>")
	assertInOrder(t, html, ">Gamma</a>", ">Beta</a>")
}

func TestDataviewMultilineTableColumns(t *testing.T) {
	v := makeDataviewVault(t)
	n, err := v.ReadNote("Dashboards/Multiline.md")
	if err != nil {
		t.Fatal(err)
	}
	html := NewRenderer(v).Render(n).HTML
	for _, want := range []string{"Statut", "Note", "Projet", `href="/Projects/Alpha.md">Alpha</a>`, `href="/Projects/Gamma.md">Gamma</a>`} {
		if !strings.Contains(html, want) {
			t.Fatalf("multiline table missing %q in:\n%s", want, html)
		}
	}
	if strings.Contains(html, "dataview-error") || strings.Contains(html, "Beta Project") {
		t.Fatalf("multiline table should render active projects without errors:\n%s", html)
	}
}

func TestDataviewWhereLimitTagSourceFunctionsAndYamlLists(t *testing.T) {
	v := makeDataviewVault(t)
	query := `TABLE file.link as "Note", default(area, "none") as "Area", length(aliases) as "Aliases", choice(score >= 9, "high", "normal") as "Tier"
FROM #project
WHERE status = "active" AND contains(file.tags, "project")
SORT created DESC
LIMIT 2`
	html := string(RenderDataviewBlock(v, query))
	for _, want := range []string{`href="/Projects/Gamma.md">Gamma</a>`, `href="/Projects/Alpha.md">Alpha</a>`, `<td class="number">2</td>`, `<td>high</td>`, `<td>normal</td>`} {
		if !strings.Contains(html, want) {
			t.Fatalf("query result missing %q in:\n%s", want, html)
		}
	}
	if strings.Contains(html, "Beta Project") || strings.Contains(html, "Tagged Area") {
		t.Fatalf("WHERE/LIMIT should exclude beta and tagged area:\n%s", html)
	}
	assertInOrder(t, html, ">Gamma</a>", ">Alpha</a>")
}

func TestDataviewListTaskGroupFlattenAndDiagnostics(t *testing.T) {
	v := makeDataviewVault(t)

	listHTML := string(RenderDataviewBlock(v, `LIST status
FROM "Projects"
WHERE status != "done"
SORT file.name`))
	for _, want := range []string{`class="dataview dataview-list"`, `href="/Projects/Alpha.md">Alpha</a>`, `href="/Projects/Gamma.md">Gamma</a>`, "active"} {
		if !strings.Contains(listHTML, want) {
			t.Fatalf("LIST missing %q in:\n%s", want, listHTML)
		}
	}
	if strings.Contains(listHTML, `href="/Projects/Beta.md">Beta</a>`) {
		t.Fatalf("LIST should exclude done project:\n%s", listHTML)
	}

	taskHTML := string(RenderDataviewBlock(v, `TASK
FROM "Projects"
WHERE !completed
SORT due ASC`))
	for _, want := range []string{`class="dataview dataview-tasks"`, "Open alpha task", "Beta task", "Due 2026-06-01", "Priority High", `href="/Projects/Alpha.md"`} {
		if !strings.Contains(taskHTML, want) {
			t.Fatalf("TASK missing %q in:\n%s", want, taskHTML)
		}
	}
	if strings.Contains(taskHTML, "Closed alpha task") {
		t.Fatalf("TASK should exclude completed task:\n%s", taskHTML)
	}
	assertInOrder(t, taskHTML, "Open alpha task", "Beta task")

	groupHTML := string(RenderDataviewBlock(v, `TABLE rows.file.link as "Projet", length(rows) as "Count"
FROM "Projects"
GROUP BY area
SORT key`))
	for _, want := range []string{"Core", "Ops", "Alpha, Beta", "Gamma", `<td class="number">2</td>`, `<td class="number">1</td>`} {
		if !strings.Contains(groupHTML, want) {
			t.Fatalf("GROUP BY missing %q in:\n%s", want, groupHTML)
		}
	}

	flattenHTML := string(RenderDataviewBlock(v, `TABLE aliases as "Alias", file.link as "Projet"
FROM "Projects"
FLATTEN aliases
WHERE aliases != ""
SORT aliases`))
	for _, want := range []string{"A1", "Alpha", `href="/Projects/Alpha.md">Alpha</a>`} {
		if !strings.Contains(flattenHTML, want) {
			t.Fatalf("FLATTEN missing %q in:\n%s", want, flattenHTML)
		}
	}

	errHTML := string(RenderDataviewBlock(v, `CALENDAR file.mtime
FROM "Projects"`))
	for _, want := range []string{`class="dataview dataview-calendar"`, `href="/Projects/Alpha.md">Alpha</a>`, `href="/Projects/Gamma.md">Gamma</a>`} {
		if !strings.Contains(errHTML, want) {
			t.Fatalf("CALENDAR render missing %q in:\n%s", want, errHTML)
		}
	}
}

func TestDataviewEscapesHTMLAndUsesTypedDateSort(t *testing.T) {
	v := makeDataviewVault(t)
	writeDataviewFixture(t, v, "Projects/Evil.md", "---\ntitle: '<script>alert(1)</script>'\ntags: [project]\nstatus: active\nfact_count: 1\ncreated: 2026-12-01\n---\n# Evil\n")
	html := string(RenderDataviewBlock(v, `TABLE file.link as "Note", created
FROM "Projects"
WHERE status = "active"
SORT created DESC
LIMIT 1`))
	if strings.Contains(html, "<script>") {
		t.Fatalf("dataview output must escape frontmatter/title HTML:\n%s", html)
	}
	if !strings.Contains(html, `href="/Projects/Evil.md">Evil</a>`) {
		t.Fatalf("file.link filename label missing in:\n%s", html)
	}
	if !strings.Contains(html, "2026-12-01") {
		t.Fatalf("typed date value missing in:\n%s", html)
	}
}

func TestDataviewSupportsInlineClausesDateArithmeticListLinkAndFileContent(t *testing.T) {
	v := makeDataviewVault(t)
	writeDataviewFixture(t, v, "Areas/Health/Events/Recent.md", "---\ntitle: Recent Event\ntype: health-event\ndate: 2026-05-20\ncategory: appointment\nrelated:\n  - '[[Kyste pilonidal]]'\n---\n# Recent\nCiclopirox mention\n")
	writeDataviewFixture(t, v, "Areas/Health/Events/Old.md", "---\ntitle: Old Event\ntype: health-event\ndate: 2026-04-01\ncategory: note\nrelated:\n  - '[[Other]]'\n---\n# Old\n")

	html := string(RenderDataviewBlock(v, `TABLE date, category FROM "Areas/Health/Events" WHERE type = "health-event" AND date >= date("2026-05-28") - dur(30 days) SORT date DESC`))
	if !strings.Contains(html, "2026-05-20") || strings.Contains(html, "2026-04-01") {
		t.Fatalf("date arithmetic query mismatch:\n%s", html)
	}

	statusHTML := string(RenderDataviewBlock(v, `TABLE file.link as "Projet", status AS "Statut" FROM "Projects" WHERE contains(list("active", "waiting"), status) SORT file.name`))
	if !strings.Contains(statusHTML, `href="/Projects/Alpha.md">Alpha</a>`) || !strings.Contains(statusHTML, `href="/Projects/Gamma.md">Gamma</a>`) || strings.Contains(statusHTML, `href="/Projects/Beta.md">Beta</a>`) {
		t.Fatalf("list() contains query mismatch:\n%s", statusHTML)
	}

	linkHTML := string(RenderDataviewBlock(v, `TABLE file.link as "Event", date FROM "Areas/Health/Events" WHERE contains(related, link("Kyste pilonidal"))`))
	if !strings.Contains(linkHTML, `href="/Areas/Health/Events/Recent.md">Recent</a>`) || strings.Contains(linkHTML, `href="/Areas/Health/Events/Old.md">Old</a>`) {
		t.Fatalf("link() contains query mismatch:\n%s", linkHTML)
	}

	contentHTML := string(RenderDataviewBlock(v, `TABLE file.name as "Log" FROM "Areas/Health/Events" WHERE contains(file.content, "Ciclopirox")`))
	if !strings.Contains(contentHTML, "Recent") || strings.Contains(contentHTML, "Old") {
		t.Fatalf("file.content query mismatch:\n%s", contentHTML)
	}
}

func TestDataviewCalendarInlinksAndDiagnosticsEndpoint(t *testing.T) {
	v := makeDataviewVault(t)
	writeDataviewFixture(t, v, "Projects/Linked.md", "---\ntitle: Linked Project\ncreated: 2026-05-12\n---\n# Linked\n[[Alpha]]\n")
	writeDataviewFixture(t, v, "Dashboards/Broken.md", "# Broken\n\n```dataview\nTABLE file.link FROM \"Projects\" WHERE totally unsupported syntax\n```\n")

	calHTML := string(RenderDataviewBlock(v, `CALENDAR created
FROM "Projects"`))
	for _, want := range []string{`class="dataview dataview-calendar"`, "2026-05-01", `href="/Projects/Alpha.md">Alpha</a>`} {
		if !strings.Contains(calHTML, want) {
			t.Fatalf("CALENDAR missing %q in:\n%s", want, calHTML)
		}
	}

	inlinksHTML := string(RenderDataviewBlock(v, `TABLE file.inlinks as "Inlinks" FROM "Projects" WHERE file.name = "Alpha"`))
	if !strings.Contains(inlinksHTML, `href="/Projects/Linked.md">Linked</a>`) {
		t.Fatalf("file.inlinks should include backlinking note:\n%s", inlinksHTML)
	}

	req := httptest.NewRequest("GET", "/_dataview", nil)
	rr := httptest.NewRecorder()
	NewServer(v, "", "").ServeHTTP(rr, req)
	body := rr.Body.String()
	for _, want := range []string{"Dataview diagnostics", "Dashboards/Dataview.md", "Dashboards/Broken.md", "supported", "unsupported"} {
		if !strings.Contains(body, want) {
			t.Fatalf("/_dataview missing %q in:\n%s", want, body)
		}
	}
}

func TestVaultIndexCacheReusesUnchangedIndexAndInvalidatesOnMarkdownChange(t *testing.T) {
	v := makeDataviewVault(t)
	idx1, err := v.BuildIndex()
	if err != nil {
		t.Fatal(err)
	}
	idx2, err := v.BuildIndex()
	if err != nil {
		t.Fatal(err)
	}
	if idx1 != idx2 {
		t.Fatalf("unchanged vault should reuse cached index pointer")
	}

	writeDataviewFixture(t, v, "Projects/New.md", "---\ntitle: New\nstatus: active\n---\n# New\n")
	idx3, err := v.BuildIndex()
	if err != nil {
		t.Fatal(err)
	}
	if idx3 == idx2 {
		t.Fatalf("changed vault should invalidate cached index")
	}
	if _, ok := idx3.ByRel["Projects/New.md"]; !ok {
		t.Fatalf("invalidated index should include new note")
	}
}

func TestDataviewRowsComputeHeavyFieldsOnlyWhenQueryNeedsThem(t *testing.T) {
	v := makeDataviewVault(t)
	writeDataviewFixture(t, v, "Projects/Linked.md", "# Linked\n[[Alpha]]\n")
	idx, err := v.BuildIndex()
	if err != nil {
		t.Fatal(err)
	}

	lightQuery, err := parseDataviewQuery(`TABLE file.link, status FROM "Projects" WHERE status = "active" SORT file.name`)
	if err != nil {
		t.Fatal(err)
	}
	lightRows, err := evalDataviewRows(v, idx, lightQuery)
	if err != nil {
		t.Fatal(err)
	}
	if len(lightRows) == 0 {
		t.Fatal("expected light dataview rows")
	}
	for _, row := range lightRows {
		if _, ok := row.Data["file.content"]; ok {
			t.Fatalf("light query should not populate file.content: %#v", row.Data)
		}
		if _, ok := row.Data["file.inlinks"]; ok {
			t.Fatalf("light query should not populate file.inlinks: %#v", row.Data)
		}
	}

	heavyQuery, err := parseDataviewQuery(`TABLE file.inlinks FROM "Projects" WHERE contains(file.content, "Alpha")`)
	if err != nil {
		t.Fatal(err)
	}
	heavyRows, err := evalDataviewRows(v, idx, heavyQuery)
	if err != nil {
		t.Fatal(err)
	}
	if len(heavyRows) == 0 {
		t.Fatal("expected heavy dataview rows")
	}
	foundAlpha := false
	for _, row := range heavyRows {
		if row.Note != nil && row.Note.RelPath == "Projects/Alpha.md" {
			foundAlpha = true
			if _, ok := row.Data["file.content"]; !ok {
				t.Fatalf("query using file.content should populate it")
			}
			inlinks, ok := row.Data["file.inlinks"].([]dataviewLink)
			if !ok || len(inlinks) == 0 || inlinks[0].Text != "Linked" {
				t.Fatalf("query using file.inlinks should use precomputed backlinks, got %#v", row.Data["file.inlinks"])
			}
		}
	}
	if !foundAlpha {
		t.Fatalf("expected Alpha row in heavy query: %#v", heavyRows)
	}
}

func TestDataviewTableClientEnhancementAssetsArePresent(t *testing.T) {
	for _, want := range []string{".dataview-table-wrap", ".dataview-error", "overflow-x:auto", ".dataview-filter", ".dataview-pager"} {
		if !strings.Contains(css, want) {
			t.Fatalf("CSS missing %q", want)
		}
	}
	for _, want := range []string{"data-dataview-sort", "aria-sort", "dataview-table", "dataview-filter", "dataview-page-size"} {
		if !strings.Contains(js, want) {
			t.Fatalf("JS missing %q", want)
		}
	}
}

func TestParseFrontmatterUsesRealYAMLTypes(t *testing.T) {
	fm, body := parseFrontmatter("---\ncount: 7\nactive: true\nday: 2026-05-28\ntags:\n  - project\n  - active\n---\n# Body\n")
	if strings.TrimSpace(body) != "# Body" {
		t.Fatalf("body mismatch: %q", body)
	}
	if fm["count"] != 7 || fm["active"] != true {
		t.Fatalf("typed scalar mismatch: %#v", fm)
	}
	if _, ok := fm["day"].(time.Time); !ok {
		t.Fatalf("date should parse as time.Time, got %#v", fm["day"])
	}
	tags, ok := fm["tags"].([]any)
	if !ok || len(tags) != 2 || tags[0] != "project" {
		t.Fatalf("list mismatch: %#v", fm["tags"])
	}
}
