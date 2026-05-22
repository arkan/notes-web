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
	d := Dashboard{LatestDaily: v.LatestDaily(), BrokenLinkCount: v.CountBrokenWikiLinks(idx), OrphanNoteCount: v.CountOrphanNotes(idx)}
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

func (v *Vault) CountBrokenWikiLinks(idx *VaultIndex) int {
	if idx == nil {
		return 0
	}
	count := 0
	for _, note := range idx.Notes {
		for _, target := range note.OutgoingWikiLinks {
			if v.ResolveWikiLink(target).Kind == "missing" {
				count++
			}
		}
	}
	return count
}

func (v *Vault) CountOrphanNotes(idx *VaultIndex) int {
	if idx == nil {
		return 0
	}
	incoming := map[string]int{}
	for _, note := range idx.Notes {
		for _, target := range note.OutgoingWikiLinks {
			res := v.ResolveWikiLink(target)
			if res.Kind == "unique" {
				incoming[res.Matches[0].RelPath]++
			}
		}
	}
	count := 0
	for _, note := range idx.Notes {
		if incoming[note.RelPath] == 0 {
			count++
		}
	}
	return count
}
