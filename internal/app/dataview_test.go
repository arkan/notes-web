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

func TestDataviewDateformatRendersFileMtimeWithTime(t *testing.T) {
	v := makeDataviewVault(t)
	mtime := time.Date(2026, 6, 18, 14, 35, 0, 0, time.Local)
	path := filepath.Join(v.Root, "Projects", "Alpha.md")
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatal(err)
	}

	html := string(RenderDataviewBlock(v, `TABLE dateformat(file.mtime, "yyyy-MM-dd HH:mm") as "Dernière update"
FROM "Projects"
WHERE file.name = "Alpha"`))
	if !strings.Contains(html, "2026-06-18 14:35") {
		t.Fatalf("dateformat(file.mtime) should include date and time:\n%s", html)
	}
	if strings.Contains(html, ">—</td>") {
		t.Fatalf("dateformat(file.mtime) should not render as an empty dash:\n%s", html)
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
	for _, want := range []string{".dataview-table-wrap", ".dataview-error", "overflow-x:auto", "-webkit-overflow-scrolling:touch", ".dataview-table-scroll{width:100%;min-width:100%", ".dataview-table-wrap .dataview-table{display:table;width:100%;max-width:none;min-width:var(--dataview-min-width);margin:0", ".dataview-controls{box-sizing:border-box;display:flex", ".dataview-pager{box-sizing:border-box;display:flex", ".dataview-filter", ".dataview-pager"} {
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

// ---------------------------------------------------------------------------
// Dataview FILTER parser tests
// ---------------------------------------------------------------------------

func TestDataviewFilterParseBasic(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    dataviewFilter
		wantErr string
	}{
		{
			name:  "simple default scalar",
			input: `status DEFAULT "active"`,
			want:  dataviewFilter{Field: "status", Defaults: []string{"active"}, Mode: filterModeSingle, Clearable: false},
		},
		{
			name:  "default with clearable",
			input: `status DEFAULT "active" CLEARABLE`,
			want:  dataviewFilter{Field: "status", Defaults: []string{"active"}, Mode: filterModeSingle, Clearable: true},
		},
		{
			name:  "default mode single",
			input: `status DEFAULT "active" MODE single`,
			want:  dataviewFilter{Field: "status", Defaults: []string{"active"}, Mode: filterModeSingle},
		},
		{
			name:  "mode multi default list",
			input: `tags DEFAULT [#project, #dashboard] MODE multi`,
			want:  dataviewFilter{Field: "tags", Defaults: []string{"#project", "#dashboard"}, Mode: filterModeMulti},
		},
		{
			name:  "mode multi with clearable",
			input: `tags MODE multi CLEARABLE`,
			want:  dataviewFilter{Field: "tags", Mode: filterModeMulti, Clearable: true},
		},
		{
			name:  "no default no clearable",
			input: `status`,
			want:  dataviewFilter{Field: "status", Mode: filterModeSingle},
		},
		{
			name:  "flexible option order: clearable first",
			input: `status CLEARABLE DEFAULT "done"`,
			want:  dataviewFilter{Field: "status", Defaults: []string{"done"}, Mode: filterModeSingle, Clearable: true},
		},
		{
			name:  "flexible option order: mode before default",
			input: `status MODE single DEFAULT "active" CLEARABLE`,
			want:  dataviewFilter{Field: "status", Defaults: []string{"active"}, Mode: filterModeSingle, Clearable: true},
		},
		{
			name:  "file.tags field with multi",
			input: `file.tags MODE multi`,
			want:  dataviewFilter{Field: "file.tags", Mode: filterModeMulti},
		},
		{
			name:  "file.etags field",
			input: `file.etags MODE multi CLEARABLE`,
			want:  dataviewFilter{Field: "file.etags", Mode: filterModeMulti, Clearable: true},
		},
		{
			name:  "default list with single value",
			input: `status DEFAULT [active] MODE multi`,
			want:  dataviewFilter{Field: "status", Defaults: []string{"active"}, Mode: filterModeMulti},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseDataviewFilter(tc.input)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("parseDataviewFilter(%q) error = %v, want %q", tc.input, err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseDataviewFilter(%q) unexpected error: %v", tc.input, err)
			}
			if got.Field != tc.want.Field || got.Clearable != tc.want.Clearable || got.Mode != tc.want.Mode {
				t.Fatalf("parseDataviewFilter(%q) = %+v, want %+v", tc.input, got, tc.want)
			}
			if len(got.Defaults) != len(tc.want.Defaults) {
				t.Fatalf("parseDataviewFilter(%q) defaults=%v, want %v", tc.input, got.Defaults, tc.want.Defaults)
			}
			for i := range got.Defaults {
				if got.Defaults[i] != tc.want.Defaults[i] {
					t.Fatalf("parseDataviewFilter(%q) defaults[%d]=%q, want %q", tc.input, i, got.Defaults[i], tc.want.Defaults[i])
				}
			}
		})
	}
}

func TestDataviewFilterParseErrors(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{
			name:    "empty",
			input:   "",
			wantErr: "empty FILTER",
		},
		{
			name:    "list default in single mode",
			input:   `status DEFAULT [a, b] MODE single`,
			wantErr: "single cannot have multiple DEFAULT",
		},
		{
			name:    "scalar default in multi mode",
			input:   `status DEFAULT "active" MODE multi`,
			wantErr: "multi requires DEFAULT [...]",
		},
		{
			name:    "invalid mode value",
			input:   `status MODE invalid`,
			wantErr: "invalid MODE",
		},
		{
			name:    "duplicate default",
			input:   `status DEFAULT "a" DEFAULT "b"`,
			wantErr: "duplicate DEFAULT",
		},
		{
			name:    "duplicate mode",
			input:   `status MODE single MODE multi`,
			wantErr: "duplicate MODE",
		},
		{
			name:    "duplicate clearable",
			input:   `status CLEARABLE CLEARABLE`,
			wantErr: "duplicate CLEARABLE",
		},
		{
			name:    "unexpected token",
			input:   `status UNKNOWN`,
			wantErr: "unexpected token",
		},
		{
			name:    "default without value",
			input:   `status DEFAULT`,
			wantErr: "requires a value",
		},
		{
			name:    "mode without value",
			input:   `status MODE`,
			wantErr: "requires a value",
		},
		{
			name:    "unclosed default list",
			input:   `status DEFAULT [a, b`,
			wantErr: "unclosed DEFAULT list",
		},
		{
			name:    "empty list default",
			input:   `status DEFAULT [] MODE multi`,
			wantErr: "list is empty",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseDataviewFilter(tc.input)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("parseDataviewFilter(%q) error = %v, want %q", tc.input, err, tc.wantErr)
			}
		})
	}
}

func TestDataviewFilterParseCaseInsensitiveKeywords(t *testing.T) {
	f, err := parseDataviewFilter(`STATUS DEFAULT "active" MODE SINGLE CLEARABLE`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Field != "STATUS" || len(f.Defaults) != 1 || f.Defaults[0] != "active" || f.Mode != filterModeSingle || !f.Clearable {
		t.Fatalf("unexpected filter: %+v", f)
	}

	// Mixed case mode values.
	f2, err := parseDataviewFilter(`tags MODE Multi CLEARABLE`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f2.Mode != filterModeMulti {
		t.Fatalf("expected multi mode, got %d", f2.Mode)
	}
}

// ---------------------------------------------------------------------------
// Dataview FILTER integration with query parser
// ---------------------------------------------------------------------------

func TestDataviewFilterInQueryParse(t *testing.T) {
	q, err := parseDataviewQuery(`TABLE status, file.link
FROM "Projects"
FILTER status DEFAULT "active" CLEARABLE
FILTER tags MODE multi
SORT status`)
	if err != nil {
		t.Fatalf("parseDataviewQuery error: %v", err)
	}
	if len(q.Filters) != 2 {
		t.Fatalf("expected 2 filters, got %d", len(q.Filters))
	}
	if q.Filters[0].Field != "status" || q.Filters[0].Defaults[0] != "active" || !q.Filters[0].Clearable {
		t.Fatalf("first filter mismatch: %+v", q.Filters[0])
	}
	if q.Filters[1].Field != "tags" || q.Filters[1].Mode != filterModeMulti {
		t.Fatalf("second filter mismatch: %+v", q.Filters[1])
	}
}

func TestDataviewFilterDuplicateFieldError(t *testing.T) {
	_, err := parseDataviewQuery(`TABLE status, file.link
FROM "Projects"
FILTER status DEFAULT "active"
FILTER status CLEARABLE`)
	if err == nil || !strings.Contains(err.Error(), "duplicate FILTER") {
		t.Fatalf("expected duplicate FILTER error, got %v", err)
	}
}

func TestDataviewFilterNonTableError(t *testing.T) {
	q, err := parseDataviewQuery(`LIST status FROM "Projects" FILTER status DEFAULT "active"`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	err = validateFiltersForQuery(q)
	if err == nil || !strings.Contains(err.Error(), "only supported for TABLE") {
		t.Fatalf("expected TABLE-only error, got %v", err)
	}
}

func TestDataviewFilterNonVisibleFieldError(t *testing.T) {
	q, err := parseDataviewQuery(`TABLE status, file.link FROM "Projects" FILTER score DEFAULT "10"`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	err = validateFiltersForQuery(q)
	if err == nil || !strings.Contains(err.Error(), "does not match any visible table column") {
		t.Fatalf("expected column mismatch error, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Dataview FILTER pipeline integration tests
// ---------------------------------------------------------------------------

func TestDataviewFilterPipelineDefaultFilters(t *testing.T) {
	v := makeDataviewVault(t)
	idx, err := v.BuildIndex()
	if err != nil {
		t.Fatal(err)
	}

	// Query with FILTER status default "active"
	q, err := parseDataviewQuery(`TABLE status, file.link FROM "Projects" FILTER status DEFAULT "active" SORT file.name`)
	if err != nil {
		t.Fatal(err)
	}

	// Initial render with defaults: should only show active projects.
	params := dataviewTableParams{}
	rows, states, err := evalDataviewTableRows(v, idx, q, params)
	if err != nil {
		t.Fatal(err)
	}

	// Should have Active (Alpha, Gamma) but not done (Beta).
	if len(rows) != 2 {
		t.Fatalf("expected 2 filtered rows (active), got %d: %+v", len(rows), rows)
	}

	// Verify filter state.
	if len(states) != 1 {
		t.Fatalf("expected 1 filter state, got %d", len(states))
	}
	if len(states[0].Selected) != 1 || states[0].Selected[0] != "active" {
		t.Fatalf("expected active selected, got %v", states[0].Selected)
	}
}

func TestDataviewFilterPipelineAJAXParams(t *testing.T) {
	v := makeDataviewVault(t)
	idx, err := v.BuildIndex()
	if err != nil {
		t.Fatal(err)
	}

	q, err := parseDataviewQuery(`TABLE status, file.link FROM "Projects" FILTER status DEFAULT "active" CLEARABLE SORT file.name`)
	if err != nil {
		t.Fatal(err)
	}

	// AJAX: filter by "done" instead of default.
	params := dataviewTableParams{Filters: map[string][]string{"status": {"done"}}}
	rows, states, err := evalDataviewTableRows(v, idx, q, params)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 done row, got %d", len(rows))
	}
	if len(states[0].Selected) != 1 || states[0].Selected[0] != "done" {
		t.Fatalf("expected done selected, got %v", states[0].Selected)
	}

	// AJAX: "All" (empty selected).
	params2 := dataviewTableParams{Filters: map[string][]string{"status": {}}}
	rows2, _, err := evalDataviewTableRows(v, idx, q, params2)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows2) != 3 {
		t.Fatalf("expected 3 rows for All, got %d", len(rows2))
	}
}

func TestDataviewFilterPipelineTextQ(t *testing.T) {
	v := makeDataviewVault(t)
	idx, err := v.BuildIndex()
	if err != nil {
		t.Fatal(err)
	}

	q, err := parseDataviewQuery(`TABLE status, file.link FROM "Projects" FILTER status MODE multi SORT file.name`)
	if err != nil {
		t.Fatal(err)
	}

	// Global text q: filter to rows containing "Gamma".
	params := dataviewTableParams{Q: "Gamma"}
	rows, _, err := evalDataviewTableRows(v, idx, q, params)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row matching q=Gamma, got %d", len(rows))
	}
}

func TestDataviewFilterPipelineUserSort(t *testing.T) {
	v := makeDataviewVault(t)
	idx, err := v.BuildIndex()
	if err != nil {
		t.Fatal(err)
	}

	q, err := parseDataviewQuery(`TABLE status, fact_count FROM "Projects" SORT file.name`)
	if err != nil {
		t.Fatal(err)
	}

	// User sort by fact_count descending.
	params := dataviewTableParams{Sort: "fact_count", Dir: "desc"}
	rows, _, err := evalDataviewTableRows(v, idx, q, params)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	// First row should have highest fact_count (Gamma=12).
	firstFact := evalValue(rows[0], "fact_count")
	if displayPlain(firstFact) != "12" {
		t.Fatalf("expected first row fact_count=12, got %v", firstFact)
	}
}

func TestDataviewFilterUserSortReplacesQuerySort(t *testing.T) {
	v := makeDataviewVault(t)
	idx, err := v.BuildIndex()
	if err != nil {
		t.Fatal(err)
	}

	// Query SORTs by fact_count DESC. User SORTs by fact_count ASC.
	// User sort should replace query sort entirely, not just be a tie-breaker.
	q, err := parseDataviewQuery(`TABLE status, fact_count FROM "Projects" SORT fact_count DESC`)
	if err != nil {
		t.Fatal(err)
	}

	// User sort ASC: should override query DESC.
	params := dataviewTableParams{Sort: "fact_count", Dir: "asc"}
	rows, _, err := evalDataviewTableRows(v, idx, q, params)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	// First row should be lowest fact_count (Beta=2), not highest.
	firstFact := evalValue(rows[0], "fact_count")
	if displayPlain(firstFact) != "2" {
		t.Fatalf("expected first row fact_count=2 (ASC), got %v", firstFact)
	}
}

func TestDataviewFilterPipelineLimitAfterQ(t *testing.T) {
	v := makeDataviewVault(t)
	idx, err := v.BuildIndex()
	if err != nil {
		t.Fatal(err)
	}

	// Query with LIMIT 1, filter with q should apply q before LIMIT.
	q, err := parseDataviewQuery(`TABLE file.link FROM "Projects" LIMIT 1`)
	if err != nil {
		t.Fatal(err)
	}

	// Without q: LIMIT 1 returns first row (Alpha).
	rows, _, err := evalDataviewTableRows(v, idx, q, dataviewTableParams{})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row with LIMIT 1, got %d", len(rows))
	}

	// With q=Gamma: should find Gamma first, then LIMIT.
	rows2, _, err := evalDataviewTableRows(v, idx, q, dataviewTableParams{Q: "Gamma"})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows2) != 1 {
		t.Fatalf("expected 1 row with LIMIT 1 + q=Gamma, got %d", len(rows2))
	}
	firstNote := rows2[0].Note
	if firstNote == nil || !strings.Contains(firstNote.RelPath, "Gamma") {
		t.Fatalf("expected Gamma with q=Gamma, got %+v", firstNote)
	}
}

func TestDataviewFilterPipelineGroupBy(t *testing.T) {
	v := makeDataviewVault(t)
	idx, err := v.BuildIndex()
	if err != nil {
		t.Fatal(err)
	}

	// GROUP BY with filters.
	q, err := parseDataviewQuery(`TABLE rows.file.link as "Projet", length(rows) as "Count" FROM "Projects" GROUP BY area SORT key`)
	if err != nil {
		t.Fatal(err)
	}

	// Without filters: should have Core (Alpha, Beta) and Ops (Gamma).
	rows, _, err := evalDataviewTableRows(v, idx, q, dataviewTableParams{})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 GROUP BY rows, got %d", len(rows))
	}
}

func TestDataviewFilterPipelineFlatten(t *testing.T) {
	v := makeDataviewVault(t)
	idx, err := v.BuildIndex()
	if err != nil {
		t.Fatal(err)
	}

	q, err := parseDataviewQuery(`TABLE aliases as "Alias", file.link as "Projet" FROM "Projects" FLATTEN aliases WHERE aliases != "" SORT aliases`)
	if err != nil {
		t.Fatal(err)
	}

	// Without filters: should have A1, Alpha flattens.
	rows, _, err := evalDataviewTableRows(v, idx, q, dataviewTableParams{})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 FLATTEN rows, got %d", len(rows))
	}
}

// ---------------------------------------------------------------------------
// Dataview scanner tests
// ---------------------------------------------------------------------------

func TestDataviewScannerCountsTables(t *testing.T) {
	// Build the text with fenced dataview blocks using fmt so backticks are valid.
	block1 := "```dataview\nTABLE status, file.link FROM \"Projects\"\n```"
	block2 := "```dataview\nLIST status FROM \"Projects\"\n```"
	block3 := "```dataview\nTABLE file.name FROM \"Areas\"\n```"
	text := "Some text\n\n" + block1 + "\n\nMore text\n\n" + block2 + "\n\n" + block3 + "\n"

	blocks := scanDataviewBlocks(text)
	if len(blocks) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(blocks))
	}
	// First block should be TABLE index 1
	if !blocks[0].IsTable || blocks[0].TableIndex != 1 {
		t.Fatalf("first block should be table index 1, got isTable=%v index=%d", blocks[0].IsTable, blocks[0].TableIndex)
	}
	// Second block should NOT be a TABLE
	if blocks[1].IsTable {
		t.Fatalf("second block (LIST) should not be a table")
	}
	if blocks[1].TableIndex != 0 {
		t.Fatalf("second block table index should be 0, got %d", blocks[1].TableIndex)
	}
	// Third block should be TABLE index 2
	if !blocks[2].IsTable || blocks[2].TableIndex != 2 {
		t.Fatalf("third block should be table index 2, got isTable=%v index=%d", blocks[2].IsTable, blocks[2].TableIndex)
	}
}

func TestDataviewScannerSingleLineFenced(t *testing.T) {
	text := "```dataview\nTABLE status FROM \"Projects\" SORT file.name\n```\n"
	blocks := scanDataviewBlocks(text)
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if !blocks[0].IsTable || blocks[0].TableIndex != 1 {
		t.Fatalf("single-line table should be index 1, got index=%d", blocks[0].TableIndex)
	}
}

// ---------------------------------------------------------------------------
// Dataview filter HTML rendering tests
// ---------------------------------------------------------------------------

func TestDataviewFilterRenderControls(t *testing.T) {
	v := makeDataviewVault(t)
	idx, err := v.BuildIndex()
	if err != nil {
		t.Fatal(err)
	}

	q, err := parseDataviewQuery(`TABLE status, file.link FROM "Projects" FILTER status DEFAULT "active" CLEARABLE SORT file.name`)
	if err != nil {
		t.Fatal(err)
	}

	html := renderDataviewTableBlock(v, idx, q, 1)
	str := string(html)

	// Data attributes.
	if !strings.Contains(str, `data-dataview-action="renderDataviewTable"`) {
		t.Fatalf("missing action data attribute in:\n%s", str)
	}
	if !strings.Contains(str, `data-dataview-table="1"`) {
		t.Fatalf("missing table index data attribute in:\n%s", str)
	}

	// Filter control for "status".
	if !strings.Contains(str, `data-dataview-filter="status"`) {
		t.Fatalf("missing filter control for status in:\n%s", str)
	}

	// Default "active" should be selected.
	if !strings.Contains(str, `value="active" selected`) && !strings.Contains(str, `value="active"`+` selected`) {
		t.Fatalf("expected active selected in:\n%s", str)
	}

	// "All" option should be present (clearable).
	if !strings.Contains(str, `>All</option>`) {
		t.Fatalf("expected All option for clearable filter in:\n%s", str)
	}

	// Table headers.
	if !strings.Contains(str, `>status</th>`) {
		t.Fatalf("missing status column header in:\n%s", str)
	}
	if !strings.Contains(str, `>file.link</th>`) {
		t.Fatalf("missing file.link column header in:\n%s", str)
	}
}

func TestDataviewFilterRenderNoMatchingRows(t *testing.T) {
	v := makeDataviewVault(t)
	idx, err := v.BuildIndex()
	if err != nil {
		t.Fatal(err)
	}

	// Filter to a value that doesn't exist.
	q, err := parseDataviewQuery(`TABLE status, file.link FROM "Projects" FILTER status CLEARABLE SORT file.name`)
	if err != nil {
		t.Fatal(err)
	}
	params := dataviewTableParams{Filters: map[string][]string{"status": {"nonexistent"}}}
	html := renderDataviewTableBlockWithParams(v, idx, q, 1, params)
	str := string(html)

	// Should render "No matching rows" message.
	if !strings.Contains(str, "No matching rows") {
		t.Fatalf("expected No matching rows in:\n%s", str)
	}

	// Headers should still be present.
	if !strings.Contains(str, `>status</th>`) {
		t.Fatalf("missing status header in no-rows state:\n%s", str)
	}
}

func TestDataviewFilterRenderDefaultAbsentSyntheticOption(t *testing.T) {
	v := makeDataviewVault(t)
	idx, err := v.BuildIndex()
	if err != nil {
		t.Fatal(err)
	}

	// Default value "urgent" doesn't exist in data. Should still render as synthetic option.
	q, err := parseDataviewQuery(`TABLE status, file.link FROM "Projects" FILTER status DEFAULT "urgent" MODE single SORT file.name`)
	if err != nil {
		t.Fatal(err)
	}
	html := renderDataviewTableBlock(v, idx, q, 1)
	str := string(html)

	// Should contain "urgent" as an option.
	if !strings.Contains(str, `>urgent</option>`) && !strings.Contains(str, `value="urgent"`) {
		t.Fatalf("expected synthetic urgent option in:\n%s", str)
	}

	// Should show 0 rows because urgent matches nothing.
	if !strings.Contains(str, "No matching rows") {
		t.Fatalf("expected No matching rows for absent default in:\n%s", str)
	}
}

func TestDataviewFilterRenderNoClearableNoDefaultDisabledPlaceholder(t *testing.T) {
	v := makeDataviewVault(t)
	idx, err := v.BuildIndex()
	if err != nil {
		t.Fatal(err)
	}

	// No default, not clearable.
	q, err := parseDataviewQuery(`TABLE status, file.link FROM "Projects" FILTER status SORT file.name`)
	if err != nil {
		t.Fatal(err)
	}
	html := renderDataviewTableBlock(v, idx, q, 1)
	str := string(html)

	// Should have a disabled placeholder.
	if !strings.Contains(str, `disabled`) {
		t.Fatalf("expected disabled placeholder for non-clearable no-default filter in:\n%s", str)
	}
}

func TestDataviewFilterRenderEscapesDynamicContent(t *testing.T) {
	v := makeDataviewVault(t)
	writeDataviewFixture(t, v, "Projects/XSS.md", "---\ntitle: '<script>alert(1)</script>'\nstatus: '<script>evil</script>'\n---\n# XSS\n")
	defer func() {
		os.Remove(filepath.Join(v.Root, "Projects", "XSS.md"))
	}()

	idx, err := v.BuildIndex()
	if err != nil {
		t.Fatal(err)
	}

	q, err := parseDataviewQuery(`TABLE status, file.link FROM "Projects" FILTER status CLEARABLE SORT file.name`)
	if err != nil {
		t.Fatal(err)
	}
	html := renderDataviewTableBlock(v, idx, q, 1)
	str := string(html)

	// HTML in content should be escaped.
	if strings.Contains(str, "<script>") && !strings.Contains(str, "&lt;script&gt;") {
		t.Fatalf("dynamic content should be HTML-escaped in:\n%s", str)
	}
}

// ---------------------------------------------------------------------------
// Dataview filter tag handling
// ---------------------------------------------------------------------------

func TestDataviewFilterTagsPrefix(t *testing.T) {
	v := makeDataviewVault(t)
	writeDataviewFixture(t, v, "Projects/Tagged2.md", "---\ntitle: Tagged Note\ntags: [project, filter-test]\nstatus: active\n---\n# Tagged\n")
	defer func() {
		os.Remove(filepath.Join(v.Root, "Projects", "Tagged2.md"))
	}()

	idx, err := v.BuildIndex()
	if err != nil {
		t.Fatal(err)
	}

	// Test with tags field filter.
	q, err := parseDataviewQuery(`TABLE tags, file.link FROM "Projects" FILTER tags MODE multi SORT file.name`)
	if err != nil {
		t.Fatal(err)
	}
	html := renderDataviewTableBlock(v, idx, q, 1)
	str := string(html)

	// Tag values should have # prefix in display.
	if !strings.Contains(str, "#project") && !strings.Contains(str, "#filter-test") {
		t.Fatalf("expected #-prefixed tag options in:\n%s", str)
	}

	// AJAX filter with # prefix should match.
	params := dataviewTableParams{Filters: map[string][]string{"tags": {"#filter-test"}}}
	rows, _, err := evalDataviewTableRows(v, idx, q, params)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row filtered by #filter-test, got %d", len(rows))
	}

	// Filter without # should NOT match (strict # required).
	params2 := dataviewTableParams{Filters: map[string][]string{"tags": {"filter-test"}}}
	rows2, _, err := evalDataviewTableRows(v, idx, q, params2)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows2) != 0 {
		t.Fatalf("expected 0 rows filtered by filter-test without #, got %d", len(rows2))
	}
}

func TestDataviewFilterFileTags(t *testing.T) {
	v := makeDataviewVault(t)
	writeDataviewFixture(t, v, "Projects/Tagged3.md", "---\ntitle: Tagged Three\ntags: [three, test]\nstatus: active\n---\n# Tagged Three\n")
	defer func() {
		os.Remove(filepath.Join(v.Root, "Projects", "Tagged3.md"))
	}()

	idx, err := v.BuildIndex()
	if err != nil {
		t.Fatal(err)
	}

	// file.tags filter.
	q, err := parseDataviewQuery(`TABLE file.tags, file.link FROM "Projects" FILTER file.tags MODE multi CLEARABLE SORT file.name`)
	if err != nil {
		t.Fatal(err)
	}

	// Render should include #-prefixed options.
	html := renderDataviewTableBlock(v, idx, q, 1)
	str := string(html)
	if !strings.Contains(str, `#three`) && !strings.Contains(str, `#test`) {
		t.Fatalf("expected #-prefixed tag options for file.tags in:\n%s", str)
	}
}

func TestDataviewFilterTagDefaultsRequireHashPrefix(t *testing.T) {
	q, err := parseDataviewQuery(`TABLE tags, file.link FROM "Projects" FILTER tags DEFAULT [project] MODE multi`)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateFiltersForQuery(q); err == nil || !strings.Contains(err.Error(), "# prefix") {
		t.Fatalf("expected tag default prefix validation error, got %v", err)
	}
}

func TestDataviewTagCellEmptyValueDoesNotRenderHash(t *testing.T) {
	v := makeDataviewVault(t)
	writeDataviewFixture(t, v, "Projects/EmptyTag.md", "---\ntitle: Empty Tag\ntags: [\"\"]\nstatus: active\n---\n# Empty Tag\n")
	defer func() {
		os.Remove(filepath.Join(v.Root, "Projects", "EmptyTag.md"))
	}()
	idx, err := v.BuildIndex()
	if err != nil {
		t.Fatal(err)
	}
	q, err := parseDataviewQuery(`TABLE tags, file.link FROM "Projects" WHERE file.name = "EmptyTag"`)
	if err != nil {
		t.Fatal(err)
	}
	html := string(renderDataviewTableBlock(v, idx, q, 1))
	if strings.Contains(html, `>#<`) || strings.Contains(html, `#,`) {
		t.Fatalf("empty tag value should not render #:\n%s", html)
	}
}

// ---------------------------------------------------------------------------
// Dataview TABLE implicit LIMIT cap tests
// ---------------------------------------------------------------------------

func TestDataviewTableImplicitCapCapsAt10(t *testing.T) {
	v := makeVault(t)
	// Create 150 notes to trigger the cap.
	for i := 0; i < 150; i++ {
		name := fmt.Sprintf("CapTest/Note%03d.md", i)
		writeDataviewFixture(t, v, name, "---\ntitle: Note "+fmt.Sprint(i)+"\nnum: "+fmt.Sprint(i)+"\n---\n# Note\n")
	}
	idx, err := v.BuildIndex()
	if err != nil {
		t.Fatal(err)
	}

	// No explicit LIMIT -> should cap at 10.
	q, err := parseDataviewQuery(`TABLE file.link FROM "CapTest" SORT file.name`)
	if err != nil {
		t.Fatal(err)
	}
	html := renderDataviewTableBlock(v, idx, q, 1)
	str := string(html)
	if !strings.Contains(str, "Showing first 10 of 150") {
		t.Fatalf("expected cap note for 150 rows, got:\n%s", str)
	}
	// Rough check: should have many rows but not 150.
	if strings.Count(str, "<tr>") > 150 {
		t.Fatalf("expected capped rows (<150 <tr>), got many in:\n%s", str)
	}
}

func TestDataviewTableImplicitCapExplicitLimitRespected(t *testing.T) {
	v := makeVault(t)
	for i := 0; i < 150; i++ {
		name := fmt.Sprintf("CapLimitTest/Note%03d.md", i)
		writeDataviewFixture(t, v, name, "---\ntitle: Note "+fmt.Sprint(i)+"\n---\n# Note\n")
	}
	idx, err := v.BuildIndex()
	if err != nil {
		t.Fatal(err)
	}

	// Explicit LIMIT 200 should not be capped (renders all 150 matching rows).
	q, err := parseDataviewQuery(`TABLE file.link FROM "CapLimitTest" LIMIT 200 SORT file.name`)
	if err != nil {
		t.Fatal(err)
	}
	html := renderDataviewTableBlock(v, idx, q, 1)
	str := string(html)
	if strings.Contains(str, "Showing first 10") {
		t.Fatalf("explicit LIMIT 200 should not trigger cap message:\n%s", str)
	}

	// Explicit LIMIT 10 should not be capped.
	q2, err := parseDataviewQuery(`TABLE file.link FROM "CapLimitTest" LIMIT 10 SORT file.name`)
	if err != nil {
		t.Fatal(err)
	}
	html2 := renderDataviewTableBlock(v, idx, q2, 1)
	str2 := string(html2)
	if strings.Contains(str2, "Showing first 10") {
		t.Fatalf("explicit LIMIT 10 should not trigger cap message:\n%s", str2)
	}
}

// ---------------------------------------------------------------------------
// Dataview action handler tests (app_test.go for HTTP-level tests)
// ---------------------------------------------------------------------------

func TestDataviewFilterNonTableFullRenderError(t *testing.T) {
	v := makeDataviewVault(t)
	// Full render of a LIST with FILTER should show a visible Dataview error.
	input := "# Test\n\n```dataview\nLIST status FROM \"Projects\" FILTER status DEFAULT \"active\"\n```\n"
	html := preprocessDataviewBlocks(input, v)
	if !strings.Contains(html, "dataview-error") {
		t.Fatalf("expected dataview-error for LIST with FILTER, got:\n%s", html)
	}
	if !strings.Contains(html, "only supported for TABLE") {
		t.Fatalf("expected TABLE-only error message in:\n%s", html)
	}
}

func TestDataviewFilterValidateFilterFieldInColumns(t *testing.T) {
	cols := []dataviewColumn{{Expr: "status", Label: "Status"}, {Expr: "file.link", Label: "File"}}
	if err := validateFilterFieldInColumns(dataviewFilter{Field: "status"}, cols); err != nil {
		t.Fatalf("status should be valid: %v", err)
	}
	if err := validateFilterFieldInColumns(dataviewFilter{Field: "score"}, cols); err == nil {
		t.Fatalf("score should be invalid")
	}
}

func TestDataviewFilterIsValidFilterField(t *testing.T) {
	if !isValidFilterField("status") {
		t.Fatal("status should be valid")
	}
	if !isValidFilterField("file.tags") {
		t.Fatal("file.tags should be valid")
	}
	if !isValidFilterField("file.etags") {
		t.Fatal("file.etags should be valid")
	}
	if isValidFilterField("") {
		t.Fatal("empty should be invalid")
	}
	if isValidFilterField("$invalid") {
		t.Fatal("$invalid should be invalid")
	}
	if isValidFilterField("dateformat(...)") {
		t.Fatal("function calls should be invalid")
	}
}

func TestDataviewFilterParseFilterParams(t *testing.T) {
	params := map[string][]string{
		"filter.status": {"active"},
		"filter.tags":   {"#project", "#dashboard"},
		"q":             {"search"},
		"sort":          {"status"},
		"dir":           {"desc"},
	}
	filters := parseFilterParams(params)
	if len(filters) != 2 {
		t.Fatalf("expected 2 filter params, got %d", len(filters))
	}
	if len(filters["status"]) != 1 || filters["status"][0] != "active" {
		t.Fatalf("status filter mismatch: %v", filters["status"])
	}
	if len(filters["tags"]) != 2 || filters["tags"][0] != "#project" {
		t.Fatalf("tags filter mismatch: %v", filters["tags"])
	}
}

func TestDataviewFilterDedupeFilterValues(t *testing.T) {
	deduped := dedupeFilterValues([]string{"a", "b", "a", "c"})
	if len(deduped) != 3 || deduped[0] != "a" || deduped[1] != "b" || deduped[2] != "c" {
		t.Fatalf("deduped = %v, want [a b c]", deduped)
	}
}
