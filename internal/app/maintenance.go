package app

import "net/http"

// ---------------------------------------------------------------------------
// Maintenance data model
// ---------------------------------------------------------------------------

// MaintenanceView is the read-only view-model for the maintenance page.
type MaintenanceView struct {
	BrokenTotal           int    `json:"broken_total,omitempty"`
	BrokenDistinctTargets int    `json:"broken_distinct_targets,omitempty"`
	BrokenAffectedNotes   int    `json:"broken_affected_notes,omitempty"`
	OrphanTotal           int    `json:"orphan_total,omitempty"`
	DataviewTotal         int    `json:"dataview_total,omitempty"`
	DataviewUnsupported   int    `json:"dataview_unsupported,omitempty"`
	TrashCount            int    `json:"trash_count,omitempty"`
	TrashAvailable        bool   `json:"trash_available"`
	Err                   string `json:"err,omitempty"`
}

// buildMaintenanceData aggregates existing diagnostic data for the maintenance
// page. It reuses the same sources as the dedicated detail pages — no new
// filesystem scans or interpretations.
func (s *Server) buildMaintenanceData() MaintenanceView {
	idx, idxErr := s.vault.BuildIndex()
	m := MaintenanceView{}
	if idxErr != nil {
		m.Err = "Maintenance data unavailable."
		return m
	}
	if idx == nil {
		return m
	}

	resolver := NewIndexResolver(idx)

	// Broken links.
	links := BrokenWikiLinks(idx, resolver)
	m.BrokenTotal = len(links)
	m.BrokenDistinctTargets = BrokenDistinctTargetCount(links)
	m.BrokenAffectedNotes = BrokenAffectedNoteCount(links)

	// Orphans.
	orphans := OrphanNotes(idx, resolver)
	m.OrphanTotal = len(orphans)

	// Dataview diagnostics.
	items := ScanDataviewDiagnosticsFromIndex(idx)
	m.DataviewTotal = len(items)
	unsupported := 0
	for _, item := range items {
		if item.Status != "supported" {
			unsupported++
		}
	}
	m.DataviewUnsupported = unsupported

	// Trash health (only when editing enabled).
	cfg := s.vault.LoadConfig()
	if cfg.Editing.Enabled {
		entries, tErr := s.vault.trashList()
		if tErr == nil {
			m.TrashAvailable = true
			m.TrashCount = len(entries)
		}
	}

	return m
}

// ---------------------------------------------------------------------------
// API: GET /_maintenance
// ---------------------------------------------------------------------------

func (s *Server) maintenancePage(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.NotFound(w, r)
		return
	}

	data := s.buildMaintenanceData()
	c := setCurrentAppRoute(s.common("Maintenance"), "maintenance")
	c["Maintenance"] = data
	s.render(w, "maintenance", c)
}
