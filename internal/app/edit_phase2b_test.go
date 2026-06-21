package app

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Wikilink rewrite helpers
// ---------------------------------------------------------------------------

func TestRefsForOldBareStem(t *testing.T) {
	refs := refsForOld("Areas/Old.md")
	found := false
	for _, r := range refs {
		if r.raw == "Old" && !r.hasExt && !r.hasPath {
			found = true
		}
	}
	if !found {
		t.Fatalf("refsForOld should include bare stem: %+v", refs)
	}
}

func TestRefsForOldStemWithExt(t *testing.T) {
	refs := refsForOld("Areas/Old.md")
	found := false
	for _, r := range refs {
		if r.raw == "Old.md" && r.hasExt && !r.hasPath {
			found = true
		}
	}
	if !found {
		t.Fatalf("refsForOld should include stem with .md: %+v", refs)
	}
}

func TestRefsForOldPathTarget(t *testing.T) {
	refs := refsForOld("Areas/Old.md")
	foundBare := false
	foundPath := false
	for _, r := range refs {
		if r.raw == "Areas/Old" && r.hasPath && !r.hasExt {
			foundPath = true
		}
		if r.raw == "Areas/Old.md" && r.hasPath && r.hasExt {
			foundBare = true
		}
	}
	if !foundPath || !foundBare {
		t.Fatalf("refsForOld should include both path forms: %+v", refs)
	}
}

func TestNewWikilinkForBarePreservesBare(t *testing.T) {
	ref := oldRef{raw: "Old", hasExt: false, hasPath: false}
	got := newWikilinkFor(ref, "Archive/New.md")
	if got != "New" {
		t.Fatalf("bare ref -> %q, want %q", got, "New")
	}
}

func TestNewWikilinkForExtPreservesExt(t *testing.T) {
	ref := oldRef{raw: "Old.md", hasExt: true, hasPath: false}
	got := newWikilinkFor(ref, "Archive/New.md")
	if got != "New.md" {
		t.Fatalf("ext ref -> %q, want %q", got, "New.md")
	}
}

func TestNewWikilinkForPathPreservesPathNoExt(t *testing.T) {
	ref := oldRef{raw: "Areas/Old", hasExt: false, hasPath: true}
	got := newWikilinkFor(ref, "Archive/New.md")
	if got != "Archive/New" {
		t.Fatalf("path ref -> %q, want %q", got, "Archive/New")
	}
}

func TestNewWikilinkForPathExtPreservesPathAndExt(t *testing.T) {
	ref := oldRef{raw: "Areas/Old.md", hasExt: true, hasPath: true}
	got := newWikilinkFor(ref, "Archive/New.md")
	if got != "Archive/New.md" {
		t.Fatalf("path+ext ref -> %q, want %q", got, "Archive/New.md")
	}
}

func TestRewriteWikilinksBare(t *testing.T) {
	content := "See [[Old]] and [[Old|alias]] for details."
	count, result := rewriteWikilinksInContent(content, "Areas/Old.md", "Archive/New.md")
	if count != 2 {
		t.Fatalf("expected 2 rewrites, got %d", count)
	}
	if !strings.Contains(result, "[[New]]") {
		t.Fatalf("bare wikilink should become [[New]]: %s", result)
	}
	if !strings.Contains(result, "[[New|alias]]") {
		t.Fatalf("wikilink with alias should become [[New|alias]]: %s", result)
	}
}

func TestRewriteWikilinksExt(t *testing.T) {
	content := "See [[Old.md]]."
	count, result := rewriteWikilinksInContent(content, "Areas/Old.md", "Archive/New.md")
	if count != 1 {
		t.Fatalf("expected 1 rewrite, got %d", count)
	}
	if !strings.Contains(result, "[[New.md]]") {
		t.Fatalf("ext wikilink should become [[New.md]]: %s", result)
	}
}

func TestRewriteWikilinksPath(t *testing.T) {
	content := "See [[Areas/Old]]."
	count, result := rewriteWikilinksInContent(content, "Areas/Old.md", "Archive/New.md")
	if count != 1 {
		t.Fatalf("expected 1 rewrite, got %d", count)
	}
	if !strings.Contains(result, "[[Archive/New]]") {
		t.Fatalf("path wikilink should become [[Archive/New]]: %s", result)
	}
}

func TestRewriteWikilinksPathExt(t *testing.T) {
	content := "See [[Areas/Old.md]]."
	count, result := rewriteWikilinksInContent(content, "Areas/Old.md", "Archive/New.md")
	if count != 1 {
		t.Fatalf("expected 1 rewrite, got %d", count)
	}
	if !strings.Contains(result, "[[Archive/New.md]]") {
		t.Fatalf("path+ext wikilink should become [[Archive/New.md]]: %s", result)
	}
}

func TestRewriteWikilinksPreservesHeadingAndAlias(t *testing.T) {
	content := "See [[Areas/Old.md#section|click here]]."
	count, result := rewriteWikilinksInContent(content, "Areas/Old.md", "Archive/New.md")
	if count != 1 {
		t.Fatalf("expected 1 rewrite, got %d", count)
	}
	if !strings.Contains(result, "[[Archive/New.md#section|click here]]") {
		t.Fatalf("should preserve heading+alias: %s", result)
	}
}

func TestRewriteWikilinksOnlyMatchingTarget(t *testing.T) {
	content := "See [[Old]] but not [[Other]] or [[Another.md]]."
	count, result := rewriteWikilinksInContent(content, "Areas/Old.md", "Archive/New.md")
	if count != 1 {
		t.Fatalf("expected 1 rewrite (Old), got %d", count)
	}
	if !strings.Contains(result, "[[Other]]") || !strings.Contains(result, "[[Another.md]]") {
		t.Fatal("unrelated wikilinks should remain unchanged")
	}
	if strings.Contains(result, "[[Old]]") {
		t.Fatal("matching wikilink should be rewritten")
	}
}

// ---------------------------------------------------------------------------
// Markdown link rewrite helpers
// ---------------------------------------------------------------------------

func TestRewriteMarkdownLinksRelativeFromRoot(t *testing.T) {
	// From a root note, [old](Areas/Old.md) resolves to Areas/Old.md (relative from source dir).
	content := `See [old note](Areas/Old.md) for context.`
	count, result := rewriteMarkdownLinksInContent(content, "Root.md", "Areas/Old.md", "Archive/New.md")
	if count != 1 {
		t.Fatalf("expected 1 rewrite, got %d", count)
	}
	if !strings.Contains(result, "Archive/New.md") {
		t.Fatalf("root-relative md link should be rewritten: %s", result)
	}
}

func TestRewriteMarkdownLinksVaultRootRelative(t *testing.T) {
	// Link with / prefix is vault-root-relative.
	content := `See [old](/Areas/Old.md) for context.`
	count, result := rewriteMarkdownLinksInContent(content, "Any/Source.md", "Areas/Old.md", "Archive/New.md")
	if count != 1 {
		t.Fatalf("expected 1 rewrite, got %d", count)
	}
	want := `See [old](../Archive/New.md) for context.`
	if result != want {
		t.Fatalf("/-prefixed md link should still be rewritten relative to source = %q, want %q", result, want)
	}
}

func TestRewriteMarkdownLinksDoesNotTreatNestedRelativeAsRootRelative(t *testing.T) {
	content := `See [old](Areas/Old.md) for context.`
	count, result := rewriteMarkdownLinksInContent(content, "Areas/Source.md", "Areas/Old.md", "Archive/New.md")
	if count != 0 || result != content {
		t.Fatalf("nested relative link should not be treated as vault-root-relative: count=%d result=%q", count, result)
	}
}

func TestRewriteMarkdownLinksSameDir(t *testing.T) {
	content := `See [old](Old.md).`
	count, result := rewriteMarkdownLinksInContent(content, "Areas/Source.md", "Areas/Old.md", "Areas/New.md")
	if count != 1 {
		t.Fatalf("expected 1 rewrite, got %d", count)
	}
	if !strings.Contains(result, "[old](New.md)") {
		t.Fatalf("same-dir link should use bare name: %s", result)
	}
}

func TestRewriteMarkdownLinksPreservesFragment(t *testing.T) {
	content := `See [old](Old.md#section).`
	count, result := rewriteMarkdownLinksInContent(content, "Areas/Source.md", "Areas/Old.md", "Areas/New.md")
	if count != 1 {
		t.Fatalf("expected 1 rewrite, got %d", count)
	}
	if !strings.Contains(result, "New.md#section") {
		t.Fatalf("should preserve fragment: %s", result)
	}
}

func TestRewriteMarkdownLinksSkipsExternal(t *testing.T) {
	content := `See [web](https://example.com).`
	count, result := rewriteMarkdownLinksInContent(content, "Areas/Source.md", "Areas/Old.md", "Areas/New.md")
	if count != 0 {
		t.Fatalf("should not rewrite external links, got %d", count)
	}
	if !strings.Contains(result, content) {
		t.Fatalf("content should be unchanged: %s", result)
	}
}

func TestRewriteMarkdownLinksSkipsImages(t *testing.T) {
	content := `![image](Old.md)`
	count, _ := rewriteMarkdownLinksInContent(content, "Areas/Source.md", "Areas/Old.md", "Areas/New.md")
	if count != 0 {
		t.Fatalf("should not rewrite image links, got %d", count)
	}
}

func TestRewriteMarkdownLinksSkipsAnchors(t *testing.T) {
	content := `[jump](#section).`
	count, _ := rewriteMarkdownLinksInContent(content, "Areas/Source.md", "Areas/Old.md", "Areas/New.md")
	if count != 0 {
		t.Fatalf("should not rewrite anchor-only links, got %d", count)
	}
}

func TestRewriteMarkdownLinksFreeTextUnchanged(t *testing.T) {
	content := "The word Old.md in prose should remain as-is."
	count, result := rewriteMarkdownLinksInContent(content, "Areas/Source.md", "Areas/Old.md", "Areas/New.md")
	if count != 0 || result != content {
		t.Fatalf("free text should not be rewritten: count=%d result=%q", count, result)
	}
}

// ---------------------------------------------------------------------------
// Rename dry-run
// ---------------------------------------------------------------------------

func TestRenameDryRunNote(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	// Create a note that links to Target.
	if err := os.WriteFile(filepath.Join(v.Root, "Areas", "Linker.md"), []byte("See [[Target]].\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewServer(v, "", "")

	body := `{"path":"Areas/Target.md","new_path":"Areas/RenamedTarget.md","dry_run":true}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/rename", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("dry-run rename: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp renameSuccessResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("expected JSON: %v", err)
	}
	if resp.Status != "preview" {
		t.Fatalf("status = %q, want %q", resp.Status, "preview")
	}
	if resp.Impact == nil {
		t.Fatal("dry-run should return impact")
	}
	if len(resp.Impact.Visible) == 0 {
		t.Fatalf("expected impacted files, got none: %+v", resp.Impact)
	}
	// Source file should still exist (dry-run).
	if _, err := os.Stat(filepath.Join(v.Root, "Areas", "Target.md")); err != nil {
		t.Fatal("source should still exist after dry-run")
	}
}

func TestRenameDryRunDoesNotRewriteAmbiguousBareWikilink(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	if err := os.MkdirAll(filepath.Join(v.Root, "Other"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(v.Root, "Other", "Target.md"), []byte("# Other Target\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(v.Root, "Areas", "Ambiguous Linker.md"), []byte("See [[Target]].\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewServer(v, "", "")

	body := `{"path":"Areas/Target.md","new_path":"Areas/RenamedTarget.md","dry_run":true}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/rename", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("dry-run ambiguous bare rename: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp renameSuccessResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("expected JSON: %v", err)
	}
	if resp.Impact != nil {
		for _, item := range resp.Impact.Visible {
			if item.Path == "Areas/Ambiguous Linker.md" {
				t.Fatalf("ambiguous bare wikilink should not be rewritten: %+v", resp.Impact.Visible)
			}
		}
	}
}

func TestRenameDryRunHasExpectedHashes(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	body := `{"path":"Areas/Target.md","new_path":"Areas/RenamedTarget2.md","dry_run":true}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/rename", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("dry-run hashes: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp renameSuccessResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Hashes == nil {
		t.Fatal("expected_hashes should be present in dry-run response")
	}
}

// ---------------------------------------------------------------------------
// Rename execute
// ---------------------------------------------------------------------------

func TestRenameExecuteNote(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	// Create a linking note.
	linkerPath := filepath.Join(v.Root, "Areas", "Linker2.md")
	if err := os.WriteFile(linkerPath, []byte("See [[Target]] and [[Target.md]].\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewServer(v, "", "")

	// First, dry-run to get expected hashes.
	body := `{"path":"Areas/Target.md","new_path":"Areas/MovedTarget.md","dry_run":true}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/rename", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("dry-run pre-execute: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var dryResp renameSuccessResponse
	json.Unmarshal(w.Body.Bytes(), &dryResp)
	hashes, _ := json.Marshal(dryResp.Hashes)

	// Execute with hashes.
	execBody := `{"path":"Areas/Target.md","new_path":"Areas/MovedTarget.md","dry_run":false,"expected_hashes":` + string(hashes) + `}`
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest("POST", "/_api/edit/rename", strings.NewReader(execBody))
	r2.Header.Set("Content-Type", "application/json")
	r2.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w2, r2)

	if w2.Code != 200 {
		t.Fatalf("execute rename: expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	var execResp renameSuccessResponse
	json.Unmarshal(w2.Body.Bytes(), &execResp)
	if execResp.Status != "renamed" {
		t.Fatalf("status = %q, want %q", execResp.Status, "renamed")
	}

	// Old file should be gone.
	if _, err := os.Stat(filepath.Join(v.Root, "Areas", "Target.md")); err == nil {
		t.Fatal("old file should be removed after rename")
	}
	// New file should exist.
	if _, err := os.Stat(filepath.Join(v.Root, "Areas", "MovedTarget.md")); err != nil {
		t.Fatal("new file should exist after rename")
	}
	// Linker should be rewritten.
	data, err := os.ReadFile(linkerPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "[[MovedTarget]]") {
		t.Fatalf("linker should reference MovedTarget: %s", content)
	}
}

func TestRenameRollbackRestoresSourceWhenTargetWriteFails(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	original := "# Target\n\nSelf [[Target]].\n"
	if err := os.WriteFile(filepath.Join(v.Root, "Areas", "Target.md"), []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewServer(v, "", "")
	newPath := "Areas/" + strings.Repeat("a", 300) + ".md"

	dryPayload, err := json.Marshal(map[string]any{
		"path":     "Areas/Target.md",
		"new_path": newPath,
		"dry_run":  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	dry := httptest.NewRecorder()
	dryReq := httptest.NewRequest("POST", "/_api/edit/rename", strings.NewReader(string(dryPayload)))
	dryReq.Header.Set("Content-Type", "application/json")
	dryReq.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(dry, dryReq)
	if dry.Code != 200 {
		t.Fatalf("dry-run rename rollback fixture: expected 200, got %d: %s", dry.Code, dry.Body.String())
	}
	var preview renameSuccessResponse
	if err := json.Unmarshal(dry.Body.Bytes(), &preview); err != nil {
		t.Fatal(err)
	}
	execPayload, err := json.Marshal(map[string]any{
		"path":            "Areas/Target.md",
		"new_path":        newPath,
		"dry_run":         false,
		"expected_hashes": preview.Hashes,
	})
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/rename", strings.NewReader(string(execPayload)))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)
	if w.Code != 409 {
		t.Fatalf("execute should fail on too-long target filename, got %d: %s", w.Code, w.Body.String())
	}
	data, err := os.ReadFile(filepath.Join(v.Root, "Areas", "Target.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != original {
		t.Fatalf("source should be rolled back, got %q want %q", string(data), original)
	}
}

func TestRenameExecuteTitleDriven(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	// Dry-run first.
	body := `{"path":"Areas/Target.md","title":"New Target","dry_run":true}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/rename", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("title-driven dry-run: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var dryResp renameSuccessResponse
	json.Unmarshal(w.Body.Bytes(), &dryResp)
	if dryResp.NewPath != "Areas/new-target.md" {
		t.Fatalf("dry-run new_path = %q, want %q", dryResp.NewPath, "Areas/new-target.md")
	}
	hashes, _ := json.Marshal(dryResp.Hashes)

	// Execute with hashes from dry-run.
	execBody := `{"path":"Areas/Target.md","title":"New Target","dry_run":false,"expected_hashes":` + string(hashes) + `}`
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest("POST", "/_api/edit/rename", strings.NewReader(execBody))
	r2.Header.Set("Content-Type", "application/json")
	r2.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w2, r2)

	if w2.Code != 200 {
		t.Fatalf("title-driven execute: expected 200, got %d: %s", w2.Code, w2.Body.String())
	}

	var execResp renameSuccessResponse
	json.Unmarshal(w2.Body.Bytes(), &execResp)
	if execResp.Status != "renamed" {
		t.Fatalf("status = %q, want %q", execResp.Status, "renamed")
	}
	if execResp.NewPath != "Areas/new-target.md" {
		t.Fatalf("new_path = %q, want %q", execResp.NewPath, "Areas/new-target.md")
	}
	// Verify old gone, new exists.
	if _, err := os.Stat(filepath.Join(v.Root, "Areas", "Target.md")); err == nil {
		t.Fatal("old file should be removed")
	}
	if _, err := os.Stat(filepath.Join(v.Root, "Areas", "new-target.md")); err != nil {
		t.Fatal("new file should exist")
	}
}

func TestRenameExecuteMissingHashBlocks(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	// Execute without expected hashes should be rejected.
	body := `{"path":"Areas/Target.md","new_path":"Areas/NoHash.md","dry_run":false}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/rename", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 409 {
		t.Fatalf("missing hash execute: expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRenameTargetCollision(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	// Create a target file that will cause collision.
	collideFile := filepath.Join(v.Root, "Areas", "CollideTarget.md")
	if err := os.WriteFile(collideFile, []byte("# Collision target\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewServer(v, "", "")

	body := `{"path":"Areas/Target.md","new_path":"Areas/CollideTarget.md","dry_run":true}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/rename", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 409 {
		t.Fatalf("collision: expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRenameSourceDotPathBlocked(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	body := `{"path":".hidden/file.md","new_path":"Areas/New.md","dry_run":true}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/rename", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 403 {
		t.Fatalf("dot path rename: expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRenameSourceTrashBlocked(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	body := `{"path":"_trash/Old.md","new_path":"Areas/New.md","dry_run":true}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/rename", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 403 {
		t.Fatalf("trash rename: expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRenameTemplateBlocked(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	// Create _template.md file.
	tmplPath := filepath.Join(v.Root, "Areas", "_template.md")
	if err := os.MkdirAll(filepath.Dir(tmplPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tmplPath, []byte("# Template\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewServer(v, "", "")

	body := `{"path":"Areas/_template.md","new_path":"Areas/NotTemplate.md","dry_run":true}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/rename", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 403 {
		t.Fatalf("template rename: expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRenameNonMDSourceBlocked(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	txtPath := filepath.Join(v.Root, "file.txt")
	if err := os.WriteFile(txtPath, []byte("text"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewServer(v, "", "")

	body := `{"path":"file.txt","new_path":"Areas/file.md","dry_run":true}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/rename", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 400 {
		t.Fatalf("non-md rename: expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRenameFolderEmpty(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	// Create an empty folder.
	emptyDir := filepath.Join(v.Root, "EmptyDir")
	if err := os.MkdirAll(emptyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	s := NewServer(v, "", "")

	body := `{"path":"EmptyDir","new_path":"RenamedDir","dry_run":false}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/rename", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("empty folder rename: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp renameSuccessResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Kind != "folder" {
		t.Fatalf("kind = %q, want %q", resp.Kind, "folder")
	}
	if resp.Status != "renamed" {
		t.Fatalf("status = %q, want %q", resp.Status, "renamed")
	}
	if _, err := os.Stat(emptyDir); err == nil {
		t.Fatal("old empty dir should be removed")
	}
	if _, err := os.Stat(filepath.Join(v.Root, "RenamedDir")); err != nil {
		t.Fatal("renamed dir should exist")
	}
}

func TestRenameFolderNonEmptyBlocks(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	nonEmptyDir := filepath.Join(v.Root, "NonEmpty")
	if err := os.MkdirAll(nonEmptyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nonEmptyDir, "note.md"), []byte("# Inside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewServer(v, "", "")

	body := `{"path":"NonEmpty","new_path":"RenamedNonEmpty","dry_run":false}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/rename", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 409 {
		t.Fatalf("non-empty folder: expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRenameFolderBlocksReadDirError(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	locked := filepath.Join(v.Root, "Locked Folder")
	if err := os.Mkdir(locked, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(locked, 0o755)
	if _, err := os.ReadDir(locked); err == nil {
		t.Skip("filesystem permissions do not make ReadDir fail in this environment")
	}
	s := NewServer(v, "", "")

	body := `{"path":"Locked Folder","new_path":"Renamed Locked Folder","dry_run":true}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/rename", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)
	if w.Code != 409 {
		t.Fatalf("expected read-dir error 409, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "folder cannot be read") {
		t.Fatalf("expected folder read error, got %s", w.Body.String())
	}
}

func TestRenameHiddenTransitionConflict(t *testing.T) {
	v := makeVault(t)
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("hidden:\n  - HiddenArea\nediting:\n  enabled: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Source note and target dir.
	srcPath := filepath.Join(v.Root, "VisibleNote.md")
	if err := os.WriteFile(srcPath, []byte("# Visible\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	hiddenDir := filepath.Join(v.Root, "HiddenArea")
	if err := os.MkdirAll(hiddenDir, 0o755); err != nil {
		t.Fatal(err)
	}
	s := NewServer(v, "", "")

	body := `{"path":"VisibleNote.md","new_path":"HiddenArea/Moved.md","dry_run":false}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/rename", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 409 {
		t.Fatalf("hidden transition: expected 409, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["requires_confirmation"] != "hidden" {
		t.Fatalf("expected hidden confirmation, got %v", resp)
	}
}

func TestRenameHiddenTransitionConfirmed(t *testing.T) {
	v := makeVault(t)
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("hidden:\n  - HiddenArea\nediting:\n  enabled: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	srcPath := filepath.Join(v.Root, "VisibleNote.md")
	if err := os.WriteFile(srcPath, []byte("# Visible\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	hiddenDir := filepath.Join(v.Root, "HiddenArea")
	if err := os.MkdirAll(hiddenDir, 0o755); err != nil {
		t.Fatal(err)
	}
	s := NewServer(v, "", "")

	// Dry-run first with confirm_hidden (since transition is visible→hidden).
	dryBody := `{"path":"VisibleNote.md","new_path":"HiddenArea/Moved.md","confirm_hidden":true,"dry_run":true}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/rename", strings.NewReader(dryBody))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("hidden dry-run: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var dryResp renameSuccessResponse
	json.Unmarshal(w.Body.Bytes(), &dryResp)
	hashes, _ := json.Marshal(dryResp.Hashes)

	// Execute with hashes.
	execBody := `{"path":"VisibleNote.md","new_path":"HiddenArea/Moved.md","confirm_hidden":true,"dry_run":false,"expected_hashes":` + string(hashes) + `}`
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest("POST", "/_api/edit/rename", strings.NewReader(execBody))
	r2.Header.Set("Content-Type", "application/json")
	r2.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w2, r2)

	if w2.Code != 200 {
		t.Fatalf("confirmed hidden rename: expected 200, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestRenameSymlinkAncestorBlocks(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)

	realDir := filepath.Join(v.Root, "RealDir")
	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatal(err)
	}
	symDir := filepath.Join(v.Root, "LinkDir")
	if err := os.Symlink("RealDir", symDir); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(symDir)

	// Put a note in the real dir.
	realNote := filepath.Join(realDir, "Note.md")
	if err := os.WriteFile(realNote, []byte("# Under Symlink\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Try to rename the note via the symlink path.
	s := NewServer(v, "", "")
	body := `{"path":"LinkDir/Note.md","new_path":"LinkDir/Renamed.md","dry_run":false}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/rename", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 403 {
		t.Fatalf("symlink rename: expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRenameHashMismatchBlocks(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	s := NewServer(v, "", "")

	body := `{"path":"Areas/Target.md","new_path":"Areas/HashCheck.md","dry_run":false,"expected_hashes":{"Areas/Target.md":"wronghash"}}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/rename", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 409 {
		t.Fatalf("hash mismatch: expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRenameCollisionBlock(t *testing.T) {
	v := makeVault(t)
	enableEditing(t, v)
	// Create a target file that would collide.
	collidePath := filepath.Join(v.Root, "Areas", "CollideTarget.md")
	if err := os.WriteFile(collidePath, []byte("# Collision\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewServer(v, "", "")

	body := `{"path":"Areas/Target.md","new_path":"Areas/CollideTarget.md","dry_run":false}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/rename", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 409 {
		t.Fatalf("collision: expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRewriteScanIncludesConfigHidden(t *testing.T) {
	v := makeVault(t)
	// Create a configured hidden note that links to Target.
	if err := os.WriteFile(filepath.Join(v.Root, ".notes-web.yaml"), []byte("hidden:\n  - Secret\nediting:\n  enabled: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	secretDir := filepath.Join(v.Root, "Secret")
	if err := os.MkdirAll(secretDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(secretDir, "Linker.md"), []byte("See [[Target]].\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewServer(v, "", "")

	body := `{"path":"Areas/Target.md","new_path":"Areas/MovedFromHidden.md","dry_run":true}`
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/_api/edit/rename", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-CSRF-Token", s.csrfToken)
	s.ServeHTTP(w, r)

	if w.Code != 200 {
		t.Fatalf("hidden scan: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp renameSuccessResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Impact == nil {
		t.Fatal("impact should not be nil")
	}
	// Should find the link in the hidden note.
	found := false
	for _, fi := range resp.Impact.Hidden {
		if strings.Contains(fi.Path, "Secret") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("hidden note's link should be in impact.Hidden: %+v", resp.Impact)
	}
}
