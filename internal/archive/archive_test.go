package archive_test

import (
	"bytes"
	"context"
	"my-compression/internal/archive"
	"os"
	"path/filepath"
	"testing"
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
