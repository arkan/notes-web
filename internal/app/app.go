package app

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
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

func (v *Vault) HiddenRel(rel string) bool {
	return v.isHiddenRel(rel, v.LoadConfig().Hidden)
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
	return v.tree(v.Root, 0, maxDepth, filepath.ToSlash(strings.TrimPrefix(activeRel, "/")), v.LoadConfig().Hidden)
}
func (v *Vault) tree(dir string, depth, max int, activeRel string, hidden []string) []TreeNode {
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
		rel := v.Rel(p)
		if v.isHiddenRel(rel, hidden) {
			continue
		}
		if !e.IsDir() && !v.IsMarkdown(p) {
			continue
		}
		n := TreeNode{Name: e.Name(), Rel: rel, URL: v.URLForRel(rel), IsDir: e.IsDir()}
		if e.IsDir() && depth < max {
			n.Children = v.tree(p, depth+1, max, activeRel, hidden)
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
	s.templates = template.Must(template.New("all").Funcs(template.FuncMap{"safe": func(x string) template.HTML { return template.HTML(x) }, "url": v.URLForRel}).ParseFS(templateFS, "templates/*.html"))
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

const sidebarTreeDepth = 3

func (s *Server) commonForActive(title, activeRel string) map[string]any {
	return map[string]any{"Title": title, "Tree": s.vault.TreeForActive(sidebarTreeDepth, activeRel), "Favorites": s.vault.Favorites(), "ActiveRel": activeRel}
}

func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	c := s.common("Home")
	var selected time.Time
	if raw := r.URL.Query().Get("date"); raw != "" {
		if parsed, err := time.Parse("2006-01-02", raw); err == nil {
			selected = parsed
		}
	} else if raw := r.URL.Query().Get("month"); raw != "" {
		if parsed, err := time.Parse("2006-01", raw); err == nil {
			selected = parsed
		}
	}
	dashboard, err := s.vault.BuildDashboardFor(selected)
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
		items = append(items, paletteItem{Title: fav.Label, URL: fav.URL, Kind: "favorite", Path: fav.Path})
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
	var notes []NoteMeta
	if idx != nil {
		notes = idx.Tags[tag]
		c["Notes"] = notes
	}
	c["TagNoteCountLabel"] = noteCountLabel(len(notes))
	s.render(w, "tag", c)
}

func noteCountLabel(n int) string {
	if n == 1 {
		return "1 note"
	}
	return fmt.Sprintf("%d notes", n)
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
	if s.vault.HiddenRel(s.vault.Rel(p)) {
		http.NotFound(w, r)
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

type folderSort struct {
	Field string
	Dir   string
}

func normalizeFolderSort(raw string) folderSort {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "name_desc":
		return folderSort{Field: "name", Dir: "desc"}
	case "modified_asc", "mod_asc", "mtime_asc":
		return folderSort{Field: "modified", Dir: "asc"}
	case "modified_desc", "mod_desc", "mtime_desc":
		return folderSort{Field: "modified", Dir: "desc"}
	default:
		return folderSort{Field: "name", Dir: "asc"}
	}
}

func folderSortFromRequest(r *http.Request, cfg Config) folderSort {
	selected := normalizeFolderSort(cfg.FolderSort)
	field := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("sort")))
	dir := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("dir")))
	if field == "" && dir == "" {
		return selected
	}
	if field != "modified" {
		field = "name"
	}
	if dir != "desc" {
		dir = "asc"
	}
	return folderSort{Field: field, Dir: dir}
}

func folderSortLinks(baseURL string, selected folderSort) []map[string]any {
	options := []struct {
		Label string
		Field string
		Dir   string
	}{
		{Label: "Name ↑", Field: "name", Dir: "asc"},
		{Label: "Name ↓", Field: "name", Dir: "desc"},
		{Label: "Modified ↓", Field: "modified", Dir: "desc"},
		{Label: "Modified ↑", Field: "modified", Dir: "asc"},
	}
	links := make([]map[string]any, 0, len(options))
	for _, opt := range options {
		links = append(links, map[string]any{
			"Label":   opt.Label,
			"URL":     baseURL + "?sort=" + opt.Field + "&dir=" + opt.Dir,
			"Current": selected.Field == opt.Field && selected.Dir == opt.Dir,
		})
	}
	return links
}

func (s *Server) folder(w http.ResponseWriter, r *http.Request, p string) {
	cfg := s.vault.LoadConfig()
	selectedSort := folderSortFromRequest(r, cfg)
	ents, _ := os.ReadDir(p)
	var items []map[string]any
	for _, e := range ents {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		pp := filepath.Join(p, e.Name())
		rel := s.vault.Rel(pp)
		if s.vault.HiddenRel(rel) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		items = append(items, map[string]any{"Name": e.Name(), "Dir": e.IsDir(), "URL": s.vault.URLForRel(rel), "ModTime": info.ModTime()})
	}
	sort.SliceStable(items, func(i, j int) bool {
		if selectedSort.Field == "modified" {
			left := items[i]["ModTime"].(time.Time)
			right := items[j]["ModTime"].(time.Time)
			if !left.Equal(right) {
				if selectedSort.Dir == "desc" {
					return left.After(right)
				}
				return left.Before(right)
			}
		}
		left := strings.ToLower(items[i]["Name"].(string))
		right := strings.ToLower(items[j]["Name"].(string))
		if selectedSort.Dir == "desc" && selectedSort.Field == "name" {
			return left > right
		}
		return left < right
	})
	c := s.common(filepath.Base(p))
	relPath := s.vault.Rel(p)
	c["Path"] = relPath
	c["FolderName"] = filepath.Base(p)
	c["Breadcrumbs"] = breadcrumbsForRel(s.vault, relPath)
	c["Items"] = items
	c["SortLinks"] = folderSortLinks(s.vault.URLForRel(relPath), selectedSort)
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
