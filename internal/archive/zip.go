package archive

import (
	"archive/zip"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func writeZIP(ctx context.Context, w io.Writer, scrPath, entryName string, progress ProgressFunc) error {
	zw := zip.NewWriter(w)
	defer zw.Close()

	report(progress, 20, "entry ZIP")

	f, err := os.Open(scrPath)
	if err != nil {
		return err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return err
	}

	hdr, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	hdr.Name = entryName
	hdr.Method = zip.Deflate

	dst, err := zw.CreateHeader(hdr)
	if err != nil {
		return err
	}

	report(progress, 50, "file's compression")
	if _, err := copyWithContext(ctx, dst, f); err != nil {
		return err
	}
	report(progress, 90, "finalize ZIP")
	return zw.Close()
}

func extractZIP(ctx context.Context, srcPath, destDir string, progress ProgressFunc) ([]string, error) {
	r, err := zip.OpenReader(srcPath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	names := make([]string, 0, len(r.File))
	total := len(r.File)
	if total == 0 {
		return nil, ErrEmptyInput
	}

	for i, f := range r.File {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		percent := 10 + (i * 80 / total)
		report(progress, percent, "unpacking: "+f.Name)

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

		if err := writeZipFile(ctx, f, target); err != nil {
			return nil, err
		}

		rel, err := filepath.Rel(destDir, target)
		if err != nil {
			return nil, err
		}
		names = append(names, filepath.ToSlash(rel))
	}

	return names, nil
}

func writeZipFile(ctx context.Context, f *zip.File, target string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode().Perm()&^0o077)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = copyWithContext(ctx, out, rc)
	return err
}

func zipDir(ctx context.Context, w io.Writer, root string, progress ProgressFunc) error {
	zw := zip.NewWriter(w)
	defer zw.Close()

	report(progress, 70, "building ZIP from unpacked files")

	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		hdr, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		hdr.Name = filepath.ToSlash(rel)
		hdr.Method = zip.Deflate

		dst, err := zw.CreateHeader(hdr)
		if err != nil {
			return err
		}

		src, err := os.Open(path)
		if err != nil {
			return err
		}
		defer src.Close()

		_, err = copyWithContext(ctx, dst, src)
		return err
	})
}
