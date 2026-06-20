package app

import (
	"fmt"
	"sort"
	"strings"
)

// dataviewTableParams holds optional user-provided filter/sort/q parameters.
type dataviewTableParams struct {
	Filters map[string][]string // field -> values (empty slice means no filter for that field)
	Q       string              // global text filter
	Sort    string              // user sort field
	Dir     string              // "asc" or "desc"
}

// dataviewFilterState holds the computed state of one filter for rendering.
type dataviewFilterState struct {
	Filter   dataviewFilter
	Options  []string // sorted unique values detected in column data
	Selected []string // currently selected values (empty means "All")
}

// evalDataviewTableRows evaluates the full pipeline for a TABLE query:
//
//	FROM/WHERE → FLATTEN → GROUP BY → SORT (query or user) → compute options → FILTER/q → LIMIT
func evalDataviewTableRows(v *Vault, idx *VaultIndex, q dataviewQuery, params dataviewTableParams) ([]dataviewRow, []dataviewFilterState, error) {
	rows, states, _, err := evalDataviewTableRowsInternal(v, idx, q, params, 0)
	return rows, states, err
}

func evalDataviewTableRowsForRender(v *Vault, idx *VaultIndex, q dataviewQuery, params dataviewTableParams) ([]dataviewRow, []dataviewFilterState, dataviewRenderCap, error) {
	renderLimit := 0
	if shouldApplyDataviewImplicitCap(q, params) {
		renderLimit = dataviewImplicitRenderLimit
	}
	rows, states, total, err := evalDataviewTableRowsInternal(v, idx, q, params, renderLimit)
	if err != nil {
		return nil, nil, dataviewRenderCap{}, err
	}
	cap := dataviewRenderCap{}
	if renderLimit > 0 && total > renderLimit {
		cap = dataviewRenderCap{Applied: true, Limit: renderLimit, Total: total}
	}
	return rows, states, cap, nil
}

func evalDataviewTableRowsInternal(v *Vault, idx *VaultIndex, q dataviewQuery, params dataviewTableParams, renderLimit int) ([]dataviewRow, []dataviewFilterState, int, error) {
	if q.Kind != "TABLE" {
		return nil, nil, 0, fmt.Errorf("FILTER is only supported for TABLE queries")
	}

	// 1-4: FROM/WHERE → FLATTEN → GROUP BY (skip SORT and LIMIT — applied below)
	rows, err := evalDataviewRowsFull(v, idx, q, true, params.Sort != "")
	if err != nil {
		return nil, nil, 0, err
	}
	totalBeforeRenderLimit := len(rows)

	// 5. Apply user sort if provided (replaces query SORT entirely).
	if params.Sort != "" {
		dir := strings.ToLower(params.Dir)
		desc := dir == "desc"
		rows = sortDataviewRowsForRenderLimit(rows, []dataviewSort{{Expr: params.Sort, Desc: desc}}, renderLimit)
	} else if len(q.Sorts) > 0 {
		// Apply query sort when no user sort.
		rows = sortDataviewRowsForRenderLimit(rows, q.Sorts, renderLimit)
	}

	// 6. Compute filter options from column values.
	cols := tableDisplayColumns(q)
	states := make([]dataviewFilterState, 0, len(q.Filters))
	for _, f := range q.Filters {
		// Determine which column this filter applies to.
		colIdx := -1
		for i, c := range cols {
			if c.Expr == f.Field {
				colIdx = i
				break
			}
		}
		if colIdx < 0 {
			// Should not happen if we validated earlier, but skip gracefully.
			continue
		}

		// Collect unique values from the column.
		valSet := map[string]bool{}
		for _, row := range rows {
			v := evalValue(row, f.Field)
			vals := toSlice(v)
			for _, item := range vals {
				s := displayPlain(item)
				if s == "" {
					continue
				}
				// Normalize: case-insensitive matching.
				key := normalizeFilterValue(s)
				if isTagField(f.Field) {
					key = addTagPrefix(key)
				}
				if !valSet[key] {
					valSet[key] = true
				}
			}
		}

		// Sort options alphabetically.
		options := make([]string, 0, len(valSet))
		for val := range valSet {
			options = append(options, val)
		}
		sort.Strings(options)

		// Determine selected values from params or defaults.
		selected := resolveFilterSelected(f, params, options)
		states = append(states, dataviewFilterState{
			Filter:   f,
			Options:  options,
			Selected: selected,
		})
	}

	// 7. Apply FILTER and q.
	rows = applyFilters(rows, q.Filters, states, params.Q, cols)

	// 8. Apply LIMIT (after filter/q, per clarified plan behavior).
	if q.Limit >= 0 && len(rows) > q.Limit {
		rows = rows[:q.Limit]
	} else if renderLimit > 0 && len(rows) > renderLimit {
		rows = rows[:renderLimit]
	}

	return rows, states, totalBeforeRenderLimit, nil
}

// tableDisplayColumns returns the columns to display for a TABLE query.
func tableDisplayColumns(q dataviewQuery) []dataviewColumn {
	if len(q.Columns) > 0 {
		return q.Columns
	}
	return []dataviewColumn{{Expr: "file.link", Label: "File"}}
}

// resolveFilterSelected determines which values should be selected for a filter.
// Tag fields require the `#` prefix: `project` does not match `#project`.
func resolveFilterSelected(f dataviewFilter, params dataviewTableParams, options []string) []string {
	// AJAX params take precedence over DEFAULT.
	paramVals := params.Filters[f.Field]
	if paramVals != nil {
		if len(paramVals) == 0 {
			return nil // "All"
		}
		var selected []string
		for _, pv := range paramVals {
			// Try to match against options for canonical form.
			found := false
			for _, opt := range options {
				if strings.EqualFold(normalizeFilterValue(pv), normalizeFilterValue(opt)) {
					selected = append(selected, opt)
					found = true
					break
				}
			}
			if !found {
				selected = append(selected, pv)
			}
		}
		if f.Mode == filterModeSingle && len(selected) > 1 {
			selected = selected[:1]
		}
		return selected
	}

	// No AJAX params: use DEFAULT. Default values for tag fields must also have `#`.
	if len(f.Defaults) > 0 {
		var selected []string
		for _, d := range f.Defaults {
			found := false
			for _, opt := range options {
				if strings.EqualFold(normalizeFilterValue(d), normalizeFilterValue(opt)) {
					selected = append(selected, opt)
					found = true
					break
				}
			}
			if !found {
				selected = append(selected, d)
			}
		}
		return selected
	}

	return nil // No default → All
}

// normalizeFilterValue normalizes a value for case-insensitive comparison.
func normalizeFilterValue(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

// addTagPrefix ensures a tag value has a leading #.
func addTagPrefix(v string) string {
	if strings.TrimSpace(v) == "" {
		return v
	}
	if !strings.HasPrefix(v, "#") {
		return "#" + v
	}
	return v
}

// stripTagPrefix removes leading # if present.
func stripTagPrefix(v string) string {
	return strings.TrimPrefix(v, "#")
}

// addTagPrefixToCell adds `#` prefix to each string element in a cell value for tag fields.
// Handles []string, []any, and plain string values. Avoids doubling existing `#`.
func addTagPrefixToCell(v any) any {
	switch x := v.(type) {
	case []string:
		out := make([]string, len(x))
		for i, s := range x {
			out[i] = addTagPrefix(s)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, item := range x {
			if s, ok := item.(string); ok {
				out[i] = addTagPrefix(s)
			} else {
				out[i] = item
			}
		}
		return out
	case string:
		return addTagPrefix(x)
	default:
		return v
	}
}

// applyFilters applies FILTER and q constraints to rows.
func applyFilters(rows []dataviewRow, filters []dataviewFilter, states []dataviewFilterState, q string, cols []dataviewColumn) []dataviewRow {
	var filtered []dataviewRow

	for _, row := range rows {
		// Apply each filter (AND semantics).
		include := true
		for i, f := range filters {
			if include && i < len(states) {
				include = include && filterMatchesRow(row, f, states[i])
			}
		}
		if !include {
			continue
		}

		// Apply global text q (case-insensitive contains over visible cell text).
		if q != "" {
			qLower := strings.ToLower(strings.TrimSpace(q))
			if qLower == "" {
				// OK, empty q passes all rows.
			} else {
				rowText := rowVisibleText(row, cols)
				if !strings.Contains(strings.ToLower(rowText), qLower) {
					continue
				}
			}
		}

		filtered = append(filtered, row)
	}

	return filtered
}

// filterMatchesRow checks if a row matches a single filter's selected state.
// For tag fields, comparison is strict with the `#` prefix (project does not match #project).
func filterMatchesRow(row dataviewRow, f dataviewFilter, state dataviewFilterState) bool {
	if len(state.Selected) == 0 {
		// "All" — no filtering.
		return true
	}

	rowVal := evalValue(row, f.Field)
	vals := toSlice(rowVal)

	for _, sel := range state.Selected {
		selNorm := normalizeFilterValue(sel)
		for _, v := range vals {
			vStr := displayPlain(v)
			// For tag fields, the cell value has no `#` prefix, but the selected value does.
			// Add `#` to the cell value for comparison when the field is a tag field.
			if isTagField(f.Field) {
				vStr = addTagPrefix(vStr)
			}
			vNorm := normalizeFilterValue(vStr)
			if vNorm == selNorm {
				return true
			}
		}
	}

	return false
}

// rowVisibleText builds the concatenated visible text of a row for global q filtering.
func rowVisibleText(row dataviewRow, cols []dataviewColumn) string {
	var parts []string
	for _, c := range cols {
		v := evalValue(row, c.Expr)
		parts = append(parts, displayPlain(v))
	}
	return strings.Join(parts, " ")
}

// validateFiltersForQuery validates all filters against the query.
func validateFiltersForQuery(q dataviewQuery) error {
	if len(q.Filters) == 0 {
		return nil
	}
	if q.Kind != "TABLE" {
		return fmt.Errorf("FILTER is only supported for TABLE queries, not %s", q.Kind)
	}
	cols := tableDisplayColumns(q)
	for _, f := range q.Filters {
		if err := validateFilterFieldInColumns(f, cols); err != nil {
			return err
		}
		if isTagField(f.Field) {
			for _, def := range f.Defaults {
				if strings.TrimSpace(def) != "" && !hasTagPrefix(def) {
					return fmt.Errorf("FILTER %q: tag defaults must include the # prefix", f.Field)
				}
			}
		}
	}
	return nil
}
