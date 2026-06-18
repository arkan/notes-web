package app

import (
	"fmt"
	"html"
	"html/template"
	"sort"
	"strconv"
	"strings"
)

// renderDataviewTableHTML renders the full TABLE HTML including controls, data attributes,
// filter dropdowns, sort metadata, and table body.
func renderDataviewTableHTML(q dataviewQuery, rows []dataviewRow, states []dataviewFilterState, tableIndex int, sortField, sortDir string) template.HTML {
	var b strings.Builder

	cols := tableDisplayColumns(q)

	// Open wrapper with data attributes.
	b.WriteString(`<div class="dataview dataview-table-wrap"`)
	b.WriteString(` data-dataview-action="renderDataviewTable"`)
	b.WriteString(` data-dataview-table="`)
	b.WriteString(strconv.Itoa(tableIndex))
	b.WriteString(`"`)
	b.WriteString(`>`)

	// Controls area: text filter, dropdown filters, rows select.
	b.WriteString(`<div class="dataview-table-scroll"><div class="dataview-controls">`)

	// Text filter.
	b.WriteString(`<label class="dataview-filter-label">Filter <input class="dataview-filter" type="search" data-dataview-filter placeholder="Filter table…"`)
	if q := ""; q != "" { // placeholder for future
		b.WriteString(` value="`)
		b.WriteString(html.EscapeString(q))
		b.WriteString(`"`)
	}
	b.WriteString(`></label>`)

	// Dropdown filters.
	for i, st := range states {
		f := st.Filter
		colLabel := columnLabelForField(cols, f.Field)

		if f.Mode == filterModeSingle {
			renderSingleFilter(&b, colLabel, f, st, i, tableIndex)
		} else {
			renderMultiFilter(&b, colLabel, f, st, i, tableIndex)
		}
	}

	// Rows page size select.
	b.WriteString(`<label class="dataview-page-size-label">Rows <select data-dataview-page-size><option value="0">All</option><option value="10">10</option><option value="25">25</option><option value="50">50</option></select></label>`)

	b.WriteString(`</div>`)

	// Table.
	b.WriteString(`<table class="dataview-table"><thead><tr>`)
	for _, c := range cols {
		ariaSort := `none`
		if strings.EqualFold(c.Expr, sortField) {
			if strings.EqualFold(sortDir, "desc") {
				ariaSort = `descending`
			} else {
				ariaSort = `ascending`
			}
		}
		b.WriteString(`<th scope="col"`)
		if isValidFilterField(c.Expr) {
			b.WriteString(` data-dataview-sort`)
			b.WriteString(` data-dataview-sort-field="`)
			b.WriteString(html.EscapeString(c.Expr))
			b.WriteString(`"`)
		}
		b.WriteString(` aria-sort="`)
		b.WriteString(ariaSort)
		b.WriteString(`">`)
		b.WriteString(html.EscapeString(c.Label))
		b.WriteString(`</th>`)
	}
	b.WriteString(`</tr></thead><tbody>`)

	if len(rows) == 0 {
		// Show header row with "No matching rows" message.
		b.WriteString(`<tr><td colspan="`)
		b.WriteString(strconv.Itoa(len(cols)))
		b.WriteString(`" class="dataview-no-rows">No matching rows</td></tr>`)
	} else {
		for _, r := range rows {
			if key, ok := r.Data["key"]; ok {
				b.WriteString(`<tr class="dataview-group"><th scope="row" colspan="`)
				b.WriteString(strconv.Itoa(len(cols)))
				b.WriteString(`">`)
				b.WriteString(html.EscapeString(displayPlain(key)))
				b.WriteString(`</th></tr>`)
			}
			b.WriteString(`<tr>`)
			for _, c := range cols {
				cellVal := evalValue(r, c.Expr)
				if isTagField(c.Expr) {
					cellVal = addTagPrefixToCell(cellVal)
				}
				b.WriteString(renderDataviewCell(cellVal))
			}
			b.WriteString(`</tr>`)
		}
	}

	b.WriteString(`</tbody></table><div class="dataview-pager" data-dataview-pager aria-live="polite"></div></div></div>`)

	return template.HTML(b.String())
}

// columnLabelForField finds the column label for a given field expression.
func columnLabelForField(cols []dataviewColumn, field string) string {
	for _, c := range cols {
		if c.Expr == field {
			return c.Label
		}
	}
	return field
}

// renderSingleFilter renders a native <select> for a single-mode filter.
func renderSingleFilter(b *strings.Builder, colLabel string, f dataviewFilter, st dataviewFilterState, idx, tableIndex int) {
	b.WriteString(`<label class="dataview-filter-label">`)
	b.WriteString(html.EscapeString(colLabel))
	b.WriteString(` <select data-dataview-filter="`)
	b.WriteString(html.EscapeString(f.Field))
	b.WriteString(`">`)

	if f.Clearable {
		// "All" option at top.
		b.WriteString(`<option value=""`)
		if len(st.Selected) == 0 {
			b.WriteString(` selected`)
		}
		b.WriteString(`>All</option>`)
	} else if len(f.Defaults) == 0 {
		// No default, not clearable: disabled placeholder.
		b.WriteString(`<option value="" disabled`)
		if len(st.Selected) == 0 {
			b.WriteString(` selected`)
		}
		b.WriteString(`>`)
		b.WriteString(html.EscapeString(colLabel))
		b.WriteString(`</option>`)
	}

	// Build set of all options: merge detected options + synthetic selected defaults.
	allOptions := buildFilterOptions(st)
	for _, opt := range allOptions {
		isSelected := false
		for _, sel := range st.Selected {
			if opt == sel {
				isSelected = true
				break
			}
		}
		b.WriteString(`<option value="`)
		b.WriteString(html.EscapeString(opt))
		b.WriteString(`"`)
		if isSelected {
			b.WriteString(` selected`)
		}
		b.WriteString(`>`)
		b.WriteString(html.EscapeString(displayFilterOption(f.Field, opt)))
		b.WriteString(`</option>`)
	}

	b.WriteString(`</select></label>`)
}

// renderMultiFilter renders a multi-select filter as a custom dropdown with checkboxes.
func renderMultiFilter(b *strings.Builder, colLabel string, f dataviewFilter, st dataviewFilterState, idx, tableIndex int) {
	// Multi-filter: custom dropdown structure for JS enhancement.
	// The server renders a hidden native <select multiple> as data source,
	// plus a visible button for JS to enhance.
	b.WriteString(`<label class="dataview-filter-label dataview-multi-filter">`)
	b.WriteString(html.EscapeString(colLabel))
	b.WriteString(` <button type="button" class="dataview-multi-btn" data-dataview-filter="`)
	b.WriteString(html.EscapeString(f.Field))
	b.WriteString(`" aria-haspopup="true" aria-expanded="false">`)
	b.WriteString(html.EscapeString(colLabel))
	b.WriteString(`: `)
	if len(st.Selected) == 0 {
		b.WriteString(`All`)
	} else {
		b.WriteString(strconv.Itoa(len(st.Selected)))
		b.WriteString(` selected`)
	}
	b.WriteString(`</button>`)

	// Hidden select with all options.
	b.WriteString(`<select multiple hidden data-dataview-filter-multi="`)
	b.WriteString(html.EscapeString(f.Field))
	b.WriteString(`">`)

	allOptions := buildFilterOptions(st)
	for _, opt := range allOptions {
		isSelected := false
		for _, sel := range st.Selected {
			if opt == sel {
				isSelected = true
				break
			}
		}
		b.WriteString(`<option value="`)
		b.WriteString(html.EscapeString(opt))
		b.WriteString(`"`)
		if isSelected {
			b.WriteString(` selected`)
		}
		b.WriteString(`>`)
		b.WriteString(html.EscapeString(displayFilterOption(f.Field, opt)))
		b.WriteString(`</option>`)
	}

	b.WriteString(`</select>`)

	// Menu container (hidden by default, shown by JS).
	b.WriteString(`<div class="dataview-multi-menu" hidden role="menu">`)
	if f.Clearable {
		b.WriteString(`<label class="dataview-multi-all"><input type="checkbox" value="" role="menuitemcheckbox"`)
		if len(st.Selected) == 0 {
			b.WriteString(` checked`)
		}
		b.WriteString(`> All</label>`)
	}
	for _, opt := range allOptions {
		isSelected := false
		for _, sel := range st.Selected {
			if opt == sel {
				isSelected = true
				break
			}
		}
		b.WriteString(`<label><input type="checkbox" value="`)
		b.WriteString(html.EscapeString(opt))
		b.WriteString(`" role="menuitemcheckbox"`)
		if isSelected {
			b.WriteString(` checked`)
		}
		b.WriteString(`> `)
		b.WriteString(html.EscapeString(displayFilterOption(f.Field, opt)))
		b.WriteString(`</label>`)
	}
	b.WriteString(`</div>`)

	b.WriteString(`</label>`)
}

// buildFilterOptions builds the complete list of options for a filter,
// including synthetic options for defaults not present in detected options.
func buildFilterOptions(st dataviewFilterState) []string {
	// Build set from detected options.
	optSet := map[string]bool{}
	for _, o := range st.Options {
		optSet[o] = true
	}

	// Add any selected values not already in options (synthetic defaults).
	for _, sel := range st.Selected {
		if !optSet[sel] {
			optSet[sel] = true
		}
	}

	result := make([]string, 0, len(optSet))
	for o := range optSet {
		result = append(result, o)
	}
	sort.Strings(result)
	return result
}

// displayFilterOption formats a filter option value for display.
// For tag fields, ensures # prefix.
func displayFilterOption(field, value string) string {
	if isTagField(field) {
		return addTagPrefix(value)
	}
	return value
}

// parseFilterParams extracts filter parameters from a query string.
// Returns map[field][]values.
func parseFilterParams(queryParams map[string][]string) map[string][]string {
	filters := map[string][]string{}
	const prefix = "filter."
	for key, vals := range queryParams {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		field := key[len(prefix):]
		if field == "" {
			continue
		}
		var nonEmpty []string
		for _, v := range vals {
			v = strings.TrimSpace(v)
			if v != "" {
				nonEmpty = append(nonEmpty, v)
			}
		}
		if len(nonEmpty) > 0 {
			filters[field] = nonEmpty
		} else {
			// Empty value means "All" — signal with empty slice.
			filters[field] = []string{}
		}
	}
	return filters
}

// parseFilterParamsFromHTTP extracts filter/sort/q params from HTTP request query params.
// Deduplicates multi-filter values.
func parseFilterParamsFromHTTP(getParams map[string][]string) (dataviewTableParams, error) {
	params := dataviewTableParams{
		Filters: parseFilterParams(getParams),
		Q:       "",
		Sort:    "",
		Dir:     "",
	}

	if qs, ok := getParams["q"]; ok && len(qs) > 0 {
		params.Q = strings.TrimSpace(qs[0])
	}

	sortVals, hasSort := getParams["sort"]
	dirVals, hasDir := getParams["dir"]

	if hasSort && !hasDir {
		return params, fmt.Errorf("sort parameter requires dir")
	}
	if hasDir && !hasSort {
		return params, fmt.Errorf("dir parameter requires sort")
	}

	if hasSort && len(sortVals) > 0 {
		params.Sort = strings.TrimSpace(sortVals[0])
	}
	if hasDir && len(dirVals) > 0 {
		d := strings.ToLower(strings.TrimSpace(dirVals[0]))
		if d != "asc" && d != "desc" {
			return params, fmt.Errorf("invalid dir %q (expected asc or desc)", d)
		}
		params.Dir = d
	}

	return params, nil
}

// validateActionFilters validates that AJAX-provided filters correspond to declared filters.
func validateActionFilters(q dataviewQuery, params dataviewTableParams) error {
	declared := map[string]bool{}
	for _, f := range q.Filters {
		declared[f.Field] = true
	}
	for field := range params.Filters {
		if !declared[field] {
			return fmt.Errorf("undeclared filter field %q", field)
		}
	}
	return nil
}

// validateActionFilterValues validates that AJAX filter values exist in detected options.
// Tag values require the `#` prefix — `project` does not match `#project`.
func validateActionFilterValues(states []dataviewFilterState, params dataviewTableParams) error {
	for _, st := range states {
		vals := params.Filters[st.Filter.Field]
		if vals == nil {
			continue
		}
		for _, v := range vals {
			// Tag fields require `#` prefix.
			if isTagField(st.Filter.Field) && !strings.HasPrefix(v, "#") {
				return fmt.Errorf("filter value %q for tag field %q must include the # prefix", v, st.Filter.Field)
			}
			found := false
			for _, opt := range st.Options {
				if strings.EqualFold(normalizeFilterValue(v), normalizeFilterValue(opt)) {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("filter value %q not found in options for field %q", v, st.Filter.Field)
			}
		}
	}
	return nil
}
