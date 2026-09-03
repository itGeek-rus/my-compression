package archive

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"

	"go-algoritms/container"
)

func writeGOAL(ctx context.Context, w io.Writer, srcPath string, progress ProgressFunc) error {
	report(progress, 20, "encode GOAL")

	src, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	report(progress, 50, "file compression")
	blob := container.Pack(src)
	if err := ctx.Err(); err != nil {
		return err
	}

	if _, err := w.Write(blob); err != nil {
		return err
	}
	report(progress, 90, "finalize GOAL")
	return nil
}

func extractGOAL(ctx context.Context, srcPath, destDir string, progress ProgressFunc) ([]string, error) {
	report(progress, 20, "decode GOAL")

	src, err := os.ReadFile(srcPath)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	out, _, err := container.Unpack(src)
	if err != nil {
		return nil, err
	}

	base := strings.TrimSuffix(filepath.Base(srcPath), ".goal")
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

	report(progress, 55, "unpacking: "+base)
	if err := os.WriteFile(target, out, 0o644); err != nil {
		return nil, err
	}
	report(progress, 90, "finalize GOAL")
	return []string{base}, nil
}
