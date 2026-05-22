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
	"os/exec"
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
}
type TOCItem struct {
	Level    int
	Text, ID string
}
type SearchResult struct {
	RelPath, URL, Line, Snippet string
	LineNo                      int
}
type Config struct {
	Favorites []string
	DailyGlob string
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
	var out []string
	_ = filepath.WalkDir(v.Root, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && v.IsMarkdown(p) {
			out = append(out, p)
		}
		return nil
	})
	sort.Strings(out)
	return out
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
	inFav := false
	for scanner.Scan() {
		line := scanner.Text()
		tr := strings.TrimSpace(line)
		if strings.HasPrefix(tr, "daily_glob:") {
			cfg.DailyGlob = strings.Trim(strings.TrimSpace(strings.TrimPrefix(tr, "daily_glob:")), "'\"")
		}
		if strings.HasPrefix(tr, "favorites:") {
			inFav = true
			continue
		}
		if inFav && strings.HasPrefix(tr, "-") {
			cfg.Favorites = append(cfg.Favorites, strings.Trim(strings.TrimSpace(strings.TrimPrefix(tr, "-")), "'\""))
			continue
		}
		if inFav && tr != "" && !strings.HasPrefix(tr, "#") {
			inFav = false
		}
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
	return RenderedDoc{Title: r.vault.Title(n), HTML: htmlBody, Toc: tocFromMarkdown(n.Body), Frontmatter: n.Frontmatter}
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
	s = dueRe.ReplaceAllString(s, `<span class="task-meta due-date" title="Due date">📅 $1</span>`)
	doneRe := regexp.MustCompile(`✅\s*(\d{4}-\d{2}-\d{2})`)
	s = doneRe.ReplaceAllString(s, `<span class="task-meta done-date" title="Done date">✅ $1</span>`)
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

func NewSearcher(v *Vault) *Searcher { return &Searcher{vault: v} }
func (s *Searcher) Search(q string) ([]SearchResult, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return nil, nil
	}
	if _, err := exec.LookPath("rg"); err == nil {
		return s.searchRG(q)
	}
	return s.searchFallback(q), nil
}

func (s *Searcher) searchRG(q string) ([]SearchResult, error) {
	cmd := exec.Command("rg", "--json", "-i", "--glob", "*.md", q, s.vault.Root)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 1 {
			return nil, nil
		}
		return nil, err
	}
	var res []SearchResult
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		var obj map[string]any
		if json.Unmarshal(scanner.Bytes(), &obj) != nil || obj["type"] != "match" {
			continue
		}
		data := obj["data"].(map[string]any)
		path := data["path"].(map[string]any)["text"].(string)
		lines := data["lines"].(map[string]any)["text"].(string)
		ln := int(data["line_number"].(float64))
		rel := s.vault.Rel(path)
		res = append(res, SearchResult{RelPath: rel, URL: s.vault.URLForRel(rel), Line: fmt.Sprint(ln), LineNo: ln, Snippet: strings.TrimSpace(lines)})
	}
	return res, nil
}

func (s *Searcher) searchFallback(q string) []SearchResult {
	var res []SearchResult
	ql := strings.ToLower(q)
	for _, p := range s.vault.MarkdownFiles() {
		f, _ := os.Open(p)
		if f == nil {
			continue
		}
		scanner := bufio.NewScanner(f)
		ln := 0
		for scanner.Scan() {
			ln++
			line := scanner.Text()
			if strings.Contains(strings.ToLower(line), ql) {
				rel := s.vault.Rel(p)
				res = append(res, SearchResult{RelPath: rel, URL: s.vault.URLForRel(rel), Line: fmt.Sprint(ln), LineNo: ln, Snippet: strings.TrimSpace(line)})
			}
		}
		_ = f.Close()
	}
	return res
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
	case "/_resolve":
		s.resolve(w, r)
	case "/_missing":
		s.missing(w, r)
	default:
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
	c["Latest"] = s.vault.LatestDaily()
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
		c["Backlinks"] = s.vault.BacklinksTo(n.RelPath)
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
	c["Path"] = s.vault.Rel(p)
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
