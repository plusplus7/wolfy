package service

import "testing"

func testTicketMaster() *MaimaiTicketMaster {
	return &MaimaiTicketMaster{
		tickets: []*MaimaiTicket{{
			Keyword: "test",
			Creator: "alice",
			Record: &MaimaiRecord{
				Title:     "Test Song",
				ImagePath: "cover.png",
				Category:  "舞萌",
				Levels: []MaimaiLevel{{
					Type:       "std",
					Difficulty: "bas",
					Level:      "1.0",
				}},
			},
		}},
		maxTicketSize: 3,
		storage: &MaimaiStorage{
			records: map[int]*MaimaiRecord{
				1: {
					Title: "Test Song",
					Levels: []MaimaiLevel{{
						Type:       "std",
						Difficulty: "bas",
						Level:      "1.0",
					}},
				},
			},
			aliases: map[int][]string{1: {"test"}},
		},
	}
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
			if _, err := tc.call(testTicketMaster()); err == nil {
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
	emptyLevels := &MaimaiTicket{Record: &MaimaiRecord{Title: "empty"}}
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
		storage: &MaimaiStorage{
			records: map[int]*MaimaiRecord{},
			aliases: map[int][]string{},
		},
	}
	if _, err := tm.AddTicket("alice", "missing"); err == nil {
		t.Fatal("expected error")
	}
	if len(tm.tickets) != 0 {
		t.Fatalf("expected no ticket to be added, got %d", len(tm.tickets))
	}
}

func TestAddTicketLimitsEachCreatorToThreePendingTickets(t *testing.T) {
	tm := testTicketMaster()
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
