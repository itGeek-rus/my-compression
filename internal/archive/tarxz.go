package archive

import (
	"archive/tar"
	"context"
	"io"
	"os"

	"github.com/ulikunitz/xz"
)

func writeTarXz(ctx context.Context, w io.Writer, srcPath, entryName string, progress ProgressFunc) error {
	xw, err := xz.NewWriter(w)
	if err != nil {
		return err
	}
	defer xw.Close()

	tw := tar.NewWriter(xw)
	defer tw.Close()

	report(progress, 20, "entry TAR.XZ")

	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return err
	}

	hdr, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	hdr.Name = entryName

	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}

	report(progress, 50, "file compression")
	if _, err := copyWithContext(ctx, tw, f); err != nil {
		return err
	}

	report(progress, 90, "finalize TAR.XZ")
	if err := tw.Close(); err != nil {
		return err
	}
	return xw.Close()
}

func extractTarXz(ctx context.Context, srcPath, destDir string, progress ProgressFunc) ([]string, error) {
	f, err := os.Open(srcPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	xr, err := xz.NewReader(f)
	if err != nil {
		return nil, err
	}

	report(progress, 20, "read TAR.XZ")
	return extractTar(ctx, tar.NewReader(xr), destDir, progress)
}
