package app

import (
	"fmt"
	"html"
	"html/template"
	"net/http"
	"strconv"
)

const actionRenderDataviewTable = "renderDataviewTable"

// handleDataviewTableAction checks if the request is a Dataview table AJAX action
// and if so, handles it and returns true. Otherwise returns false.
//
// This is called from within Server.path, after path resolution and direct-read path policy checks.
func (s *Server) handleDataviewTableAction(w http.ResponseWriter, r *http.Request, absPath string) bool {
	action := r.URL.Query().Get("action")
	if action == "" {
		return false
	}
	if action != actionRenderDataviewTable {
		writeDataviewError(w, fmt.Sprintf("Unknown action %q", html.EscapeString(action)), 400)
		return true
	}

	// GET only.
	if r.Method != http.MethodGet {
		writeDataviewError(w, "Method not allowed (use GET)", http.StatusMethodNotAllowed)
		return true
	}

	// Must be a Markdown note.
	if !s.vault.IsMarkdown(absPath) {
		writeDataviewError(w, "Action requires a Markdown note", http.StatusBadRequest)
		return true
	}

	// Prevent path parameter (security: must use URL path only).
	// Check key presence even for empty value ("?path=").
	if _, ok := r.URL.Query()["path"]; ok {
		writeDataviewError(w, "Path parameter is not allowed for this action", http.StatusBadRequest)
		return true
	}

	// Parse table index.
	tableStr := r.URL.Query().Get("table")
	if tableStr == "" {
		writeDataviewError(w, "Missing table parameter", http.StatusBadRequest)
		return true
	}
	tableIdx, err := strconv.Atoi(tableStr)
	if err != nil || tableIdx < 1 {
		writeDataviewError(w, "Invalid table parameter", http.StatusBadRequest)
		return true
	}

	// Parse filter, sort, q params.
	params, err := parseFilterParamsFromHTTP(r.URL.Query())
	if err != nil {
		writeDataviewError(w, html.EscapeString(err.Error()), http.StatusBadRequest)
		return true
	}

	// Read the note.
	n, err := s.vault.ReadNote(absPath)
	if err != nil {
		writeDataviewError(w, "Note not found", http.StatusNotFound)
		return true
	}

	// Build index.
	idx, err := s.vault.BuildIndex()
	if err != nil {
		writeDataviewError(w, html.EscapeString(err.Error()), http.StatusInternalServerError)
		return true
	}

	// Scan blocks and find the Nth table.
	blocks := scanDataviewBlocks(n.Text)
	count := 0
	var targetBlock dataviewBlockSpan
	found := false
	for _, b := range blocks {
		if b.IsTable {
			count++
			if count == tableIdx {
				targetBlock = b
				found = true
				break
			}
		}
	}
	if !found {
		writeDataviewError(w, fmt.Sprintf("Table index %d out of range (found %d tables)", tableIdx, count), http.StatusBadRequest)
		return true
	}

	// Parse the query.
	q, err := parseDataviewQuery(targetBlock.QueryText)
	if err != nil {
		writeDataviewError(w, fmt.Sprintf("dataview-error: %v", err), http.StatusBadRequest)
		return true
	}

	// Run all validations against the parsed query.
	if err := validateActionRequest(q, &params); err != nil {
		writeDataviewError(w, html.EscapeString(err.Error()), http.StatusBadRequest)
		return true
	}

	// Per plan: during AJAX, omitted filter params mean "All" (no filter),
	// even when the query has a DEFAULT. Fill in empty slices for any declared
	// filters not present in the request.
	for _, f := range q.Filters {
		if _, ok := params.Filters[f.Field]; !ok {
			params.Filters[f.Field] = []string{}
		}
	}

	// Run the pipeline.
	rows, states, cap, err := evalDataviewTableRowsForRender(s.vault, idx, q, params)
	if err != nil {
		writeDataviewError(w, fmt.Sprintf("dataview-error: %v", err), http.StatusBadRequest)
		return true
	}

	// Validate filter values against detected options.
	if err := validateActionFilterValues(states, params); err != nil {
		writeDataviewError(w, html.EscapeString(err.Error()), http.StatusBadRequest)
		return true
	}
	// Determine sort metadata.
	sortField := params.Sort
	sortDir := params.Dir
	if sortField == "" && len(q.Sorts) > 0 {
		sortField = q.Sorts[0].Expr
		if q.Sorts[0].Desc {
			sortDir = "desc"
		} else {
			sortDir = "asc"
		}
	}

	// Render the partial HTML.
	out := renderDataviewTableHTML(q, rows, states, tableIdx, sortField, sortDir)
	if cap.Applied {
		out = template.HTML(renderDataviewCapNote(cap) + string(out))
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(out))
	return true
}

// validateActionRequest validates and normalizes filter and sort params against the parsed query.
func validateActionRequest(q dataviewQuery, params *dataviewTableParams) error {
	// 1. Validate filters against query (TABLE only, visible columns).
	if err := validateFiltersForQuery(q); err != nil {
		return fmt.Errorf("dataview-error: %v", err)
	}

	// 2. Validate declared fields.
	declared := map[string]dataviewFilter{}
	for _, f := range q.Filters {
		declared[f.Field] = f
	}
	for field, vals := range params.Filters {
		f, ok := declared[field]
		if !ok {
			return fmt.Errorf("undeclared filter field %q", field)
		}
		// Reject repeated values for single-mode filters.
		if f.Mode == filterModeSingle && len(vals) > 1 {
			return fmt.Errorf("filter %q accepts at most one value (single mode)", field)
		}
		if f.Mode == filterModeMulti && len(vals) > 1 {
			params.Filters[field] = dedupeFilterValues(vals)
			vals = params.Filters[field]
		}
		// Reject tag filter values missing the `#` prefix.
		if isTagField(f.Field) {
			for _, v := range vals {
				if v != "" && !hasTagPrefix(v) {
					return fmt.Errorf("filter value %q for tag field %q must include the # prefix", v, f.Field)
				}
			}
		}
	}

	// 3. Validate sort field: must be a visible simple-field column expression.
	if params.Sort != "" {
		if !isValidFilterField(params.Sort) {
			return fmt.Errorf("sort field %q is not a valid simple field", params.Sort)
		}
		// Must be a visible column with an exact match.
		cols := tableDisplayColumns(q)
		validSort := false
		for _, c := range cols {
			if c.Expr == params.Sort {
				validSort = true
				break
			}
		}
		if !validSort {
			return fmt.Errorf("sort field %q does not match any visible column", params.Sort)
		}
	}

	return nil
}

// hasTagPrefix returns true if the value starts with #.
func hasTagPrefix(v string) bool {
	return len(v) > 0 && v[0] == '#'
}

// writeDataviewError writes a Dataview-style HTML error fragment with the given status code.
func writeDataviewError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	w.Write([]byte(`<div class="dataview-error"><strong>Dataview non rendu</strong><p>` + html.EscapeString(message) + `</p></div>`))
}
