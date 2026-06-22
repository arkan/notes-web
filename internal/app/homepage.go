package app

import (
	"path/filepath"
	"strings"
)

// HomepageView is the typed view-model for the homepage template.
type HomepageView struct {
	Blocks      []HomepageBlock
	MainBlocks  []HomepageBlock
	SideBlocks  []HomepageBlock
	QuickJump   []HomepageLink
	Dashboard   Dashboard
	TodayNote   *Note
	TodayDoc    *RenderedDoc
	TodayDate   string // "2006-01-02" format
	TodoOverdue []TaskItem
	TodoToday   []TaskItem
}

// HomepageBlock describes one visible block on the homepage.
type HomepageBlock struct {
	ID     string // block ID constant
	Column string // "main" or "side"
	Order  int    // global homepage order index for mobile ordering
}

// HomepageLink is a single quick-jump or navigation link.
type HomepageLink struct {
	Label string
	Path  string // original path from config or default
	URL   string // resolved URL
}

// buildHomepageView constructs the typed view-model for the homepage.
func (s *Server) buildHomepageView(cfg Config, dashboard Dashboard) HomepageView {
	v := s.vault

	blocks := cfg.OrderedVisibleBlocks()
	for i := range blocks {
		blocks[i].Order = i
	}
	var mainBlocks, sideBlocks []HomepageBlock
	for _, block := range blocks {
		if block.Column == "main" {
			mainBlocks = append(mainBlocks, block)
		} else {
			sideBlocks = append(sideBlocks, block)
		}
	}
	quickJump := v.QuickJumpItems()

	// Today block — uses the selected dashboard date.
	// Falls back to now() when dashboard has no selected date (e.g. tests).
	todayStr := dashboard.SelectedDate
	if todayStr == "" {
		todayStr = now().Format("2006-01-02")
	}
	var todayNote *Note
	var todayDoc *RenderedDoc

	note := v.DailyNoteForDate(todayStr)
	if note != nil {
		todayNote = note
		doc := s.renderer.Render(*note)
		todayDoc = &doc
	}

	// Homepage TODO buckets: use pre-loaded dashboard tasks when available
	// (populated from index in BuildDashboardFor), falling back to a fresh
	// BuildTaskBoard call when dashboard.AllTasks is empty.
	nowStr := now().Format("2006-01-02")
	var todoOverdue, todoToday []TaskItem
	if len(dashboard.AllTasks) > 0 {
		board := buildBoardFromTasks(dashboard.AllTasks, nowStr)
		todoOverdue = board.Overdue
		todoToday = board.Today
	} else {
		board, err := v.BuildTaskBoard(nowStr)
		if err == nil {
			todoOverdue = board.Overdue
			todoToday = board.Today
		}
	}

	return HomepageView{
		Blocks:      blocks,
		MainBlocks:  mainBlocks,
		SideBlocks:  sideBlocks,
		QuickJump:   quickJump,
		Dashboard:   dashboard,
		TodayNote:   todayNote,
		TodayDoc:    todayDoc,
		TodayDate:   todayStr,
		TodoOverdue: todoOverdue,
		TodoToday:   todoToday,
	}
}

// templateDataForHome builds the full template data map for the home page,
// keeping both the legacy .Dashboard and the new .HomepageView.Dashboard consistent.
func (s *Server) templateDataForHome(title string, dashboard Dashboard, err error) map[string]any {
	cfg := s.vault.LoadConfig()
	c := setCurrentAppRoute(s.commonForActive(title, ""), "home")
	c["Dashboard"] = dashboard
	c["Err"] = err
	c["Latest"] = dashboard.LatestDaily
	c["HomepageView"] = s.buildHomepageView(cfg, dashboard)
	return c
}

// DailyForDate returns the daily note for a specific date string (YYYY-MM-DD)
// using the configured daily_glob and dateFromRel. Returns nil if not found.
func (v *Vault) DailyForDate(date string) *Note {
	cfg := v.LoadConfig()
	return v.dailyNoteByGlob(date, cfg.DailyGlob)
}

// DailyNoteForDate returns the daily note for a specific date string (YYYY-MM-DD)
// using the configured daily_notes_glob. Returns nil if not found.
func (v *Vault) DailyNoteForDate(date string) *Note {
	cfg := v.LoadConfig()
	return v.dailyNoteByGlob(date, cfg.DailyNotesGlob)
}

// dailyNoteByGlob scans MarkdownFiles matching glob and returns the note whose
// rel path dateFromRel matches the given date. Returns nil if not found.
func (v *Vault) dailyNoteByGlob(date, glob string) *Note {
	for _, p := range v.MarkdownFiles() {
		rel := v.Rel(p)
		ok, _ := filepath.Match(filepath.ToSlash(glob), rel)
		if !ok {
			continue
		}
		if !strings.Contains(rel, date) {
			continue
		}
		d, found := dateFromRel(rel)
		if found && d.Format("2006-01-02") == date {
			n, err := v.ReadNote(p)
			if err == nil {
				return &n
			}
		}
	}
	return nil
}
