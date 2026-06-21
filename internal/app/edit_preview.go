package app

import (
	"encoding/json"
	"net/http"
	"os"
)

type editPreviewRequest struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// editPreview handles POST /_api/edit/preview.
// It renders unsaved content through the normal Markdown renderer and returns
// the resulting HTML. It is protected like a mutation because the rendered
// HTML is served under the Notes Web origin.
func (s *Server) editPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeEditError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.editAPICheck(w, r, true) {
		return
	}

	var req editPreviewRequest
	if err := decodeEditJSONBody(r, &req); err != nil {
		writeEditError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Path == "" {
		writeEditError(w, "path is required", http.StatusBadRequest)
		return
	}

	// Resolve and validate path.
	absPath, rel, err := s.vault.resolveEditPath(req.Path)
	if err != nil {
		writeEditError(w, "invalid path", http.StatusBadRequest)
		return
	}

	// Path policy: same as mutation (dot/trash blocked).
	if !s.vault.DirectMutationAllowed(rel) {
		writeEditError(w, "path not allowed", http.StatusForbidden)
		return
	}

	// Must be .md file for edit mode (strict).
	if !isMarkdownEditable(rel) {
		writeEditError(w, "only .md files are editable", http.StatusBadRequest)
		return
	}

	// Symlink check: block preview through symlink ancestors.
	// The target itself may not exist yet (preview before create), so we
	// use requireTarget=false to only check existing ancestor components.
	if err := checkSymlinkAncestor(s.vault.Root, absPath, false); err != nil {
		writeEditError(w, "path is not editable", http.StatusForbidden)
		return
	}

	// Build the note from unsaved content with the provided path as context.
	note := Note{
		Path:    absPath,
		RelPath: rel,
	}
	if st, err := os.Stat(absPath); err == nil {
		note.ModTime = st.ModTime()
	}

	// Parse frontmatter from the unsaved content.
	fm, body := parseFrontmatter(req.Content)
	note.Frontmatter = fm
	note.Body = body

	// Use index-backed renderer when possible, matching read-mode behavior.
	// Fall back to the base renderer if index build fails.
	renderer := s.renderer
	if idx, err := s.vault.BuildIndex(); err == nil && idx != nil {
		renderer = s.renderer.WithIndex(idx)
	}

	doc := renderer.Render(note)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(map[string]any{
		"html": doc.HTML,
	})
}
