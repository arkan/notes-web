package app

import (
	"net/http"
	"sort"
)

// ---------------------------------------------------------------------------
// Projects data model
// ---------------------------------------------------------------------------

// ProjectsView is the view-model for the Projects page.
type ProjectsView struct {
	Projects    []ActiveProject
	TotalCount  int
	ActiveCount int
	NoteTotal   int
}

// ActiveProjectsAll returns all active projects without a limit, unlike
// ActiveProjects which caps at the configured homepage limit.
// It explicitly excludes _template.md paths even when hide_templates is false.
func (v *Vault) ActiveProjectsAll(idx *VaultIndex) []ActiveProject {
	if idx == nil {
		return nil
	}

	cfg := v.LoadConfig()
	tmplName := cfg.Editing.TemplateName

	// Phase 1: determine which project rels have at least one active note.
	// Skip template paths explicitly.
	activeProjectRels := map[string]bool{}
	for _, note := range idx.Notes {
		if tmplName != "" && v.isTemplateRel(note.RelPath, tmplName) {
			continue
		}
		_, rel, ok := activeProjectForRel(note.RelPath)
		if !ok || !projectStatusActive(note) {
			continue
		}
		activeProjectRels[rel] = true
	}

	// Phase 2: collect all notes under active projects, excluding templates.
	projects := map[string]*ActiveProject{}
	for _, note := range idx.Notes {
		if tmplName != "" && v.isTemplateRel(note.RelPath, tmplName) {
			continue
		}
		label, rel, ok := activeProjectForRel(note.RelPath)
		if !ok || !activeProjectRels[rel] {
			continue
		}
		project := projects[rel]
		if project == nil {
			project = &ActiveProject{
				Label:       label,
				RelPath:     rel,
				URL:         v.URLForRel(rel),
				Description: projectDescription(label, rel),
			}
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
	return out
}

// ---------------------------------------------------------------------------
// Projects page handler
// ---------------------------------------------------------------------------

func (s *Server) projectsPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.NotFound(w, r)
		return
	}

	idx, err := s.vault.BuildIndex()
	view := ProjectsView{}

	if err == nil && idx != nil {
		projects := s.vault.ActiveProjectsAll(idx)
		view.Projects = projects
		view.TotalCount = len(projects)
		activeCount := 0
		for _, p := range projects {
			if p.NoteCount > 0 {
				activeCount++
			}
			view.NoteTotal += p.NoteCount
		}
		view.ActiveCount = activeCount
	}

	c := setCurrentAppRoute(s.common("Projects"), "projects")
	c["ProjectsView"] = view
	c["Err"] = err
	s.render(w, "projects", c)
}
