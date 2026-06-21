package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type missingLinkRequest struct {
	Target             string            `json:"target"`
	SourcePath         string            `json:"source_path"`
	DryRunRaw          *bool             `json:"dry_run"`
	ConfirmMissingDirs bool              `json:"confirm_missing_dirs"`
	ConfirmHidden      bool              `json:"confirm_hidden"`
	ExpectedHashes     map[string]string `json:"expected_hashes"`
}

type missingLinkResponse struct {
	Status       string            `json:"status"`
	Path         string            `json:"path"`
	Target       string            `json:"target"`
	SourcePath   string            `json:"source_path"`
	Impact       *renameImpact     `json:"impact,omitempty"`
	Hashes       map[string]string `json:"expected_hashes,omitempty"`
	TemplatePath string            `json:"template_path,omitempty"`
	Content      string            `json:"content,omitempty"`
}

// editMissingLinkCreate handles POST /_api/edit/missing-link-create.
func (s *Server) editMissingLinkCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeEditError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.editAPICheck(w, r, true) {
		return
	}

	var req missingLinkRequest
	if err := decodeEditJSONBody(r, &req); err != nil {
		writeEditError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Target == "" {
		writeEditError(w, "target is required", http.StatusBadRequest)
		return
	}
	if req.SourcePath == "" {
		writeEditError(w, "source_path is required", http.StatusBadRequest)
		return
	}

	cfg := s.vault.LoadConfig()
	isDryRun := req.DryRunRaw == nil || *req.DryRunRaw

	// Validate source path.
	srcAbs, srcRel, err := s.vault.resolveEditPath(req.SourcePath)
	if err != nil {
		writeEditError(w, "invalid source path", http.StatusBadRequest)
		return
	}
	if s.vault.isDotBlocked(srcRel) || s.vault.isTrashRel(srcRel, cfg.Editing.TrashPath) {
		writeEditError(w, "source path not allowed", http.StatusForbidden)
		return
	}
	if s.vault.isTemplateRel(srcRel, cfg.Editing.TemplateName) {
		writeEditError(w, "template files are not valid source notes", http.StatusForbidden)
		return
	}
	if !isMarkdownEditable(srcRel) {
		writeEditError(w, "source must be a .md file", http.StatusBadRequest)
		return
	}
	if _, err := os.Stat(srcAbs); err != nil {
		writeEditError(w, "source note not found", http.StatusNotFound)
		return
	}
	if err := checkSymlinkAncestor(s.vault.Root, srcAbs, true); err != nil {
		writeEditError(w, "source path is not editable", http.StatusForbidden)
		return
	}

	// Determine target path for the new note.
	targetRel := resolveMissingLinkTarget(req.Target, srcRel)
	// Validate target path resolves within vault.
	targetAbs, _, err := s.vault.resolveEditPath(targetRel)
	if err != nil {
		writeEditError(w, "invalid target path", http.StatusBadRequest)
		return
	}

	// Block dot, trash, template.
	if s.vault.isDotBlocked(targetRel) || s.vault.isTrashRel(targetRel, cfg.Editing.TrashPath) {
		writeEditError(w, "target path not allowed", http.StatusForbidden)
		return
	}
	if s.vault.isTemplateRel(targetRel, cfg.Editing.TemplateName) {
		writeEditError(w, "template files cannot be created through missing-link", http.StatusForbidden)
		return
	}

	// Collision.
	if _, err := os.Stat(targetAbs); err == nil {
		writeEditError(w, "target already exists", http.StatusConflict)
		return
	}

	// Symlink ancestor check.
	if err := checkSymlinkAncestor(s.vault.Root, targetAbs, false); err != nil {
		writeEditError(w, "target path is not editable", http.StatusForbidden)
		return
	}

	// Hidden target (check before missing dirs: path policy before logistics).
	if s.vault.isConfiguredHidden(targetRel, cfg.Hidden) && !req.ConfirmHidden {
		writeCreateHiddenConfirmation(w, targetRel)
		return
	}

	// Missing parent dirs.
	parentAbs := filepath.Dir(targetAbs)
	missingDirs := missingParentDirs(parentAbs)
	if len(missingDirs) > 0 && !req.ConfirmMissingDirs {
		writeCreateConfirmation(w, "missing_dirs", s.relDirsForResponse(missingDirs))
		return
	}

	// Resolve content.
	title := displayTitleForTarget(req.Target)
	content, templatePath := resolveMissingContent(cfg, s.vault, targetRel, title)

	// Compute impact: scan all scannable .md files for exact wikilinks matching the target.
	impact, modified, err := computeMissingLinkImpact(s.vault, cfg, req.Target, targetRel)
	if err != nil {
		writeEditError(w, "cannot compute impact", http.StatusInternalServerError)
		return
	}

	// Compute expected hashes (include source and all impacted files).
	expectedHashes := computeExpectedHashes(s.vault, modified)
	// Add source hash.
	if srcData, rErr := os.ReadFile(srcAbs); rErr == nil {
		expectedHashes[srcRel] = contentHash(srcData)
	}

	if isDryRun {
		resp := missingLinkResponse{
			Status:       "preview",
			Path:         targetRel,
			Target:       req.Target,
			SourcePath:   srcRel,
			Impact:       impact,
			Hashes:       expectedHashes,
			TemplatePath: templatePath,
			Content:      content,
		}
		if resp.Hashes == nil {
			resp.Hashes = map[string]string{}
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Execute (use request's expected_hashes for conflict detection).
	execHashes := req.ExpectedHashes
	if execHashes == nil {
		execHashes = map[string]string{}
	}
	if err := executeMissingLinkCreate(s.vault, targetRel, targetAbs, srcRel, content, req.Target, &req, modified, execHashes); err != nil {
		writeEditError(w, "missing-link creation could not be completed; refresh and try again", http.StatusConflict)
		return
	}

	resp := missingLinkResponse{
		Status:     "created",
		Path:       targetRel,
		Target:     req.Target,
		SourcePath: srcRel,
		Impact:     impact,
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(resp)
}

// resolveMissingLinkTarget computes the relative path for a new note created
// from a missing wikilink. If the target contains "/", directory segments are
// preserved and only the final segment is slugified. Otherwise the note is
// created in the source note's directory with a slugified filename.
func resolveMissingLinkTarget(target, sourceRel string) string {
	target = strings.TrimSpace(target)
	// Strip any heading fragment (ignore after #).
	if idx := strings.Index(target, "#"); idx >= 0 {
		target = target[:idx]
	}
	// Strip any .md extension if present.
	target = strings.TrimSuffix(target, ".md")

	if strings.Contains(target, "/") {
		// Preserve directory segments, slugify only the final filename segment.
		// If the target already starts with the source dir prefix, use as-is;
		// otherwise prefix with source directory.
		sourceDir := filepath.Dir(sourceRel)
		dir := filepath.Dir(target)
		base := filepath.Base(target)
		slug := editSlugify(base)
		fullPath := dir + "/" + slug + ".md"
		// If the target's directory is a relative path (not starting with sourceDir),
		// prefix it with the source directory.
		if sourceDir != "." && !strings.HasPrefix(fullPath, sourceDir+"/") && !strings.HasPrefix(fullPath, "/") {
			fullPath = sourceDir + "/" + fullPath
		}
		return fullPath
	}

	// Bare target: create in source note's directory.
	sourceDir := filepath.Dir(sourceRel)
	slug := editSlugify(target)
	if sourceDir == "." {
		return slug + ".md"
	}
	return sourceDir + "/" + slug + ".md"
}

// displayTitleForTarget returns a human-readable title for the target,
// derived from the wikilink target text (removing path prefix and extension).
func displayTitleForTarget(target string) string {
	target = strings.TrimSpace(target)
	if idx := strings.Index(target, "#"); idx >= 0 {
		target = target[:idx]
	}
	target = strings.TrimSuffix(target, ".md")
	base := filepath.Base(target)
	return base
}

// resolveMissingContent resolves template content and returns it along with
// the template path. Returns empty content if no template is found.
func resolveMissingContent(cfg Config, v *Vault, targetRel, title string) (content, tmplPath string) {
	_, tmplRel, tmplContent, tErr := v.resolveNearestTemplate(targetRel, cfg)
	if tErr != nil || tmplContent == "" {
		return "", ""
	}
	slug := editSlugify(title)
	return applyTemplate(tmplContent, templateVars{
		Title:  title,
		Slug:   slug,
		Path:   targetRel,
		Folder: filepath.Dir(targetRel),
		Date:   todayDate(),
	}), tmplRel
}

// computeMissingLinkImpact scans all rewrite-scannable .md files for exact
// wikilinks matching the given target string (ignoring alias/heading).
func computeMissingLinkImpact(v *Vault, cfg Config, rawTarget, newRel string) (*renameImpact, map[string]string, error) {
	files := v.rewriteVisibleMD(cfg)
	impact := &renameImpact{}
	modified := map[string]string{}

	// Build oldRef list from the raw target (bare stem forms).
	stem := strings.TrimSuffix(rawTarget, filepath.Ext(rawTarget))
	var matchTargets []string
	matchTargets = append(matchTargets, rawTarget)
	matchTargets = append(matchTargets, stem)
	if filepath.Ext(rawTarget) == "" {
		matchTargets = append(matchTargets, rawTarget+".md")
	}

	for _, p := range files {
		rel := v.Rel(p)
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		content := string(data)

		count, newContent := rewriteExactWikilinksMatch(content, matchTargets, newRel)
		if count == 0 {
			continue
		}

		fi := fileImpact{Path: rel, Wikilinks: count}
		if v.isConfiguredHidden(rel, cfg.Hidden) {
			impact.Hidden = append(impact.Hidden, fi)
		} else {
			impact.Visible = append(impact.Visible, fi)
		}
		modified[rel] = newContent
	}

	sort.Slice(impact.Visible, func(i, j int) bool { return impact.Visible[i].Path < impact.Visible[j].Path })
	sort.Slice(impact.Hidden, func(i, j int) bool { return impact.Hidden[i].Path < impact.Hidden[j].Path })
	return impact, modified, nil
}

// rewriteExactWikilinksMatch finds exact wikilinks whose parsed target
// (ignoring alias/heading) matches any of matchTargets, and rewrites them to
// the new relative path. Preserves alias, heading, and style (bare vs path).
func rewriteExactWikilinksMatch(content string, matchTargets []string, newRel string) (int, string) {
	count := 0
	stem := strings.TrimSuffix(filepath.Base(newRel), filepath.Ext(newRel))
	noExt := strings.TrimSuffix(newRel, filepath.Ext(newRel))

	result := editWikiRe.ReplaceAllStringFunc(content, func(match string) string {
		inner := strings.TrimSuffix(strings.TrimPrefix(match, "[["), "]]")
		parsed, ok := parseEditWikiLink(inner)
		if !ok {
			return match
		}

		// Check if the parsed target matches any of the matchTargets.
		isMatch := false
		for _, mt := range matchTargets {
			if parsed.target == mt {
				isMatch = true
				break
			}
		}
		if !isMatch {
			return match
		}

		count++
		// Rewrite to the new note. Preserve style: this is always a new note
		// in the source folder or explicit path. Use the newRel as-is.
		// For bare stems, use the base stem; for path targets, use full path.
		newInner := newRel
		if !strings.Contains(parsed.target, "/") {
			// Originally a bare stem → use just the stem.
			newInner = stem
			if strings.HasSuffix(parsed.target, ".md") {
				newInner = stem + ".md"
			}
		} else if strings.HasSuffix(parsed.target, ".md") {
			newInner = newRel
		} else {
			newInner = noExt
		}

		if parsed.hasHeading {
			newInner += "#" + parsed.heading
		}
		if parsed.hasAlias {
			newInner += "|" + parsed.alias
		}
		return "[[" + newInner + "]]"
	})
	return count, result
}

// executeMissingLinkCreate creates the new note and rewrites all matching wikilinks.
func executeMissingLinkCreate(v *Vault, targetRel, targetAbs, srcRel, content, rawTarget string, req *missingLinkRequest, modified map[string]string, expectedHashes map[string]string) error {
	// Preflight hashes.
	if err := preflightHashes(v, srcRel, targetRel, expectedHashes, modified); err != nil {
		return err
	}

	// Create missing parent dirs if confirmed.
	parentAbs := filepath.Dir(targetAbs)
	createdDirs := missingParentDirs(parentAbs)
	if parentAbs != v.Root {
		if fi, pErr := os.Stat(parentAbs); pErr != nil || !fi.IsDir() {
			if req.ConfirmMissingDirs {
				if err := os.MkdirAll(parentAbs, 0o755); err != nil {
					return fmt.Errorf("cannot create directories: %w", err)
				}
			} else {
				return fmt.Errorf("target parent directory does not exist")
			}
		}
	}

	// Backup impacted files for rollback.
	type backup struct {
		absPath string
		data    []byte
		mode    os.FileMode
		existed bool
	}
	var backups []backup

	for rel := range modified {
		abs, _, _ := v.resolveEditPath(rel)
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
		// Also remove the newly created note.
		os.Remove(targetAbs)
		for i := len(createdDirs) - 1; i >= 0; i-- {
			_ = os.Remove(createdDirs[i])
		}
	}

	// Write impacted files.
	for rel, newContent := range modified {
		abs, _, _ := v.resolveEditPath(rel)
		if err := atomicWriteFile(abs, []byte(newContent)); err != nil {
			rollback()
			return fmt.Errorf("cannot rewrite %s: %w", rel, err)
		}
	}

	// Write the new note.
	if err := atomicWriteFile(targetAbs, []byte(content)); err != nil {
		rollback()
		return fmt.Errorf("cannot create note: %w", err)
	}

	v.ClearIndexCache()
	return nil
}

// missingParentDirs returns a list of missing ancestor directories for the
// given absolute path, outer-most first. Returns empty slice if all exist.
func missingParentDirs(absPath string) []string {
	var missing []string
	current := absPath
	for {
		if _, err := os.Stat(current); err == nil {
			break
		}
		missing = append(missing, current)
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	// Reverse to get outermost first.
	result := make([]string, len(missing))
	for i, m := range missing {
		result[len(missing)-1-i] = m
	}
	return result
}
