package service

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTestSongPackage(t *testing.T, root string, notes string) string {
	t.Helper()
	songDir := filepath.Join(root, "A0001")
	if err := os.MkdirAll(songDir, 0755); err != nil {
		t.Fatal(err)
	}
	xml := `<?xml version="1.0" encoding="utf-8"?>
<MusicData>
  <name><id>1</id><str>Test Song</str></name>
  <genreName><str>maimai</str></genreName>
  <notesData>` + notes + `</notesData>
</MusicData>`
	path := filepath.Join(songDir, "Music.xml")
	if err := os.WriteFile(path, []byte(xml), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeAliasFile(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "alias.json")
	if err := os.WriteFile(path, []byte(`{"aliases":[{"song_id":1,"aliases":["test"]}]}`), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCollectSongInfoFromPackageUsesLocalAlias(t *testing.T) {
	root := t.TempDir()
	writeTestSongPackage(t, root, `<Notes><level>1</level><levelDecimal>0</levelDecimal></Notes>`)
	storage, err := collectSongInfoFromPackage(root, writeAliasFile(t, root))
	if err != nil {
		t.Fatal(err)
	}
	record, err := storage.PickOne("test", 0)
	if err != nil {
		t.Fatal(err)
	}
	if record.Title != "Test Song" {
		t.Fatalf("unexpected title %q", record.Title)
	}
}

func TestParseSongInfoSkipsUnsupportedDifficultyIndex(t *testing.T) {
	root := t.TempDir()
	var notes string
	for i := 0; i < 6; i++ {
		notes += `<Notes><level>1</level><levelDecimal>` + string(rune('0'+i)) + `</levelDecimal></Notes>`
	}
	path := writeTestSongPackage(t, root, notes)
	record, err := parseSongInfoFromXML(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(record.Levels) != 5 {
		t.Fatalf("expected 5 supported levels, got %d", len(record.Levels))
	}
}

func TestPickOneEmptyStorageReturnsError(t *testing.T) {
	storage := &MaimaiStorage{
		records: map[int]*MaimaiRecord{},
		aliases: map[int][]string{},
	}
	if _, err := storage.PickOne("anything", 0); err == nil {
		t.Fatal("expected error for empty storage")
	}
}
