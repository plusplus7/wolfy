package blivedm

import (
	"context"
	"testing"
	"time"
	"wolfy/components"
	servercomponent "wolfy/components/server"
	"wolfy/model"

	"github.com/Akegarasu/blivedm-go/message"
)

type fakeClient struct {
	cookie  string
	starts  int
	stops   int
	handler func(*message.Danmaku)
}

func (f *fakeClient) SetCookie(cookie string) {
	f.cookie = cookie
}

func (f *fakeClient) OnDanmaku(handler func(*message.Danmaku)) {
	f.handler = handler
}

func (f *fakeClient) Start() error {
	f.starts++
	return nil
}

func (f *fakeClient) Stop() {
	f.stops++
}

func TestStartRequiresRoomID(t *testing.T) {
	component := NewBlivedmComponent(make(chan *model.Task, 1), nil)
	component.SetDanmuSourceFunc(func() string { return servercomponent.DanmuSourceBlivedm })

	if err := component.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	snapshot := component.Snapshot()
	if snapshot.Status != components.StatusWaiting || snapshot.Error == "" {
		t.Fatalf("expected waiting room_id error, got %#v", snapshot)
	}
}

func TestStartRejectsInvalidRoomID(t *testing.T) {
	component := NewBlivedmComponent(make(chan *model.Task, 1), nil)
	component.SetDanmuSourceFunc(func() string { return servercomponent.DanmuSourceBlivedm })
	component.UpdateParams(map[string]string{ParamRoomID: "0", ParamCookie: ""})

	if err := component.Start(context.Background()); err == nil {
		t.Fatal("expected error")
	}
	if snapshot := component.Snapshot(); snapshot.Status != components.StatusError {
		t.Fatalf("expected error status, got %#v", snapshot)
	}
}

func TestStartForwardsDanmakuTasks(t *testing.T) {
	taskSink := make(chan *model.Task, 1)
	messageSink := make(chan string, 1)
	component := NewBlivedmComponent(taskSink, messageSink)
	component.SetDanmuSourceFunc(func() string { return servercomponent.DanmuSourceBlivedm })
	component.UpdateParams(map[string]string{ParamRoomID: "732", ParamCookie: "cookie-value"})
	client := &fakeClient{}
	component.factory = func(roomID int) dmClient {
		if roomID != 732 {
			t.Fatalf("unexpected room id %d", roomID)
		}
		return client
	}

	if err := component.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if client.starts != 1 || client.cookie != "cookie-value" || client.handler == nil {
		t.Fatalf("client was not initialized correctly: %#v", client)
	}
	if !hasEvent(component.Snapshot().Events, components.ComponentEventBlivedmClientStarted) {
		t.Fatalf("expected client started event, got %#v", component.Snapshot().Events)
	}
	client.handler(&message.Danmaku{
		Sender:  &message.User{Uname: "alice"},
		Content: "点歌 Oshama Scramble",
	})

	select {
	case task := <-taskSink:
		if task.Command != model.CommandPick || task.Caller != "alice" || task.Content != "Oshama Scramble" {
			t.Fatalf("unexpected task %#v", task)
		}
	case <-time.After(time.Second):
		t.Fatal("expected task")
	}
	select {
	case msg := <-messageSink:
		if msg != "inf alice 点歌 Oshama Scramble" {
			t.Fatalf("unexpected message %q", msg)
		}
	case <-time.After(time.Second):
		t.Fatal("expected message")
	}
	events := component.Snapshot().Events
	if !hasEvent(events, components.ComponentEventBlivedmDanmuReceived) {
		t.Fatalf("expected danmu received event, got %#v", events)
	}
	if !hasEvent(events, components.ComponentEventBlivedmTaskDelivered) {
		t.Fatalf("expected task delivered event, got %#v", events)
	}
}

func TestInactiveSourceDoesNotStart(t *testing.T) {
	component := NewBlivedmComponent(make(chan *model.Task, 1), nil)
	component.SetDanmuSourceFunc(func() string { return servercomponent.DanmuSourceDanmu })
	component.UpdateParams(map[string]string{ParamRoomID: "732", ParamCookie: ""})
	client := &fakeClient{}
	component.factory = func(roomID int) dmClient {
		return client
	}

	if err := component.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if client.starts != 0 {
		t.Fatalf("inactive component should not start, starts=%d", client.starts)
	}
	if snapshot := component.Snapshot(); snapshot.Status != components.StatusWaiting {
		t.Fatalf("expected waiting status, got %#v", snapshot)
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
