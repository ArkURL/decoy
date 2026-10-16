package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "progress.json")

	s, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	s.Set("/books/a.epub", "Book A", Position{SpineIndex: 3, Para: 12, LineInPara: 2})
	if err := s.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file not written: %v", err)
	}

	s2, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	pos, ok := s2.Get("/books/a.epub")
	if !ok {
		t.Fatal("position not found")
	}
	if pos != (Position{SpineIndex: 3, Para: 12, LineInPara: 2}) {
		t.Errorf("got %+v", pos)
	}
	if _, ok := s2.Get("/books/other.epub"); ok {
		t.Error("unexpected entry for other book")
	}
}

func TestLoadCorrupt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "progress.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Load(path)
	if err != nil {
		t.Fatalf("Load corrupt should not error: %v", err)
	}
	if len(s.Books) != 0 {
		t.Errorf("expected empty store, got %d", len(s.Books))
	}
}
