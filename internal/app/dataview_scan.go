package app

import (
	"fmt"
	"html"
	"html/template"
	"regexp"
	"strings"
)

var dataviewFenceRe = regexp.MustCompile("(?s)```dataview\\s*\\n(.*?)\\n```")

// dataviewBlockSpan describes one fenced dataview block in the source text.
type dataviewBlockSpan struct {
	Start, End int    // byte indices in text including fences
	QueryText  string // raw query text inside fences
	IsTable    bool   // whether this produces a TABLE output
	TableIndex int    // 1-based TABLE index (0 if not a TABLE)
}

// scanDataviewBlocks finds all fenced dataview blocks in the text and classifies them.
// It returns blocks in document order.
func scanDataviewBlocks(text string) []dataviewBlockSpan {
	matches := dataviewFenceRe.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return nil
	}

	blocks := make([]dataviewBlockSpan, 0, len(matches))
	tableIdx := 0
	for _, m := range matches {
		start := m[0]
		end := m[1]
		queryRaw := text[m[2]:m[3]]
		queryText := strings.TrimSpace(queryRaw)

		// Quick parse: check if it's a TABLE query (first line).
		isTable := false
		firstLine := queryText
		if idx := strings.Index(queryText, "\n"); idx >= 0 {
			firstLine = strings.TrimSpace(queryText[:idx])
		}
		upper := strings.ToUpper(firstLine)
		if strings.HasPrefix(upper, "TABLE") || strings.HasPrefix(upper, "TABLE ") {
			isTable = true
		}

		block := dataviewBlockSpan{
			Start:     start,
			End:       end,
			QueryText: queryText,
			IsTable:   isTable,
		}
		if isTable {
			tableIdx++
			block.TableIndex = tableIdx
		}
		blocks = append(blocks, block)
	}
	return blocks
}

// renderAllDataviewBlocks renders all fenced dataview blocks in the text.
// Non-TABLE blocks are rendered with the existing RenderDataviewBlockWithIndex.
// TABLE blocks get the new pipeline with controls and data attributes.
func renderAllDataviewBlocks(text string, v *Vault, idx *VaultIndex) string {
	blocks := scanDataviewBlocks(text)
	if len(blocks) == 0 {
		return text
	}

	// Process blocks in reverse order to preserve indices.
	var result strings.Builder
	lastEnd := 0
	for _, b := range blocks {
		// Copy text from lastEnd to b.Start.
		result.WriteString(text[lastEnd:b.Start])
		lastEnd = b.End

		q, err := parseDataviewQuery(b.QueryText)
		if err != nil {
			result.WriteString(string(dataviewError(b.QueryText, err)))
			continue
		}

		if b.IsTable {
			// Render with new pipeline (controls, data attributes, filter support).
			html := renderDataviewTableBlock(v, idx, q, b.TableIndex)
			result.WriteString(string(html))
		} else {
			// Non-TABLE blocks with FILTER clauses should show a visible Dataview error.
			if len(q.Filters) > 0 {
				result.WriteString(string(dataviewError(b.QueryText, fmt.Errorf("FILTER is only supported for TABLE queries, not %s", q.Kind))))
				continue
			}
			result.WriteString(string(RenderDataviewBlockWithIndex(v, idx, b.QueryText)))
		}
	}
	result.WriteString(text[lastEnd:])
	return result.String()
}

// renderOneDataviewTable renders only the Nth TABLE block (1-based) as a partial HTML fragment.
func renderOneDataviewTable(text string, v *Vault, idx *VaultIndex, tableIndex int, params dataviewTableParams) (template.HTML, error) {
	if tableIndex < 1 {
		return "", fmt.Errorf("invalid table index %d", tableIndex)
	}

	blocks := scanDataviewBlocks(text)
	// Count tables in document order.
	count := 0
	for _, b := range blocks {
		if b.IsTable {
			count++
			if count == tableIndex {
				q, err := parseDataviewQuery(b.QueryText)
				if err != nil {
					return "", fmt.Errorf("dataview-error: %v", err)
				}
				html := renderDataviewTableBlockWithParams(v, idx, q, tableIndex, params)
				return html, nil
			}
		}
	}

	return "", fmt.Errorf("table index %d out of range (found %d tables)", tableIndex, count)
}

// renderDataviewTableBlock renders a TABLE block with the new pipeline and full controls.
func renderDataviewTableBlock(v *Vault, idx *VaultIndex, q dataviewQuery, tableIndex int) template.HTML {
	return renderDataviewTableBlockWithParams(v, idx, q, tableIndex, dataviewTableParams{})
}

// renderDataviewTableBlockWithParams renders a TABLE block with user-provided params.
func renderDataviewTableBlockWithParams(v *Vault, idx *VaultIndex, q dataviewQuery, tableIndex int, params dataviewTableParams) template.HTML {
	// Validate filters.
	if err := validateFiltersForQuery(q); err != nil {
		return dataviewError(q.String(), err)
	}

	rows, states, err := evalDataviewTableRows(v, idx, q, params)
	if err != nil {
		return dataviewError(q.String(), err)
	}

	capped := false
	totalRows := len(rows)
	if q.Limit < 0 && totalRows > 10 {
		rows = rows[:10]
		capped = true
	}

	// Determine sort metadata for the rendered output.
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

	html := renderDataviewTableHTML(q, rows, states, tableIndex, sortField, sortDir)
	if capped {
		note := `<div class="dataview-cap-note" role="note">Showing first 10 of ` +
			fmt.Sprintf("%d", totalRows) +
			` rows. Add LIMIT to the Dataview query to control this view.</div>`
		return template.HTML(note + string(html))
	}
	return html
}

// queryString returns the query as a string for error display.
func (q dataviewQuery) String() string {
	var b strings.Builder
	b.WriteString(q.Kind)
	if q.Kind == "TABLE" {
		b.WriteString(" ")
		for i, c := range q.Columns {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(c.Expr)
			if c.Label != c.Expr {
				b.WriteString(" as \"")
				b.WriteString(html.EscapeString(c.Label))
				b.WriteString("\"")
			}
		}
	}
	if q.From != "" {
		b.WriteString("\nFROM ")
		b.WriteString(q.From)
	}
	if q.Where != "" {
		b.WriteString("\nWHERE ")
		b.WriteString(q.Where)
	}
	for _, f := range q.Filters {
		b.WriteString("\nFILTER ")
		b.WriteString(f.Field)
		if len(f.Defaults) > 0 {
			if f.Mode == filterModeMulti {
				b.WriteString(" DEFAULT [")
				for i, d := range f.Defaults {
					if i > 0 {
						b.WriteString(", ")
					}
					b.WriteString(d)
				}
				b.WriteString("]")
			} else {
				b.WriteString(" \"")
				b.WriteString(html.EscapeString(f.Defaults[0]))
				b.WriteString("\"")
			}
		}
		if f.Mode == filterModeMulti {
			b.WriteString(" MODE multi")
		}
		if f.Clearable {
			b.WriteString(" CLEARABLE")
		}
	}
	return b.String()
}

func preprocessDataviewBlocks(s string, v *Vault) string {
	if !dataviewFenceRe.MatchString(s) {
		return s
	}
	idx, err := v.BuildIndex()
	if err != nil {
		return dataviewFenceRe.ReplaceAllStringFunc(s, func(m string) string {
			parts := dataviewFenceRe.FindStringSubmatch(m)
			if len(parts) != 2 {
				return m
			}
			return string(dataviewError(parts[1], err))
		})
	}
	return renderAllDataviewBlocks(s, v, idx)
}

// preprocessDataviewBlocksWithIndex is the index-backed variant that reuses
// a pre-built VaultIndex, avoiding a second BuildIndex() call.
func preprocessDataviewBlocksWithIndex(s string, v *Vault, idx *VaultIndex) string {
	if !dataviewFenceRe.MatchString(s) {
		return s
	}
	return renderAllDataviewBlocks(s, v, idx)
}
