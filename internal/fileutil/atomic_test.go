package fileutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAtomicWriteFileCreatesDirsAndOverwrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "state.json")
	if err := AtomicWriteFile(path, []byte("one"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := AtomicWriteFile(path, []byte("two"), 0644); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "two" {
		t.Fatalf("unexpected data %q", string(data))
	}
}
