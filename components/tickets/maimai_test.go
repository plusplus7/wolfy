package tickets

import (
	"os"
	"path/filepath"
	"testing"
	"wolfy/components/songs"
	"wolfy/model"
)

func testTicketMaster(t *testing.T) *MaimaiTicketMaster {
	t.Helper()
	storage := testStorage(t)
	return &MaimaiTicketMaster{
		tickets: []*MaimaiTicket{{
			Keyword: "test",
			Creator: "alice",
			Record: &songs.MaimaiRecord{
				Title:     "Test Song",
				ImagePath: "cover.png",
				Category:  "舞萌",
				Levels: []songs.MaimaiLevel{{
					Type:       "std",
					Difficulty: "bas",
					Level:      "1.0",
				}},
			},
		}},
		maxTicketSize: 3,
		storage:       storage,
	}
}

func testStorage(t *testing.T) *songs.MaimaiStorage {
	t.Helper()
	root := t.TempDir()
	songDir := filepath.Join(root, "A0001")
	if err := os.MkdirAll(songDir, 0755); err != nil {
		t.Fatal(err)
	}
	xml := `<?xml version="1.0" encoding="utf-8"?>
<MusicData>
  <name><id>1</id><str>Test Song</str></name>
  <genreName><str>maimai</str></genreName>
  <notesData><Notes><level>1</level><levelDecimal>0</levelDecimal></Notes></notesData>
</MusicData>`
	if err := os.WriteFile(filepath.Join(songDir, "Music.xml"), []byte(xml), 0644); err != nil {
		t.Fatal(err)
	}
	aliasPath := filepath.Join(root, "alias.json")
	if err := os.WriteFile(aliasPath, []byte(`{"aliases":[{"song_id":1,"aliases":["test"]}]}`), 0644); err != nil {
		t.Fatal(err)
	}
	return songs.NewMaimaiStorage(root, aliasPath)
}

func TestTicketMasterInvalidIndexesReturnErrors(t *testing.T) {
	cases := []struct {
		name string
		call func(*MaimaiTicketMaster) (string, error)
	}{
		{"finish negative", func(tm *MaimaiTicketMaster) (string, error) { return tm.FinishTicket("alice", -2) }},
		{"next rank negative", func(tm *MaimaiTicketMaster) (string, error) { return tm.NextRank("alice", -2) }},
		{"next level negative", func(tm *MaimaiTicketMaster) (string, error) { return tm.NextLevel("alice", -2) }},
		{"finish missing own ticket", func(tm *MaimaiTicketMaster) (string, error) { return tm.FinishTicket("bob", -1) }},
		{"next rank out of range", func(tm *MaimaiTicketMaster) (string, error) { return tm.NextRank("alice", 99) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := tc.call(testTicketMaster(t)); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestTicketGettersHandleMissingRecordAndLevels(t *testing.T) {
	nilRecord := &MaimaiTicket{}
	if nilRecord.GetTitle() == "" {
		t.Fatal("expected placeholder title for nil record")
	}
	emptyLevels := &MaimaiTicket{Record: &songs.MaimaiRecord{Title: "empty"}}
	if emptyLevels.GetCoverInfo() != "-" {
		t.Fatalf("unexpected cover info %q", emptyLevels.GetCoverInfo())
	}
	if emptyLevels.GetSongInfo() != "-_-" {
		t.Fatalf("unexpected song info %q", emptyLevels.GetSongInfo())
	}
}

func TestAddTicketEmptyStorageReturnsError(t *testing.T) {
	tm := &MaimaiTicketMaster{
		maxTicketSize: 3,
		storage:       songs.NewMaimaiStorage(t.TempDir(), emptyAliasFile(t)),
	}
	if _, err := tm.AddTicket("alice", "missing"); err == nil {
		t.Fatal("expected error")
	}
	if len(tm.tickets) != 0 {
		t.Fatalf("expected no ticket to be added, got %d", len(tm.tickets))
	}
}

func emptyAliasFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "alias.json")
	if err := os.WriteFile(path, []byte(`{"aliases":[]}`), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestAddTicketLimitsEachCreatorToThreePendingTickets(t *testing.T) {
	tm := testTicketMaster(t)
	tm.maxTicketSize = 10
	for i := 0; i < 2; i++ {
		if _, err := tm.AddTicket("alice", "test"); err != nil {
			t.Fatalf("unexpected add error: %v", err)
		}
	}
	if len(tm.tickets) != 3 {
		t.Fatalf("expected 3 alice tickets, got %d", len(tm.tickets))
	}
	if _, err := tm.AddTicket("alice", "test"); err == nil {
		t.Fatal("expected fourth ticket from same creator to fail")
	}
	if len(tm.tickets) != 3 {
		t.Fatalf("expected failed add not to change tickets, got %d", len(tm.tickets))
	}
	if _, err := tm.AddTicket("bob", "test"); err != nil {
		t.Fatalf("expected another creator to add ticket: %v", err)
	}
}

func TestTicketMasterDoesNotOverwriteCorruptCheckpointOnLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tickets.json")
	if err := os.WriteFile(path, []byte("{bad"), 0644); err != nil {
		t.Fatal(err)
	}

	tm := NewMaimaiTicketMasterWithStorage(testStorage(t), path, 3)
	if tm == nil {
		t.Fatal("expected ticket master")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "{bad" {
		t.Fatalf("expected corrupt checkpoint to remain untouched, got %q", string(data))
	}
}

func TestForEachTicketAllowsCallbackToMutateTickets(t *testing.T) {
	tm := testTicketMaster(t)
	mutated := false

	tm.ForEachTicket(func(ticket model.ITicket) {
		if mutated {
			return
		}
		mutated = true
		if _, err := tm.AddTicket("bob", "test"); err != nil {
			t.Fatalf("unexpected add error: %v", err)
		}
	})

	if !mutated {
		t.Fatal("expected callback to run")
	}
	if len(tm.tickets) != 2 {
		t.Fatalf("expected callback mutation to add ticket, got %d", len(tm.tickets))
	}
}
