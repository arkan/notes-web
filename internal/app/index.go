package app

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

type NoteMeta struct {
	RelPath           string
	URL               string
	Title             string
	Tags              []string
	OutgoingWikiLinks []string
	ModTime           time.Time
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
		if tag == "" || seen[tag] {
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

func extractOutgoingWikiLinks(body string) []string {
	seen := map[string]bool{}
	var links []string
	re := regexp.MustCompile(`\[\[([^\]]+)\]\]`)
	for _, match := range re.FindAllStringSubmatch(body, -1) {
		target := strings.TrimSpace(strings.Split(strings.Split(match[1], "|")[0], "#")[0])
		if target == "" || seen[target] {
			continue
		}
		seen[target] = true
		links = append(links, target)
	}
	sort.Strings(links)
	return links
}
