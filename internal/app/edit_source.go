package app

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// editSource handles GET /_api/edit/source/<vault-rel-path>.
// Returns the file's content, sha256 hash, and mod_time as JSON.
func (s *Server) editSource(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		writeEditError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.editAPICheck(w, r, false) {
		return
	}

	// Extract path from URL: /_api/edit/source/<rel-path>.
	// Use EscapedPath so encoded characters such as %3F stay in the path
	// rather than being confused with query delimiters.
	escapedRelPath := strings.TrimPrefix(r.URL.EscapedPath(), "/_api/edit/source/")
	if escapedRelPath == "" || escapedRelPath == r.URL.EscapedPath() {
		writeEditError(w, "missing path", http.StatusBadRequest)
		return
	}
	relPath, err := url.PathUnescape(escapedRelPath)
	if err != nil {
		writeEditError(w, "invalid path", http.StatusBadRequest)
		return
	}

	// Resolve and validate.
	absPath, rel, err := s.vault.resolveEditPath(relPath)
	if err != nil {
		writeEditError(w, "invalid path", http.StatusBadRequest)
		return
	}

	// Path policy check.
	if !s.vault.DirectMutationAllowed(rel) {
		writeEditError(w, "path not allowed", http.StatusForbidden)
		return
	}

	// Must be a .md file for edit mode (strict: no .markdown).
	if !isMarkdownEditable(rel) {
		writeEditError(w, "only .md files are editable", http.StatusBadRequest)
		return
	}

	// Symlink check: block reading through symlinks before reading the file.
	if err := checkSymlinkAncestor(s.vault.Root, absPath, true); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writeEditError(w, "note not found", http.StatusNotFound)
			return
		}
		writeEditError(w, "path not allowed", http.StatusForbidden)
		return
	}

	// Must be an existing file.
	data, err := s.vault.ReadNote(absPath)
	if err != nil {
		writeEditError(w, "note not found", http.StatusNotFound)
		return
	}

	hash := contentHash([]byte(data.Text))

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(map[string]any{
		"path":     rel,
		"content":  data.Text,
		"hash":     hash,
		"mod_time": data.ModTime,
	})
}
