package app

import (
	"fmt"
	stdhtml "html"
	"html/template"
	"net/url"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

var now = time.Now

type Dashboard struct {
	LatestDaily     *Note
	TodayLabel      string
	SelectedDate    string
	ActiveProjects  []ActiveProject
	RecentNotes     []NoteMeta
	Calendar        MonthCalendar
	SelectedDay     SelectedDaySummary
	OpenTasks       []TaskItem
	BrokenLinkCount int
	OrphanNoteCount int
}

type ActiveProject struct {
	Label       string
	Description string
	RelPath     string
	URL         string
	LatestTitle string
	LatestURL   string
	LatestRel   string
	Updated     string
	NoteCount   int
	ModTime     time.Time
}

type MonthCalendar struct {
	MonthLabel string
	PrevURL    string
	NextURL    string
	Weeks      [][]CalendarDay
}

type CalendarDay struct {
	Label    string
	Date     string
	URL      string
	InMonth  bool
	HasNote  bool
	Selected bool
	Today    bool
}

func (d CalendarDay) Class() string {
	classes := []string{"calendar-day"}
	if !d.InMonth {
		classes = append(classes, "outside-month")
	}
	if d.HasNote {
		classes = append(classes, "has-note")
	}
	if d.Selected {
		classes = append(classes, "selected")
	}
	if d.Today {
		classes = append(classes, "today")
	}
	return strings.Join(classes, " ")
}

type SelectedDaySummary struct {
	Label string
	Date  string
	Notes []NoteMeta
}

type TaskItem struct {
	Text         string
	ID           string
	Due          string
	Done         string
	Added        string
	Repeat       string
	Priority     string
	SourceRel    string
	SourceURL    string
	Project      string
	ProjectURL   string
	LineNo       int
	PriorityRank int
	Completed    bool
	DateClass    string
	Tags         []string
}

func (t TaskItem) TextHTML() template.HTML {
	return template.HTML(linkifyTaskText(t.Text))
}

func linkifyTaskText(text string) string {
	var out strings.Builder
	last := 0
	for _, match := range taskMarkdownURLRe.FindAllStringSubmatchIndex(text, -1) {
		start, end := match[0], match[1]
		labelStart, labelEnd := match[2], match[3]
		urlStart, urlEnd := match[4], match[5]
		if start < last {
			continue
		}
		out.WriteString(linkifyRawTaskURLs(text[last:start]))
		out.WriteString(taskLinkHTML(text[urlStart:urlEnd], text[labelStart:labelEnd]))
		last = end
	}
	out.WriteString(linkifyRawTaskURLs(text[last:]))
	return out.String()
}

func linkifyRawTaskURLs(text string) string {
	var out strings.Builder
	last := 0
	for _, match := range taskURLRe.FindAllStringIndex(text, -1) {
		start, end := match[0], match[1]
		if start < last {
			continue
		}
		urlText := text[start:end]
		trimmed := strings.TrimRight(urlText, `.,;:!?)]}`+"\"")
		trailing := urlText[len(trimmed):]
		if trimmed == "" {
			continue
		}
		out.WriteString(stdhtml.EscapeString(text[last:start]))
		out.WriteString(taskLinkHTML(trimmed, trimmed))
		out.WriteString(stdhtml.EscapeString(trailing))
		last = end
	}
	out.WriteString(stdhtml.EscapeString(text[last:]))
	return out.String()
}

func taskLinkHTML(href, label string) string {
	escapedHref := stdhtml.EscapeString(href)
	escapedLabel := stdhtml.EscapeString(label)
	return `<a href="` + escapedHref + `" target="_hover" rel="noopener noreferrer">` + escapedLabel + `</a>`
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

func (t TaskItem) SourceLineURL() string {
	if t.SourceURL == "" || t.LineNo <= 0 {
		return t.SourceURL
	}
	return fmt.Sprintf("%s#line-%d", t.SourceURL, t.LineNo)
}

func (t TaskItem) PriorityLabel() string {
	if t.Priority == "" {
		return "—"
	}
	return t.Priority
}

func (t TaskItem) PriorityClass() string {
	if t.Priority == "" {
		return "none"
	}
	return strings.ToLower(t.Priority)
}

func (t TaskItem) CopyCommand() string {
	if t.ID == "" {
		return ""
	}
	return "td done " + t.ID
}

func (v *Vault) BuildDashboard() (Dashboard, error) {
	return v.BuildDashboardFor(time.Time{})
}

func (v *Vault) BuildDashboardFor(selectedOverride time.Time) (Dashboard, error) {
	idx, err := v.BuildIndex()
	if err != nil {
		return Dashboard{}, err
	}
	tasks, err := v.AllTasks()
	if err != nil {
		return Dashboard{}, err
	}
	resolver := NewIndexResolver(idx)
	latestDaily := v.LatestDaily()
	selected := selectedOverride
	if selected.IsZero() {
		selected = selectedDashboardDate(latestDaily)
	}
	d := Dashboard{
		LatestDaily:     latestDaily,
		SelectedDate:    selected.Format("2006-01-02"),
		TodayLabel:      selected.Format("Monday, January 2"),
		ActiveProjects:  v.ActiveProjects(idx, 6),
		RecentNotes:     recentNoteMetas(idx, 5),
		Calendar:        v.MonthCalendar(idx, selected),
		SelectedDay:     selectedDaySummary(idx, selected),
		BrokenLinkCount: CountBrokenWikiLinks(idx, resolver),
		OrphanNoteCount: CountOrphanNotes(idx, resolver),
	}
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

func selectedDashboardDate(latestDaily *Note) time.Time {
	if latestDaily != nil {
		if d, ok := dateFromRel(latestDaily.RelPath); ok {
			return d
		}
		if !latestDaily.ModTime.IsZero() {
			return latestDaily.ModTime
		}
	}
	return time.Now()
}

func recentNoteMetas(idx *VaultIndex, limit int) []NoteMeta {
	if idx == nil {
		return nil
	}
	notes := append([]NoteMeta(nil), idx.Notes...)
	sort.SliceStable(notes, func(i, j int) bool { return notes[i].ModTime.After(notes[j].ModTime) })
	if len(notes) > limit {
		notes = notes[:limit]
	}
	return notes
}

func (v *Vault) ActiveProjects(idx *VaultIndex, limit int) []ActiveProject {
	if idx == nil {
		return nil
	}
	projects := map[string]*ActiveProject{}
	for _, note := range idx.Notes {
		label, rel := projectForRel(note.RelPath)
		if label == "Daily Briefings" || label == "Daily Notes" || label == "Areas" {
			continue
		}
		project := projects[rel]
		if project == nil {
			project = &ActiveProject{Label: label, RelPath: rel, URL: v.URLForRel(rel), Description: projectDescription(label, rel)}
			projects[rel] = project
		}
		project.NoteCount++
		if note.ModTime.After(project.ModTime) || project.LatestRel == "" {
			project.ModTime = note.ModTime
			project.LatestTitle = note.Title
			project.LatestURL = note.URL
			project.LatestRel = note.RelPath
			project.Updated = humanDate(note.ModTime)
		}
	}
	out := make([]ActiveProject, 0, len(projects))
	for _, project := range projects {
		out = append(out, *project)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].ModTime.Equal(out[j].ModTime) {
			return out[i].ModTime.After(out[j].ModTime)
		}
		return out[i].Label < out[j].Label
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func projectForRel(rel string) (string, string) {
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) >= 3 && parts[0] == "Areas" {
		return parts[1], strings.Join(parts[:2], "/")
	}
	if len(parts) >= 2 {
		return parts[0], parts[0]
	}
	stem := strings.TrimSuffix(filepath.Base(rel), filepath.Ext(rel))
	return stem, rel
}

func projectDescription(label, rel string) string {
	if rel == "" || rel == label {
		return label
	}
	return rel
}

func humanDate(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	return t.Format("Jan 2")
}

func (v *Vault) MonthCalendar(idx *VaultIndex, selected time.Time) MonthCalendar {
	selected = selected.In(time.Local)
	first := time.Date(selected.Year(), selected.Month(), 1, 0, 0, 0, 0, selected.Location())
	start := first.AddDate(0, 0, -int((int(first.Weekday())+6)%7))
	hasDaily := map[string]bool{}
	if idx != nil {
		cfg := v.LoadConfig()
		for _, note := range idx.Notes {
			ok, _ := filepath.Match(filepath.ToSlash(cfg.DailyGlob), note.RelPath)
			if !ok {
				continue
			}
			if d, ok := dateFromRel(note.RelPath); ok {
				hasDaily[d.Format("2006-01-02")] = true
			}
		}
	}
	cal := MonthCalendar{
		MonthLabel: first.Format("January 2006"),
		PrevURL:    "/?month=" + first.AddDate(0, -1, 0).Format("2006-01"),
		NextURL:    "/?month=" + first.AddDate(0, 1, 0).Format("2006-01"),
	}
	for week := 0; week < 6; week++ {
		var days []CalendarDay
		for dow := 0; dow < 7; dow++ {
			day := start.AddDate(0, 0, week*7+dow)
			date := day.Format("2006-01-02")
			days = append(days, CalendarDay{
				Label:    fmt.Sprint(day.Day()),
				Date:     date,
				URL:      "/?date=" + date,
				InMonth:  day.Month() == selected.Month(),
				HasNote:  hasDaily[date],
				Selected: sameDate(day, selected),
				Today:    sameDate(day, now()),
			})
		}
		cal.Weeks = append(cal.Weeks, days)
	}
	return cal
}

func selectedDaySummary(idx *VaultIndex, selected time.Time) SelectedDaySummary {
	date := selected.Format("2006-01-02")
	summary := SelectedDaySummary{Date: date, Label: selected.Format("Jan 2")}
	if idx == nil {
		return summary
	}
	for _, note := range idx.Notes {
		if strings.Contains(note.RelPath, date) {
			summary.Notes = append(summary.Notes, note)
		}
	}
	sort.SliceStable(summary.Notes, func(i, j int) bool { return summary.Notes[i].Title < summary.Notes[j].Title })
	return summary
}

var relDateRe = regexp.MustCompile(`(\d{4})-(\d{2})-(\d{2})`)

func dateFromRel(rel string) (time.Time, bool) {
	m := relDateRe.FindStringSubmatch(rel)
	if len(m) == 0 {
		return time.Time{}, false
	}
	d, err := time.Parse("2006-01-02", m[0])
	return d, err == nil
}

func sameDate(a, b time.Time) bool {
	y1, m1, d1 := a.Date()
	y2, m2, d2 := b.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

type TaskBoard struct {
	Overdue  []TaskItem
	Today    []TaskItem
	Upcoming []TaskItem
	NoDate   []TaskItem
	Done     []TaskItem
}

func (b TaskBoard) OpenCount() int {
	return len(b.Overdue) + len(b.Today) + len(b.Upcoming) + len(b.NoDate)
}

func (b TaskBoard) Summary() string {
	return fmt.Sprintf("%d open · %d overdue · %d today · %d upcoming · %d no date hidden", b.OpenCount(), len(b.Overdue), len(b.Today), len(b.Upcoming), len(b.NoDate))
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
	sort.SliceStable(board.Done, func(i, j int) bool {
		if board.Done[i].Done != board.Done[j].Done {
			if board.Done[i].Done == "" {
				return false
			}
			if board.Done[j].Done == "" {
				return true
			}
			return board.Done[i].Done > board.Done[j].Done
		}
		if board.Done[i].SourceRel != board.Done[j].SourceRel {
			return board.Done[i].SourceRel < board.Done[j].SourceRel
		}
		return board.Done[i].LineNo < board.Done[j].LineNo
	})
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
		project, projectRel := projectForRel(note.RelPath)
		lines := strings.Split(note.Body, "\n")
		for i, line := range lines {
			if task, ok := parseTaskLine(line); ok {
				task.SourceRel = note.RelPath
				task.SourceURL = v.URLForRel(note.RelPath)
				task.Project, task.ProjectURL = taskProject(task, project, v.URLForRel(projectRel))
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
		if tasks[i].PriorityRank != tasks[j].PriorityRank {
			return normalizePriorityRank(tasks[i].PriorityRank) < normalizePriorityRank(tasks[j].PriorityRank)
		}
		if tasks[i].SourceRel != tasks[j].SourceRel {
			return tasks[i].SourceRel < tasks[j].SourceRel
		}
		return tasks[i].LineNo < tasks[j].LineNo
	})
	return tasks, nil
}

var (
	taskLineRe        = regexp.MustCompile(`^\s*- \[([ xX])\]\s+(.+)$`)
	tidRe             = regexp.MustCompile(`<!--\s*tid:([A-Za-z0-9_-]+)\s*-->`)
	dueRe             = regexp.MustCompile(`📅\s*(\d{4}-\d{2}-\d{2})`)
	doneRe            = regexp.MustCompile(`✅\s*(\d{4}-\d{2}-\d{2})`)
	addedRe           = regexp.MustCompile(`➕\s*(\d{4}-\d{2}-\d{2})`)
	recurRe           = regexp.MustCompile(`🔁\s*([^📅✅➕⏫🔼🔽⏬]+)`)
	priorityRe        = regexp.MustCompile(`[⏫🔼🔽⏬]`)
	tagRe             = regexp.MustCompile(`(^|\s)#([[:alnum:]_/-]+)`)
	taskURLRe         = regexp.MustCompile(`https?://[^\s<]+`)
	taskMarkdownURLRe = regexp.MustCompile(`\[([^\]]+)\]\((https?://[^\s<)]+)\)`)
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
	if added := addedRe.FindStringSubmatch(body); len(added) > 1 {
		task.Added = added[1]
	}
	metadataBody := tidRe.ReplaceAllString(body, "")
	if repeat := recurRe.FindStringSubmatch(metadataBody); len(repeat) > 1 {
		task.Repeat = strings.TrimSpace(repeat[1])
	}
	task.Priority, task.PriorityRank = parseTaskPriority(body)
	task.Tags = extractTaskTags(body)
	return task, true
}

func cleanTaskText(s string) string {
	s = tidRe.ReplaceAllString(s, "")
	s = dueRe.ReplaceAllString(s, "")
	s = doneRe.ReplaceAllString(s, "")
	s = addedRe.ReplaceAllString(s, "")
	s = recurRe.ReplaceAllString(s, "")
	s = priorityRe.ReplaceAllString(s, "")
	s = tagRe.ReplaceAllString(s, " ")
	return strings.Join(strings.Fields(s), " ")
}

func parseTaskPriority(s string) (string, int) {
	switch {
	case strings.Contains(s, "⏫"):
		return "P1", 1
	case strings.Contains(s, "🔼"):
		return "P2", 2
	case strings.Contains(s, "🔽"):
		return "P3", 3
	case strings.Contains(s, "⏬"):
		return "P4", 4
	default:
		return "", 0
	}
}

func normalizePriorityRank(rank int) int {
	if rank == 0 {
		return 99
	}
	return rank
}

func extractTaskTags(s string) []string {
	matches := tagRe.FindAllStringSubmatch(s, -1)
	seen := map[string]bool{}
	tags := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 3 || match[2] == "" || seen[match[2]] {
			continue
		}
		seen[match[2]] = true
		tags = append(tags, match[2])
	}
	return tags
}

func taskProject(task TaskItem, fallbackLabel, fallbackURL string) (string, string) {
	for _, tag := range task.Tags {
		if strings.HasPrefix(tag, "project/") {
			label := strings.TrimPrefix(tag, "project/")
			if label != "" {
				return strings.TrimSpace(strings.ReplaceAll(label, "-", " ")), "/_tags/" + url.PathEscape(tag)
			}
		}
	}
	for _, tag := range task.Tags {
		switch tag {
		case "admin", "nid", "fp", "santé", "sante":
			return strings.Title(strings.ReplaceAll(tag, "-", " ")), "/_tags/" + url.PathEscape(tag)
		}
	}
	if fallbackLabel == "Areas" || fallbackLabel == "TODO" {
		return "Inbox", fallbackURL
	}
	return fallbackLabel, fallbackURL
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
