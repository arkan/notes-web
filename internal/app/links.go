package app

import (
	"net/url"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type ForwardLink struct {
	Target  string
	Display string
	Kind    string
	URL     string
}

type BacklinkContext struct {
	Source  Note
	Context string
	LineNo  int
}

var wikiLinkRe = regexp.MustCompile(`\[\[([^\]]+)\]\]`)

type parsedWikiLink struct {
	Raw               string
	Target            string
	TargetWithHeading string
	Display           string
}

func parseWikiLink(raw string) (parsedWikiLink, bool) {
	raw = strings.TrimSpace(raw)
	parts := strings.SplitN(raw, "|", 2)
	targetWithHeading := strings.TrimSpace(parts[0])
	target := strings.TrimSpace(strings.Split(targetWithHeading, "#")[0])
	if target == "" {
		return parsedWikiLink{}, false
	}
	display := target
	if len(parts) == 2 && strings.TrimSpace(parts[1]) != "" {
		display = strings.TrimSpace(parts[1])
	}
	return parsedWikiLink{Raw: raw, Target: target, TargetWithHeading: targetWithHeading, Display: display}, true
}

func (v *Vault) preprocessWikiLinks(s string) string {
	return wikiLinkRe.ReplaceAllStringFunc(s, func(match string) string {
		inner := strings.TrimSuffix(strings.TrimPrefix(match, "[["), "]]")
		return v.wikiLinkMarkdown(inner, match)
	})
}

func (v *Vault) wikiLinkMarkdown(inner, fallback string) string {
	link, ok := parseWikiLink(inner)
	if !ok {
		return fallback
	}
	res := v.ResolveWikiLink(link.Raw)
	switch res.Kind {
	case "unique":
		u := v.URLForRel(res.Matches[0].RelPath)
		if res.Heading != "" {
			u += "#" + slugify(res.Heading)
		}
		return "[" + link.Display + "](" + u + ")"
	case "ambiguous":
		return "[" + link.Display + "](/_resolve?name=" + url.QueryEscape(res.Target) + ")"
	default:
		return "[" + link.Display + "](/_missing?name=" + url.QueryEscape(res.Target) + ")"
	}
}

func wikiLinksIn(line string) []parsedWikiLink {
	var links []parsedWikiLink
	for _, match := range wikiLinkRe.FindAllStringSubmatch(line, -1) {
		link, ok := parseWikiLink(match[1])
		if ok {
			links = append(links, link)
		}
	}
	return links
}

func (v *Vault) ForwardLinksFrom(note Note) []ForwardLink {
	var links []ForwardLink
	seen := map[string]bool{}
	for _, parsed := range wikiLinksIn(note.Body) {
		if seen[parsed.Raw] {
			continue
		}
		seen[parsed.Raw] = true
		res := v.ResolveWikiLink(parsed.Raw)
		link := ForwardLink{Target: parsed.Target, Display: parsed.Display, Kind: res.Kind}
		if res.Kind == "unique" {
			link.URL = v.URLForRel(res.Matches[0].RelPath)
		}
		links = append(links, link)
	}
	return links
}

func (v *Vault) BacklinksWithContext(relPath string) []BacklinkContext {
	var out []BacklinkContext
	base := strings.TrimSuffix(filepath.Base(relPath), filepath.Ext(relPath))
	noExt := strings.TrimSuffix(relPath, filepath.Ext(relPath))
	for _, p := range v.MarkdownFiles() {
		note, err := v.ReadNote(p)
		if err != nil || note.RelPath == relPath {
			continue
		}
		lines := strings.Split(note.Body, "\n")
		for i, line := range lines {
			if strings.Contains(line, "[["+base) || strings.Contains(line, "[["+noExt) || strings.Contains(line, relPath) {
				out = append(out, BacklinkContext{Source: note, Context: strings.TrimSpace(line), LineNo: i + 1})
				break
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].Source.ModTime.Equal(out[j].Source.ModTime) {
			return out[i].Source.ModTime.After(out[j].Source.ModTime)
		}
		return out[i].Source.RelPath < out[j].Source.RelPath
	})
	return out
}
