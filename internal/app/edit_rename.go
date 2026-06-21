package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type editRenameRequest struct {
	Path           string            `json:"path"`
	NewPath        string            `json:"new_path"`
	Title          string            `json:"title"`
	DryRunRaw      *bool             `json:"dry_run"`
	ConfirmHidden  bool              `json:"confirm_hidden"`
	ExpectedHashes map[string]string `json:"expected_hashes"`
}

type renameSuccessResponse struct {
	Status  string            `json:"status"`
	Kind    string            `json:"kind"`
	Path    string            `json:"path"`
	NewPath string            `json:"new_path"`
	Impact  *renameImpact     `json:"impact,omitempty"`
	Hashes  map[string]string `json:"expected_hashes,omitempty"`
}

// editRename handles POST /_api/edit/rename.
func (s *Server) editRename(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeEditError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.editAPICheck(w, r, true) {
		return
	}

	var req editRenameRequest
	if err := decodeEditJSONBody(r, &req); err != nil {
		writeEditError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Path == "" {
		writeEditError(w, "path is required", http.StatusBadRequest)
		return
	}

	cfg := s.vault.LoadConfig()
	isDryRun := req.DryRunRaw == nil || *req.DryRunRaw

	absPath, rel, err := s.vault.resolveEditPath(req.Path)
	if err != nil {
		writeEditError(w, "invalid path", http.StatusBadRequest)
		return
	}

	// Path policy: block dot and trash.
	if s.vault.isDotBlocked(rel) || s.vault.isTrashRel(rel, cfg.Editing.TrashPath) {
		writeEditError(w, "path not allowed", http.StatusForbidden)
		return
	}

	// Block template rename (always, regardless of hide_templates).
	if s.vault.isTemplateRel(rel, cfg.Editing.TemplateName) {
		writeEditError(w, "template files cannot be renamed", http.StatusForbidden)
		return
	}

	// Check source exists.
	st, err := os.Stat(absPath)
	if err != nil {
		writeEditError(w, "source not found", http.StatusNotFound)
		return
	}

	// Determine kind.
	isFolder := st.IsDir()
	kind := "note"
	if isFolder {
		kind = "folder"
	} else if !isMarkdownEditable(rel) {
		writeEditError(w, "only .md files can be renamed", http.StatusBadRequest)
		return
	}

	// Symlink check on source.
	if err := checkSymlinkAncestor(s.vault.Root, absPath, true); err != nil {
		writeEditError(w, "path is not editable", http.StatusForbidden)
		return
	}

	// Determine target path.
	newRel, err := s.resolveRenameTarget(rel, isFolder, &req, cfg)
	if err != nil {
		writeEditError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Target must differ from source.
	if newRel == rel {
		writeEditError(w, "new path must differ from current path", http.StatusBadRequest)
		return
	}

	// Block target dot, trash, template.
	if s.vault.isDotBlocked(newRel) || s.vault.isTrashRel(newRel, cfg.Editing.TrashPath) {
		writeEditError(w, "target path not allowed", http.StatusForbidden)
		return
	}
	if s.vault.isTemplateRel(newRel, cfg.Editing.TemplateName) {
		writeEditError(w, "target path not allowed", http.StatusForbidden)
		return
	}

	// Resolve target path and check existence.
	newAbs, _, err := s.vault.resolveEditPath(newRel)
	if err != nil {
		writeEditError(w, "invalid target path", http.StatusBadRequest)
		return
	}
	if _, err := os.Stat(newAbs); err == nil {
		writeEditError(w, "target already exists", http.StatusConflict)
		return
	}

	// Symlink check on target ancestors.
	if err := checkSymlinkAncestor(s.vault.Root, newAbs, false); err != nil {
		writeEditError(w, "path is not editable", http.StatusForbidden)
		return
	}

	// Target parent must exist (no silent directory creation on rename).
	targetParent := filepath.Dir(newAbs)
	if targetParent != s.vault.Root {
		if fi, pErr := os.Stat(targetParent); pErr != nil || !fi.IsDir() {
			writeEditError(w, "target parent directory does not exist", http.StatusConflict)
			return
		}
	}

	// Folder non-empty check.
	if isFolder {
		ents, err := os.ReadDir(absPath)
		if err != nil {
			writeEditError(w, "folder cannot be read", http.StatusConflict)
			return
		}
		if len(ents) > 0 {
			writeEditError(w, "cannot rename non-empty folder", http.StatusConflict)
			return
		}
	}

	// Hidden transition check.
	srcHidden := s.vault.isConfiguredHidden(rel, cfg.Hidden)
	dstHidden := s.vault.isConfiguredHidden(newRel, cfg.Hidden)
	if srcHidden != dstHidden && !req.ConfirmHidden {
		writeHiddenRenameConfirmation(w, rel, newRel)
		return
	}

	// Compute impact for notes (folder rename has no link rewrites).
	var impact *renameImpact
	var modifiedContent map[string]string
	var expectedHashes map[string]string
	allowBareWikiRefs := true
	if !isFolder {
		allowBareWikiRefs = s.vault.bareWikilinkReferenceIsUnique(rel, s.vault.rewriteVisibleMD(cfg))
		impact, modifiedContent, err = s.vault.computeRenameImpact(cfg, rel, newRel)
		if err != nil {
			writeEditError(w, "cannot compute rename impact", http.StatusInternalServerError)
			return
		}
		expectedHashes = computeExpectedHashes(s.vault, modifiedContent)
		// Include source note hash in expected hashes.
		srcAbs, _, _ := s.vault.resolveEditPath(rel)
		if srcData, rErr := os.ReadFile(srcAbs); rErr == nil {
			expectedHashes[rel] = contentHash(srcData)
		}
	}

	if isDryRun {
		resp := renameSuccessResponse{
			Status:  "preview",
			Kind:    kind,
			Path:    rel,
			NewPath: newRel,
			Impact:  impact,
			Hashes:  expectedHashes,
		}
		// For folder rename with no link rewrites, hashes is nil; return empty.
		if resp.Hashes == nil {
			resp.Hashes = map[string]string{}
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Execute.
	if err := s.executeRename(rel, newRel, isFolder, &req, modifiedContent, allowBareWikiRefs); err != nil {
		writeEditError(w, "rename could not be completed; refresh and try again", http.StatusConflict)
		return
	}

	resp := renameSuccessResponse{
		Status:  "renamed",
		Kind:    kind,
		Path:    rel,
		NewPath: newRel,
		Impact:  impact,
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(resp)
}

// resolveRenameTarget determines the target relative path for a rename.
func (s *Server) resolveRenameTarget(rel string, isFolder bool, req *editRenameRequest, cfg Config) (string, error) {
	if strings.TrimSpace(req.NewPath) != "" {
		p := strings.TrimSpace(req.NewPath)
		if isFolder {
			return strings.TrimSuffix(p, "/"), nil
		}
		if !isMarkdownEditable(p) {
			return "", fmt.Errorf("note path must end with .md")
		}
		return p, nil
	}

	if isFolder {
		return "", fmt.Errorf("new_path is required for folder rename")
	}

	if strings.TrimSpace(req.Title) == "" {
		return "", fmt.Errorf("new_path or title is required")
	}

	slug := editSlugify(req.Title)
	if slug == "" {
		return "", fmt.Errorf("title produces an empty filename")
	}
	dir := filepath.Dir(rel)
	if dir != "." && dir != "" {
		return dir + "/" + slug + ".md", nil
	}
	return slug + ".md", nil
}

// executeRename performs the actual rename: moves the source file/folder and
// rewrites all impacted files. Best-effort rollback on failure.
func (s *Server) executeRename(oldRel, newRel string, isFolder bool, req *editRenameRequest, modified map[string]string, allowBareWikiRefs bool) error {
	if modified == nil {
		modified = map[string]string{}
	}

	// Preflight: check expected hashes match current disk content.
	if err := preflightHashes(s.vault, oldRel, newRel, req.ExpectedHashes, modified); err != nil {
		return err
	}

	// Read source content (for note) to rewrite the source note itself.
	var sourceContent string
	if !isFolder {
		abs, _, _ := s.vault.resolveEditPath(oldRel)
		data, err := os.ReadFile(abs)
		if err != nil {
			return fmt.Errorf("cannot read source: %w", err)
		}
		sourceContent = string(data)
	}

	// Build backup map: store original bytes for rollback.
	type backup struct {
		absPath string
		data    []byte
		mode    os.FileMode
		existed bool // true if file existed before, false if newly created
	}
	var backups []backup

	// Backup existing files that will be modified.
	for rel := range modified {
		abs, _, _ := s.vault.resolveEditPath(rel)
		if data, err := os.ReadFile(abs); err == nil {
			fi, _ := os.Stat(abs)
			m := os.FileMode(0o644)
			if fi != nil {
				m = fi.Mode().Perm()
			}
			backups = append(backups, backup{absPath: abs, data: data, mode: m, existed: true})
		}
	}

	rollback := func() {
		for _, b := range backups {
			if b.existed {
				os.WriteFile(b.absPath, b.data, b.mode)
			} else {
				os.Remove(b.absPath)
			}
		}
	}

	// Write modified content to impacted files.
	for rel, newContent := range modified {
		abs, _, _ := s.vault.resolveEditPath(rel)
		if err := atomicWriteFile(abs, []byte(newContent)); err != nil {
			rollback()
			return fmt.Errorf("cannot write %s: %w", rel, err)
		}
	}

	// Add the source note to modified list (for the renamed target).
	if !isFolder {
		// Apply rewrites to the source content itself (the source note may
		// contain wikilinks to itself, which need updating).
		wkCount, wkContent := rewriteWikilinksInContentWithBare(sourceContent, oldRel, newRel, allowBareWikiRefs)
		mdCount, mdContent := rewriteMarkdownLinksInContent(wkContent, oldRel, oldRel, newRel)
		if wkCount == 0 && mdCount == 0 {
			mdContent = sourceContent // no change
		} else if wkCount > 0 && mdCount == 0 {
			mdContent = wkContent
		}
		// Use mdContent (which has both wikilink and md rewrites applied).

		// Write the new file at target path (newly created — rollback must remove).
		newAbs, _, _ := s.vault.resolveEditPath(newRel)
		// Preserve source file mode.
		srcMode := os.FileMode(0o644)
		srcAbs, _, _ := s.vault.resolveEditPath(oldRel)
		if srcFi, sErr := os.Stat(srcAbs); sErr == nil {
			srcMode = srcFi.Mode().Perm()
		}
		backups = append(backups, backup{absPath: newAbs, data: []byte(mdContent), mode: srcMode, existed: false})
		// Use atomicWriteFile but force the source mode.
		tmpData := []byte(mdContent)
		if wErr := writeFileWithMode(newAbs, tmpData, srcMode); wErr != nil {
			rollback()
			return fmt.Errorf("cannot write renamed note: %w", wErr)
		}
	}

	// Move source to target (for note: new file; for folder: rename directory).
	if isFolder {
		oldAbs, _, _ := s.vault.resolveEditPath(oldRel)
		newAbs, _, _ := s.vault.resolveEditPath(newRel)
		// Target parent must already exist (silent directory creation is rejected).
		parent := filepath.Dir(newAbs)
		if parent != s.vault.Root {
			if fi, pErr := os.Stat(parent); pErr != nil || !fi.IsDir() {
				rollback()
				return fmt.Errorf("target parent directory does not exist")
			}
		}
		if err := os.Rename(oldAbs, newAbs); err != nil {
			rollback()
			return fmt.Errorf("cannot rename folder: %w", err)
		}
	} else {
		oldAbs, _, _ := s.vault.resolveEditPath(oldRel)
		if err := os.Remove(oldAbs); err != nil {
			rollback()
			return fmt.Errorf("cannot remove old file: %w", err)
		}
	}

	s.vault.ClearIndexCache()
	return nil
}

// preflightHashes checks that expected hashes match current disk content.
// For note renames: expected hashes must be provided for the source and every
// impacted file. Missing expected hash blocks with error.
// For folder renames: no hash checking (folders have no content hash).
func preflightHashes(v *Vault, oldRel, newRel string, expected map[string]string, modified map[string]string) error {
	if expected == nil {
		expected = map[string]string{}
	}

	// Source must exist.
	oldAbs, _, err := v.resolveEditPath(oldRel)
	if err != nil {
		return fmt.Errorf("source not found")
	}
	oldFi, err := os.Stat(oldAbs)
	if err != nil {
		return fmt.Errorf("source not found")
	}

	// Target must not exist.
	newAbs, _, _ := v.resolveEditPath(newRel)
	if _, err := os.Stat(newAbs); err == nil {
		return fmt.Errorf("target already exists")
	}

	// Folders have no content hash — skip hash check for folders.
	if oldFi.IsDir() {
		return nil
	}

	// Collect all files that need hash checking: modified files + source.
	checkRels := []string{oldRel}
	for rel := range modified {
		checkRels = append(checkRels, rel)
	}

	for _, rel := range checkRels {
		abs, _, _ := v.resolveEditPath(rel)
		// Reject symlink files (no reads/rewrites through symlinks).
		if fi, lErr := os.Lstat(abs); lErr == nil && fi.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s is a symlink", rel)
		}
		if _, err := os.Stat(abs); err != nil {
			return fmt.Errorf("%s not found for hash check", rel)
		}
		data, err := os.ReadFile(abs)
		if err != nil {
			return fmt.Errorf("cannot read %s: %w", rel, err)
		}
		diskHash := contentHash(data)
		expectedHash, ok := expected[rel]
		if !ok || expectedHash == "" {
			return fmt.Errorf("expected hash missing for %s", rel)
		}
		if diskHash != expectedHash {
			return fmt.Errorf("hash mismatch for %s: disk changed since dry run", rel)
		}
	}

	return nil
}

func writeHiddenRenameConfirmation(w http.ResponseWriter, oldRel, newRel string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusConflict)
	json.NewEncoder(w).Encode(map[string]any{
		"error":                 "path transition requires confirmation",
		"requires_confirmation": "hidden",
		"path":                  oldRel,
		"new_path":              newRel,
		"message":               "Configured hidden paths remain accessible by direct URL but are excluded from normal listings, search, palette, and diagnostics.",
	})
}
