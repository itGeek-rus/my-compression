package archive

import "testing"

func TestSafeJoin(t *testing.T) {
	root := t.TempDir()
	ok, err := safeJoin(root, "a/b.txt")
	if err != nil {
		t.Fatal(err)
	}
	if ok == "" {
		t.Fatal("empty path")
	}
	if _, err := safeJoin(root, "../etc/passwd"); err == nil {
		t.Fatal("expected zip-slip error")
	}
	if _, err := safeJoin(root, "foo/../../etc/passwd"); err == nil {
		t.Fatal("expected zip-slip error")
	}
}
