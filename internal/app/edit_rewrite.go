package app

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// ---------- wikilink matching / rewriting ----------

// wikiLinkRe matches [[...]] wikilinks.
var editWikiRe = regexp.MustCompile(`\[\[([^\]]+)\]\]`)

// markdownLinkRe matches [label](target) Markdown links (but not images ![alt](src)).
// Negative lookbehind is not supported by Go's RE2; we handle image exclusion in the
// replacement function.
var markdownLinkRe = regexp.MustCompile(`(?m)\[([^\]]*)\]\(([^)]+)\)`)

// fileImpact describes one file that would be changed by a rename.
type fileImpact struct {
	Path          string `json:"path"`
	Wikilinks     int    `json:"wikilinks"`
	MarkdownLinks int    `json:"markdown_links"`
}

// renameImpact groups impacted files by visibility.
type renameImpact struct {
	Visible   []fileImpact `json:"visible"`
	Hidden    []fileImpact `json:"hidden"`
	Untouched []string     `json:"untouched,omitempty"`
}

// renameScanResult is the full dry-run result.
type renameScanResult struct {
	Path           string            `json:"path"`
	NewPath        string            `json:"new_path"`
	Kind           string            `json:"kind"`
	Impact         renameImpact      `json:"impact"`
	ExpectedHashes map[string]string `json:"expected_hashes"`
}

// ---------- scan enumeration ----------

// rewriteVisibleMD returns all .md files that are link-rewrite-scannable:
// visible notes + configured-hidden addressable notes. Excludes dot-prefixed,
// _trash, and _template.md. Does NOT scan .markdown files.
func (v *Vault) rewriteVisibleMD(cfg Config) []string {
	var out []string
	_ = filepath.WalkDir(v.Root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel := v.Rel(p)

		// Skip dot-prefixed entirely (including .hidden dirs).
		if v.isDotBlocked(rel) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip _trash entirely.
		if v.isTrashRel(rel, cfg.Editing.TrashPath) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip _template.md regardless of hide_templates: templates are outside
		// the normal rewrite/index surface.
		if v.isTemplateRel(rel, cfg.Editing.TemplateName) {
			return nil
		}

		if !d.IsDir() && strings.HasSuffix(strings.ToLower(filepath.Ext(p)), ".md") {
			// Skip symlink files (they are not safe to read/rewrite through).
			if fi, lErr := os.Lstat(p); lErr == nil && fi.Mode()&os.ModeSymlink != 0 {
				return nil
			}
			out = append(out, p)
		}
		return nil
	})
	sort.Strings(out)
	return out
}

// ---------- wikilink matching ----------

// oldRef describes one way a note can be referenced in a wikilink.
type oldRef struct {
	// The exact string that appears in [[...]] (without brackets).
	raw string
	// Whether this ref used .md extension.
	hasExt bool
	// Whether this ref includes a directory path.
	hasPath bool
}

// refsForOld computes all wikilink reference forms for the old relative path.
// oldRel is like "Areas/Old.md".
func refsForOld(oldRel string) []oldRef {
	return refsForOldWithBare(oldRel, true)
}

func refsForOldWithBare(oldRel string, allowBare bool) []oldRef {
	stem := strings.TrimSuffix(filepath.Base(oldRel), filepath.Ext(oldRel))
	dir := filepath.Dir(oldRel)
	noExt := strings.TrimSuffix(oldRel, filepath.Ext(oldRel))

	var refs []oldRef
	if allowBare {
		// Bare stem: [[Old]]
		refs = append(refs, oldRef{raw: stem, hasExt: false, hasPath: false})
		// Stem with .md: [[Old.md]]
		refs = append(refs, oldRef{raw: stem + ".md", hasExt: true, hasPath: false})
	}

	if dir != "." {
		// Path without .md: [[Areas/Old]]
		refs = append(refs, oldRef{raw: noExt, hasExt: false, hasPath: true})
		// Path with .md: [[Areas/Old.md]]
		refs = append(refs, oldRef{raw: oldRel, hasExt: true, hasPath: true})
	}
	return refs
}

// newWikilinkFor computes the new wikilink text (the [[...]] inner part)
// given the old ref style and the new relative path.
// newRel is like "Archive/New.md".
func newWikilinkFor(ref oldRef, newRel string) string {
	stem := strings.TrimSuffix(filepath.Base(newRel), filepath.Ext(newRel))
	noExt := strings.TrimSuffix(newRel, filepath.Ext(newRel))

	if !ref.hasPath {
		// Bare stem or stem with .md: use only the stem (no directory).
		if ref.hasExt {
			return stem + ".md"
		}
		return stem
	}
	// Path target: use full path (with or without .md to match original style).
	if ref.hasExt {
		return newRel
	}
	return noExt
}

// ---------- wikilink rewriting in a single note ----------

// rewriteWikilinksInContent scans content for wikilinks referencing oldRel
// and rewrites them to target the newRel, preserving style, alias, heading.
func rewriteWikilinksInContent(content, oldRel, newRel string) (int, string) {
	return rewriteWikilinksInContentWithBare(content, oldRel, newRel, true)
}

func rewriteWikilinksInContentWithBare(content, oldRel, newRel string, allowBare bool) (int, string) {
	refs := refsForOldWithBare(oldRel, allowBare)
	count := 0
	result := editWikiRe.ReplaceAllStringFunc(content, func(match string) string {
		inner := strings.TrimSuffix(strings.TrimPrefix(match, "[["), "]]")
		parsed, ok := parseEditWikiLink(inner)
		if !ok {
			return match
		}

		// Check if this wikilink references oldRel in any form.
		var matchedRef *oldRef
		for i := range refs {
			if parsed.target == refs[i].raw {
				matchedRef = &refs[i]
				break
			}
		}
		if matchedRef == nil {
			return match
		}

		count++
		newInner := newWikilinkFor(*matchedRef, newRel)
		if parsed.hasHeading {
			newInner += "#" + parsed.heading
		}
		if parsed.hasAlias {
			newInner += "|" + parsed.alias
		}
		return "[[" + newInner + "]]"
	})
	return count, result
}

type editParsedWikiLink struct {
	target     string
	hasAlias   bool
	alias      string
	hasHeading bool
	heading    string
	raw        string
}

func parseEditWikiLink(inner string) (editParsedWikiLink, bool) {
	inner = strings.TrimSpace(inner)
	if inner == "" {
		return editParsedWikiLink{}, false
	}
	// Split alias: [[target|alias]]
	var alias string
	hasAlias := false
	if idx := strings.LastIndex(inner, "|"); idx >= 0 {
		alias = strings.TrimSpace(inner[idx+1:])
		inner = strings.TrimSpace(inner[:idx])
		hasAlias = alias != ""
	}
	// Split heading: [[target#heading]]
	var heading string
	hasHeading := false
	if idx := strings.Index(inner, "#"); idx >= 0 {
		heading = strings.TrimSpace(inner[idx+1:])
		inner = strings.TrimSpace(inner[:idx])
		hasHeading = heading != ""
	}
	if inner == "" {
		return editParsedWikiLink{}, false
	}
	return editParsedWikiLink{
		target:     inner,
		hasAlias:   hasAlias,
		alias:      alias,
		hasHeading: hasHeading,
		heading:    heading,
		raw:        inner,
	}, true
}

// ---------- Markdown link matching / rewriting ----------

// rewriteMarkdownLinksInContent scans content for [label](target) links whose
// resolved relative path points to oldRel and rewrites them to newRel.
func rewriteMarkdownLinksInContent(content, sourceRel, oldRel, newRel string) (int, string) {
	sourceDir := filepath.Dir(sourceRel)

	matches := markdownLinkRe.FindAllStringSubmatchIndex(content, -1)
	var b strings.Builder
	b.Grow(len(content))
	lastEnd := 0
	count := 0

	for _, m := range matches {
		fullStart, fullEnd := m[0], m[1]
		labelStart, labelEnd := m[2], m[3]
		targetStart, targetEnd := m[4], m[5]

		// Append text before this match.
		b.WriteString(content[lastEnd:fullStart])

		// Skip images: preceding char must not be `!`.
		if fullStart > 0 && content[fullStart-1] == '!' {
			b.WriteString(content[fullStart:fullEnd])
			lastEnd = fullEnd
			continue
		}

		label := content[labelStart:labelEnd]
		target := content[targetStart:targetEnd]

		// Skip external schemes, anchors.
		if strings.Contains(target, "://") || strings.HasPrefix(target, "#") || strings.HasPrefix(target, "mailto:") {
			b.WriteString(content[fullStart:fullEnd])
			lastEnd = fullEnd
			continue
		}

		// Split heading fragment.
		linkTarget := target
		heading := ""
		if idx := strings.Index(target, "#"); idx >= 0 {
			linkTarget = target[:idx]
			heading = target[idx:]
		}

		// Try matching the link target against oldRel.
		matched := false
		wasRootRelative := strings.HasPrefix(linkTarget, "/")
		oldNoExt := strings.TrimSuffix(oldRel, filepath.Ext(oldRel))

		if wasRootRelative {
			// Vault-root-relative: /Areas/Old.md → compare with oldRel.
			absTarget := strings.TrimPrefix(linkTarget, "/")
			absNoExt := strings.TrimSuffix(absTarget, filepath.Ext(absTarget))
			if absNoExt == oldNoExt || absTarget == oldRel {
				matched = true
			}
		} else if !filepath.IsAbs(linkTarget) {
			// Relative path: resolve from source note's directory.
			resolved := filepath.ToSlash(filepath.Join(sourceDir, linkTarget))
			resolved = strings.TrimPrefix(resolved, "/")
			resolvedNoExt := strings.TrimSuffix(resolved, filepath.Ext(resolved))
			if resolvedNoExt == oldNoExt || resolved == oldRel {
				matched = true
			}
		}

		if !matched {
			b.WriteString(content[fullStart:fullEnd])
			lastEnd = fullEnd
			continue
		}

		count++

		// Compute new relative path from source to newRel.
		newRelDir := filepath.Dir(newRel)
		relPath, err := filepath.Rel(sourceDir, newRel)
		if err != nil {
			b.WriteString(content[fullStart:fullEnd])
			lastEnd = fullEnd
			continue
		}
		relPath = filepath.ToSlash(relPath)

		// Collapse to basename only when source and new target share the same directory.
		if sourceDir == newRelDir {
			relPath = filepath.Base(newRel)
		}

		newTarget := relPath + heading
		b.WriteString("[" + label + "](" + newTarget + ")")
		lastEnd = fullEnd
	}
	b.WriteString(content[lastEnd:])
	if count == 0 {
		return 0, content
	}
	return count, b.String()
}

// ---------- impact scan ----------

// computeRenameImpact scans all rewrite-scannable .md files for wikilinks
// and Markdown links that reference oldRel. Returns the impact grouped by
// visibility and the set of modified contents (for execution).
func (v *Vault) computeRenameImpact(cfg Config, oldRel, newRel string) (*renameImpact, map[string]string, error) {
	files := v.rewriteVisibleMD(cfg)
	allowBareWikiRefs := v.bareWikilinkReferenceIsUnique(oldRel, files)
	impact := &renameImpact{}
	modified := map[string]string{}

	for _, p := range files {
		rel := v.Rel(p)
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		content := string(data)

		wkCount, wkResult := rewriteWikilinksInContentWithBare(content, oldRel, newRel, allowBareWikiRefs)
		mdCount, mdResult := rewriteMarkdownLinksInContent(content, rel, oldRel, newRel)

		if wkCount == 0 && mdCount == 0 {
			continue
		}

		fi := fileImpact{Path: rel, Wikilinks: wkCount, MarkdownLinks: mdCount}

		// Classify as visible or hidden.
		if v.isConfiguredHidden(rel, cfg.Hidden) {
			impact.Hidden = append(impact.Hidden, fi)
		} else {
			impact.Visible = append(impact.Visible, fi)
		}

		// Use the wikilink result (which includes both wikilink and md rewrites).
		// If only md links changed, use that result.
		finalContent := wkResult
		if wkCount == 0 && mdCount > 0 {
			finalContent = mdResult
		} else if wkCount > 0 && mdCount > 0 {
			// Both changed: apply md rewrites on top of wikilink result.
			_, finalContent = rewriteMarkdownLinksInContent(finalContent, rel, oldRel, newRel)
		}

		modified[rel] = finalContent
	}

	// Sort visible/hidden by path.
	sort.Slice(impact.Visible, func(i, j int) bool { return impact.Visible[i].Path < impact.Visible[j].Path })
	sort.Slice(impact.Hidden, func(i, j int) bool { return impact.Hidden[i].Path < impact.Hidden[j].Path })

	return impact, modified, nil
}

func (v *Vault) bareWikilinkReferenceIsUnique(oldRel string, files []string) bool {
	oldStem := strings.TrimSuffix(filepath.Base(oldRel), filepath.Ext(oldRel))
	count := 0
	for _, p := range files {
		rel := v.Rel(p)
		stem := strings.TrimSuffix(filepath.Base(rel), filepath.Ext(rel))
		if stem == oldStem {
			count++
		}
	}
	return count == 1
}

// computeExpectedHashes reads each modified file from disk and returns its
// sha256 hex digest. This is used for the dry-run response so the client can
// pass the hashes back for conflict detection during execution.
func computeExpectedHashes(v *Vault, modified map[string]string) map[string]string {
	hashes := map[string]string{}
	for rel := range modified {
		abs, _, err := v.resolveEditPath(rel)
		if err != nil {
			continue
		}
		data, err := os.ReadFile(abs)
		if err != nil {
			continue
		}
		hashes[rel] = contentHash(data)
	}
	return hashes
}
