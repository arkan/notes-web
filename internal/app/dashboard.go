package app

import (
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type Dashboard struct {
	LatestDaily     *Note
	OpenTasks       []TaskItem
	BrokenLinkCount int
	OrphanNoteCount int
}

type TaskItem struct {
	Text      string
	ID        string
	Due       string
	Done      string
	SourceRel string
	SourceURL string
	LineNo    int
	Completed bool
	DateClass string
}

func (t TaskItem) DueClass(today string) string {
	if t.Due == "" {
		return ""
	}
	if t.Due < today {
		return "overdue-date"
	}
	if t.Due == today {
		return "today-date"
	}
	return "upcoming-date"
}

func (t TaskItem) StatusLabel() string {
	if t.Completed {
		return "Done"
	}
	return "Open"
}

func (v *Vault) BuildDashboard() (Dashboard, error) {
	idx, err := v.BuildIndex()
	if err != nil {
		return Dashboard{}, err
	}
	tasks, err := v.AllTasks()
	if err != nil {
		return Dashboard{}, err
	}
	resolver := NewIndexResolver(idx)
	d := Dashboard{LatestDaily: v.LatestDaily(), BrokenLinkCount: CountBrokenWikiLinks(idx, resolver), OrphanNoteCount: CountOrphanNotes(idx, resolver)}
	for _, task := range tasks {
		if !task.Completed {
			d.OpenTasks = append(d.OpenTasks, task)
		}
	}
	if len(d.OpenTasks) > 8 {
		d.OpenTasks = d.OpenTasks[:8]
	}
	return d, nil
}

type TaskBoard struct {
	Overdue  []TaskItem
	Today    []TaskItem
	Upcoming []TaskItem
	NoDate   []TaskItem
	Done     []TaskItem
}

func (v *Vault) BuildTaskBoard(today string) (TaskBoard, error) {
	tasks, err := v.AllTasks()
	if err != nil {
		return TaskBoard{}, err
	}
	board := TaskBoard{}
	for _, task := range tasks {
		task.DateClass = task.DueClass(today)
		switch {
		case task.Completed:
			board.Done = append(board.Done, task)
		case task.Due == "":
			board.NoDate = append(board.NoDate, task)
		case task.Due < today:
			board.Overdue = append(board.Overdue, task)
		case task.Due == today:
			board.Today = append(board.Today, task)
		default:
			board.Upcoming = append(board.Upcoming, task)
		}
	}
	return board, nil
}

func (v *Vault) AllTasks() ([]TaskItem, error) {
	var tasks []TaskItem
	for _, p := range v.MarkdownFiles() {
		note, err := v.ReadNote(p)
		if err != nil {
			return nil, err
		}
		if strings.ToLower(filepath.Base(note.RelPath)) != "todo.md" {
			continue
		}
		lines := strings.Split(note.Body, "\n")
		for i, line := range lines {
			if task, ok := parseTaskLine(line); ok {
				task.SourceRel = note.RelPath
				task.SourceURL = v.URLForRel(note.RelPath)
				task.LineNo = i + 1
				tasks = append(tasks, task)
			}
		}
	}
	sort.SliceStable(tasks, func(i, j int) bool {
		if tasks[i].Completed != tasks[j].Completed {
			return !tasks[i].Completed
		}
		if tasks[i].Due != tasks[j].Due {
			if tasks[i].Due == "" {
				return false
			}
			if tasks[j].Due == "" {
				return true
			}
			return tasks[i].Due < tasks[j].Due
		}
		if tasks[i].SourceRel != tasks[j].SourceRel {
			return tasks[i].SourceRel < tasks[j].SourceRel
		}
		return tasks[i].LineNo < tasks[j].LineNo
	})
	return tasks, nil
}

var (
	taskLineRe = regexp.MustCompile(`^\s*- \[([ xX])\]\s+(.+)$`)
	tidRe      = regexp.MustCompile(`<!--\s*tid:([A-Za-z0-9_-]+)\s*-->`)
	dueRe      = regexp.MustCompile(`📅\s*(\d{4}-\d{2}-\d{2})`)
	doneRe     = regexp.MustCompile(`✅\s*(\d{4}-\d{2}-\d{2})`)
)

func parseTaskLine(line string) (TaskItem, bool) {
	m := taskLineRe.FindStringSubmatch(line)
	if len(m) == 0 {
		return TaskItem{}, false
	}
	body := strings.TrimSpace(m[2])
	task := TaskItem{Completed: strings.EqualFold(m[1], "x"), Text: cleanTaskText(body)}
	if tid := tidRe.FindStringSubmatch(body); len(tid) > 1 {
		task.ID = tid[1]
	}
	if due := dueRe.FindStringSubmatch(body); len(due) > 1 {
		task.Due = due[1]
	}
	if done := doneRe.FindStringSubmatch(body); len(done) > 1 {
		task.Done = done[1]
	}
	return task, true
}

func cleanTaskText(s string) string {
	s = tidRe.ReplaceAllString(s, "")
	s = dueRe.ReplaceAllString(s, "")
	s = doneRe.ReplaceAllString(s, "")
	return strings.Join(strings.Fields(s), " ")
}

type IndexResolution struct {
	Kind    string
	RelPath string
}

type IndexResolver struct {
	byRel  map[string][]string
	byStem map[string][]string
}

func NewIndexResolver(idx *VaultIndex) *IndexResolver {
	resolver := &IndexResolver{byRel: map[string][]string{}, byStem: map[string][]string{}}
	if idx == nil {
		return resolver
	}
	for _, note := range idx.Notes {
		rel := note.RelPath
		resolver.byRel[rel] = append(resolver.byRel[rel], rel)
		noExt := strings.TrimSuffix(rel, filepath.Ext(rel))
		resolver.byRel[noExt] = append(resolver.byRel[noExt], rel)
		base := filepath.Base(rel)
		resolver.byStem[base] = append(resolver.byStem[base], rel)
		stem := strings.TrimSuffix(base, filepath.Ext(base))
		resolver.byStem[stem] = append(resolver.byStem[stem], rel)
	}
	return resolver
}

func (r *IndexResolver) Resolve(raw string) IndexResolution {
	if r == nil {
		return IndexResolution{Kind: "missing"}
	}
	target := strings.TrimSpace(strings.Split(strings.Split(raw, "|")[0], "#")[0])
	if target == "" {
		return IndexResolution{Kind: "missing"}
	}
	var matches []string
	if strings.Contains(target, "/") || strings.HasSuffix(strings.ToLower(target), ".md") {
		matches = append(matches, r.byRel[target]...)
		if filepath.Ext(target) == "" {
			matches = append(matches, r.byRel[target+".md"]...)
		}
	} else {
		matches = append(matches, r.byStem[target]...)
	}
	matches = uniqueStrings(matches)
	switch len(matches) {
	case 0:
		return IndexResolution{Kind: "missing"}
	case 1:
		return IndexResolution{Kind: "unique", RelPath: matches[0]}
	default:
		return IndexResolution{Kind: "ambiguous"}
	}
}

func uniqueStrings(items []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

type BrokenWikiLink struct {
	Source  NoteMeta
	Target  string
	Display string
	Context string
	LineNo  int
}

type BrokenLinkGroup struct {
	Target string
	Items  []BrokenWikiLink
	Total  int
	Open   bool
}

func GroupBrokenWikiLinks(links []BrokenWikiLink, limit int) []BrokenLinkGroup {
	byTarget := map[string][]BrokenWikiLink{}
	for _, link := range links {
		byTarget[link.Target] = append(byTarget[link.Target], link)
	}
	groups := make([]BrokenLinkGroup, 0, len(byTarget))
	for target, items := range byTarget {
		total := len(items)
		if len(items) > 20 {
			items = items[:20]
		}
		groups = append(groups, BrokenLinkGroup{Target: target, Items: items, Total: total})
	}
	sort.SliceStable(groups, func(i, j int) bool {
		if groups[i].Total != groups[j].Total {
			return groups[i].Total > groups[j].Total
		}
		return groups[i].Target < groups[j].Target
	})
	for i := range groups {
		groups[i].Open = i == 0
	}
	if limit > 0 && len(groups) > limit {
		return groups[:limit]
	}
	return groups
}

func BrokenDistinctTargetCount(links []BrokenWikiLink) int {
	seen := map[string]bool{}
	for _, link := range links {
		seen[link.Target] = true
	}
	return len(seen)
}

func BrokenAffectedNoteCount(links []BrokenWikiLink) int {
	seen := map[string]bool{}
	for _, link := range links {
		seen[link.Source.RelPath] = true
	}
	return len(seen)
}

func BrokenWikiLinks(idx *VaultIndex, resolver *IndexResolver) []BrokenWikiLink {
	if idx == nil {
		return nil
	}
	var broken []BrokenWikiLink
	for _, note := range idx.Notes {
		for _, link := range note.OutgoingLinks {
			if resolver.Resolve(link.Target).Kind == "missing" {
				broken = append(broken, BrokenWikiLink{Source: note, Target: link.Target, Display: link.Display, Context: link.Context, LineNo: link.LineNo})
			}
		}
	}
	sort.SliceStable(broken, func(i, j int) bool {
		if broken[i].Source.RelPath != broken[j].Source.RelPath {
			return broken[i].Source.RelPath < broken[j].Source.RelPath
		}
		if broken[i].LineNo != broken[j].LineNo {
			return broken[i].LineNo < broken[j].LineNo
		}
		return broken[i].Target < broken[j].Target
	})
	return broken
}

func OrphanNotes(idx *VaultIndex, resolver *IndexResolver) []NoteMeta {
	if idx == nil {
		return nil
	}
	incoming := map[string]int{}
	for _, note := range idx.Notes {
		for _, target := range note.OutgoingWikiLinks {
			res := resolver.Resolve(target)
			if res.Kind == "unique" {
				incoming[res.RelPath]++
			}
		}
	}
	var orphans []NoteMeta
	for _, note := range idx.Notes {
		if incoming[note.RelPath] == 0 {
			orphans = append(orphans, note)
		}
	}
	sort.SliceStable(orphans, func(i, j int) bool { return orphans[i].RelPath < orphans[j].RelPath })
	return orphans
}

func CountBrokenWikiLinks(idx *VaultIndex, resolver *IndexResolver) int {
	return len(BrokenWikiLinks(idx, resolver))
}

func CountOrphanNotes(idx *VaultIndex, resolver *IndexResolver) int {
	return len(OrphanNotes(idx, resolver))
}
