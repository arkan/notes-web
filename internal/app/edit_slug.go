package app

import (
	"regexp"
	"strings"
)

var editSlugNonAlnumRe = regexp.MustCompile(`[^a-z0-9]+`)

// editSlugify transforms a title into a kebab-case filename-safe slug.
// It transliterates common Latin accented characters, lowercases,
// replaces non-alphanumeric runs with hyphens, and trims hyphens.
func editSlugify(s string) string {
	s = transliterateLatinAccents(s)
	s = strings.ToLower(strings.TrimSpace(s))
	s = editSlugNonAlnumRe.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// transliterateLatinAccents replaces common accented Latin characters with
// their ASCII equivalents. Covers Western European languages (French, German,
// Spanish, Italian, Portuguese, Scandinavian, etc).
func transliterateLatinAccents(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case 'À', 'Á', 'Â', 'Ã', 'Ä', 'Å', 'à', 'á', 'â', 'ã', 'ä', 'å':
			b.WriteByte('a')
		case 'Æ', 'æ':
			b.WriteString("ae")
		case 'Ç', 'ç':
			b.WriteByte('c')
		case 'È', 'É', 'Ê', 'Ë', 'è', 'é', 'ê', 'ë':
			b.WriteByte('e')
		case 'Ì', 'Í', 'Î', 'Ï', 'ì', 'í', 'î', 'ï':
			b.WriteByte('i')
		case 'Ð', 'ð':
			b.WriteByte('d')
		case 'Ñ', 'ñ':
			b.WriteByte('n')
		case 'Ò', 'Ó', 'Ô', 'Õ', 'Ö', 'Ø', 'ò', 'ó', 'ô', 'õ', 'ö', 'ø':
			b.WriteByte('o')
		case 'Œ', 'œ':
			b.WriteString("oe")
		case 'Š', 'š':
			b.WriteByte('s')
		case 'ß':
			b.WriteString("ss")
		case 'Þ', 'þ':
			b.WriteString("th")
		case 'Ù', 'Ú', 'Û', 'Ü', 'ù', 'ú', 'û', 'ü':
			b.WriteByte('u')
		case 'Ý', 'Ÿ', 'ý', 'ÿ':
			b.WriteByte('y')
		case 'Ž', 'ž':
			b.WriteByte('z')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
