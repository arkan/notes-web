package app

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var trashSnapshotPattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{6}-[0-9a-f]{6}$`)

// ---------- trash snapshot metadata ----------

type trashMetadata struct {
	OriginalPath string `json:"original_path"`
	TrashedAt    string `json:"trashed_at"`
	Kind         string `json:"kind"`
}

// ---------- move to trash ----------

type trashRequest struct {
	Path string `json:"path"`
}

type trashResponse struct {
	Status   string `json:"status"`
	Path     string `json:"path"`
	Snapshot string `json:"snapshot"`
}

// editTrash handles POST /_api/edit/trash (move to trash).
func (s *Server) editTrash(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeEditError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.editAPICheck(w, r, true) {
		return
	}

	var req trashRequest
	if err := decodeEditJSONBody(r, &req); err != nil {
		writeEditError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Path == "" {
		writeEditError(w, "path is required", http.StatusBadRequest)
		return
	}

	cfg := s.vault.LoadConfig()

	absPath, rel, err := s.vault.resolveEditPath(req.Path)
	if err != nil {
		writeEditError(w, "invalid path", http.StatusBadRequest)
		return
	}

	// Block dot, trash, template source.
	if s.vault.isDotBlocked(rel) || s.vault.isTrashRel(rel, cfg.Editing.TrashPath) {
		writeEditError(w, "path not allowed", http.StatusForbidden)
		return
	}
	if s.vault.isTemplateRel(rel, cfg.Editing.TemplateName) {
		writeEditError(w, "template files cannot be trashed", http.StatusForbidden)
		return
	}

	// Source must exist.
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
		writeEditError(w, "only .md files can be trashed", http.StatusBadRequest)
		return
	}

	// Symlink check.
	if err := checkSymlinkAncestor(s.vault.Root, absPath, true); err != nil {
		writeEditError(w, "path is not editable", http.StatusForbidden)
		return
	}

	// Non-empty folder check.
	if isFolder {
		ents, err := os.ReadDir(absPath)
		if err != nil {
			writeEditError(w, "cannot read folder contents", http.StatusInternalServerError)
			return
		}
		if len(ents) > 0 {
			writeEditError(w, "cannot trash non-empty folder", http.StatusConflict)
			return
		}
	}

	// Configured hidden source is allowed — no extra confirmation needed for trash.

	// Build snapshot path.
	trashRoot := filepath.Join(s.vault.Root, filepath.FromSlash(cfg.Editing.TrashPath))

	// Validate trash root and ancestors are not symlinks before creating snapshot.
	if err := checkSymlinkAncestor(s.vault.Root, trashRoot, false); err != nil {
		writeEditError(w, "trash path is not editable", http.StatusForbidden)
		return
	}

	snapshotDir, _, err := trashSnapshotName(trashRoot)
	if err != nil {
		writeEditError(w, "cannot create trash location", http.StatusInternalServerError)
		return
	}

	// Snapshot path: <trash_root>/<timestamp>-<random>/<original_rel_path>
	payloadRel := filepath.ToSlash(rel)
	payloadAbs := filepath.Join(snapshotDir, filepath.FromSlash(payloadRel))

	// Validate payload parent is not a symlink (snapshotDir was just created, but verify).
	if err := checkSymlinkAncestor(s.vault.Root, filepath.Dir(payloadAbs), false); err != nil {
		_ = os.RemoveAll(snapshotDir)
		writeEditError(w, "trash path is not editable", http.StatusForbidden)
		return
	}

	// Create parent dirs for the payload within snapshot.
	if err := os.MkdirAll(filepath.Dir(payloadAbs), 0o755); err != nil {
		_ = os.RemoveAll(snapshotDir)
		writeEditError(w, "cannot create trash directory", http.StatusInternalServerError)
		return
	}

	// Move the source file/folder.
	var renameErr error
	if isFolder {
		renameErr = os.Rename(absPath, payloadAbs)
	} else {
		renameErr = os.Rename(absPath, payloadAbs)
	}
	if renameErr != nil {
		writeEditError(w, "cannot move to trash", http.StatusInternalServerError)
		return
	}

	// Write metadata.
	meta := trashMetadata{
		OriginalPath: rel,
		TrashedAt:    time.Now().Format(time.RFC3339),
		Kind:         kind,
	}
	metaData, _ := json.Marshal(meta)
	metaPath := filepath.Join(snapshotDir, ".notes-web-trash.json")
	if err := os.WriteFile(metaPath, metaData, 0o644); err != nil {
		_ = os.Rename(payloadAbs, absPath)
		_ = os.RemoveAll(snapshotDir)
		writeEditError(w, "cannot write trash metadata", http.StatusInternalServerError)
		return
	}

	// Clear index.
	s.vault.ClearIndexCache()

	snapshotName := filepath.Base(snapshotDir)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(trashResponse{
		Status:   "trashed",
		Path:     rel,
		Snapshot: snapshotName,
	})
}

// ---------- trash listing ----------

type trashEntry struct {
	Snapshot     string `json:"snapshot"`
	OriginalPath string `json:"original_path"`
	TrashedAt    string `json:"trashed_at"`
	Kind         string `json:"kind"`
}

// trashList returns all trash snapshots with their metadata.
func (v *Vault) trashList() ([]trashEntry, error) {
	cfg := v.LoadConfig()
	trashRoot := filepath.Join(v.Root, filepath.FromSlash(cfg.Editing.TrashPath))
	if err := checkSymlinkAncestor(v.Root, trashRoot, false); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(trashRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return []trashEntry{}, nil
		}
		return nil, err
	}

	var result []trashEntry
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		snapshotAbs := filepath.Join(trashRoot, e.Name())
		if err := checkSymlinkAncestor(v.Root, snapshotAbs, true); err != nil {
			continue
		}
		metaPath := filepath.Join(trashRoot, e.Name(), ".notes-web-trash.json")
		if err := checkSymlinkAncestor(v.Root, metaPath, true); err != nil {
			continue
		}
		data, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}
		var meta trashMetadata
		if err := json.Unmarshal(data, &meta); err != nil {
			continue
		}
		result = append(result, trashEntry{
			Snapshot:     e.Name(),
			OriginalPath: meta.OriginalPath,
			TrashedAt:    meta.TrashedAt,
			Kind:         meta.Kind,
		})
	}
	return result, nil
}

// editTrashPage renders the dedicated trash listing page.
func (s *Server) editTrashPage(w http.ResponseWriter, r *http.Request) {
	// Editing must be enabled to see trash.
	if !s.vault.LoadConfig().Editing.Enabled {
		http.NotFound(w, r)
		return
	}
	entries, err := s.vault.trashList()
	c := setCurrentAppRoute(s.common("Trash"), "trash")
	if err != nil {
		w.WriteHeader(http.StatusForbidden)
		c["Err"] = "Unable to load trash."
	} else {
		c["Entries"] = entries
	}
	s.render(w, "trash", c)
}

// ---------- restore ----------

type restoreRequest struct {
	Snapshot    string `json:"snapshot"`
	RestorePath string `json:"restore_path"`
}

type restoreResponse struct {
	Status       string `json:"status"`
	OriginalPath string `json:"original_path,omitempty"`
	RestoredPath string `json:"restored_path"`
	Snapshot     string `json:"snapshot,omitempty"`
	Requires     string `json:"requires_confirmation,omitempty"`
	ExistsAt     string `json:"exists_at,omitempty"`
}

// editTrashRestore handles POST /_api/edit/trash/restore.
func (s *Server) editTrashRestore(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeEditError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.editAPICheck(w, r, true) {
		return
	}

	var req restoreRequest
	if err := decodeEditJSONBody(r, &req); err != nil {
		writeEditError(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Snapshot == "" {
		writeEditError(w, "snapshot is required", http.StatusBadRequest)
		return
	}
	// Snapshot must be one direct child created by trashSnapshotName.
	if !trashSnapshotPattern.MatchString(req.Snapshot) {
		writeEditError(w, "invalid snapshot name", http.StatusBadRequest)
		return
	}

	cfg := s.vault.LoadConfig()
	trashRoot := filepath.Join(s.vault.Root, filepath.FromSlash(cfg.Editing.TrashPath))
	if err := checkSymlinkAncestor(s.vault.Root, trashRoot, false); err != nil {
		writeEditError(w, "trash is not accessible", http.StatusForbidden)
		return
	}
	snapshotAbs := filepath.Join(trashRoot, req.Snapshot)
	if relSnapshot, relErr := filepath.Rel(trashRoot, snapshotAbs); relErr != nil || relSnapshot != req.Snapshot || strings.Contains(relSnapshot, string(filepath.Separator)) {
		writeEditError(w, "invalid snapshot name", http.StatusBadRequest)
		return
	}

	// Read metadata.
	if err := checkSymlinkAncestor(s.vault.Root, snapshotAbs, true); err != nil {
		writeEditError(w, "snapshot not found", http.StatusNotFound)
		return
	}
	metaPath := filepath.Join(snapshotAbs, ".notes-web-trash.json")
	if err := checkSymlinkAncestor(s.vault.Root, metaPath, true); err != nil {
		writeEditError(w, "snapshot not found", http.StatusNotFound)
		return
	}
	metaData, err := os.ReadFile(metaPath)
	if err != nil {
		writeEditError(w, "snapshot not found", http.StatusNotFound)
		return
	}
	var meta trashMetadata
	if err := json.Unmarshal(metaData, &meta); err != nil {
		writeEditError(w, "invalid snapshot metadata", http.StatusInternalServerError)
		return
	}
	if meta.Kind != "note" && meta.Kind != "folder" {
		writeEditError(w, "invalid snapshot metadata", http.StatusInternalServerError)
		return
	}

	// Determine restore target.
	restoreRel := meta.OriginalPath
	if req.RestorePath != "" {
		restoreRel = strings.TrimSpace(req.RestorePath)
	}
	restoreRel = filepath.ToSlash(strings.TrimPrefix(restoreRel, "/"))

	// Validate restore path is safe: no traversal, within vault.
	restoreAbs, _, err := s.vault.resolveEditPath(restoreRel)
	if err != nil {
		writeEditError(w, "invalid restore path", http.StatusBadRequest)
		return
	}

	// Block restore to dot, trash, template.
	if s.vault.isDotBlocked(restoreRel) || s.vault.isTrashRel(restoreRel, cfg.Editing.TrashPath) {
		writeEditError(w, "restore path not allowed", http.StatusForbidden)
		return
	}
	if s.vault.isTemplateRel(restoreRel, cfg.Editing.TemplateName) {
		writeEditError(w, "cannot restore to a template path", http.StatusForbidden)
		return
	}

	// Restore target must match the source kind:
	//   note → .md strict
	//   folder → not a .md file, not dot/trash/template (checked above)
	if meta.Kind == "note" && !isMarkdownEditable(restoreRel) {
		writeEditError(w, "restore target for a note must be a .md file", http.StatusBadRequest)
		return
	}
	if meta.Kind == "folder" && strings.HasSuffix(restoreRel, ".md") {
		writeEditError(w, "restore target for a folder must be a directory, not a .md file", http.StatusBadRequest)
		return
	}

	// Check destination collision.
	if _, err := os.Stat(restoreAbs); err == nil {
		// Collision: require restore_as.
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(restoreResponse{
			Status:       "conflict",
			Requires:     "restore_as",
			OriginalPath: meta.OriginalPath,
			ExistsAt:     restoreRel,
			Snapshot:     req.Snapshot,
		})
		return
	}

	// Symlink ancestor check on restore target.
	if err := checkSymlinkAncestor(s.vault.Root, restoreAbs, false); err != nil {
		writeEditError(w, "restore path is not editable", http.StatusForbidden)
		return
	}

	// Validate meta.OriginalPath is a clean vault-relative path (no traversal segments).
	if !isCleanVaultRelativePath(meta.OriginalPath) {
		writeEditError(w, "invalid snapshot metadata", http.StatusInternalServerError)
		return
	}

	// Snapshot payload path: snapshot dir + validated original relative path.
	payloadRel := filepath.ToSlash(meta.OriginalPath)
	payloadAbs := filepath.Join(snapshotAbs, filepath.FromSlash(payloadRel))

	// Verify payloadAbs stays under snapshotAbs (defense-in-depth against traversal).
	payloadRelCheck, err := filepath.Rel(snapshotAbs, payloadAbs)
	if err != nil || strings.HasPrefix(payloadRelCheck, "..") || filepath.IsAbs(payloadRelCheck) {
		writeEditError(w, "invalid snapshot metadata", http.StatusInternalServerError)
		return
	}

	// Ensure payload exists.
	if _, err := os.Stat(payloadAbs); err != nil {
		writeEditError(w, "snapshot payload not found", http.StatusNotFound)
		return
	}

	// Validate payload source and ancestors are not symlinks.
	if err := checkSymlinkAncestor(s.vault.Root, payloadAbs, true); err != nil {
		writeEditError(w, "snapshot payload is not editable", http.StatusForbidden)
		return
	}

	// Create parent directory for restore target.
	if err := os.MkdirAll(filepath.Dir(restoreAbs), 0o755); err != nil {
		writeEditError(w, "cannot create restore directory", http.StatusInternalServerError)
		return
	}

	// Move payload to restore target.
	if err := os.Rename(payloadAbs, restoreAbs); err != nil {
		// If rename across devices fails, copy+delete.
		writeEditError(w, "cannot restore file", http.StatusInternalServerError)
		return
	}

	// Clean up metadata and now-empty snapshot directory tree.
	_ = os.RemoveAll(snapshotAbs)

	s.vault.ClearIndexCache()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(restoreResponse{
		Status:       "restored",
		OriginalPath: meta.OriginalPath,
		RestoredPath: restoreRel,
	})
}

func isCleanVaultRelativePath(rel string) bool {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	if rel == "" || strings.HasPrefix(rel, "/") {
		return false
	}
	clean := path.Clean(rel)
	if clean == "." || clean != rel {
		return false
	}
	for _, segment := range strings.Split(rel, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
	}
	return true
}

// ---------- helpers ----------

// trashSnapshotName generates a unique trash snapshot directory name:
// <timestamp>-<short-random>. Creates the directory and returns its absolute
// path and the random suffix.
func trashSnapshotName(trashRoot string) (absPath, shortRand string, err error) {
	if err := os.MkdirAll(trashRoot, 0o755); err != nil {
		return "", "", err
	}
	ts := time.Now().UTC().Format("2006-01-02T150405")
	randBytes := make([]byte, 3) // 6 hex chars
	if _, rErr := rand.Read(randBytes); rErr != nil {
		return "", "", rErr
	}
	shortRand = hex.EncodeToString(randBytes)
	name := ts + "-" + shortRand
	abs := filepath.Join(trashRoot, name)
	if err := os.Mkdir(abs, 0o755); err != nil {
		return "", "", err
	}
	return abs, shortRand, nil
}
