package app

import (
	"bufio"
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html"
	"html/template"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	htmlrenderer "github.com/yuin/goldmark/renderer/html"
)

type (
	Vault struct{ Root string }
	Note  struct {
		Path, RelPath, Text, Body string
		Frontmatter               map[string]any
		ModTime                   time.Time
	}
)
type WikiResolution struct {
	Kind, Target, Heading string
	Matches               []Note
}
type RenderedDoc struct {
	Title, HTML string
	Toc         []TOCItem
	Frontmatter map[string]any
	Tags        []string
}
type TOCItem struct {
	Level    int
	Text, ID string
}
type SearchResult struct {
	RelPath, URL, Title, Line, Snippet, SnippetHTML string
	LineNo, Score                                   int
}
type Config struct {
	Favorites []string
	DailyGlob string
	Hidden    []string
}
type TreeNode struct {
	Name, Rel, URL string
	IsDir          bool
	IsActive       bool
	ContainsActive bool
	Children       []TreeNode
}

func NewVault(root string) (*Vault, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("vault is not a directory: %s", abs)
	}
	return &Vault{Root: abs}, nil
}

func (v *Vault) Rel(p string) string { rel, _ := filepath.Rel(v.Root, p); return filepath.ToSlash(rel) }

func (v *Vault) URLForRel(rel string) string {
	rel = filepath.ToSlash(strings.TrimPrefix(rel, "/"))
	parts := strings.Split(rel, "/")
	for i, p := range parts {
		parts[i] = strings.ReplaceAll(url.QueryEscape(p), "+", "%20")
	}
	return "/" + strings.Join(parts, "/")
}

func (v *Vault) ResolveURLPath(urlPath string) (string, error) {
	clean := strings.TrimPrefix(strings.Split(urlPath, "?")[0], "/")
	dec, err := url.QueryUnescape(clean)
	if err != nil {
		return "", err
	}
	joined := filepath.Join(v.Root, filepath.FromSlash(dec))
	abs, err := filepath.Abs(joined)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(v.Root, abs)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return "", errors.New("path escapes vault")
	}
	return abs, nil
}

func (v *Vault) IsMarkdown(p string) bool {
	e := strings.ToLower(filepath.Ext(p))
	return e == ".md" || e == ".markdown"
}

func (v *Vault) MarkdownFiles() []string {
	cfg := v.LoadConfig()
	var out []string
	_ = filepath.WalkDir(v.Root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel := v.Rel(p)
		if rel != "." && v.isHiddenRel(rel, cfg.Hidden) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.IsDir() && v.IsMarkdown(p) {
			out = append(out, p)
		}
		return nil
	})
	sort.Strings(out)
	return out
}

func (v *Vault) isHiddenRel(rel string, configured []string) bool {
	rel = filepath.ToSlash(strings.Trim(rel, "/"))
	if rel == "." || rel == "" {
		return false
	}
	for _, part := range strings.Split(rel, "/") {
		if strings.HasPrefix(part, ".") {
			return true
		}
	}
	for _, hidden := range configured {
		hidden = filepath.ToSlash(strings.Trim(hidden, " /"))
		if hidden == "" {
			continue
		}
		if rel == hidden || strings.HasPrefix(rel, hidden+"/") {
			return true
		}
	}
	return false
}

func (v *Vault) ReadNote(relOrAbs string) (Note, error) {
	p := relOrAbs
	if !filepath.IsAbs(p) {
		p = filepath.Join(v.Root, filepath.FromSlash(p))
	}
	abs, _ := filepath.Abs(p)
	rel, err := filepath.Rel(v.Root, abs)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return Note{}, errors.New("path escapes vault")
	}
	b, err := os.ReadFile(abs)
	if err != nil {
		return Note{}, err
	}
	st, _ := os.Stat(abs)
	fm, body := parseFrontmatter(string(b))
	mt := time.Time{}
	if st != nil {
		mt = st.ModTime()
	}
	return Note{Path: abs, RelPath: filepath.ToSlash(rel), Text: string(b), Body: body, Frontmatter: fm, ModTime: mt}, nil
}

func (v *Vault) Title(n Note) string {
	if s, ok := n.Frontmatter["title"].(string); ok && strings.TrimSpace(s) != "" {
		return strings.TrimSpace(s)
	}
	re := regexp.MustCompile(`(?m)^#\s+(.+)$`)
	if m := re.FindStringSubmatch(n.Body); len(m) > 1 {
		return stripMD(m[1])
	}
	return strings.TrimSuffix(filepath.Base(n.RelPath), filepath.Ext(n.RelPath))
}

func (v *Vault) ResolveWikiLink(raw string) WikiResolution {
	target := strings.TrimSpace(strings.Split(raw, "|")[0])
	heading := ""
	if i := strings.Index(target, "#"); i >= 0 {
		heading = target[i+1:]
		target = target[:i]
	}
	var files []string
	explicit := strings.Contains(target, "/") || strings.HasSuffix(strings.ToLower(target), ".md")
	if explicit {
		t := target
		if filepath.Ext(t) == "" {
			t += ".md"
		}
		p := filepath.Join(v.Root, filepath.FromSlash(t))
		if st, err := os.Stat(p); err == nil && !st.IsDir() && v.IsMarkdown(p) {
			files = append(files, p)
		}
	} else {
		for _, p := range v.MarkdownFiles() {
			stem := strings.TrimSuffix(filepath.Base(p), filepath.Ext(p))
			if stem == target || filepath.Base(p) == target {
				files = append(files, p)
			}
		}
	}
	var notes []Note
	for _, p := range files {
		if n, err := v.ReadNote(p); err == nil {
			notes = append(notes, n)
		}
	}
	kind := "missing"
	if len(notes) == 1 {
		kind = "unique"
	} else if len(notes) > 1 {
		kind = "ambiguous"
	}
	return WikiResolution{Kind: kind, Target: target, Heading: heading, Matches: notes}
}

func (v *Vault) BacklinksTo(relPath string) []Note {
	var out []Note
	base := strings.TrimSuffix(filepath.Base(relPath), filepath.Ext(relPath))
	noExt := strings.TrimSuffix(relPath, filepath.Ext(relPath))
	for _, p := range v.MarkdownFiles() {
		n, _ := v.ReadNote(p)
		if n.RelPath == relPath {
			continue
		}
		if strings.Contains(n.Text, "[["+base) || strings.Contains(n.Text, "[["+noExt) || strings.Contains(n.Text, relPath) || strings.Contains(n.Text, url.QueryEscape(relPath)) {
			out = append(out, n)
		}
	}
	return out
}

func (v *Vault) Favorites() []map[string]string {
	cfg := v.LoadConfig()
	if len(cfg.Favorites) == 0 {
		cfg.Favorites = []string{"Areas/Daily Briefings", "Areas/TODO.md", "Projects"}
	}
	var out []map[string]string
	for _, f := range cfg.Favorites {
		f = strings.Trim(f, " /")
		out = append(out, map[string]string{"Label": filepath.Base(f), "Rel": f, "URL": v.URLForRel(f)})
	}
	return out
}

func (v *Vault) LoadConfig() Config {
	cfg := Config{DailyGlob: "Areas/Daily Briefings/*-briefing.md"}
	b, err := os.ReadFile(filepath.Join(v.Root, ".notes-web.yaml"))
	if err != nil {
		return cfg
	}
	scanner := bufio.NewScanner(strings.NewReader(string(b)))
	section := ""
	for scanner.Scan() {
		tr := strings.TrimSpace(scanner.Text())
		if tr == "" || strings.HasPrefix(tr, "#") {
			continue
		}
		if strings.HasPrefix(tr, "daily_glob:") {
			cfg.DailyGlob = strings.Trim(strings.TrimSpace(strings.TrimPrefix(tr, "daily_glob:")), "'\"")
			section = ""
			continue
		}
		if strings.HasPrefix(tr, "favorites:") {
			section = "favorites"
			continue
		}
		if strings.HasPrefix(tr, "hidden:") {
			section = "hidden"
			continue
		}
		if strings.HasPrefix(tr, "-") {
			value := strings.Trim(strings.TrimSpace(strings.TrimPrefix(tr, "-")), "'\"")
			switch section {
			case "favorites":
				cfg.Favorites = append(cfg.Favorites, value)
			case "hidden":
				cfg.Hidden = append(cfg.Hidden, value)
			}
			continue
		}
		section = ""
	}
	return cfg
}

func (v *Vault) LatestDaily() *Note {
	cfg := v.LoadConfig()
	var best *Note
	for _, p := range v.MarkdownFiles() {
		rel := v.Rel(p)
		ok, _ := filepath.Match(filepath.ToSlash(cfg.DailyGlob), rel)
		if ok {
			n, _ := v.ReadNote(p)
			if best == nil || n.ModTime.After(best.ModTime) {
				best = &n
			}
		}
	}
	return best
}

func (v *Vault) RecentNotes(limit int) []Note {
	files := v.MarkdownFiles()
	notes := []Note{}
	for _, p := range files {
		n, _ := v.ReadNote(p)
		notes = append(notes, n)
	}
	sort.Slice(notes, func(i, j int) bool { return notes[i].ModTime.After(notes[j].ModTime) })
	if len(notes) > limit {
		notes = notes[:limit]
	}
	return notes
}
func (v *Vault) Tree(maxDepth int) []TreeNode { return v.TreeForActive(maxDepth, "") }
func (v *Vault) TreeForActive(maxDepth int, activeRel string) []TreeNode {
	return v.tree(v.Root, 0, maxDepth, filepath.ToSlash(strings.TrimPrefix(activeRel, "/")))
}
func (v *Vault) tree(dir string, depth, max int, activeRel string) []TreeNode {
	ents, _ := os.ReadDir(dir)
	sort.Slice(ents, func(i, j int) bool {
		if ents[i].IsDir() != ents[j].IsDir() {
			return ents[i].IsDir()
		}
		return strings.ToLower(ents[i].Name()) < strings.ToLower(ents[j].Name())
	})
	var out []TreeNode
	for _, e := range ents {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		p := filepath.Join(dir, e.Name())
		if !e.IsDir() && !v.IsMarkdown(p) {
			continue
		}
		rel := v.Rel(p)
		n := TreeNode{Name: e.Name(), Rel: rel, URL: v.URLForRel(rel), IsDir: e.IsDir()}
		if e.IsDir() && depth < max {
			n.Children = v.tree(p, depth+1, max, activeRel)
		}
		if activeRel != "" {
			n.IsActive = rel == activeRel
			n.ContainsActive = n.IsActive || strings.HasPrefix(activeRel, rel+"/")
		}
		out = append(out, n)
	}
	return out
}

func parseFrontmatter(s string) (map[string]any, string) {
	fm := map[string]any{}
	if strings.HasPrefix(s, "---\n") {
		if idx := strings.Index(s[4:], "\n---"); idx >= 0 {
			raw := s[4 : 4+idx]
			body := s[4+idx+len("\n---"):]
			body = strings.TrimPrefix(body, "\n")
			for _, line := range strings.Split(raw, "\n") {
				if k, v, ok := strings.Cut(line, ":"); ok {
					val := strings.TrimSpace(v)
					val = strings.Trim(val, "[]\"")
					fm[strings.TrimSpace(k)] = val
				}
			}
			return fm, body
		}
	}
	return fm, s
}

func stripMD(s string) string {
	return strings.TrimSpace(regexp.MustCompile("[`*_~]").ReplaceAllString(s, ""))
}

func slugify(s string) string {
	s = strings.ToLower(stripMD(s))
	s = regexp.MustCompile(`[^a-z0-9\pL]+`).ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

type Renderer struct {
	vault *Vault
	md    goldmark.Markdown
}

func NewRenderer(v *Vault) *Renderer {
	return &Renderer{vault: v, md: goldmark.New(goldmark.WithExtensions(extension.GFM, extension.Footnote, highlighting.NewHighlighting(highlighting.WithStyle("github"))), goldmark.WithParserOptions(parser.WithAutoHeadingID()), goldmark.WithRendererOptions(htmlrenderer.WithUnsafe()))}
}

func (r *Renderer) Render(n Note) RenderedDoc {
	body := r.preprocess(n.Body)
	var buf bytes.Buffer
	_ = r.md.Convert([]byte(body), &buf)
	fm := renderFrontmatter(n.Frontmatter)
	htmlBody := normalizeRenderedHTML(fm + buf.String())
	return RenderedDoc{Title: r.vault.Title(n), HTML: htmlBody, Toc: tocFromMarkdown(n.Body), Frontmatter: n.Frontmatter, Tags: extractTags(n)}
}

func normalizeRenderedHTML(s string) string {
	s = decorateCalloutsHTML(s)
	s = strings.ReplaceAll(s, `<input checked="" disabled="" type="checkbox">`, `<input type="checkbox" checked disabled>`)
	s = strings.ReplaceAll(s, `<input disabled="" type="checkbox">`, `<input type="checkbox" disabled>`)
	s = decorateTaskLists(s)
	s = decorateTaskMetadata(s)
	tidRe := regexp.MustCompile(`<!--\s*tid:([A-Za-z0-9_-]+)\s*-->`)
	s = tidRe.ReplaceAllString(s, `<button class="task-id" data-copy="$1" title="Copy task ID">tid:$1</button>`)
	return s
}

func decorateTaskLists(s string) string {
	if !strings.Contains(s, `type="checkbox"`) {
		return s
	}
	s = strings.ReplaceAll(s, "<ul>\n<li><input", "<ul class=\"contains-task-list\">\n<li class=\"task-list-item\"><input")
	s = strings.ReplaceAll(s, "</li>\n<li><input", "</li>\n<li class=\"task-list-item\"><input")
	return s
}

func decorateTaskMetadata(s string) string {
	dueRe := regexp.MustCompile(`📅\s*(\d{4}-\d{2}-\d{2})`)
	s = dueRe.ReplaceAllString(s, `<span class="task-meta due-date" title="Due date">Due $1</span>`)
	doneRe := regexp.MustCompile(`✅\s*(\d{4}-\d{2}-\d{2})`)
	s = doneRe.ReplaceAllString(s, `<span class="task-meta done-date" title="Done date">Done $1</span>`)
	recurRe := regexp.MustCompile(`🔁\s*([^<
]+)`)
	s = recurRe.ReplaceAllString(s, `<span class="task-meta repeat-meta" title="Repeats">Repeats $1</span>`)
	priorityRe := regexp.MustCompile(`[⏫🔼🔽⏬]`)
	s = priorityRe.ReplaceAllString(s, `<span class="task-meta priority-meta" title="Priority">Priority</span>`)
	return s
}

func (r *Renderer) preprocess(s string) string {
	s = preprocessCallouts(s)
	s = preprocessMermaid(s)
	re := regexp.MustCompile(`\[\[([^\]]+)\]\]`)
	return re.ReplaceAllStringFunc(s, func(m string) string {
		inner := strings.TrimSuffix(strings.TrimPrefix(m, "[["), "]]")
		parts := strings.SplitN(inner, "|", 2)
		label := parts[0]
		if len(parts) == 2 {
			label = parts[1]
		}
		res := r.vault.ResolveWikiLink(inner)
		switch res.Kind {
		case "unique":
			u := r.vault.URLForRel(res.Matches[0].RelPath)
			if res.Heading != "" {
				u += "#" + slugify(res.Heading)
			}
			return "[" + label + "](" + u + ")"
		case "ambiguous":
			return "[" + label + "](/_resolve?name=" + url.QueryEscape(res.Target) + ")"
		default:
			return "[" + label + "](/_missing?name=" + url.QueryEscape(res.Target) + ")"
		}
	})
}

func preprocessCallouts(s string) string {
	return s
}

func decorateCalloutsHTML(s string) string {
	re := regexp.MustCompile(`(?s)<blockquote>\s*<p>\[!(\w+)\]([+-]?)\s*([^\n<]*)\n(.*?)</p>\s*</blockquote>`)
	return re.ReplaceAllStringFunc(s, func(match string) string {
		parts := re.FindStringSubmatch(match)
		if len(parts) != 5 {
			return match
		}
		kind := strings.ToLower(parts[1])
		fold := parts[2]
		title := strings.TrimSpace(parts[3])
		body := strings.TrimSpace(parts[4])
		if title == "" {
			title = defaultCalloutTitle(kind)
		}
		classes := "callout " + html.EscapeString(kind) + " callout-" + html.EscapeString(kind)
		if fold == "-" {
			classes += " is-collapsed"
		}
		var b strings.Builder
		b.WriteString(`<div class="` + classes + `" data-callout="` + html.EscapeString(kind) + `">`)
		b.WriteString(`<div class="callout-title"><span class="callout-icon" aria-hidden="true">` + calloutIcon(kind) + `</span><span class="callout-title-text">` + html.EscapeString(title) + `</span></div>`)
		if body != "" {
			b.WriteString(`<div class="callout-body"><p>` + body + `</p></div>`)
		}
		b.WriteString(`</div>`)
		return b.String()
	})
}

func defaultCalloutTitle(kind string) string {
	switch strings.ToLower(kind) {
	case "note":
		return "Note"
	case "abstract", "summary", "tldr":
		return "Summary"
	case "info":
		return "Info"
	case "todo":
		return "Todo"
	case "tip", "hint", "important":
		return "Tip"
	case "success", "check", "done":
		return "Success"
	case "question", "help", "faq":
		return "Question"
	case "warning", "caution", "attention":
		return "Warning"
	case "failure", "fail", "missing":
		return "Failure"
	case "danger", "error":
		return "Danger"
	case "bug":
		return "Bug"
	case "example":
		return "Example"
	case "quote", "cite":
		return "Quote"
	default:
		if kind == "" {
			return "Note"
		}
		return strings.ToUpper(kind[:1]) + kind[1:]
	}
}

func calloutIcon(kind string) string {
	switch strings.ToLower(kind) {
	case "note":
		return "📝"
	case "abstract", "summary", "tldr":
		return "📌"
	case "info":
		return "ℹ️"
	case "todo":
		return "☑️"
	case "tip", "hint", "important":
		return "💡"
	case "success", "check", "done":
		return "✅"
	case "question", "help", "faq":
		return "❓"
	case "warning", "caution", "attention":
		return "⚠️"
	case "failure", "fail", "missing":
		return "✖️"
	case "danger", "error":
		return "🚨"
	case "bug":
		return "🐞"
	case "example":
		return "📎"
	case "quote", "cite":
		return "❝"
	default:
		return "📝"
	}
}

func preprocessMermaid(s string) string {
	re := regexp.MustCompile("(?s)```mermaid\n(.*?)\n```")
	return re.ReplaceAllString(s, "<pre class=\"mermaid\">$1</pre>")
}

func renderFrontmatter(fm map[string]any) string {
	if len(fm) == 0 {
		return ""
	}
	keys := make([]string, 0, len(fm))
	for k := range fm {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString(`<details class="frontmatter" open><summary>Frontmatter</summary><dl>`)
	for _, k := range keys {
		b.WriteString("<dt>" + html.EscapeString(k) + "</dt><dd>" + html.EscapeString(fmt.Sprint(fm[k])) + "</dd>")
	}
	b.WriteString("</dl></details>")
	return b.String()
}

func tocFromMarkdown(s string) []TOCItem {
	re := regexp.MustCompile(`(?m)^(#{1,6})\s+(.+)$`)
	var out []TOCItem
	for _, m := range re.FindAllStringSubmatch(s, -1) {
		txt := stripMD(m[2])
		out = append(out, TOCItem{Level: len(m[1]), Text: txt, ID: slugify(txt)})
	}
	return out
}

type Searcher struct{ vault *Vault }

type ParsedSearchQuery struct {
	Terms       []string
	Tag         string
	Path        string
	Title       string
	Frontmatter map[string]string
}

func NewSearcher(v *Vault) *Searcher { return &Searcher{vault: v} }

func (s *Searcher) Search(q string) ([]SearchResult, error) {
	parsed := ParseSearchQuery(q)
	if parsed.Empty() {
		return nil, nil
	}
	idx, err := s.vault.BuildIndex()
	if err != nil {
		return nil, err
	}
	var results []SearchResult
	for _, meta := range idx.Notes {
		note, err := s.vault.ReadNote(meta.RelPath)
		if err != nil {
			return nil, err
		}
		if !parsed.MatchesFilters(meta, note) {
			continue
		}
		line, lineNo := bestSearchLineForTerms(note, parsed.Terms)
		score := searchScoreForQuery(meta, note, parsed, lineNo > 0)
		if score == 0 {
			continue
		}
		if line == "" {
			line = meta.Title
		}
		highlightTerms := parsed.HighlightTerms()
		results = append(results, SearchResult{
			RelPath:     meta.RelPath,
			URL:         meta.URL,
			Title:       meta.Title,
			Line:        fmt.Sprint(lineNo),
			LineNo:      lineNo,
			Snippet:     strings.TrimSpace(line),
			SnippetHTML: highlightSearchSnippet(line, highlightTerms),
			Score:       score,
		})
	}
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		if strings.ToLower(results[i].Title) != strings.ToLower(results[j].Title) {
			return strings.ToLower(results[i].Title) < strings.ToLower(results[j].Title)
		}
		return results[i].RelPath < results[j].RelPath
	})
	return results, nil
}

func ParseSearchQuery(q string) ParsedSearchQuery {
	parsed := ParsedSearchQuery{Frontmatter: map[string]string{}}
	for _, token := range tokenizeSearchQuery(q) {
		key, value, ok := strings.Cut(token, ":")
		if ok {
			key = strings.ToLower(strings.TrimSpace(key))
			value = strings.TrimSpace(value)
			switch key {
			case "tag":
				parsed.Tag = normalizeTag(value)
				continue
			case "path":
				parsed.Path = strings.ToLower(value)
				continue
			case "title":
				parsed.Title = strings.ToLower(value)
				continue
			case "frontmatter", "fm":
				if fmKey, fmValue, ok := strings.Cut(value, "="); ok {
					parsed.Frontmatter[strings.ToLower(strings.TrimSpace(fmKey))] = strings.ToLower(strings.TrimSpace(fmValue))
					continue
				}
			}
		}
		term := strings.TrimSpace(token)
		if term != "" {
			parsed.Terms = append(parsed.Terms, strings.ToLower(term))
		}
	}
	return parsed
}

func tokenizeSearchQuery(q string) []string {
	var tokens []string
	var b strings.Builder
	inQuote := false
	for _, r := range q {
		switch r {
		case '"':
			inQuote = !inQuote
		case ' ', '\t', '\n':
			if inQuote {
				b.WriteRune(r)
			} else if b.Len() > 0 {
				tokens = append(tokens, b.String())
				b.Reset()
			}
		default:
			b.WriteRune(r)
		}
	}
	if b.Len() > 0 {
		tokens = append(tokens, b.String())
	}
	return tokens
}

func (q ParsedSearchQuery) Empty() bool {
	return q.Tag == "" && q.Path == "" && q.Title == "" && len(q.Frontmatter) == 0 && len(q.Terms) == 0
}

func (q ParsedSearchQuery) MatchesFilters(meta NoteMeta, note Note) bool {
	if q.Tag != "" && !containsString(meta.Tags, q.Tag) {
		return false
	}
	if q.Path != "" && !strings.Contains(strings.ToLower(meta.RelPath), q.Path) {
		return false
	}
	if q.Title != "" && !strings.Contains(strings.ToLower(meta.Title), q.Title) {
		return false
	}
	for key, want := range q.Frontmatter {
		got, ok := note.Frontmatter[key]
		if !ok || !strings.Contains(strings.ToLower(fmt.Sprint(got)), want) {
			return false
		}
	}
	return true
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func (q ParsedSearchQuery) HighlightTerms() []string {
	if len(q.Terms) > 0 {
		return q.Terms
	}
	var out []string
	if q.Tag != "" {
		out = append(out, q.Tag)
	}
	if q.Path != "" {
		out = append(out, q.Path)
	}
	if q.Title != "" {
		out = append(out, q.Title)
	}
	for _, value := range q.Frontmatter {
		out = append(out, value)
	}
	return out
}

func searchScoreForQuery(meta NoteMeta, note Note, query ParsedSearchQuery, hasContentMatch bool) int {
	score := 0
	title := strings.ToLower(meta.Title)
	rel := strings.ToLower(meta.RelPath)
	body := strings.ToLower(note.Body)
	for _, term := range query.Terms {
		switch {
		case title == term:
			score += 1000
		case strings.Contains(title, term):
			score += 700
		}
		if strings.Contains(rel, term) {
			score += 300
		}
		if strings.Contains(body, term) {
			score += 100
		}
	}
	if query.Tag != "" {
		score += 250
	}
	if query.Path != "" {
		score += 200
	}
	if query.Title != "" {
		score += 500
	}
	if len(query.Frontmatter) > 0 {
		score += 250
	}
	if hasContentMatch {
		score += 25
	}
	return score
}

func bestSearchLineForTerms(note Note, terms []string) (string, int) {
	if len(terms) == 0 {
		return "", 0
	}
	lines := strings.Split(note.Body, "\n")
	for i, line := range lines {
		lower := strings.ToLower(line)
		for _, term := range terms {
			if strings.Contains(lower, term) {
				return line, i + 1
			}
		}
	}
	return "", 0
}

func highlightSearchSnippet(snippet string, terms []string) string {
	escaped := html.EscapeString(strings.TrimSpace(snippet))
	for _, term := range terms {
		term = strings.TrimSpace(term)
		if term == "" {
			continue
		}
		re := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(html.EscapeString(term)))
		escaped = re.ReplaceAllStringFunc(escaped, func(match string) string { return `<mark>` + match + `</mark>` })
	}
	return escaped
}

type Breadcrumb struct {
	Label   string
	URL     string
	Current bool
}

func breadcrumbsForRel(v *Vault, rel string) []Breadcrumb {
	parts := strings.Split(filepath.ToSlash(rel), "/")
	crumbs := []Breadcrumb{{Label: "Home", URL: "/"}}
	for i, part := range parts {
		if part == "" {
			continue
		}
		partial := strings.Join(parts[:i+1], "/")
		crumbs = append(crumbs, Breadcrumb{Label: part, URL: v.URLForRel(partial), Current: i == len(parts)-1})
	}
	return crumbs
}

type Server struct {
	vault      *Vault
	renderer   *Renderer
	searcher   *Searcher
	user, pass string
	templates  *template.Template
}

func NewServer(v *Vault, user, pass string) *Server {
	s := &Server{vault: v, renderer: NewRenderer(v), searcher: NewSearcher(v), user: user, pass: pass}
	s.templates = template.Must(template.New("all").Funcs(template.FuncMap{"safe": func(x string) template.HTML { return template.HTML(x) }, "url": v.URLForRel}).Parse(templates))
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !s.auth(w, r) {
		return
	}
	if strings.HasPrefix(r.URL.Path, "/_static/") {
		s.static(w, r)
		return
	}
	switch r.URL.Path {
	case "/":
		s.home(w, r)
	case "/_search":
		s.search(w, r)
	case "/_api/palette":
		s.paletteAPI(w, r)
	case "/_resolve":
		s.resolve(w, r)
	case "/_missing":
		s.missing(w, r)
	case "/_tags":
		s.tags(w, r)
	case "/_todo":
		s.todo(w, r)
	case "/_broken-links":
		s.brokenLinks(w, r)
	case "/_orphans":
		s.orphans(w, r)
	default:
		if strings.HasPrefix(r.URL.Path, "/_tags/") {
			s.tag(w, r)
			return
		}
		s.path(w, r)
	}
}

func (s *Server) auth(w http.ResponseWriter, r *http.Request) bool {
	if s.user == "" {
		return true
	}
	u, p, ok := r.BasicAuth()
	if ok && subtle.ConstantTimeCompare([]byte(u), []byte(s.user)) == 1 && subtle.ConstantTimeCompare([]byte(p), []byte(s.pass)) == 1 {
		return true
	}
	w.Header().Set("WWW-Authenticate", `Basic realm="notes-web"`)
	http.Error(w, "authentication required", http.StatusUnauthorized)
	return false
}

func (s *Server) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = s.templates.ExecuteTemplate(w, name, data)
}

func (s *Server) common(title string) map[string]any {
	return s.commonForActive(title, "")
}

func (s *Server) commonForActive(title, activeRel string) map[string]any {
	return map[string]any{"Title": title, "Tree": s.vault.TreeForActive(2, activeRel), "Favorites": s.vault.Favorites(), "ActiveRel": activeRel}
}

func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	c := s.common("Home")
	dashboard, err := s.vault.BuildDashboard()
	c["Dashboard"] = dashboard
	c["Err"] = err
	c["Latest"] = dashboard.LatestDaily
	c["Recent"] = s.vault.RecentNotes(10)
	s.render(w, "home", c)
}

func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	res, err := s.searcher.Search(q)
	c := s.common("Search")
	c["Q"] = q
	c["Results"] = res
	c["Err"] = err
	s.render(w, "search", c)
}

func (s *Server) resolve(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	res := s.vault.ResolveWikiLink(name)
	c := s.common("Choose a note")
	c["Name"] = name
	c["Matches"] = res.Matches
	s.render(w, "resolve", c)
}

func (s *Server) missing(w http.ResponseWriter, r *http.Request) {
	c := s.common("Note not found")
	c["Name"] = r.URL.Query().Get("name")
	w.WriteHeader(404)
	s.render(w, "missing", c)
}

type paletteItem struct {
	Title string `json:"title"`
	URL   string `json:"url"`
	Kind  string `json:"kind"`
	Path  string `json:"path,omitempty"`
}

func (s *Server) paletteAPI(w http.ResponseWriter, r *http.Request) {
	idx, err := s.vault.BuildIndex()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	items := make([]paletteItem, 0, len(idx.Notes)+len(idx.Tags)+len(s.vault.Favorites()))
	for _, fav := range s.vault.Favorites() {
		items = append(items, paletteItem{Title: fav["Label"], URL: fav["URL"], Kind: "favorite", Path: fav["Rel"]})
	}
	for _, note := range idx.Notes {
		items = append(items, paletteItem{Title: note.Title, URL: note.URL, Kind: "note", Path: note.RelPath})
	}
	for tag := range idx.Tags {
		items = append(items, paletteItem{Title: "#" + tag, URL: "/_tags/" + url.PathEscape(tag), Kind: "tag"})
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Kind != items[j].Kind {
			return items[i].Kind < items[j].Kind
		}
		return strings.ToLower(items[i].Title) < strings.ToLower(items[j].Title)
	})
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(items)
}

func (s *Server) brokenLinks(w http.ResponseWriter, r *http.Request) {
	idx, err := s.vault.BuildIndex()
	c := s.common("Broken links")
	c["Err"] = err
	if idx != nil {
		links := BrokenWikiLinks(idx, NewIndexResolver(idx))
		c["BrokenLinks"] = links
		c["BrokenGroups"] = GroupBrokenWikiLinks(links, 50)
		c["BrokenTotal"] = len(links)
		c["BrokenDistinctTargets"] = BrokenDistinctTargetCount(links)
		c["BrokenAffectedNotes"] = BrokenAffectedNoteCount(links)
		c["BrokenTopLimit"] = 50
	}
	s.render(w, "broken-links", c)
}

func (s *Server) orphans(w http.ResponseWriter, r *http.Request) {
	idx, err := s.vault.BuildIndex()
	c := s.common("Orphan notes")
	c["Err"] = err
	if idx != nil {
		orphans := OrphanNotes(idx, NewIndexResolver(idx))
		c["Orphans"] = orphans
		c["OrphanTotal"] = len(orphans)
	}
	s.render(w, "orphans", c)
}

func (s *Server) todo(w http.ResponseWriter, r *http.Request) {
	today := r.URL.Query().Get("today")
	if today == "" {
		today = time.Now().Format("2006-01-02")
	}
	board, err := s.vault.BuildTaskBoard(today)
	c := s.common("TODOs")
	c["Err"] = err
	c["Today"] = today
	c["Board"] = board
	s.render(w, "todo", c)
}

func (s *Server) tags(w http.ResponseWriter, r *http.Request) {
	idx, err := s.vault.BuildIndex()
	c := s.common("Tags")
	c["Err"] = err
	tags := tagSummaries(idx)
	c["Tags"] = tags
	c["PopularTags"] = popularTagSummaries(tags, 24)
	c["RareTags"] = rareTagSummaries(tags, 80)
	c["TagGroups"] = tagAlphabeticalGroups(tags)
	c["TagTotal"] = len(tags)
	c["OneOffTagCount"] = countOneOffTags(tags)
	s.render(w, "tags", c)
}

func (s *Server) tag(w http.ResponseWriter, r *http.Request) {
	tag := normalizeTag(strings.TrimPrefix(r.URL.Path, "/_tags/"))
	idx, err := s.vault.BuildIndex()
	c := s.common("#" + tag)
	c["Err"] = err
	c["Tag"] = tag
	if idx != nil {
		c["Notes"] = idx.Tags[tag]
	}
	s.render(w, "tag", c)
}

type TagSummary struct {
	Tag   string
	Count int
	URL   string
}

type TagGroup struct {
	Letter string
	Tags   []TagSummary
}

func tagSummaries(idx *VaultIndex) []TagSummary {
	if idx == nil {
		return nil
	}
	keys := make([]string, 0, len(idx.Tags))
	for tag := range idx.Tags {
		keys = append(keys, tag)
	}
	sort.Strings(keys)
	out := make([]TagSummary, 0, len(keys))
	for _, tag := range keys {
		out = append(out, TagSummary{Tag: tag, Count: len(idx.Tags[tag]), URL: "/_tags/" + url.PathEscape(tag)})
	}
	return out
}

func popularTagSummaries(tags []TagSummary, limit int) []TagSummary {
	out := append([]TagSummary(nil), tags...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Tag < out[j].Tag
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

func rareTagSummaries(tags []TagSummary, limit int) []TagSummary {
	var out []TagSummary
	for _, tag := range tags {
		if tag.Count <= 1 {
			out = append(out, tag)
		}
	}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

func countOneOffTags(tags []TagSummary) int {
	count := 0
	for _, tag := range tags {
		if tag.Count <= 1 {
			count++
		}
	}
	return count
}

func tagAlphabeticalGroups(tags []TagSummary) []TagGroup {
	groups := []TagGroup{}
	byLetter := map[string][]TagSummary{}
	for _, tag := range tags {
		if tag.Count <= 1 {
			continue
		}
		letter := "#"
		if tag.Tag != "" {
			r := []rune(strings.ToUpper(tag.Tag))[0]
			if r >= 'A' && r <= 'Z' {
				letter = string(r)
			}
		}
		byLetter[letter] = append(byLetter[letter], tag)
	}
	letters := make([]string, 0, len(byLetter))
	for letter := range byLetter {
		letters = append(letters, letter)
	}
	sort.Strings(letters)
	for _, letter := range letters {
		items := byLetter[letter]
		if len(items) > 40 {
			items = items[:40]
		}
		groups = append(groups, TagGroup{Letter: letter, Tags: items})
	}
	return groups
}

func (s *Server) path(w http.ResponseWriter, r *http.Request) {
	p, err := s.vault.ResolveURLPath(r.URL.Path)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	st, err := os.Stat(p)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if st.IsDir() {
		s.folder(w, r, p)
		return
	}
	if s.vault.IsMarkdown(p) {
		n, err := s.vault.ReadNote(p)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		doc := s.renderer.Render(n)
		c := s.commonForActive(doc.Title, n.RelPath)
		c["Note"] = n
		c["Doc"] = doc
		c["Breadcrumbs"] = breadcrumbsForRel(s.vault, n.RelPath)
		c["ForwardLinks"] = s.vault.ForwardLinksFrom(n)
		c["Backlinks"] = s.vault.BacklinksWithContext(n.RelPath)
		s.render(w, "note", c)
		return
	}
	s.file(w, r, p, st)
}

func (s *Server) folder(w http.ResponseWriter, r *http.Request, p string) {
	ents, _ := os.ReadDir(p)
	var items []map[string]any
	for _, e := range ents {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		pp := filepath.Join(p, e.Name())
		rel := s.vault.Rel(pp)
		items = append(items, map[string]any{"Name": e.Name(), "Dir": e.IsDir(), "URL": s.vault.URLForRel(rel)})
	}
	c := s.common(filepath.Base(p))
	relPath := s.vault.Rel(p)
	c["Path"] = relPath
	c["FolderName"] = filepath.Base(p)
	c["Breadcrumbs"] = breadcrumbsForRel(s.vault, relPath)
	c["Items"] = items
	s.render(w, "folder", c)
}

func (s *Server) file(w http.ResponseWriter, r *http.Request, p string, st os.FileInfo) {
	mt := mime.TypeByExtension(filepath.Ext(p))
	if mt != "" {
		w.Header().Set("Content-Type", mt)
	}
	http.ServeFile(w, r, p)
}

func (s *Server) static(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/_static/style.css" {
		w.Header().Set("Content-Type", "text/css")
		io.WriteString(w, css)
		return
	}
	if r.URL.Path == "/_static/app.js" {
		w.Header().Set("Content-Type", "application/javascript")
		io.WriteString(w, js)
		return
	}
	http.NotFound(w, r)
}

func Main(args []string) error {
	fs := flag.NewFlagSet("notes-web", flag.ContinueOnError)
	vault := fs.String("vault", "/home/arkan/hermes", "vault path")
	host := fs.String("host", "127.0.0.1", "host")
	port := fs.Int("port", 8080, "port")
	user := fs.String("user", "", "basic auth user")
	passwordEnv := fs.String("password-env", "", "environment variable containing password")
	if err := fs.Parse(args); err != nil {
		return err
	}
	v, err := NewVault(*vault)
	if err != nil {
		return err
	}
	pass := ""
	if *passwordEnv != "" {
		pass = os.Getenv(*passwordEnv)
		if pass == "" {
			return fmt.Errorf("%s is empty", *passwordEnv)
		}
	}
	srv := NewServer(v, *user, pass)
	addr := fmt.Sprintf("%s:%d", *host, *port)
	fmt.Printf("notes-web serving %s on http://%s\n", v.Root, addr)
	return http.ListenAndServe(addr, srv)
}
