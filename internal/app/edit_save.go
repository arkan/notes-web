package app

import (
	"encoding/json"
	"net/http"
	"os"
)

type editSaveRequest struct {
	Path     string `json:"path"`
	Content  string `json:"content"`
	BaseHash string `json:"base_hash"`
}

// editSave handles PUT /_api/edit/save.
// It saves the provided content to the specified path, enforcing:
//   - editing enabled, CSRF, strict JSON
//   - path policy (dot/trash blocked, configured hidden/template allowed)
//   - .md extension only
//   - existing file only (no create in Phase 1 -- handler checks file exists)
//   - conflict detection via content hash
//   - symlink target/ancestor blocking
//   - atomic write with permission preservation
//   - index cache invalidation after successful save
func (s *Server) editSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PUT" {
		writeEditError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.editAPICheck(w, r, true) {
		return
	}

	var req editSaveRequest
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

	// Path policy (checked before content validation).
	if !s.vault.DirectMutationAllowed(rel) {
		writeEditError(w, "path not allowed", http.StatusForbidden)
		return
	}

	// Must be .md (strict).
	if !isMarkdownEditable(rel) {
		writeEditError(w, "only .md files are editable", http.StatusBadRequest)
		return
	}

	// Existing file only (no create in Phase 1).
	if _, err := statFile(absPath); err != nil {
		writeEditError(w, "note not found", http.StatusNotFound)
		return
	}

	// Symlink check: reject writes to symlink targets or through symlink ancestors.
	// Must come before content validation (security concern).
	if err := checkSymlinkAncestor(s.vault.Root, absPath, true); err != nil {
		writeEditError(w, "path is not editable", http.StatusForbidden)
		return
	}

	// base_hash is required for conflict detection.
	if req.BaseHash == "" {
		writeEditError(w, "base_hash is required", http.StatusBadRequest)
		return
	}

	// Conflict detection: compare base_hash with current disk content.
	currentHash, err := hashFile(absPath)
	if err != nil {
		writeEditError(w, "cannot read current file", http.StatusInternalServerError)
		return
	}
	if currentHash != req.BaseHash {
		writeEditError(w, "save conflict: file changed on disk", http.StatusConflict)
		return
	}

	// Atomic write with permission preservation.
	if err := atomicWriteFile(absPath, []byte(req.Content)); err != nil {
		writeEditError(w, "cannot write file", http.StatusInternalServerError)
		return
	}

	// Invalidate index cache after successful write.
	s.vault.ClearIndexCache()

	// Compute new hash.
	newHash, err := hashFile(absPath)
	if err != nil {
		newHash = ""
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(map[string]any{
		"status": "saved",
		"hash":   newHash,
	})
}

// statFile returns the FileInfo for the given path, or an error.
// Used as a lightweight existence check without reading content.
func statFile(path string) (interface{}, error) {
	return os.Stat(path)
}
