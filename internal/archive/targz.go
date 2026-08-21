package archive

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"io"
	"os"
)

func writeTarGz(ctx context.Context, w io.Writer, srcPath, entryName string, progress ProgressFunc) error {
	gw := gzip.NewWriter(w)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	report(progress, 20, "entry TAR.GZ")

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

	report(progress, 50, "file's compression")
	if _, err := copyWithContext(ctx, tw, f); err != nil {
		return err
	}

	report(progress, 90, "finalize TAR.GZ")
	if err := tw.Close(); err != nil {
		return err
	}
	return gw.Close()
}

func extractTarGz(ctx context.Context, srcPath, destDir string, progress ProgressFunc) ([]string, error) {
	f, err := os.Open(srcPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gr.Close()

	return extractTar(ctx, tar.NewReader(gr), destDir, progress)
}

func writeTarFile(ctx context.Context, r io.Reader, target string, mode os.FileMode) error {
	out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode&^0o077)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = copyWithContext(ctx, out, r)
	return err
}
