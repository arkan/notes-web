package app

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// contentHash returns the SHA-256 hex digest of the given data.
func contentHash(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// hashFile returns the SHA-256 hex digest of the file at absPath.
// Returns an error if the file cannot be read.
func hashFile(absPath string) (string, error) {
	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", err
	}
	return contentHash(data), nil
}

// checkSymlinkAncestor returns an error if the target absolute path or any
// ancestor directory between the vault root and target is a symlink.
// Writes to symlink targets and through symlink ancestors are denied.
// When requireTarget is false, non-existing components at any depth are
// silently skipped (only existing components are checked for symlinks).
func checkSymlinkAncestor(vaultRoot, absPath string, requireTarget bool) error {
	rel, err := filepath.Rel(vaultRoot, absPath)
	if err != nil {
		return fmt.Errorf("path not under vault root")
	}
	if strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return fmt.Errorf("path not under vault root")
	}

	// Walk from vault root to target, checking each component.
	parts := strings.Split(filepath.ToSlash(rel), "/")
	current := vaultRoot
	for _, part := range parts {
		current = filepath.Join(current, part)
		fi, err := os.Lstat(current)
		if err != nil {
			if !requireTarget {
				// Non-existing component: skip it. If it doesn't exist it
				// can't be a symlink. For the target component this also
				// means the path doesn't exist yet (preview/create).
				return nil
			}
			return fmt.Errorf("cannot stat %s: %w", current, err)
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("path %s is a symlink or has a symlink ancestor", rel)
		}
	}
	return nil
}

// writeFileWithMode writes data to a temporary file in the same directory,
// then atomically renames it to the target path, using the given mode.
// This is a lower-level variant that does not stat the target for mode.
func writeFileWithMode(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".edit-tmp-*")
	if err != nil {
		return fmt.Errorf("cannot create temp file: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			os.Remove(tmpName)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("cannot write temp file: %w", err)
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return fmt.Errorf("cannot set file mode: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("cannot close temp file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("cannot rename temp file: %w", err)
	}
	cleanup = false
	return nil
}

// atomicWriteFile writes data to a temporary file in the same directory,
// then atomically renames it to the target path, preserving the target's
// existing file mode. If the target does not exist, mode 0644 is used.
func atomicWriteFile(path string, data []byte) error {
	dir := filepath.Dir(path)

	// Determine mode: preserve existing or default 0644.
	mode := os.FileMode(0o644)
	if fi, err := os.Stat(path); err == nil {
		mode = fi.Mode().Perm()
	}

	// Write to temp file in the same directory (atomic rename guarantee).
	tmp, err := os.CreateTemp(dir, ".edit-tmp-*")
	if err != nil {
		return fmt.Errorf("cannot create temp file: %w", err)
	}
	tmpName := tmp.Name()

	// Clean up temp file on any error.
	cleanup := true
	defer func() {
		if cleanup {
			os.Remove(tmpName)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("cannot write temp file: %w", err)
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return fmt.Errorf("cannot set file mode: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("cannot close temp file: %w", err)
	}

	// Atomic rename.
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("cannot rename temp file: %w", err)
	}
	cleanup = false
	return nil
}

// resolveEditPath resolves a vault-relative path and validates it is within
// the vault root. Returns the absolute path and relative path.
func (v *Vault) resolveEditPath(relPath string) (absPath string, rel string, err error) {
	rel = filepath.ToSlash(strings.TrimSpace(relPath))
	rel = strings.TrimPrefix(rel, "/")
	if rel == "" || rel == "." {
		return "", "", fmt.Errorf("invalid path")
	}
	absPath = filepath.Join(v.Root, filepath.FromSlash(rel))
	abs, err := filepath.Abs(absPath)
	if err != nil {
		return "", "", fmt.Errorf("path resolution error")
	}
	resolvedRel, err := filepath.Rel(v.Root, abs)
	if err != nil || strings.HasPrefix(resolvedRel, "..") || filepath.IsAbs(resolvedRel) {
		return "", "", fmt.Errorf("path escapes vault")
	}
	return abs, filepath.ToSlash(resolvedRel), nil
}
