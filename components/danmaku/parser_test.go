package danmaku

import (
	"testing"
	"wolfy/model"
)

func TestParseTaskPick(t *testing.T) {
	task := ParseTask("alice", "点歌  Oshama Scramble")
	if task == nil {
		t.Fatal("expected task")
	}
	if task.Command != model.CommandPick || task.Caller != "alice" || task.Content != "Oshama Scramble" || task.Index != -1 {
		t.Fatalf("unexpected task %#v", task)
	}
}

func TestParseTaskIndexedCommands(t *testing.T) {
	tests := []struct {
		name    string
		message string
		command string
		index   int64
	}{
		{name: "next rank", message: "换歌 1", command: model.CommandNextRank, index: 0},
		{name: "next rank zero", message: "换歌 0", command: model.CommandNextRank, index: 0},
		{name: "next rank negative", message: "换歌 -1", command: model.CommandNextRank, index: 0},
		{name: "next level", message: "换谱 2", command: model.CommandNextLevel, index: 1},
		{name: "delete", message: "删除 3", command: model.CommandFinish, index: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := ParseTask("alice", tt.message)
			if task == nil {
				t.Fatal("expected task")
			}
			if task.Command != tt.command || task.Index != tt.index {
				t.Fatalf("unexpected task %#v", task)
			}
		})
	}
}

func TestParseTaskIgnoresUnknownMessages(t *testing.T) {
	if task := ParseTask("alice", "hello"); task != nil {
		t.Fatalf("expected nil, got %#v", task)
	}
	if task := ParseTask("alice", "换歌 abc"); task != nil {
		t.Fatalf("expected nil, got %#v", task)
	}
}
