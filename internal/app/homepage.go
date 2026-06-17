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

	// Today daily — always uses today's actual date via now()
	todayStr := now().Format("2006-01-02")
	var todayNote *Note
	var todayDoc *RenderedDoc

	note := v.DailyForDate(todayStr)
	if note != nil {
		todayNote = note
		doc := s.renderer.Render(*note)
		todayDoc = &doc
	}

	// Homepage TODO buckets: overdue + today only
	var todoOverdue, todoToday []TaskItem
	board, err := v.BuildTaskBoard(todayStr)
	if err == nil {
		todoOverdue = board.Overdue
		todoToday = board.Today
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
	c := s.commonForActive(title, "")
	c["Dashboard"] = dashboard
	c["Err"] = err
	c["Latest"] = dashboard.LatestDaily
	c["Recent"] = s.vault.RecentNotes(10)
	c["HomepageView"] = s.buildHomepageView(cfg, dashboard)
	return c
}

// DailyForDate returns the daily note for a specific date string (YYYY-MM-DD)
// using the configured daily_glob and dateFromRel. Returns nil if not found.
func (v *Vault) DailyForDate(date string) *Note {
	cfg := v.LoadConfig()
	for _, p := range v.MarkdownFiles() {
		rel := v.Rel(p)
		ok, _ := filepath.Match(filepath.ToSlash(cfg.DailyGlob), rel)
		if !ok {
			continue
		}
		// Only match if the rel path contains the date and dateFromRel extracts it
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
