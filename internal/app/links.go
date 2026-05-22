package app

import (
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

func (v *Vault) ForwardLinksFrom(note Note) []ForwardLink {
	var links []ForwardLink
	seen := map[string]bool{}
	for _, match := range wikiLinkRe.FindAllStringSubmatch(note.Body, -1) {
		raw := strings.TrimSpace(match[1])
		parts := strings.SplitN(raw, "|", 2)
		targetWithHeading := strings.TrimSpace(parts[0])
		target := strings.TrimSpace(strings.Split(targetWithHeading, "#")[0])
		if target == "" || seen[raw] {
			continue
		}
		seen[raw] = true
		display := target
		if len(parts) == 2 && strings.TrimSpace(parts[1]) != "" {
			display = strings.TrimSpace(parts[1])
		}
		res := v.ResolveWikiLink(raw)
		link := ForwardLink{Target: target, Display: display, Kind: res.Kind}
		if res.Kind == "unique" {
			link.URL = v.URLForRel(res.Matches[0].RelPath)
		}
		links = append(links, link)
	}
	return links
}

func (v *Vault) BacklinksWithContext(relPath string) []BacklinkContext {
	var out []BacklinkContext
	base := strings.TrimSuffix(pathBase(relPath), pathExt(relPath))
	noExt := strings.TrimSuffix(relPath, pathExt(relPath))
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

func pathBase(rel string) string {
	parts := strings.Split(rel, "/")
	return parts[len(parts)-1]
}

func pathExt(rel string) string {
	base := pathBase(rel)
	idx := strings.LastIndex(base, ".")
	if idx < 0 {
		return ""
	}
	return base[idx:]
}
