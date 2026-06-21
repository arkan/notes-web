package app

import (
	"path/filepath"
	"strings"
)

// Path policy helpers for the vault. These distinguish between dot-prefixed
// (always blocked), configured hidden (non-enumerated but direct-URL
// addressable), trash (non-enumerated and blocked for direct CRUD/read),
// and template (non-enumerated but direct-read addressable).

// isDotBlocked returns true if any segment of the relative path starts with ".".
func (v *Vault) isDotBlocked(rel string) bool {
	rel = filepath.ToSlash(strings.Trim(rel, "/"))
	if rel == "." || rel == "" {
		return false
	}
	for _, part := range strings.Split(rel, "/") {
		if strings.HasPrefix(part, ".") {
			return true
		}
	}
	return false
}

// isTrashRel checks if the relative path is under the configured trash subtree.
func (v *Vault) isTrashRel(rel string, trashPath string) bool {
	if trashPath == "" {
		return false
	}
	rel = filepath.ToSlash(strings.Trim(rel, "/"))
	trash := filepath.ToSlash(strings.Trim(trashPath, " /"))
	if trash == "" {
		return false
	}
	return rel == trash || strings.HasPrefix(rel, trash+"/")
}

// isTemplateRel checks if the relative path's final segment matches the
// configured template name (e.g. "_template.md").
func (v *Vault) isTemplateRel(rel string, templateName string) bool {
	if templateName == "" {
		return false
	}
	rel = filepath.ToSlash(strings.Trim(rel, "/"))
	tmpl := filepath.ToSlash(strings.Trim(templateName, " /"))
	if tmpl == "" {
		return false
	}
	parts := strings.Split(rel, "/")
	return parts[len(parts)-1] == tmpl
}

// isExcludedFromEnumeration returns true if the path should be excluded from
// normal enumeration: MarkdownFiles, folder listings, tree, sidebar tree,
// Favorites, QuickJump, search, palette, backlinks, Dataview, diagnostics.
//
// Excluded paths: dot-prefixed segments, configured hidden paths, trash
// subtree, and template files (when HideTemplates is true).
func (v *Vault) isExcludedFromEnumeration(rel string, cfg Config) bool {
	if v.isDotBlocked(rel) {
		return true
	}
	if v.isConfiguredHidden(rel, cfg.Hidden) {
		return true
	}
	if v.isTrashRel(rel, cfg.Editing.TrashPath) {
		return true
	}
	if cfg.Editing.HideTemplates && v.isTemplateRel(rel, cfg.Editing.TemplateName) {
		return true
	}
	return false
}

// DirectReadAllowed returns true if the vault path can be served via direct
// URL (note or folder route). Only dot-prefixed and trash paths are blocked
// from direct read. Configured hidden and template paths are accessible.
func (v *Vault) DirectReadAllowed(rel string) bool {
	if v.isDotBlocked(rel) {
		return false
	}
	cfg := v.LoadConfig()
	if v.isTrashRel(rel, cfg.Editing.TrashPath) {
		return false
	}
	return true
}

// DirectMutationAllowed returns true if the vault path can be mutated via
// the edit API (source fetch, preview, save). Only dot-prefixed and trash
// paths are blocked. Configured hidden and template paths are allowed.
// For Phase 1 this matches DirectReadAllowed; later phases may differ for
// create/rename/trash operations.
func (v *Vault) DirectMutationAllowed(rel string) bool {
	return v.DirectReadAllowed(rel)
}

// isMarkdownEditable returns true if the relative path corresponds to a
// Markdown file (.md extension). This is the extension policy for the edit
// API: only .md notes and _template.md (which ends in .md) are editable.
// Non-.md files, directories, and dot-files are rejected earlier by
// DirectMutationAllowed.
func isMarkdownEditable(rel string) bool {
	return strings.HasSuffix(strings.ToLower(rel), ".md")
}
