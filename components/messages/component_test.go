package messages

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"
	"wolfy/components"
	"wolfy/model"
)

func TestMessagesComponentConsumesMessages(t *testing.T) {
	cleanRuntime(t)
	component := NewMessagesComponent()
	if err := component.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer component.Stop(context.Background())

	sink := component.MessageChan()
	for i := 0; i < 21; i++ {
		sink <- "inf user message-" + strconv.Itoa(i)
	}

	var messages []string
	waitForMessages(t, component, 20, &messages)
	if messages[0] != "inf user message-20" || messages[len(messages)-1] != "inf user message-1" {
		t.Fatalf("unexpected messages %#v", messages)
	}
	events := component.Snapshot().Events
	if !hasEvent(events, components.ComponentEventMessagesMessageReceived) {
		t.Fatalf("expected received event, got %#v", events)
	}
	if !hasEvent(events, components.ComponentEventMessagesMessageStored) {
		t.Fatalf("expected stored event, got %#v", events)
	}
}

func TestMessagesComponentStopDoesNotCloseMessageChan(t *testing.T) {
	cleanRuntime(t)
	component := NewMessagesComponent()
	if err := component.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := component.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}

	component.MessageChan() <- "inf user after-stop"
}

func cleanRuntime(t *testing.T) {
	t.Helper()
	if err := os.RemoveAll("runtime"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll("runtime")
	})
}

func waitForMessages(t *testing.T, component *MessagesComponent, want int, messages *[]string) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		*messages = (*messages)[:0]
		if err := component.ForEachMessage(func(message *model.Message) {
			*messages = append(*messages, message.Content)
		}); err != nil {
			t.Fatal(err)
		}
		if len(*messages) == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("expected %d messages, got %#v", want, *messages)
}

func hasEvent(events []components.ComponentEvent, eventType components.ComponentEventType) bool {
	for _, event := range events {
		if event.Type == eventType {
			return true
		}
	}
	return false
}
