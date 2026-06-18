package app

import (
	"fmt"
	"regexp"
	"strings"
)

// dataviewFilter declares a dropdown filter for a Dataview TABLE column.
type dataviewFilter struct {
	Field     string
	Defaults  []string // empty means no default
	Mode      filterMode
	Clearable bool
}

type filterMode int

const (
	filterModeSingle filterMode = iota
	filterModeMulti
)

// parseDataviewFilter parses a single FILTER clause line:
//
//	FILTER <field> [DEFAULT <value-or-list>] [MODE single|multi] [CLEARABLE]
//
// Options may appear in any order after <field>.
// Keywords are case-insensitive.
func parseDataviewFilter(line string) (dataviewFilter, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return dataviewFilter{}, fmt.Errorf("empty FILTER clause")
	}

	// Split into tokens (simple word-based tokenizer).
	tokens := tokenizeFilterLine(line)
	if len(tokens) == 0 {
		return dataviewFilter{}, fmt.Errorf("empty FILTER clause")
	}

	f := dataviewFilter{Mode: filterModeSingle}
	field := tokens[0]
	if !isValidFilterField(field) {
		return dataviewFilter{}, fmt.Errorf("invalid FILTER field %q", field)
	}
	f.Field = field
	tokens = tokens[1:]

	seen := map[string]bool{}
	defaultWasList := false
	for i := 0; i < len(tokens); i++ {
		tok := strings.ToUpper(tokens[i])
		switch tok {
		case "DEFAULT":
			if seen["DEFAULT"] {
				return dataviewFilter{}, fmt.Errorf("duplicate DEFAULT in FILTER for %q", f.Field)
			}
			seen["DEFAULT"] = true
			if i+1 >= len(tokens) {
				return dataviewFilter{}, fmt.Errorf("FILTER %q: DEFAULT requires a value", f.Field)
			}
			next := tokens[i+1]
			if next == "[" {
				defaultWasList = true
				// Parse list default: [value1, value2, ...]
				var list []string
				var closed bool
				i += 2 // skip '[' and move to first item or ']'
				if i >= len(tokens) {
					return dataviewFilter{}, fmt.Errorf("FILTER %q: unclosed DEFAULT list", f.Field)
				}
				if tokens[i] == "]" {
					// Empty list — no default. Don't increment i here;
					// the for loop's i++ will advance past "]".
					continue
				}
				for i < len(tokens) {
					item := tokens[i]
					if item == "]" {
						closed = true
						// Don't increment i — let the for loop's i++ handle it.
						break
					}
					if item == "," {
						i++
						continue
					}
					val, err := parseFilterValue(item)
					if err != nil {
						return dataviewFilter{}, err
					}
					list = append(list, val)
					i++
				}
				if !closed {
					return dataviewFilter{}, fmt.Errorf("FILTER %q: unclosed DEFAULT list", f.Field)
				}
				if len(list) == 0 {
					return dataviewFilter{}, fmt.Errorf("FILTER %q: DEFAULT list is empty", f.Field)
				}
				f.Defaults = dedupeFilterValues(list)
			} else {
				// Single scalar default
				val, err := parseFilterValue(next)
				if err != nil {
					return dataviewFilter{}, err
				}
				f.Defaults = []string{val}
				i++
			}
		case "MODE":
			if seen["MODE"] {
				return dataviewFilter{}, fmt.Errorf("duplicate MODE in FILTER for %q", f.Field)
			}
			seen["MODE"] = true
			if i+1 >= len(tokens) {
				return dataviewFilter{}, fmt.Errorf("FILTER %q: MODE requires a value", f.Field)
			}
			modeVal := strings.ToLower(tokens[i+1])
			switch modeVal {
			case "single":
				f.Mode = filterModeSingle
			case "multi":
				f.Mode = filterModeMulti
			default:
				return dataviewFilter{}, fmt.Errorf("FILTER %q: invalid MODE %q (expected single or multi)", f.Field, tokens[i+1])
			}
			i++
		case "CLEARABLE":
			if seen["CLEARABLE"] {
				return dataviewFilter{}, fmt.Errorf("duplicate CLEARABLE in FILTER for %q", f.Field)
			}
			seen["CLEARABLE"] = true
			f.Clearable = true
		default:
			// Unknown token — could be an unquoted default value that's also a keyword.
			// Per plan, values equal to reserved keywords must be quoted, so reject.
			return dataviewFilter{}, fmt.Errorf("FILTER %q: unexpected token %q", f.Field, tokens[i])
		}
	}

	// Validate mode vs defaults.
	if f.Mode == filterModeSingle && len(f.Defaults) > 1 {
		return dataviewFilter{}, fmt.Errorf("FILTER %q: MODE single cannot have multiple DEFAULT values", f.Field)
	}
	if f.Mode == filterModeMulti && len(f.Defaults) == 1 && !defaultWasList {
		return dataviewFilter{}, fmt.Errorf("FILTER %q: MODE multi requires DEFAULT [...] for multiple values", f.Field)
	}
	if f.Mode == filterModeMulti && len(f.Defaults) == 0 && defaultWasList {
		return dataviewFilter{}, fmt.Errorf("FILTER %q: DEFAULT list is empty", f.Field)
	}

	return f, nil
}

// tokenizeFilterLine splits a FILTER line into tokens handling quotes and brackets.
// Spaces inside brackets act as delimiters (not written to buffer).
func tokenizeFilterLine(line string) []string {
	var tokens []string
	var cur strings.Builder
	inQuote := false
	inBracket := 0
	flush := func() {
		if cur.Len() > 0 {
			tokens = append(tokens, cur.String())
			cur.Reset()
		}
	}
	for _, r := range line {
		switch {
		case r == '"':
			inQuote = !inQuote
			cur.WriteRune(r)
		case r == '[' && !inQuote:
			flush()
			tokens = append(tokens, "[")
			inBracket++
		case r == ']' && !inQuote && inBracket > 0:
			flush()
			tokens = append(tokens, "]")
			inBracket--
		case r == ',' && !inQuote && inBracket > 0:
			flush()
			tokens = append(tokens, ",")
		case r == ' ' || r == '\t':
			if !inQuote && inBracket == 0 {
				flush()
			} else if inQuote {
				cur.WriteRune(r)
			} else {
				// Inside brackets: spaces are delimiters, ignore.
				flush()
			}
		default:
			cur.WriteRune(r)
		}
	}
	flush()
	return tokens
}

// parseFilterValue extracts the actual value from a token, removing quotes if present.
func parseFilterValue(token string) (string, error) {
	token = strings.TrimSpace(token)
	if strings.HasPrefix(token, "\"") && strings.HasSuffix(token, "\"") && len(token) >= 2 {
		inner := token[1 : len(token)-1]
		if inner == "" {
			return "", fmt.Errorf("empty quoted filter value")
		}
		return inner, nil
	}
	return token, nil
}

// isValidFilterField checks if a field name is a valid simple identifier path.
func isValidFilterField(field string) bool {
	if field == "" {
		return false
	}
	// Must match: simple identifier with dots, no spaces, no function calls.
	re := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)*$`)
	return re.MatchString(field)
}

// dedupeFilterValues removes duplicate strings while preserving order.
func dedupeFilterValues(vals []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, v := range vals {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

// validateFilterFieldInColumns checks that the filter field matches a column's source expression.
func validateFilterFieldInColumns(f dataviewFilter, cols []dataviewColumn) error {
	for _, c := range cols {
		if c.Expr == f.Field {
			return nil
		}
	}
	return fmt.Errorf("FILTER field %q does not match any visible table column", f.Field)
}

// isTagField returns true if the field is a tag-like field that should use # prefix.
func isTagField(field string) bool {
	return field == "tags" || field == "file.tags" || field == "file.etags"
}
