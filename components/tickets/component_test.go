package tickets

import (
	"errors"
	"testing"
	"wolfy/components"
	"wolfy/model"
)

type fakeTicketMaster struct {
	addErr error
}

func (f fakeTicketMaster) AddTicket(creator string, keyword string) (string, error) {
	if f.addErr != nil {
		return "", f.addErr
	}
	return "ok", nil
}

func (f fakeTicketMaster) FinishTicket(operator string, index int64) (string, error) {
	return "closed", nil
}

func (f fakeTicketMaster) ForEachTicket(fn func(model.ITicket)) {}

func (f fakeTicketMaster) NextLevel(operator string, index int64) (string, error) {
	return "level", nil
}

func (f fakeTicketMaster) NextRank(operator string, index int64) (string, error) {
	return "rank", nil
}

func TestHandleTaskRecordsReceivedAndCompletedEvents(t *testing.T) {
	messageSink := make(chan string, 1)
	component := &TicketsComponent{
		BaseComponent: components.NewBaseComponent(TicketsComponentName, nil),
		ticketMaster:  fakeTicketMaster{},
		messageSink:   messageSink,
	}

	msg, err := component.HandleTask(&model.Task{
		Command: model.CommandPick,
		Caller:  "alice",
		Content: "test",
		Index:   -1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if msg != "ok" {
		t.Fatalf("unexpected result %q", msg)
	}
	events := component.Snapshot().Events
	if !hasEvent(events, components.ComponentEventTicketsTaskReceived) {
		t.Fatalf("expected task received event, got %#v", events)
	}
	if !hasEvent(events, components.ComponentEventTicketsTaskCompleted) {
		t.Fatalf("expected task completed event, got %#v", events)
	}
	if got := <-messageSink; got != "inf alice ok" {
		t.Fatalf("unexpected message %q", got)
	}
}

func TestHandleTaskRecordsFailedEvents(t *testing.T) {
	component := &TicketsComponent{
		BaseComponent: components.NewBaseComponent(TicketsComponentName, nil),
		ticketMaster:  fakeTicketMaster{addErr: errors.New("boom")},
		messageSink:   make(chan string, 1),
	}

	_, err := component.HandleTask(&model.Task{
		Command: model.CommandPick,
		Caller:  "alice",
		Content: "test",
		Index:   -1,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	events := component.Snapshot().Events
	if !hasEvent(events, components.ComponentEventTicketsTaskReceived) {
		t.Fatalf("expected task received event, got %#v", events)
	}
	if !hasEvent(events, components.ComponentEventTicketsTaskFailed) {
		t.Fatalf("expected task failed event, got %#v", events)
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
