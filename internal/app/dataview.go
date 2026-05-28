package app

import (
	"fmt"
	"html"
	"html/template"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type dataviewQuery struct {
	Kind      string
	WithoutID bool
	Columns   []dataviewColumn
	From      string
	Where     string
	Sorts     []dataviewSort
	Limit     int
	GroupBy   string
	Flatten   string
}

type dataviewColumn struct{ Expr, Label string }
type dataviewSort struct {
	Expr string
	Desc bool
}

type dataviewRow struct {
	Note *NoteMeta
	Task *IndexedTask
	Data map[string]any
}

type IndexedTask struct {
	Text, Path, URL                            string
	Line                                       int
	Completed                                  bool
	Due, Completion, Created, Scheduled, Start string
	Priority                                   string
	Tags                                       []string
}

// RenderDataviewBlock renders a safe, server-side subset of Obsidian Dataview.
func RenderDataviewBlock(v *Vault, raw string) template.HTML {
	idx, err := v.BuildIndex()
	if err != nil {
		return dataviewError(raw, err)
	}
	return RenderDataviewBlockWithIndex(v, idx, raw)
}

func RenderDataviewBlockWithIndex(v *Vault, idx *VaultIndex, raw string) template.HTML {
	q, err := parseDataviewQuery(raw)
	if err != nil {
		return dataviewError(raw, err)
	}
	rows, err := evalDataviewRows(v, idx, q)
	if err != nil {
		return dataviewError(raw, err)
	}
	switch q.Kind {
	case "TABLE":
		return renderDataviewTable(q, rows)
	case "LIST":
		return renderDataviewList(q, rows)
	case "TASK":
		return renderDataviewTasks(q, rows)
	case "CALENDAR":
		return renderDataviewCalendar(q, rows)
	default:
		return dataviewError(raw, fmt.Errorf("unsupported Dataview query type %s", q.Kind))
	}
}

func preprocessDataviewBlocks(s string, v *Vault) string {
	re := regexp.MustCompile("(?s)```dataview\\s*\\n(.*?)\\n```")
	if !re.MatchString(s) {
		return s
	}
	idx, err := v.BuildIndex()
	if err != nil {
		return re.ReplaceAllStringFunc(s, func(m string) string {
			parts := re.FindStringSubmatch(m)
			if len(parts) != 2 {
				return m
			}
			return string(dataviewError(parts[1], err))
		})
	}
	return re.ReplaceAllStringFunc(s, func(m string) string {
		parts := re.FindStringSubmatch(m)
		if len(parts) != 2 {
			return m
		}
		return string(RenderDataviewBlockWithIndex(v, idx, parts[1]))
	})
}

func parseDataviewQuery(raw string) (dataviewQuery, error) {
	q := dataviewQuery{Limit: -1}
	lines := dataviewLines(raw)
	if len(lines) == 0 {
		return q, fmt.Errorf("empty Dataview query")
	}
	first := lines[0]
	upper := strings.ToUpper(first)
	switch {
	case strings.HasPrefix(upper, "TABLE"):
		q.Kind = "TABLE"
		rest := strings.TrimSpace(first[len("TABLE"):])
		if strings.HasPrefix(strings.ToUpper(rest), "WITHOUT ID") {
			q.WithoutID = true
			rest = strings.TrimSpace(rest[len("WITHOUT ID"):])
		}
		cols, err := parseDataviewColumns(rest)
		if err != nil {
			return q, err
		}
		q.Columns = cols
	case strings.HasPrefix(upper, "LIST"):
		q.Kind = "LIST"
		rest := strings.TrimSpace(first[len("LIST"):])
		if rest != "" {
			q.Columns = []dataviewColumn{{Expr: rest, Label: rest}}
		}
	case strings.HasPrefix(upper, "TASK"):
		q.Kind = "TASK"
	case strings.HasPrefix(upper, "CALENDAR"):
		q.Kind = "CALENDAR"
		rest := strings.TrimSpace(first[len("CALENDAR"):])
		if rest != "" {
			q.Columns = []dataviewColumn{{Expr: rest, Label: rest}}
		}
	default:
		return q, fmt.Errorf("unsupported Dataview query type in %q", first)
	}
	for _, line := range lines[1:] {
		u := strings.ToUpper(line)
		switch {
		case strings.HasPrefix(u, "FROM "):
			q.From = strings.TrimSpace(line[len("FROM "):])
		case strings.HasPrefix(u, "WHERE "):
			q.Where = strings.TrimSpace(line[len("WHERE "):])
		case strings.HasPrefix(u, "SORT "):
			q.Sorts = parseDataviewSorts(strings.TrimSpace(line[len("SORT "):]))
		case strings.HasPrefix(u, "LIMIT "):
			n, err := strconv.Atoi(strings.TrimSpace(line[len("LIMIT "):]))
			if err != nil {
				return q, fmt.Errorf("invalid LIMIT %q", line)
			}
			q.Limit = n
		case strings.HasPrefix(u, "GROUP BY "):
			q.GroupBy = strings.TrimSpace(line[len("GROUP BY "):])
		case strings.HasPrefix(u, "FLATTEN "):
			q.Flatten = strings.TrimSpace(line[len("FLATTEN "):])
		default:
			return q, fmt.Errorf("unsupported Dataview clause %q", line)
		}
	}
	return q, nil
}

func dataviewLines(raw string) []string {
	var logical []string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := splitDataviewInlineClauses(line)
		for _, part := range parts {
			upper := strings.ToUpper(strings.TrimSpace(part))
			isClause := strings.HasPrefix(upper, "TABLE") || strings.HasPrefix(upper, "LIST") || strings.HasPrefix(upper, "TASK") || strings.HasPrefix(upper, "CALENDAR") || strings.HasPrefix(upper, "FROM ") || strings.HasPrefix(upper, "WHERE ") || strings.HasPrefix(upper, "SORT ") || strings.HasPrefix(upper, "LIMIT ") || strings.HasPrefix(upper, "GROUP BY ") || strings.HasPrefix(upper, "FLATTEN ")
			if isClause || len(logical) == 0 {
				logical = append(logical, part)
			} else {
				logical[len(logical)-1] = strings.TrimSpace(logical[len(logical)-1] + " " + part)
			}
		}
	}
	return logical
}

func splitDataviewInlineClauses(line string) []string {
	clauseKeywords := []string{" FROM ", " WHERE ", " SORT ", " LIMIT ", " GROUP BY ", " FLATTEN "}
	var parts []string
	for {
		pos := -1
		for _, kw := range clauseKeywords {
			if i := findTopLevelKeyword(line, kw); i >= 0 && (pos < 0 || i < pos) {
				pos = i
			}
		}
		if pos < 0 {
			break
		}
		if strings.TrimSpace(line[:pos]) != "" {
			parts = append(parts, strings.TrimSpace(line[:pos]))
		}
		line = strings.TrimSpace(line[pos:])
	}
	if strings.TrimSpace(line) != "" {
		parts = append(parts, strings.TrimSpace(line))
	}
	return parts
}

func findTopLevelKeyword(s, kw string) int {
	upper, target := strings.ToUpper(s), strings.ToUpper(kw)
	depth := 0
	inQuote := false
	for i, r := range s {
		switch r {
		case '"':
			inQuote = !inQuote
		case '(':
			if !inQuote {
				depth++
			}
		case ')':
			if !inQuote && depth > 0 {
				depth--
			}
		}
		if !inQuote && depth == 0 && strings.HasPrefix(upper[i:], target) {
			return i
		}
	}
	return -1
}

func parseDataviewColumns(s string) ([]dataviewColumn, error) {
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}
	parts := splitTopLevel(s, ',')
	cols := make([]dataviewColumn, 0, len(parts))
	asRe := regexp.MustCompile(`(?i)^(.+?)\s+as\s+"([^"]+)"$`)
	for _, p := range parts {
		p = strings.TrimSpace(p)
		col := dataviewColumn{Expr: p, Label: p}
		if m := asRe.FindStringSubmatch(p); len(m) == 3 {
			col.Expr = strings.TrimSpace(m[1])
			col.Label = m[2]
		}
		cols = append(cols, col)
	}
	return cols, nil
}

func parseDataviewSorts(s string) []dataviewSort {
	var sorts []dataviewSort
	for _, p := range splitTopLevel(s, ',') {
		fields := strings.Fields(strings.TrimSpace(p))
		if len(fields) == 0 {
			continue
		}
		d := false
		if len(fields) > 1 && strings.EqualFold(fields[len(fields)-1], "DESC") {
			d = true
			fields = fields[:len(fields)-1]
		} else if len(fields) > 1 && strings.EqualFold(fields[len(fields)-1], "ASC") {
			fields = fields[:len(fields)-1]
		}
		sorts = append(sorts, dataviewSort{Expr: strings.Join(fields, " "), Desc: d})
	}
	return sorts
}

func splitTopLevel(s string, sep rune) []string {
	var out []string
	start, depth := 0, 0
	inQuote := false
	for i, r := range s {
		switch r {
		case '"':
			inQuote = !inQuote
		case '(':
			if !inQuote {
				depth++
			}
		case ')':
			if !inQuote && depth > 0 {
				depth--
			}
		default:
			if r == sep && !inQuote && depth == 0 {
				out = append(out, s[start:i])
				start = i + 1
			}
		}
	}
	out = append(out, s[start:])
	return out
}

func evalDataviewRows(v *Vault, idx *VaultIndex, q dataviewQuery) ([]dataviewRow, error) {
	var rows []dataviewRow
	heavy := q.requiredHeavyFields()
	if q.Kind == "TASK" {
		for _, meta := range idx.Notes {
			if sourceMatches(meta, q.From) {
				for _, task := range extractTasksForNote(v, meta) {
					r := dataviewRow{Note: &meta, Task: &task, Data: dataviewBaseData(v, idx, meta, heavy)}
					if whereMatches(r, q.Where) {
						rows = append(rows, r)
					}
				}
			}
		}
	} else {
		for _, meta := range idx.Notes {
			if sourceMatches(meta, q.From) {
				r := dataviewRow{Note: &meta, Data: dataviewBaseData(v, idx, meta, heavy)}
				if whereMatches(r, q.Where) {
					rows = append(rows, r)
				}
			}
		}
	}
	if q.Flatten != "" {
		rows = flattenRows(rows, q.Flatten)
	}
	if q.GroupBy != "" {
		rows = groupRows(rows, q.GroupBy)
	}
	sortDataviewRows(rows, q.Sorts)
	if q.Limit >= 0 && len(rows) > q.Limit {
		rows = rows[:q.Limit]
	}
	return rows, nil
}

type dataviewHeavyFields struct{ Content, Inlinks bool }

func (q dataviewQuery) requiredHeavyFields() dataviewHeavyFields {
	var exprs []string
	for _, c := range q.Columns {
		exprs = append(exprs, c.Expr)
	}
	for _, s := range q.Sorts {
		exprs = append(exprs, s.Expr)
	}
	exprs = append(exprs, q.Where, q.GroupBy, q.Flatten)
	joined := strings.Join(exprs, "\n")
	return dataviewHeavyFields{Content: strings.Contains(joined, "file.content"), Inlinks: strings.Contains(joined, "file.inlinks")}
}

func dataviewBaseData(v *Vault, idx *VaultIndex, meta NoteMeta, heavy dataviewHeavyFields) map[string]any {
	data := map[string]any{}
	if heavy.Content {
		if n, err := v.ReadNote(meta.RelPath); err == nil {
			data["file.content"] = n.Body
		}
	}
	if heavy.Inlinks {
		data["file.inlinks"] = idx.Inlinks[meta.RelPath]
	}
	return data
}

func sourceMatches(meta NoteMeta, src string) bool {
	src = strings.TrimSpace(src)
	if src == "" {
		return true
	}
	terms := strings.Split(src, " AND ")
	for _, term := range terms {
		term = strings.TrimSpace(term)
		neg := strings.HasPrefix(term, "-")
		term = strings.TrimPrefix(term, "-")
		match := false
		switch {
		case strings.HasPrefix(term, "\"") && strings.HasSuffix(term, "\""):
			folder := strings.Trim(term, "\"")
			match = meta.RelPath == folder || strings.HasPrefix(meta.RelPath, strings.TrimSuffix(folder, "/")+"/")
		case strings.HasPrefix(term, "#"):
			tag := normalizeTag(strings.TrimPrefix(term, "#"))
			for _, t := range meta.Tags {
				if t == tag {
					match = true
					break
				}
			}
		case strings.HasPrefix(term, "[["):
			target := strings.TrimSuffix(strings.TrimPrefix(term, "[["), "]]")
			match = strings.Contains(meta.RelPath, target)
		default:
			match = true
		}
		if neg {
			match = !match
		}
		if !match {
			return false
		}
	}
	return true
}

func flattenRows(rows []dataviewRow, expr string) []dataviewRow {
	var out []dataviewRow
	for _, r := range rows {
		v := evalValue(r, expr)
		vals := toSlice(v)
		if len(vals) == 0 {
			rr := r
			rr.Data = copyMap(r.Data)
			rr.Data[expr] = ""
			out = append(out, rr)
			continue
		}
		for _, item := range vals {
			rr := r
			rr.Data = copyMap(r.Data)
			rr.Data[expr] = item
			out = append(out, rr)
		}
	}
	return out
}

func groupRows(rows []dataviewRow, expr string) []dataviewRow {
	by := map[string][]dataviewRow{}
	order := []string{}
	for _, r := range rows {
		key := displayPlain(evalValue(r, expr))
		if _, ok := by[key]; !ok {
			order = append(order, key)
		}
		by[key] = append(by[key], r)
	}
	var out []dataviewRow
	for _, key := range order {
		out = append(out, dataviewRow{Data: map[string]any{"key": key, "rows": by[key]}})
	}
	return out
}

func sortDataviewRows(rows []dataviewRow, sorts []dataviewSort) {
	if len(sorts) == 0 {
		return
	}
	sort.SliceStable(rows, func(i, j int) bool {
		for _, s := range sorts {
			c := compareValues(evalValue(rows[i], s.Expr), evalValue(rows[j], s.Expr))
			if c != 0 {
				if s.Desc {
					return c > 0
				}
				return c < 0
			}
		}
		return false
	})
}

func whereMatches(r dataviewRow, where string) bool {
	where = strings.TrimSpace(where)
	if where == "" {
		return true
	}
	return evalBoolExpr(r, where)
}

func evalBoolExpr(r dataviewRow, expr string) bool {
	expr = stripOuterParens(strings.TrimSpace(expr))
	if parts := splitBool(expr, " OR "); len(parts) > 1 {
		for _, p := range parts {
			if evalBoolExpr(r, p) {
				return true
			}
		}
		return false
	}
	if parts := splitBool(expr, " AND "); len(parts) > 1 {
		for _, p := range parts {
			if !evalBoolExpr(r, p) {
				return false
			}
		}
		return true
	}
	if strings.HasPrefix(expr, "!") {
		return !truthy(evalValue(r, strings.TrimSpace(expr[1:])))
	}
	for _, op := range []string{"!=", ">=", "<=", "=", ">", "<"} {
		if l, rr, ok := splitCompare(expr, op); ok {
			c := compareValues(evalValue(r, l), evalValue(r, rr))
			switch op {
			case "=":
				return c == 0
			case "!=":
				return c != 0
			case ">":
				return c > 0
			case "<":
				return c < 0
			case ">=":
				return c >= 0
			case "<=":
				return c <= 0
			}
		}
	}
	return truthy(evalValue(r, expr))
}

func splitBool(s, op string) []string {
	var out []string
	start, depth := 0, 0
	inQuote := false
	upper := strings.ToUpper(s)
	for i, r := range s {
		switch r {
		case '"':
			inQuote = !inQuote
		case '(':
			if !inQuote {
				depth++
			}
		case ')':
			if !inQuote && depth > 0 {
				depth--
			}
		}
		if !inQuote && depth == 0 && strings.HasPrefix(upper[i:], op) {
			out = append(out, strings.TrimSpace(s[start:i]))
			start = i + len(op)
		}
	}
	if len(out) > 0 {
		out = append(out, strings.TrimSpace(s[start:]))
	}
	return out
}
func splitCompare(s, op string) (string, string, bool) {
	depth := 0
	inQuote := false
	for i, r := range s {
		switch r {
		case '"':
			inQuote = !inQuote
		case '(':
			if !inQuote {
				depth++
			}
		case ')':
			if !inQuote && depth > 0 {
				depth--
			}
		}
		if !inQuote && depth == 0 && strings.HasPrefix(s[i:], op) {
			return strings.TrimSpace(s[:i]), strings.TrimSpace(s[i+len(op):]), true
		}
	}
	return "", "", false
}
func stripOuterParens(s string) string {
	for strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") {
		return strings.TrimSpace(s[1 : len(s)-1])
	}
	return s
}

func evalValue(r dataviewRow, expr string) any {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return ""
	}
	if v, ok := r.Data[expr]; ok {
		return v
	}
	if l, rr, op, ok := splitArithmetic(expr); ok {
		return evalArithmetic(evalValue(r, l), evalValue(r, rr), op)
	}
	if strings.HasPrefix(expr, "\"") && strings.HasSuffix(expr, "\"") {
		return strings.Trim(expr, "\"")
	}
	if strings.EqualFold(expr, "today") {
		now := time.Now()
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	}
	if n, err := strconv.Atoi(expr); err == nil {
		return n
	}
	if f, err := strconv.ParseFloat(expr, 64); err == nil {
		return f
	}
	if strings.EqualFold(expr, "true") {
		return true
	}
	if strings.EqualFold(expr, "false") {
		return false
	}
	if name, args, ok := parseFunc(expr); ok {
		return evalFunc(r, name, args)
	}
	if strings.HasPrefix(expr, "rows.") {
		return evalRowsValue(r, strings.TrimPrefix(expr, "rows."))
	}
	if r.Task != nil {
		if v, ok := taskField(*r.Task, expr); ok {
			return v
		}
	}
	if r.Note != nil {
		return noteField(*r.Note, expr)
	}
	return ""
}

func parseFunc(expr string) (string, []string, bool) {
	i := strings.Index(expr, "(")
	if i < 1 || !strings.HasSuffix(expr, ")") {
		return "", nil, false
	}
	return strings.TrimSpace(expr[:i]), splitTopLevel(expr[i+1:len(expr)-1], ','), true
}
func evalFunc(r dataviewRow, name string, args []string) any {
	switch strings.ToLower(name) {
	case "list":
		out := make([]any, 0, len(args))
		for _, arg := range args {
			out = append(out, evalValue(r, arg))
		}
		return out
	case "link":
		if len(args) == 1 {
			txt := displayPlain(evalValue(r, args[0]))
			return dataviewLink{URL: "#", Text: txt}
		}
	case "dur", "duration":
		if len(args) == 1 {
			return parseDataviewDuration(args[0])
		}
	case "contains":
		if len(args) == 2 {
			return containsValue(evalValue(r, args[0]), evalValue(r, args[1]))
		}
	case "default":
		if len(args) == 2 {
			v := evalValue(r, args[0])
			if isEmpty(v) {
				return evalValue(r, args[1])
			}
			return v
		}
	case "date":
		if len(args) == 1 {
			return parseDateAny(evalValue(r, args[0]))
		}
	case "choice":
		if len(args) == 3 {
			if evalBoolExpr(r, args[0]) {
				return evalValue(r, args[1])
			}
			return evalValue(r, args[2])
		}
	case "startswith":
		if len(args) == 2 {
			return strings.HasPrefix(displayPlain(evalValue(r, args[0])), displayPlain(evalValue(r, args[1])))
		}
	case "endswith":
		if len(args) == 2 {
			return strings.HasSuffix(displayPlain(evalValue(r, args[0])), displayPlain(evalValue(r, args[1])))
		}
	case "regexmatch":
		if len(args) == 2 {
			ok, _ := regexp.MatchString(displayPlain(evalValue(r, args[0])), displayPlain(evalValue(r, args[1])))
			return ok
		}
	case "length":
		if len(args) == 1 {
			return lenOf(evalValue(r, args[0]))
		}
	case "sum", "min", "max", "average":
		return aggregateFunc(r, name, args)
	}
	return ""
}
func aggregateFunc(r dataviewRow, name string, args []string) any {
	if len(args) != 1 {
		return ""
	}
	vals := toSlice(evalValue(r, args[0]))
	if len(vals) == 0 {
		return 0
	}
	nums := []float64{}
	for _, v := range vals {
		if f, ok := toFloat(v); ok {
			nums = append(nums, f)
		}
	}
	if len(nums) == 0 {
		return 0
	}
	res := nums[0]
	sum := 0.0
	for _, f := range nums {
		sum += f
		if name == "min" && f < res {
			res = f
		}
		if name == "max" && f > res {
			res = f
		}
	}
	if name == "sum" {
		res = sum
	}
	if name == "average" {
		res = sum / float64(len(nums))
	}
	if res == float64(int(res)) {
		return int(res)
	}
	return res
}
func evalRowsValue(r dataviewRow, expr string) any {
	rows, ok := r.Data["rows"].([]dataviewRow)
	if !ok {
		return ""
	}
	if expr == "" {
		return rows
	}
	vals := make([]any, 0, len(rows))
	for _, row := range rows {
		vals = append(vals, evalValue(row, expr))
	}
	return vals
}

func noteField(n NoteMeta, expr string) any {
	switch expr {
	case "file.link":
		return dataviewLink{URL: n.URL, Text: noteFileName(n)}
	case "file.name":
		return noteFileName(n)
	case "file.path":
		return n.RelPath
	case "file.folder":
		return filepath.ToSlash(filepath.Dir(n.RelPath))
	case "file.ext":
		return strings.TrimPrefix(filepath.Ext(n.RelPath), ".")
	case "file.mtime", "file.ctime":
		return n.ModTime
	case "file.tags", "file.etags":
		return n.Tags
	case "file.outlinks":
		return n.OutgoingWikiLinks
	}
	if n.Frontmatter != nil {
		if v, ok := n.Frontmatter[expr]; ok {
			return v
		}
	}
	return ""
}
func taskField(t IndexedTask, expr string) (any, bool) {
	switch expr {
	case "text":
		return t.Text, true
	case "completed":
		return t.Completed, true
	case "due":
		return t.Due, true
	case "completion":
		return t.Completion, true
	case "created":
		return t.Created, true
	case "scheduled":
		return t.Scheduled, true
	case "start":
		return t.Start, true
	case "priority":
		return t.Priority, true
	case "tags":
		return t.Tags, true
	case "line":
		return t.Line, true
	case "path":
		return t.Path, true
	case "link":
		return dataviewLink{URL: t.URL, Text: t.Path}, true
	}
	return nil, false
}

func noteFileName(n NoteMeta) string {
	return strings.TrimSuffix(filepath.Base(n.RelPath), filepath.Ext(n.RelPath))
}

type dataviewLink struct{ URL, Text string }

func renderDataviewTable(q dataviewQuery, rows []dataviewRow) template.HTML {
	var b strings.Builder
	b.WriteString(`<div class=\"dataview dataview-table-wrap\"><div class=\"dataview-controls\"><label class=\"dataview-filter-label\">Filter <input class=\"dataview-filter\" type=\"search\" data-dataview-filter placeholder=\"Filter table…\"></label><label class=\"dataview-page-size-label\">Rows <select data-dataview-page-size><option value=\"0\">All</option><option value=\"10\">10</option><option value=\"25\">25</option><option value=\"50\">50</option></select></label></div><table class=\"dataview-table\"><thead><tr>`)
	cols := q.Columns
	if len(cols) == 0 {
		cols = []dataviewColumn{{Expr: "file.link", Label: "File"}}
	}
	for _, c := range cols {
		b.WriteString(`<th scope=\"col\" data-dataview-sort aria-sort=\"none\">` + html.EscapeString(c.Label) + `</th>`)
	}
	b.WriteString(`</tr></thead><tbody>`)
	for _, r := range rows {
		if key, ok := r.Data["key"]; ok {
			b.WriteString(`<tr class=\"dataview-group\"><th scope=\"row\" colspan=\"` + strconv.Itoa(len(cols)) + `\">` + html.EscapeString(displayPlain(key)) + `</th></tr>`)
		}
		b.WriteString(`<tr>`)
		for _, c := range cols {
			b.WriteString(renderDataviewCell(evalValue(r, c.Expr)))
		}
		b.WriteString(`</tr>`)
	}
	b.WriteString(`</tbody></table><div class=\"dataview-pager\" data-dataview-pager aria-live=\"polite\"></div></div>`)
	return template.HTML(b.String())
}
func renderDataviewCell(v any) string {
	cls := ""
	if _, ok := toFloat(v); ok {
		cls = ` class=\"number\"`
	}
	return `<td` + cls + `>` + renderDataviewValue(v) + `</td>`
}
func renderDataviewList(q dataviewQuery, rows []dataviewRow) template.HTML {
	var b strings.Builder
	b.WriteString(`<ul class=\"dataview dataview-list\">`)
	for _, r := range rows {
		label := ""
		if r.Note != nil {
			label = renderDataviewValue(noteField(*r.Note, "file.link"))
		}
		if len(q.Columns) > 0 {
			value := renderDataviewValue(evalValue(r, q.Columns[0].Expr))
			if label == "" {
				label = value
			} else {
				label += ` <span class=\"dataview-list-value\">` + value + `</span>`
			}
		}
		if label == "" {
			label = "—"
		}
		b.WriteString(`<li>` + label + `</li>`)
	}
	b.WriteString(`</ul>`)
	return template.HTML(b.String())
}
func renderDataviewTasks(q dataviewQuery, rows []dataviewRow) template.HTML {
	var b strings.Builder
	b.WriteString(`<ul class=\"dataview dataview-tasks\">`)
	for _, r := range rows {
		if r.Task == nil {
			continue
		}
		t := r.Task
		checked := ""
		if t.Completed {
			checked = " checked"
		}
		b.WriteString(`<li class=\"task-list-item\"><input type=\"checkbox\" disabled` + checked + `> ` + html.EscapeString(t.Text))
		if t.Due != "" {
			b.WriteString(` <span class=\"task-meta due-date\">Due ` + html.EscapeString(t.Due) + `</span>`)
		}
		if t.Priority != "" {
			b.WriteString(` <span class=\"task-meta priority-meta\">Priority ` + html.EscapeString(t.Priority) + `</span>`)
		}
		b.WriteString(` <a href=\"` + html.EscapeString(t.URL) + `\">` + html.EscapeString(t.Path) + `</a></li>`)
	}
	b.WriteString(`</ul>`)
	return template.HTML(b.String())
}
func renderDataviewCalendar(q dataviewQuery, rows []dataviewRow) template.HTML {
	expr := "file.mtime"
	if len(q.Columns) > 0 && strings.TrimSpace(q.Columns[0].Expr) != "" {
		expr = q.Columns[0].Expr
	}
	byDay := map[string][]dataviewRow{}
	var days []string
	for _, r := range rows {
		t := parseDateAny(evalValue(r, expr))
		if t.IsZero() {
			continue
		}
		day := t.Format("2006-01-02")
		if _, ok := byDay[day]; !ok {
			days = append(days, day)
		}
		byDay[day] = append(byDay[day], r)
	}
	sort.Strings(days)
	var b strings.Builder
	b.WriteString(`<div class=\"dataview dataview-calendar\">`)
	if len(days) == 0 {
		b.WriteString(`<p>Aucune date.</p>`)
	}
	for _, day := range days {
		b.WriteString(`<section class=\"dataview-calendar-day\"><h4>` + html.EscapeString(day) + `</h4><ul>`)
		for _, r := range byDay[day] {
			if r.Note != nil {
				b.WriteString(`<li>` + renderDataviewValue(noteField(*r.Note, "file.link")) + `</li>`)
			}
		}
		b.WriteString(`</ul></section>`)
	}
	b.WriteString(`</div>`)
	return template.HTML(b.String())
}

func renderDataviewValue(v any) string {
	switch x := v.(type) {
	case dataviewLink:
		return `<a href=\"` + html.EscapeString(x.URL) + `\">` + html.EscapeString(x.Text) + `</a>`
	case []any:
		parts := []string{}
		for _, i := range x {
			parts = append(parts, html.EscapeString(displayPlain(i)))
		}
		return strings.Join(parts, ", ")
	case []string:
		parts := []string{}
		for _, i := range x {
			parts = append(parts, html.EscapeString(i))
		}
		return strings.Join(parts, ", ")
	case []dataviewLink:
		parts := []string{}
		for _, link := range x {
			parts = append(parts, renderDataviewValue(link))
		}
		return strings.Join(parts, ", ")
	case []dataviewRow:
		parts := []string{}
		for _, row := range x {
			if row.Note != nil {
				parts = append(parts, renderDataviewValue(noteField(*row.Note, "file.link")))
			}
		}
		return strings.Join(parts, ", ")
	case time.Time:
		if x.IsZero() {
			return "—"
		}
		return html.EscapeString(x.Format("2006-01-02"))
	default:
		s := displayPlain(v)
		if s == "" {
			return "—"
		}
		return html.EscapeString(s)
	}
}
func dataviewError(raw string, err error) template.HTML {
	return template.HTML(`<div class=\"dataview-error\"><strong>Dataview non rendu</strong><p>` + html.EscapeString(err.Error()) + `</p><pre>` + html.EscapeString(raw) + `</pre></div>`)
}

func extractTasksForNote(v *Vault, meta NoteMeta) []IndexedTask {
	n, err := v.ReadNote(meta.RelPath)
	if err != nil {
		return nil
	}
	var tasks []IndexedTask
	for i, line := range strings.Split(n.Body, "\n") {
		trimmed := strings.TrimSpace(line)
		if !(strings.HasPrefix(trimmed, "- [ ]") || strings.HasPrefix(trimmed, "- [x]") || strings.HasPrefix(trimmed, "- [X]")) {
			continue
		}
		completed := strings.HasPrefix(trimmed, "- [x]") || strings.HasPrefix(trimmed, "- [X]")
		text := strings.TrimSpace(trimmed[5:])
		task := IndexedTask{Text: cleanDataviewTaskText(text), Path: meta.RelPath, URL: meta.URL, Line: i + 1, Completed: completed, Tags: extractInlineTags(text)}
		task.Due = firstDateAfter(text, "📅")
		task.Completion = firstDateAfter(text, "✅")
		task.Created = firstDateAfter(text, "➕")
		task.Scheduled = firstDateAfter(text, "⏳")
		task.Start = firstDateAfter(text, "🛫")
		if strings.Contains(text, "⏫") {
			task.Priority = "High"
		} else if strings.Contains(text, "🔼") {
			task.Priority = "Medium"
		} else if strings.Contains(text, "🔽") || strings.Contains(text, "⏬") {
			task.Priority = "Low"
		}
		tasks = append(tasks, task)
	}
	return tasks
}
func cleanDataviewTaskText(s string) string {
	s = regexp.MustCompile(`<!--.*?-->`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`[📅✅➕⏳🛫]\s*\d{4}-\d{2}-\d{2}`).ReplaceAllString(s, "")
	s = strings.NewReplacer("⏫", "", "🔼", "", "🔽", "", "⏬", "").Replace(s)
	return strings.TrimSpace(s)
}
func firstDateAfter(s, marker string) string {
	re := regexp.MustCompile(regexp.QuoteMeta(marker) + `\s*(\d{4}-\d{2}-\d{2})`)
	m := re.FindStringSubmatch(s)
	if len(m) == 2 {
		return m[1]
	}
	return ""
}
func extractInlineTags(s string) []string {
	re := regexp.MustCompile(`(^|\s)#([\pL\pN][\pL\pN/_-]*)`)
	seen := map[string]bool{}
	var tags []string
	for _, m := range re.FindAllStringSubmatch(s, -1) {
		tag := normalizeTag(m[2])
		if tag != "" && !seen[tag] {
			seen[tag] = true
			tags = append(tags, tag)
		}
	}
	return tags
}

type dataviewDiagnostic struct {
	Path, Kind, Status, Message, Query string
	Line                               int
}

func ScanDataviewDiagnostics(v *Vault) []dataviewDiagnostic {
	var out []dataviewDiagnostic
	blockRe := regexp.MustCompile("(?is)```(dataview|dataviewjs)\\s*\\n(.*?)\\n```")
	for _, p := range v.MarkdownFiles() {
		note, err := v.ReadNote(p)
		if err != nil {
			continue
		}
		for _, m := range blockRe.FindAllStringSubmatchIndex(note.Text, -1) {
			kind := strings.ToLower(note.Text[m[2]:m[3]])
			query := note.Text[m[4]:m[5]]
			line := 1 + strings.Count(note.Text[:m[0]], "\n")
			d := dataviewDiagnostic{Path: note.RelPath, Kind: kind, Query: strings.TrimSpace(query), Line: line, Status: "supported"}
			if kind == "dataviewjs" {
				d.Status, d.Message = "unsupported", "dataviewjs is intentionally not executed"
			} else if q, err := parseDataviewQuery(query); err != nil {
				d.Status, d.Message = "unsupported", err.Error()
			} else if msg := dataviewUnsupportedReason(q); msg != "" {
				d.Status, d.Message = "unsupported", msg
			}
			out = append(out, d)
		}
	}
	return out
}

func dataviewUnsupportedReason(q dataviewQuery) string {
	if q.Kind == "" {
		return "empty query kind"
	}
	if q.Kind != "TABLE" && q.Kind != "LIST" && q.Kind != "TASK" && q.Kind != "CALENDAR" {
		return "unsupported query kind"
	}
	if invalidBareBoolExpr(q.Where) {
		return "unsupported WHERE expression"
	}
	return ""
}

func invalidBareBoolExpr(expr string) bool {
	expr = stripOuterParens(strings.TrimSpace(expr))
	if expr == "" {
		return false
	}
	if strings.Contains(strings.ToUpper(expr), " AND ") || strings.Contains(strings.ToUpper(expr), " OR ") {
		return false
	}
	if strings.HasPrefix(expr, "!") {
		return invalidBareBoolExpr(strings.TrimSpace(expr[1:]))
	}
	if _, _, ok := parseFunc(expr); ok {
		return false
	}
	for _, op := range []string{"!=", ">=", "<=", "=", ">", "<"} {
		if _, _, ok := splitCompare(expr, op); ok {
			return false
		}
	}
	return len(strings.Fields(expr)) > 1
}

type dataviewDuration time.Duration

func splitArithmetic(s string) (string, string, string, bool) {
	depth := 0
	inQuote := false
	for i, r := range s {
		switch r {
		case '"':
			inQuote = !inQuote
		case '(':
			if !inQuote {
				depth++
			}
		case ')':
			if !inQuote && depth > 0 {
				depth--
			}
		case '+', '-':
			if !inQuote && depth == 0 && i > 0 && i < len(s)-1 {
				return strings.TrimSpace(s[:i]), strings.TrimSpace(s[i+1:]), string(r), true
			}
		}
	}
	return "", "", "", false
}

func evalArithmetic(a, b any, op string) any {
	if ta := parseDateAny(a); !ta.IsZero() {
		if d, ok := b.(dataviewDuration); ok {
			if op == "-" {
				return ta.Add(-time.Duration(d))
			}
			if op == "+" {
				return ta.Add(time.Duration(d))
			}
		}
	}
	fa, aok := toFloat(a)
	fb, bok := toFloat(b)
	if aok && bok {
		if op == "-" {
			return fa - fb
		}
		return fa + fb
	}
	return ""
}

func parseDataviewDuration(raw string) dataviewDuration {
	raw = strings.TrimSpace(strings.Trim(raw, "\""))
	fields := strings.Fields(raw)
	if len(fields) == 0 {
		return 0
	}
	n, err := strconv.Atoi(fields[0])
	if err != nil {
		return 0
	}
	unit := "days"
	if len(fields) > 1 {
		unit = strings.ToLower(fields[1])
	}
	switch {
	case strings.HasPrefix(unit, "day"):
		return dataviewDuration(time.Duration(n) * 24 * time.Hour)
	case strings.HasPrefix(unit, "week"):
		return dataviewDuration(time.Duration(n) * 7 * 24 * time.Hour)
	case strings.HasPrefix(unit, "month"):
		return dataviewDuration(time.Duration(n) * 30 * 24 * time.Hour)
	case strings.HasPrefix(unit, "year"):
		return dataviewDuration(time.Duration(n) * 365 * 24 * time.Hour)
	}
	return 0
}

func compareValues(a, b any) int {
	ta, tb := parseDateAny(a), parseDateAny(b)
	if !ta.IsZero() || !tb.IsZero() {
		if ta.Before(tb) {
			return -1
		}
		if ta.After(tb) {
			return 1
		}
		return 0
	}
	if fa, ok := toFloat(a); ok {
		if fb, ok := toFloat(b); ok {
			if fa < fb {
				return -1
			}
			if fa > fb {
				return 1
			}
			return 0
		}
	}
	sa, sb := strings.ToLower(displayPlain(a)), strings.ToLower(displayPlain(b))
	if sa < sb {
		return -1
	}
	if sa > sb {
		return 1
	}
	return 0
}
func parseDateAny(v any) time.Time {
	switch x := v.(type) {
	case time.Time:
		return x
	case string:
		for _, layout := range []string{"2006-01-02", time.RFC3339} {
			if t, err := time.Parse(layout, x); err == nil {
				return t
			}
		}
	}
	return time.Time{}
}
func toFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case string:
		f, err := strconv.ParseFloat(x, 64)
		return f, err == nil
	default:
		return 0, false
	}
}
func displayPlain(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case int, int64, float64, float32, bool:
		return fmt.Sprint(x)
	case time.Time:
		if x.IsZero() {
			return ""
		}
		return x.Format("2006-01-02")
	case dataviewLink:
		return x.Text
	case []string:
		return strings.Join(x, ", ")
	case []any:
		parts := []string{}
		for _, i := range x {
			parts = append(parts, displayPlain(i))
		}
		return strings.Join(parts, ", ")
	case []dataviewLink:
		parts := []string{}
		for _, link := range x {
			parts = append(parts, link.Text)
		}
		return strings.Join(parts, ", ")
	case []dataviewRow:
		return fmt.Sprint(len(x))
	default:
		return fmt.Sprint(x)
	}
}
func truthy(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case string:
		return x != "" && x != "false"
	case nil:
		return false
	default:
		if f, ok := toFloat(x); ok {
			return f != 0
		}
		return true
	}
}
func containsValue(container, item any) bool {
	needle := normalizeDataviewComparable(item)
	for _, v := range toSlice(container) {
		if normalizeDataviewComparable(v) == needle {
			return true
		}
	}
	return strings.Contains(normalizeDataviewComparable(container), needle)
}

func normalizeDataviewComparable(v any) string {
	s := displayPlain(v)
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "[[") && strings.HasSuffix(s, "]]") {
		s = strings.TrimSuffix(strings.TrimPrefix(s, "[["), "]]")
		if i := strings.Index(s, "|"); i >= 0 {
			s = s[:i]
		}
	}
	return s
}

func toSlice(v any) []any {
	switch x := v.(type) {
	case []any:
		return x
	case []string:
		out := make([]any, len(x))
		for i, v := range x {
			out[i] = v
		}
		return out
	case []dataviewLink:
		out := make([]any, len(x))
		for i, v := range x {
			out[i] = v
		}
		return out
	case []dataviewRow:
		out := make([]any, len(x))
		for i, v := range x {
			out[i] = v
		}
		return out
	default:
		if isEmpty(v) {
			return nil
		}
		return []any{v}
	}
}
func lenOf(v any) int {
	switch x := v.(type) {
	case []any:
		return len(x)
	case []string:
		return len(x)
	case []dataviewLink:
		return len(x)
	case []dataviewRow:
		return len(x)
	case string:
		if x == "" {
			return 0
		}
		return len([]rune(x))
	default:
		if isEmpty(v) {
			return 0
		}
		return 1
	}
}
func isEmpty(v any) bool {
	if v == nil {
		return true
	}
	switch x := v.(type) {
	case string:
		return x == ""
	case []any:
		return len(x) == 0
	case []string:
		return len(x) == 0
	}
	return false
}
func copyMap(in map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range in {
		out[k] = v
	}
	return out
}
