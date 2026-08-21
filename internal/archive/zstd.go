package archive

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/zstd"
)

func writeZstd(ctx context.Context, w io.Writer, srcPath string, progress ProgressFunc) error {
	report(progress, 20, "encode zstd")

	enc, err := zstd.NewWriter(w, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	if err != nil {
		return err
	}
	defer enc.Close()

	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer f.Close()

	report(progress, 50, "file compression")
	if _, err := copyWithContext(ctx, enc, f); err != nil {
		return err
	}
	report(progress, 90, "finalize zstd")
	return enc.Close()
}

func extractZstd(ctx context.Context, srcPath, destDir string, progress ProgressFunc) ([]string, error) {
	report(progress, 20, "decode zstd")

	src, err := os.Open(srcPath)
	if err != nil {
		return nil, err
	}
	defer src.Close()

	dec, err := zstd.NewReader(src)
	if err != nil {
		return nil, err
	}
	defer dec.Close()

	base := strings.TrimSuffix(filepath.Base(srcPath), ".zst")
	base = strings.TrimSuffix(base, ".zstd")
	if base == "" || base == filepath.Base(srcPath) {
		base = "file.bin"
	}

	target, err := safeJoin(destDir, base)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return nil, err
	}

	out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, err
	}
	defer out.Close()

	report(progress, 55, "unpacking: "+base)
	if _, err := copyWithContext(ctx, out, dec); err != nil {
		return nil, err
	}
	report(progress, 90, "finalize zstd")
	return []string{base}, nil
}
