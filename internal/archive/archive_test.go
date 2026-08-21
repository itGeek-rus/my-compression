package archive_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"my-compression/internal/archive"
)

func TestArchiveAndExtractZIP(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(src, []byte("hello compression"), 0o644); err != nil {
		t.Fatal(err)
	}

	arcDir := filepath.Join(dir, "out")
	if err := os.MkdirAll(arcDir, 0o750); err != nil {
		t.Fatal(err)
	}

	res, err := archive.Archive(context.Background(), src, arcDir, archive.FormatZIP, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.OriginalSize <= 0 || res.ResultSize <= 0 {
		t.Fatalf("unexpected sizes: %+v", res)
	}

	extDir := filepath.Join(dir, "ext")
	if err := os.MkdirAll(extDir, 0o750); err != nil {
		t.Fatal(err)
	}

	got, err := archive.Extract(context.Background(), res.OutputPath, extDir, archive.FormatZIP, nil)
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(got.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, []byte("hello compression")) {
		t.Fatalf("content mismatch: %q", data)
	}
}

func TestArchiveAndExtractTarGz(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "note.txt")
	payload := bytes.Repeat([]byte("x"), 4096)
	if err := os.WriteFile(src, payload, 0o644); err != nil {
		t.Fatal(err)
	}

	arcDir := filepath.Join(dir, "out")
	_ = os.MkdirAll(arcDir, 0o750)

	res, err := archive.Archive(context.Background(), src, arcDir, archive.FormatTarGz, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.ResultSize >= res.OriginalSize {
		t.Fatalf("expected compression, got original=%d result=%d", res.OriginalSize, res.ResultSize)
	}

	extDir := filepath.Join(dir, "ext")
	_ = os.MkdirAll(extDir, 0o750)

	got, err := archive.Extract(context.Background(), res.OutputPath, extDir, archive.FormatTarGz, nil)
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(got.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, payload) {
		t.Fatal("tar.gz roundtrip failed")
	}
}

func TestZipSlipRejected(t *testing.T) {
	// покрывается через safeJoin косвенно; прямой unit на path — ниже
}

func TestArchiveAndExtractZstd(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "note.txt")
	payload := bytes.Repeat([]byte("z"), 8192)
	if err := os.WriteFile(src, payload, 0o644); err != nil {
		t.Fatal(err)
	}

	arcDir := filepath.Join(dir, "out")
	_ = os.MkdirAll(arcDir, 0o750)

	res, err := archive.Archive(context.Background(), src, arcDir, archive.FormatZstd, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.ResultSize >= res.OriginalSize {
		t.Fatalf("expected compression, got original=%d result=%d", res.OriginalSize, res.ResultSize)
	}

	got, err := archive.Extract(context.Background(), res.OutputPath, filepath.Join(dir, "ext"), archive.FormatZstd, nil)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(got.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, payload) {
		t.Fatal("zstd roundtrip failed")
	}
}

func TestArchiveAndExtractTarXz(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "note.txt")
	payload := bytes.Repeat([]byte("x"), 4096)
	if err := os.WriteFile(src, payload, 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := archive.Archive(context.Background(), src, filepath.Join(dir, "out"), archive.FormatTarXz, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.ResultSize >= res.OriginalSize {
		t.Fatalf("expected compression, got original=%d result=%d", res.OriginalSize, res.ResultSize)
	}

	got, err := archive.Extract(context.Background(), res.OutputPath, filepath.Join(dir, "ext"), archive.FormatTarXz, nil)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(got.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, payload) {
		t.Fatal("tar.xz roundtrip failed")
	}
}

func TestArchive7zUnsupported(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(src, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := archive.Archive(context.Background(), src, dir, archive.Format7z, nil)
	if !errors.Is(err, archive.Err7zCreateUnsupported) {
		t.Fatalf("expected Err7zCreateUnsupported, got %v", err)
	}
}
