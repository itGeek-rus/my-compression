package archive

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Format string

const (
	FormatZIP   Format = "zip"
	FormatTarGz Format = "tar.gz"
	FormatZstd  Format = "zstd"
	Format7z    Format = "7z"
	FormatTarXz Format = "tar.xz"
)

var (
	ErrUnsupportedFormat   = errors.New("unsupported format")
	ErrEmptyInput          = errors.New("empty input")
	Err7zCreateUnsupported = errors.New("7z packing is not supported; use zstd or tar.xz")
)

// ProgressFunc report progress 0...100. Maybe nil
type ProgressFunc func(percent int, message string)

type Result struct {
	OriginalSize int64
	ResultSize   int64
	OutputPath   string
	OutputName   string
}

func ParseFormat(s string) (Format, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "zip":
		return FormatZIP, nil
	case "tar.gz", "tgz":
		return FormatTarGz, nil
	case "zstd", "zst":
		return FormatZstd, nil
	case "7z":
		return Format7z, nil
	case "tar.xz", "txz":
		return FormatTarXz, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedFormat, s)
	}
}

// Archive packs a single file from srcPath into an archive of the selected format in destDir.
func Archive(ctx context.Context, srcPath, destDir string, format Format, progress ProgressFunc) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}

	info, err := os.Stat(srcPath)
	if err != nil {
		return Result{}, err
	}
	if info.IsDir() {
		return Result{}, errors.New("directories are not supported yet")
	}

	if err := os.MkdirAll(destDir, 0o750); err != nil {
		return Result{}, err
	}

	report(progress, 5, "prepare archive")

	base := filepath.Base(srcPath)
	outName := base + extension(format)
	if format == Format7z {
		return Result{}, Err7zCreateUnsupported
	}
	outPath := filepath.Join(destDir, outName)

	out, err := os.Create(outPath)
	if err != nil {
		return Result{}, err
	}
	defer out.Close()

	switch format {
	case FormatZIP:
		err = writeZIP(ctx, out, srcPath, base, progress)
	case FormatTarGz:
		err = writeTarGz(ctx, out, srcPath, base, progress)
	case FormatZstd:
		err = writeZstd(ctx, out, srcPath, progress)
	case FormatTarXz:
		err = writeTarXz(ctx, out, srcPath, base, progress)
	case Format7z:
		err = Err7zCreateUnsupported
	default:
		err = ErrUnsupportedFormat
	}
	if err != nil {
		_ = os.Remove(outPath)
		return Result{}, err
	}

	if err := out.Sync(); err != nil {
		_ = os.Remove(outPath)
		return Result{}, err
	}

	st, err := out.Stat()
	if err != nil {
		return Result{}, err
	}

	report(progress, 100, "archive is ready")
	return Result{
		OriginalSize: info.Size(),
		ResultSize:   st.Size(),
		OutputPath:   outPath,
		OutputName:   outName,
	}, nil
}

// Extract unpacks the archive into destDir.
// If there is only one file inside, it returns its path.
// If there are multiple files, it packs the contents of destDir into a ZIP file and returns that ZIP file
// (useful for downloading everything as a single file later).
func Extract(ctx context.Context, srcPath, destDir string, format Format, progress ProgressFunc) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}

	info, err := os.Stat(srcPath)
	if err != nil {
		return Result{}, err
	}

	report(progress, 5, "archive reading")

	extractRoot := filepath.Join(destDir, "extracted")
	if err := os.MkdirAll(extractRoot, 0o755); err != nil {
		return Result{}, err
	}

	var names []string
	switch format {
	case FormatZIP:
		names, err = extractZIP(ctx, srcPath, extractRoot, progress)
	case FormatTarGz:
		names, err = extractTarGz(ctx, srcPath, extractRoot, progress)
	case FormatZstd:
		names, err = extractZstd(ctx, srcPath, extractRoot, progress)
	case Format7z:
		names, err = extract7z(ctx, srcPath, extractRoot, progress)
	case FormatTarXz:
		names, err = extractTarXz(ctx, srcPath, extractRoot, progress)
	default:
		err = ErrUnsupportedFormat
	}
	if err != nil {
		return Result{}, err
	}
	if len(names) == 0 {
		return Result{}, ErrEmptyInput
	}

	var (
		outPath string
		outName string
		outSize int64
	)

	if len(names) == 1 {
		outPath = filepath.Join(extractRoot, names[0])
		outName = filepath.Base(names[0])
		st, err := os.Stat(outPath)
		if err != nil {
			return Result{}, err
		}
		outSize = st.Size()
	} else {
		// multiple files → one ZIP file for download
		outName = strings.TrimSuffix(filepath.Base(srcPath), extension(format)) + "-extracted.zip"
		outPath = filepath.Join(destDir, outName)
		out, err := os.Create(outPath)
		if err != nil {
			return Result{}, err
		}
		defer out.Close()

		if err := zipDir(ctx, out, extractRoot, progress); err != nil {
			_ = os.Remove(outPath)
			return Result{}, err
		}
		st, err := out.Stat()
		if err != nil {
			return Result{}, err
		}
		outSize = st.Size()
	}

	report(progress, 100, "unpacking completed")
	return Result{
		OriginalSize: info.Size(),
		ResultSize:   outSize,
		OutputPath:   outPath,
		OutputName:   outName,
	}, nil
}

func extension(format Format) string {
	switch format {
	case FormatZIP:
		return ".zip"
	case FormatTarGz:
		return ".tar.gz"
	case FormatZstd:
		return ".zst"
	case Format7z:
		return ".7z"
	case FormatTarXz:
		return ".tar.xz"
	default:
		return ""
	}
}

func report(progress ProgressFunc, percent int, message string) {
	if progress != nil {
		progress(percent, message)
	}
}

func copyWithContext(ctx context.Context, dst io.Writer, src io.Reader) (int64, error) {
	buf := make([]byte, 32*1024)
	var written int64
	for {
		if err := ctx.Err(); err != nil {
			return written, err
		}
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[:nr])
			written += int64(nw)
			if ew != nil {
				return written, ew
			}
			if nw != nr {
				return written, io.ErrShortWrite
			}
		}
		if er != nil {
			if errors.Is(er, io.EOF) {
				return written, nil
			}
			return written, er
		}
	}
}
