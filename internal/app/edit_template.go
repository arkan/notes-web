package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// templateVars holds the variables available for template substitution.
type templateVars struct {
	Title  string
	Slug   string
	Path   string
	Folder string
	Date   string
}

// resolveNearestTemplate walks upward from targetRel's parent directory to
// the vault root, looking for the configured template file. Returns the
// absolute path, relative path, and content of the first template found, or
// ("", "", nil) if no template is found. Symlink templates are rejected.
func (v *Vault) resolveNearestTemplate(targetRel string, cfg Config) (absPath, relPath, content string, err error) {
	tmplName := cfg.Editing.TemplateName
	if tmplName == "" {
		return "", "", "", nil
	}

	// Start from the target's parent directory and walk up.
	dir := filepath.Dir(strings.Trim(targetRel, "/"))
	for {
		candidateRel := tmplName
		if dir != "." && dir != "" {
			candidateRel = dir + "/" + tmplName
		}
		candidateAbs := filepath.Join(v.Root, filepath.FromSlash(candidateRel))

		if fi, err := os.Lstat(candidateAbs); err == nil {
			if fi.Mode()&os.ModeSymlink != 0 {
				return "", "", "", fmt.Errorf("template %s is a symlink", candidateRel)
			}
			if fi.IsDir() {
				return "", "", "", fmt.Errorf("template %s is a directory", candidateRel)
			}
			data, rerr := os.ReadFile(candidateAbs)
			if rerr != nil {
				return "", "", "", fmt.Errorf("cannot read template %s: %w", candidateRel, rerr)
			}
			return candidateAbs, candidateRel, string(data), nil
		} else if !os.IsNotExist(err) {
			return "", "", "", fmt.Errorf("cannot inspect template %s: %w", candidateRel, err)
		}

		// Walk up.
		if dir == "." || dir == "" {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", "", "", nil
}

// applyTemplate substitutes template variables in the content string.
// Variables use the `{{name}}` syntax. Unknown variables are left as-is.
func applyTemplate(content string, vars templateVars) string {
	repl := strings.NewReplacer(
		"{{title}}", vars.Title,
		"{{slug}}", vars.Slug,
		"{{path}}", vars.Path,
		"{{folder}}", vars.Folder,
		"{{date}}", vars.Date,
	)
	return repl.Replace(content)
}

// todayDate returns the current local date formatted as YYYY-MM-DD.
// Exported as a var for testability (override in tests).
var todayDate = func() string {
	return time.Now().Format("2006-01-02")
}
