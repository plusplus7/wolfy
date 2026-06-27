package components

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestBaseComponentRecordsEventsInSnapshot(t *testing.T) {
	component := NewBaseComponent("demo", []string{"token"})

	component.UpdateParams(map[string]string{"token": "abc"})
	component.SetStatus(StatusRunning, nil)
	component.RecordEvent(ComponentEventMessagesMessageReceived, "manual")

	snapshot := component.Snapshot()
	if len(snapshot.Events) != 3 {
		t.Fatalf("expected 3 events, got %#v", snapshot.Events)
	}
	if snapshot.Events[0].Component != "demo" || snapshot.Events[0].Type != ComponentEventParamsUpdated {
		t.Fatalf("unexpected params event %#v", snapshot.Events[0])
	}
	if snapshot.Events[1].Type != ComponentEventStatusChanged {
		t.Fatalf("unexpected status event %#v", snapshot.Events[1])
	}
	if snapshot.Events[2].Type != ComponentEventMessagesMessageReceived {
		t.Fatalf("unexpected manual event %#v", snapshot.Events[2])
	}
	if !strings.Contains(snapshot.Events[2].CodeLocation, "components/events_test.go:") {
		t.Fatalf("unexpected code location %q", snapshot.Events[2].CodeLocation)
	}
	if _, err := time.Parse(time.RFC3339Nano, snapshot.Events[2].Time); err != nil {
		t.Fatalf("event time is not RFC3339Nano: %q", snapshot.Events[2].Time)
	}

	snapshot.Events[2].Message = "mutated"
	if got := component.Snapshot().Events[2].Message; got != "manual" {
		t.Fatalf("snapshot events should be cloned, got %q", got)
	}
}

func TestBaseComponentEventBufferKeepsLatest100(t *testing.T) {
	component := NewBaseComponent("demo", nil)

	for i := 0; i < 105; i++ {
		component.RecordEvent(ComponentEventStatusChanged, strconv.Itoa(i))
	}

	events := component.Snapshot().Events
	if len(events) != 100 {
		t.Fatalf("expected 100 events, got %d", len(events))
	}
	if events[0].Message != "5" || events[len(events)-1].Message != "104" {
		t.Fatalf("unexpected rolling events first=%q last=%q", events[0].Message, events[len(events)-1].Message)
	}
}

func TestComponentEventTypesReturnsCopy(t *testing.T) {
	types := ComponentEventTypes()
	if len(types) == 0 {
		t.Fatal("expected event types")
	}
	types[0].Type = "mutated"

	if ComponentEventTypes()[0].Type == "mutated" {
		t.Fatal("event type catalog should be returned as a copy")
	}
}
