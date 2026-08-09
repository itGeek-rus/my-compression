package archive

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"
)

// safeJoin prevents Zip Slip: the name of the file in the archive must not extend beyond the root directory.
func safeJoin(root, name string) (string, error) {
	normalized := strings.ReplaceAll(name, "\\", "/")
	if normalized == "" || strings.HasPrefix(normalized, "/") {
		return "", fmt.Errorf("invalid entry name %q", name)
	}

	// Reject ".." before Clean: path.Clean("/../etc/passwd") collapses to "/etc/passwd"
	// and would otherwise be treated as a safe in-root path.
	for _, seg := range strings.Split(normalized, "/") {
		if seg == ".." {
			return "", fmt.Errorf("illegal path: %s", name)
		}
	}

	clean := path.Clean("/" + normalized)
	rel := strings.TrimPrefix(clean, "/")
	if rel == "" || rel == "." {
		return "", fmt.Errorf("invalid entry name %q", name)
	}

	target := filepath.Join(root, filepath.FromSlash(rel))
	relToRoot, err := filepath.Rel(root, target)
	if err != nil {
		return "", err
	}
	if relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes root: %s", name)
	}
	return target, nil
}
