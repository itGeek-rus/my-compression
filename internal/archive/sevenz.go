package archive

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/bodgit/sevenzip"
)

func extract7z(ctx context.Context, srcPath, destDir string, progress ProgressFunc) ([]string, error) {
	r, err := sevenzip.OpenReader(srcPath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	total := len(r.File)
	if total == 0 {
		return nil, ErrEmptyInput
	}

	names := make([]string, 0, total)
	for i, f := range r.File {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		report(progress, 10+(i*80/total), "unpacking: "+f.Name)

		target, err := safeJoin(destDir, f.Name)
		if err != nil {
			return nil, err
		}

		if f.FileInfo().IsDir() || strings.HasSuffix(f.Name, "/") {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return nil, err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return nil, err
		}

		rc, err := f.Open()
		if err != nil {
			return nil, err
		}

		out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode().Perm()&^0o077)
		if err != nil {
			_ = rc.Close()
			return nil, err
		}
		_, copyErr := copyWithContext(ctx, out, rc)
		_ = out.Close()
		_ = rc.Close()
		if copyErr != nil {
			return nil, copyErr
		}

		rel, err := filepath.Rel(destDir, target)
		if err != nil {
			return nil, err
		}
		names = append(names, filepath.ToSlash(rel))

	}
	if len(names) == 0 {
		return nil, ErrEmptyInput
	}
	return names, nil
}
