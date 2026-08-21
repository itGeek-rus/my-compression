package archive

import (
	"archive/tar"
	"context"
	"io"
	"os"
	"path/filepath"
)

func extractTar(ctx context.Context, tr *tar.Reader, destDir string, progress ProgressFunc) ([]string, error) {
	names := make([]string, 0, 8)

	for i := 0; ; i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		report(progress, 10+min(80, i*10), "unpacking: "+hdr.Name)

		target, err := safeJoin(destDir, hdr.Name)
		if err != nil {
			return nil, err
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return nil, err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return nil, err
			}
			if err := writeTarFile(ctx, tr, target, hdr.FileInfo().Mode().Perm()); err != nil {
				return nil, err
			}
			rel, err := filepath.Rel(destDir, target)
			if err != nil {
				return nil, err
			}
			names = append(names, filepath.ToSlash(rel))
		default:
			continue
		}
	}

	if len(names) == 0 {
		return nil, ErrEmptyInput
	}
	return names, nil
}
