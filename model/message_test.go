package model

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMessageManagerDoesNotOverwriteCorruptCheckpointOnLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "messages.json")
	if err := os.WriteFile(path, []byte("{bad"), 0644); err != nil {
		t.Fatal(err)
	}

	manager := NewMessageManager(path, 3)
	if manager == nil {
		t.Fatal("expected manager")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "{bad" {
		t.Fatalf("expected corrupt checkpoint to remain untouched, got %q", string(data))
	}
}

func TestForEachMessageAllowsCallbackToPush(t *testing.T) {
	manager := NewMessageManager("", 3)
	manager.Push("one")

	manager.ForEachMessage(func(message *Message) {
		manager.Push("two")
	})
}

func TestMessageManagerKeepsOnlyLatestMessages(t *testing.T) {
	manager := NewMessageManager("", 20)
	for i := 0; i < 21; i++ {
		manager.Push(string(rune('a' + i)))
	}

	var messages []string
	manager.ForEachMessage(func(message *Message) {
		messages = append(messages, message.Content)
	})

	if len(messages) != 20 {
		t.Fatalf("expected 20 messages, got %d", len(messages))
	}
	if messages[0] != "u" || messages[len(messages)-1] != "b" {
		t.Fatalf("expected latest 20 messages from newest to oldest, got %#v", messages)
	}
}

func TestMessageManagerIgnoresLegacyExpireTime(t *testing.T) {
	path := filepath.Join(t.TempDir(), "messages.json")
	if err := os.WriteFile(path, []byte(`[{"content":"old","expire_time":1}]`), 0644); err != nil {
		t.Fatal(err)
	}

	manager := NewMessageManager(path, 20)
	var messages []string
	manager.ForEachMessage(func(message *Message) {
		messages = append(messages, message.Content)
	})

	if len(messages) != 1 || messages[0] != "old" {
		t.Fatalf("expected legacy message to be returned, got %#v", messages)
	}
}

func TestMessageManagerTrimsLoadedCheckpoint(t *testing.T) {
	path := filepath.Join(t.TempDir(), "messages.json")
	data := `[{"content":"20"},{"content":"19"},{"content":"18"},{"content":"17"},{"content":"16"},{"content":"15"},{"content":"14"},{"content":"13"},{"content":"12"},{"content":"11"},{"content":"10"},{"content":"9"},{"content":"8"},{"content":"7"},{"content":"6"},{"content":"5"},{"content":"4"},{"content":"3"},{"content":"2"},{"content":"1"},{"content":"0"}]`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	manager := NewMessageManager(path, 20)
	var messages []string
	manager.ForEachMessage(func(message *Message) {
		messages = append(messages, message.Content)
	})

	if len(messages) != 20 || messages[0] != "20" || messages[len(messages)-1] != "1" {
		t.Fatalf("expected loaded checkpoint to be trimmed to latest 20, got %#v", messages)
	}
}
