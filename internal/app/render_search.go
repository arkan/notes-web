package app

import (
	"bytes"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	htmlrenderer "github.com/yuin/goldmark/renderer/html"
)

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
	return wikiLinkRe.ReplaceAllStringFunc(s, func(m string) string {
		inner := strings.TrimSuffix(strings.TrimPrefix(m, "[["), "]]")
		link, ok := parseWikiLink(inner)
		if !ok {
			return m
		}
		res := r.vault.ResolveWikiLink(link.Raw)
		switch res.Kind {
		case "unique":
			u := r.vault.URLForRel(res.Matches[0].RelPath)
			if res.Heading != "" {
				u += "#" + slugify(res.Heading)
			}
			return "[" + link.Display + "](" + u + ")"
		case "ambiguous":
			return "[" + link.Display + "](/_resolve?name=" + url.QueryEscape(res.Target) + ")"
		default:
			return "[" + link.Display + "](/_missing?name=" + url.QueryEscape(res.Target) + ")"
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
	b.WriteString(`<details class="frontmatter" data-panel-state="frontmatter" open><summary>Frontmatter</summary><dl>`)
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
