package app

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Calendar data helpers
// ---------------------------------------------------------------------------

// CalendarView is the view-model for the Calendar page.
type CalendarView struct {
	Calendar   MonthCalendar
	Selected   *CalendarDayDetail
	TodayDate  string
	TodayLabel string
}

// CalendarDayDetail is the selected day detail in the Calendar page.
type CalendarDayDetail struct {
	Date    string
	Label   string
	Note    *Note
	Doc     *RenderedDoc
	HasNote bool
}

// buildCalendarView builds the calendar page data for a given month and optional date.
// Invalid query params fall back safely to current month/today.
func (s *Server) buildCalendarView(monthStr, dateStr string) CalendarView {
	now := now().In(time.Local)

	selectedMonth := now
	selectedDay := now
	if monthStr != "" {
		if t, err := time.ParseInLocation("2006-01", monthStr, time.Local); err == nil {
			selectedMonth = t
			selectedDay = selectedMonth
		}
	}
	if dateStr != "" {
		if t, err := time.ParseInLocation("2006-01-02", dateStr, time.Local); err == nil {
			selectedDay = t
			selectedMonth = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.Local)
		}
	}

	cal := s.calendarMonthGrid(selectedMonth, selectedDay)
	selDate := selectedDay.Format("2006-01-02")

	// Build selected day detail.
	detail := &CalendarDayDetail{
		Date:  selDate,
		Label: selectedDay.Format("Jan 2, 2006"),
	}

	// Get the daily note for the selected date.
	note := s.calendarDailyNoteForDate(selDate)
	if note != nil {
		detail.HasNote = true
		detail.Note = note
		doc := s.renderer.Render(*note)
		detail.Doc = &doc
	}

	return CalendarView{
		Calendar:   cal,
		Selected:   detail,
		TodayDate:  now.Format("2006-01-02"),
		TodayLabel: "Today",
	}
}

// calendarMonthGrid builds a daily-notes-only MonthCalendar with URLs pointing to /_calendar.
func (s *Server) calendarMonthGrid(month, selectedDay time.Time) MonthCalendar {
	month = month.In(time.Local)
	selectedDay = selectedDay.In(time.Local)
	first := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, month.Location())
	start := first.AddDate(0, 0, -int((int(first.Weekday())+6)%7))

	// Determine which dates have daily notes, respecting path policy.
	hasDaily := map[string]bool{}
	cfg := s.vault.LoadConfig()
	glob := cfg.DailyNotesGlob
	if glob == "" {
		glob = "Daily Notes/*/*/*.md"
	}
	for _, p := range s.vault.MarkdownFiles() {
		rel := s.vault.Rel(p)
		if s.vault.isTemplateRel(rel, cfg.Editing.TemplateName) {
			continue
		}
		ok, _ := filepath.Match(filepath.ToSlash(glob), rel)
		if !ok {
			continue
		}
		if d, dateOk := dateFromRel(rel); dateOk {
			hasDaily[d.Format("2006-01-02")] = true
		}
	}

	cal := MonthCalendar{
		MonthLabel: first.Format("January 2006"),
		PrevURL:    fmt.Sprintf("/_calendar?month=%s", first.AddDate(0, -1, 0).Format("2006-01")),
		NextURL:    fmt.Sprintf("/_calendar?month=%s", first.AddDate(0, 1, 0).Format("2006-01")),
	}
	for week := 0; week < 6; week++ {
		var days []CalendarDay
		for dow := 0; dow < 7; dow++ {
			day := start.AddDate(0, 0, week*7+dow)
			date := day.Format("2006-01-02")
			url := "/_calendar?date=" + date
			if monthStr := month.Format("2006-01"); monthStr != day.Format("2006-01") {
				url = fmt.Sprintf("/_calendar?month=%s&date=%s", day.Format("2006-01"), date)
			}
			days = append(days, CalendarDay{
				Label:    fmt.Sprint(day.Day()),
				Date:     date,
				URL:      url,
				InMonth:  day.Month() == month.Month(),
				HasNote:  hasDaily[date],
				Selected: sameDate(day, selectedDay),
				Today:    sameDate(day, now()),
			})
		}
		cal.Weeks = append(cal.Weeks, days)
	}
	return cal
}

func (s *Server) calendarDailyNoteForDate(date string) *Note {
	cfg := s.vault.LoadConfig()
	glob := cfg.DailyNotesGlob
	if glob == "" {
		glob = "Daily Notes/*/*/*.md"
	}
	for _, p := range s.vault.MarkdownFiles() {
		rel := s.vault.Rel(p)
		if s.vault.isTemplateRel(rel, cfg.Editing.TemplateName) {
			continue
		}
		ok, _ := filepath.Match(filepath.ToSlash(glob), rel)
		if !ok || !strings.Contains(rel, date) {
			continue
		}
		d, found := dateFromRel(rel)
		if !found || d.Format("2006-01-02") != date {
			continue
		}
		n, err := s.vault.ReadNote(p)
		if err == nil {
			return &n
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Vault method: DailyNoteForDate — already exists in homepage.go.
// We use the existing daily_notes_glob-backed implementation for selected-day details.
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// API: GET /_calendar
// ---------------------------------------------------------------------------

func (s *Server) calendarPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.NotFound(w, r)
		return
	}

	monthStr := strings.TrimSpace(r.URL.Query().Get("month"))
	dateStr := strings.TrimSpace(r.URL.Query().Get("date"))

	data := s.buildCalendarView(monthStr, dateStr)

	c := setCurrentAppRoute(s.common("Calendar"), "calendar")
	c["CalendarView"] = data
	s.render(w, "calendar", c)
}
