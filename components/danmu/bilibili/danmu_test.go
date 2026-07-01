package bilibili

import (
	"testing"
	"wolfy/model"
)

func TestParseRemoteDanmuTaskCommands(t *testing.T) {
	tests := []struct {
		name    string
		message string
		command string
		content string
		index   int64
	}{
		{name: "pick", message: "点歌  sky", command: model.CommandPick, content: "sky", index: -1},
		{name: "next rank", message: "换歌 2", command: model.CommandNextRank, content: "", index: 1},
		{name: "next level", message: "换谱 3", command: model.CommandNextLevel, content: "", index: 2},
		{name: "finish", message: "删除 1", command: model.CommandFinish, content: "", index: 0},
		{name: "zero clamps", message: "删除 0", command: model.CommandFinish, content: "", index: 0},
		{name: "negative clamps", message: "换歌 -1", command: model.CommandNextRank, content: "", index: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := ParseRemoteDanmuTask("alice", tt.message)
			if task == nil {
				t.Fatal("expected task")
			}
			if task.Command != tt.command || task.Caller != "alice" || task.Content != tt.content || task.Index != tt.index {
				t.Fatalf("unexpected task %#v", task)
			}
		})
	}
}

func TestParseRemoteDanmuTaskIgnoresInvalidMessages(t *testing.T) {
	for _, message := range []string{"hello", "换歌 abc", "换谱", "删除"} {
		if task := ParseRemoteDanmuTask("alice", message); task != nil {
			t.Fatalf("%q: expected nil, got %#v", message, task)
		}
	}
}
