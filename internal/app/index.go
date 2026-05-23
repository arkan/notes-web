package app

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type NoteMeta struct {
	RelPath           string
	URL               string
	Title             string
	Tags              []string
	OutgoingWikiLinks []string
	OutgoingLinks     []WikiLinkOccurrence
	ModTime           time.Time
}

type WikiLinkOccurrence struct {
	Target  string
	Display string
	Context string
	LineNo  int
}

type VaultIndex struct {
	Notes []NoteMeta
	ByRel map[string]NoteMeta
	Tags  map[string][]NoteMeta
}

func (v *Vault) BuildIndex() (*VaultIndex, error) {
	idx := &VaultIndex{
		ByRel: map[string]NoteMeta{},
		Tags:  map[string][]NoteMeta{},
	}
	for _, p := range v.MarkdownFiles() {
		note, err := v.ReadNote(p)
		if err != nil {
			return nil, err
		}
		meta := NoteMeta{
			RelPath:           note.RelPath,
			URL:               v.URLForRel(note.RelPath),
			Title:             v.Title(note),
			Tags:              extractTags(note),
			OutgoingWikiLinks: extractOutgoingWikiLinks(note.Body),
			OutgoingLinks:     extractWikiLinkOccurrences(note.Body),
			ModTime:           note.ModTime,
		}
		idx.Notes = append(idx.Notes, meta)
		idx.ByRel[meta.RelPath] = meta
		for _, tag := range meta.Tags {
			idx.Tags[tag] = append(idx.Tags[tag], meta)
		}
	}
	sort.Slice(idx.Notes, func(i, j int) bool { return idx.Notes[i].RelPath < idx.Notes[j].RelPath })
	for tag := range idx.Tags {
		sort.Slice(idx.Tags[tag], func(i, j int) bool { return idx.Tags[tag][i].RelPath < idx.Tags[tag][j].RelPath })
	}
	return idx, nil
}

func extractTags(note Note) []string {
	seen := map[string]bool{}
	var tags []string
	add := func(raw string) {
		tag := normalizeTag(raw)
		if tag == "" || !isDisplayTag(tag) || seen[tag] {
			return
		}
		seen[tag] = true
		tags = append(tags, tag)
	}
	if raw, ok := note.Frontmatter["tags"]; ok {
		for _, tag := range splitTags(raw) {
			add(tag)
		}
	}
	hashRe := regexp.MustCompile(`(^|\s)#([\pL\pN][\pL\pN/_-]*)`)
	for _, match := range hashRe.FindAllStringSubmatch(note.Body, -1) {
		add(match[2])
	}
	sort.Strings(tags)
	return tags
}

func splitTags(raw any) []string {
	s := strings.TrimSpace(fmt.Sprint(raw))
	s = strings.Trim(s, "[]")
	if s == "" {
		return nil
	}
	fields := strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\n' || r == '\t'
	})
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.Trim(field, `"'`)
		if field != "" {
			out = append(out, field)
		}
	}
	return out
}

func normalizeTag(raw string) string {
	tag := strings.Trim(strings.TrimSpace(raw), "#[]\"'")
	tag = strings.TrimSuffix(tag, ",")
	return strings.ToLower(tag)
}

func isDisplayTag(tag string) bool {
	if tag == "" {
		return false
	}
	for _, r := range tag {
		if r < '0' || r > '9' {
			return true
		}
	}
	year, err := strconv.Atoi(tag)
	return err == nil && year >= 1900 && year <= 2100
}

func extractOutgoingWikiLinks(body string) []string {
	seen := map[string]bool{}
	var links []string
	for _, link := range wikiLinksIn(body) {
		if seen[link.Target] {
			continue
		}
		seen[link.Target] = true
		links = append(links, link.Target)
	}
	sort.Strings(links)
	return links
}

func extractWikiLinkOccurrences(body string) []WikiLinkOccurrence {
	var links []WikiLinkOccurrence
	lines := strings.Split(body, "\n")
	for i, line := range lines {
		for _, link := range wikiLinksIn(line) {
			links = append(links, WikiLinkOccurrence{Target: link.Target, Display: link.Display, Context: strings.TrimSpace(line), LineNo: i + 1})
		}
	}
	return links
}
