package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
)

const editJSONMaxBody = 1 << 20 // 1 MB max request body for edit API

// isJSONMediaType returns true if the Content-Type value represents a strict
// JSON media type (application/json with optional charset parameter).
// Rejects application/jsonx, text/json, form types, etc.
func isJSONMediaType(ct string) bool {
	if ct == "" {
		return false
	}
	mediaType, _, err := mime.ParseMediaType(ct)
	if err != nil {
		return false
	}
	return mediaType == "application/json"
}

// decodeEditJSONBody reads the request body as strict JSON with a max size
// limit and DisallowUnknownFields. It decodes into the provided target value.
//
// An empty body is accepted (returns nil without decoding). A body exceeding
// editJSONMaxBody is rejected. Malformed or unknown-field JSON is rejected.
// Use this helper for all Phase 1+ edit API endpoints.
func decodeEditJSONBody(r *http.Request, v any) error {
	if r.Body == nil {
		return nil
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, editJSONMaxBody+1))
	if err != nil {
		return fmt.Errorf("cannot read request body")
	}
	if len(body) == 0 {
		return nil
	}
	if int64(len(body)) > editJSONMaxBody {
		return fmt.Errorf("request body too large")
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("invalid JSON body: %v", err)
	}
	// Reject trailing data after the single JSON value.
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return fmt.Errorf("invalid JSON body: trailing data after value")
	}
	return nil
}

// editAPICheck performs common security checks for edit API endpoints.
// It verifies editing is enabled, CSRF token is valid, and for non-GET
// requests that Content-Type is application/json. Returns false and writes
// an error response if a check fails.
func (s *Server) editAPICheck(w http.ResponseWriter, r *http.Request, requireJSON bool) bool {
	if !s.vault.LoadConfig().Editing.Enabled {
		writeEditError(w, "editing not enabled", http.StatusForbidden)
		return false
	}
	if !verifyCSRFToken(s.csrfToken, r.Header.Get("X-CSRF-Token")) {
		writeEditError(w, "invalid or missing CSRF token", http.StatusForbidden)
		return false
	}
	if requireJSON && r.Method != "GET" {
		if !isJSONMediaType(r.Header.Get("Content-Type")) {
			writeEditError(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
			return false
		}
	}
	return true
}

// writeEditError writes a JSON error response for the edit API.
// The error message is escaped by encoding it as JSON.
func writeEditError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	// Use json.NewEncoder for proper escaping.
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// editAPI routes edit sub-API requests under /_api/edit/.
func (s *Server) editAPI(w http.ResponseWriter, r *http.Request) {
	sub := strings.TrimPrefix(r.URL.Path, "/_api/edit/")

	// source/<path> — must be checked before exact-name cases.
	if strings.HasPrefix(sub, "source/") {
		s.editSource(w, r)
		return
	}

	switch sub {
	case "ping":
		s.editPing(w, r)
	case "preview":
		s.editPreview(w, r)
	case "save":
		s.editSave(w, r)
	case "create":
		s.editCreate(w, r)
	case "rename":
		s.editRename(w, r)
	case "missing-link-create":
		s.editMissingLinkCreate(w, r)
	case "trash":
		s.editTrash(w, r)
	case "trash/restore":
		s.editTrashRestore(w, r)
	case "capture":
		s.editCapture(w, r)
	case "inbox/archive":
		s.editInboxArchive(w, r)
	case "inbox/move":
		s.editInboxMove(w, r)
	case "inbox/convert-task":
		s.editInboxConvertTask(w, r)
	default:
		http.NotFound(w, r)
	}
}

// editPing is a minimal endpoint to test the edit API gate chain.
// It requires POST, editing enabled, valid CSRF, JSON content-type,
// and optionally a valid JSON body (empty body is accepted).
func (s *Server) editPing(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeEditError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.editAPICheck(w, r, true) {
		return
	}
	// Use the reusable decode helper. Ping accepts any valid JSON object
	// (map target) or empty body.
	var pingReq map[string]any
	if err := decodeEditJSONBody(r, &pingReq); err != nil {
		writeEditError(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}
