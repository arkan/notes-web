package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type editCreateRequest struct {
	Kind               string `json:"kind"`
	Directory          string `json:"directory"`
	Title              string `json:"title"`
	Path               string `json:"path"`
	Content            string `json:"content"`
	ConfirmMissingDirs bool   `json:"confirm_missing_dirs"`
	ConfirmHidden      bool   `json:"confirm_hidden"`
}

type editCreateResponse struct {
	Status       string   `json:"status"`
	Kind         string   `json:"kind"`
	Path         string   `json:"path"`
	TemplatePath string   `json:"template_path,omitempty"`
	Content      string   `json:"content,omitempty"`
	MissingDirs  []string `json:"missing_dirs,omitempty"`
	Requires     string   `json:"requires_confirmation,omitempty"`
}

// editCreate handles POST /_api/edit/create.
// It creates a new note (from title slugification or explicit path) or an
// empty folder, with template resolution, collision detection, hidden-path
// confirmation, and missing-parent-directory confirmation.
func (s *Server) editCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeEditError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.editAPICheck(w, r, true) {
		return
	}

	var req editCreateRequest
	if err := decodeEditJSONBody(r, &req); err != nil {
		writeEditError(w, err.Error(), http.StatusBadRequest)
		return
	}

	cfg := s.vault.LoadConfig()

	switch req.Kind {
	case "note":
		s.createNote(w, r, &req, cfg)
	case "folder":
		s.createFolder(w, r, &req, cfg)
	default:
		writeEditError(w, "kind must be 'note' or 'folder'", http.StatusBadRequest)
	}
}

func (s *Server) createNote(w http.ResponseWriter, r *http.Request, req *editCreateRequest, cfg Config) {
	// Determine target path.
	targetRel, err := s.resolveNoteTarget(req, cfg)
	if err != nil {
		writeEditError(w, err.Error(), http.StatusBadRequest)
		return
	}

	absPath, rel, err := s.vault.resolveEditPath(targetRel)
	if err != nil {
		writeEditError(w, "invalid path", http.StatusBadRequest)
		return
	}

	// Block dot-prefix.
	if s.vault.isDotBlocked(rel) {
		writeEditError(w, "path not allowed", http.StatusForbidden)
		return
	}

	// Block _trash.
	if s.vault.isTrashRel(rel, cfg.Editing.TrashPath) {
		writeEditError(w, "path not allowed", http.StatusForbidden)
		return
	}

	// Block _template.md creation.
	if s.vault.isTemplateRel(rel, cfg.Editing.TemplateName) {
		writeEditError(w, "template files cannot be created through this endpoint", http.StatusForbidden)
		return
	}

	// Must be .md (strict).
	if !isMarkdownEditable(rel) {
		writeEditError(w, "only .md files can be created", http.StatusBadRequest)
		return
	}

	// Collision check.
	if _, err := os.Stat(absPath); err == nil {
		writeEditError(w, "file already exists", http.StatusConflict)
		return
	}

	// Symlink ancestor check (before creating dirs).
	if err := checkSymlinkAncestor(s.vault.Root, absPath, false); err != nil {
		writeEditError(w, "path is not editable", http.StatusForbidden)
		return
	}

	// Missing parent directories check.
	parentRel := filepath.Dir(rel)
	parentAbs := filepath.Dir(absPath)
	missingDirs := s.missingParentDirs(parentAbs)
	if len(missingDirs) > 0 && !req.ConfirmMissingDirs {
		writeCreateConfirmation(w, "missing_dirs", s.relDirsForResponse(missingDirs))
		return
	}

	// Hidden path confirmation.
	if s.vault.isConfiguredHidden(rel, cfg.Hidden) && !req.ConfirmHidden {
		writeCreateHiddenConfirmation(w, rel)
		return
	}

	// Create missing parent directories.
	if len(missingDirs) > 0 {
		if err := os.MkdirAll(parentAbs, 0o755); err != nil {
			writeEditError(w, "cannot create directories", http.StatusInternalServerError)
			return
		}
	}

	// Resolve content.
	content := req.Content
	templatePath := ""
	if content == "" {
		_, tmplRel, tmplContent, tErr := s.vault.resolveNearestTemplate(targetRel, cfg)
		if tErr == nil && tmplContent != "" {
			content = applyTemplate(tmplContent, templateVars{
				Title:  req.Title,
				Slug:   editSlugify(req.Title),
				Path:   rel,
				Folder: parentRel,
				Date:   todayDate(),
			})
			templatePath = tmplRel
		}
	}

	// Atomic write.
	if err := atomicWriteFile(absPath, []byte(content)); err != nil {
		writeEditError(w, "cannot write file", http.StatusInternalServerError)
		return
	}

	s.vault.ClearIndexCache()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(editCreateResponse{
		Status:       "created",
		Kind:         "note",
		Path:         rel,
		TemplatePath: templatePath,
		Content:      content,
	})
}

func (s *Server) createFolder(w http.ResponseWriter, r *http.Request, req *editCreateRequest, cfg Config) {
	if req.Path == "" {
		writeEditError(w, "path is required for folder creation", http.StatusBadRequest)
		return
	}

	targetRel := strings.Trim(strings.TrimSpace(req.Path), "/")
	targetRel = strings.TrimSuffix(targetRel, "/")

	if targetRel == "" {
		writeEditError(w, "invalid path", http.StatusBadRequest)
		return
	}

	absPath, rel, err := s.vault.resolveEditPath(targetRel)
	if err != nil {
		writeEditError(w, "invalid path", http.StatusBadRequest)
		return
	}

	// Block dot-prefix.
	if s.vault.isDotBlocked(rel) {
		writeEditError(w, "path not allowed", http.StatusForbidden)
		return
	}

	// Block _trash.
	if s.vault.isTrashRel(rel, cfg.Editing.TrashPath) {
		writeEditError(w, "path not allowed", http.StatusForbidden)
		return
	}

	// Block _template.md as folder target.
	if s.vault.isTemplateRel(rel, cfg.Editing.TemplateName) {
		writeEditError(w, "path not allowed", http.StatusForbidden)
		return
	}

	// Collision check.
	if _, err := os.Stat(absPath); err == nil {
		writeEditError(w, "folder already exists", http.StatusConflict)
		return
	}

	// Symlink ancestor check.
	if err := checkSymlinkAncestor(s.vault.Root, absPath, false); err != nil {
		writeEditError(w, "path is not editable", http.StatusForbidden)
		return
	}

	// Missing parent directories check.
	missingDirs := s.missingParentDirs(filepath.Dir(absPath))
	if len(missingDirs) > 0 && !req.ConfirmMissingDirs {
		writeCreateConfirmation(w, "missing_dirs", s.relDirsForResponse(missingDirs))
		return
	}

	// Hidden path confirmation.
	if s.vault.isConfiguredHidden(rel, cfg.Hidden) && !req.ConfirmHidden {
		writeCreateHiddenConfirmation(w, rel)
		return
	}

	// Create missing parent directories.
	if len(missingDirs) > 0 {
		if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
			writeEditError(w, "cannot create directories", http.StatusInternalServerError)
			return
		}
	}

	// Create the folder.
	if err := os.Mkdir(absPath, 0o755); err != nil {
		writeEditError(w, "cannot create folder", http.StatusInternalServerError)
		return
	}

	s.vault.ClearIndexCache()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(editCreateResponse{
		Status: "created",
		Kind:   "folder",
		Path:   rel,
	})
}

// resolveNoteTarget determines the target relative path for a new note.
// If req.Path is set, it is used as the raw path (manual override).
// Otherwise, req.Title is slugified and placed inside req.Directory.
func (s *Server) resolveNoteTarget(req *editCreateRequest, cfg Config) (string, error) {
	if strings.TrimSpace(req.Path) != "" {
		// Manual path override: use exactly as provided.
		p := strings.TrimSpace(req.Path)
		if !isMarkdownEditable(p) {
			return "", fmt.Errorf("note path must end with .md")
		}
		return p, nil
	}

	if strings.TrimSpace(req.Title) == "" {
		return "", fmt.Errorf("title or path is required")
	}

	slug := editSlugify(req.Title)
	if slug == "" {
		return "", fmt.Errorf("title produces an empty filename")
	}

	dir := strings.Trim(strings.TrimSpace(req.Directory), "/")
	if dir != "" {
		return dir + "/" + slug + ".md", nil
	}
	return slug + ".md", nil
}

// missingParentDirs returns the list of missing ancestor directories for the
// given absolute path, from the outermost missing down to the innermost.
func (s *Server) missingParentDirs(absPath string) []string {
	return missingParentDirs(absPath)
}

func (s *Server) relDirsForResponse(absDirs []string) []string {
	relDirs := make([]string, 0, len(absDirs))
	for _, d := range absDirs {
		rel, err := filepath.Rel(s.vault.Root, d)
		if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
			relDirs = append(relDirs, d)
			continue
		}
		relDirs = append(relDirs, filepath.ToSlash(rel))
	}
	return relDirs
}

// writeCreateConfirmation writes a 409 JSON response asking the client to
// confirm missing directories before creating them.
func writeCreateConfirmation(w http.ResponseWriter, requires string, dirs []string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusConflict)
	json.NewEncoder(w).Encode(map[string]any{
		"error":                 "missing directories",
		"requires_confirmation": requires,
		"missing_dirs":          dirs,
	})
}

// writeCreateHiddenConfirmation writes a 409 JSON response asking the client
// to confirm the target is a configured hidden path before creating there.
func writeCreateHiddenConfirmation(w http.ResponseWriter, rel string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusConflict)
	json.NewEncoder(w).Encode(map[string]any{
		"error":                 "path is configured hidden",
		"message":               "Configured hidden paths are not listed but remain accessible by direct URL.",
		"requires_confirmation": "hidden",
		"path":                  rel,
	})
}
