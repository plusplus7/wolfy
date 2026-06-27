package songs

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"wolfy/components"
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

func TestCollectSongInfoFromPackageDumpsAndLoadsCache(t *testing.T) {
	root := t.TempDir()
	writeTestSongPackage(t, root, `<Notes><level>1</level><levelDecimal>0</levelDecimal></Notes>`)
	cachePath := filepath.Join(t.TempDir(), "cache.json")

	storage, err := NewMaimaiStorageFromPackageWithCache(root, writeAliasFile(t, root), cachePath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(cachePath); err != nil {
		t.Fatal(err)
	}
	fromCache, err := NewMaimaiStorageFromCache(cachePath)
	if err != nil {
		t.Fatal(err)
	}
	record, err := fromCache.PickOne("test", 0)
	if err != nil {
		t.Fatal(err)
	}
	if record.Title != storage.records[1].Title {
		t.Fatalf("unexpected cached title %q", record.Title)
	}
}

func TestSongsComponentStartsFromCacheBeforePackage(t *testing.T) {
	root := t.TempDir()
	writeTestSongPackage(t, root, `<Notes><level>1</level><levelDecimal>0</levelDecimal></Notes>`)
	cachePath := filepath.Join(t.TempDir(), "cache.json")
	if _, err := NewMaimaiStorageFromPackageWithCache(root, writeAliasFile(t, root), cachePath); err != nil {
		t.Fatal(err)
	}

	component := NewSongsComponent()
	component.UpdateParams(map[string]string{
		ParamCachePath:   cachePath,
		ParamSongPackage: filepath.Join(t.TempDir(), "missing"),
		ParamAliasFile:   "",
	})
	if err := component.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	record, err := component.Storage().PickOne("test", 0)
	if err != nil {
		t.Fatal(err)
	}
	if record.Title != "Test Song" {
		t.Fatalf("unexpected title %q", record.Title)
	}
	if !hasEvent(component.Snapshot().Events, components.ComponentEventSongsStorageLoaded) {
		t.Fatalf("expected storage loaded event, got %#v", component.Snapshot().Events)
	}
}

func TestConvertStaticSongsJSONToCache(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "cache.json")
	if err := ConvertStaticSongsJSONToCache(filepath.Join("..", "..", "static", "maimai", "songs.json"), cachePath); err != nil {
		t.Fatal(err)
	}
	storage, err := NewMaimaiStorageFromCache(cachePath)
	if err != nil {
		t.Fatal(err)
	}
	record, err := storage.PickOne("真爱", 0)
	if err != nil {
		t.Fatal(err)
	}
	if record.Title != "True Love Song" {
		t.Fatalf("unexpected converted title %q", record.Title)
	}
	if len(record.Levels) == 0 {
		t.Fatal("expected converted levels")
	}
	if record.Levels[0].Level != "12" || record.Levels[0].Difficulty != "mas" {
		t.Fatalf("unexpected converted level %#v", record.Levels[0])
	}
	if record.ImagePath != "https://assets2.lxns.net/maimai/jacket/8.png" {
		t.Fatalf("unexpected image path %q", record.ImagePath)
	}
}

func hasEvent(events []components.ComponentEvent, eventType components.ComponentEventType) bool {
	for _, event := range events {
		if event.Type == eventType {
			return true
		}
	}
	return false
}
