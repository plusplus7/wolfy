package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"wolfy/components/danmu/bilibili"
)

func TestRemoteDanmuServerLifecycleRoutes(t *testing.T) {
	remote := NewRemoteDanmuServerWithStarter(func(ctx context.Context, appID int64, anchorCode string, eventSink chan<- *bilibili.DanmuEvent) (*bilibili.StartAppRespData, error) {
		return &bilibili.StartAppRespData{
			GameInfo: bilibili.GameInfo{GameId: "game-1"},
			AnchorInfo: bilibili.AnchorInfo{
				RoomId: 123,
				Uname:  "alice",
			},
			WebsocketInfo: bilibili.WebSocketInfo{WssLink: []string{"unused"}},
		}, nil
	})
	remote.Register()

	startReq := bytes.NewBufferString(`{"app_id":42,"anchor_code":"anchor"}`)
	rec := httptest.NewRecorder()
	remote.router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/openapi/games", startReq))
	if rec.Code != http.StatusOK {
		t.Fatalf("start status %d: %s", rec.Code, rec.Body.String())
	}
	var startBody bilibili.StartGameResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &startBody); err != nil {
		t.Fatal(err)
	}
	if startBody.Data.Status != "running" || startBody.Data.GameID != "game-1" {
		t.Fatalf("unexpected start body %#v", startBody)
	}

	rec = httptest.NewRecorder()
	remote.router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/openapi/games/anchor", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("get status %d: %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	remote.router.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/openapi/games/anchor", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("stop status %d: %s", rec.Code, rec.Body.String())
	}
	var stopBody bilibili.StopGameResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &stopBody); err != nil {
		t.Fatal(err)
	}
	if stopBody.Data.Status != "stopped" {
		t.Fatalf("unexpected stop body %#v", stopBody)
	}

	rec = httptest.NewRecorder()
	remote.router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/sign", bytes.NewBufferString(`{}`)))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected /sign to be removed, got %d", rec.Code)
	}
}

func TestRemoteDanmuSessionPollsByCursorAndLimit(t *testing.T) {
	session := newRemoteDanmuSession("anchor", 1, func() {}, 10)
	session.appendEvent(bilibili.DanmuEvent{Caller: "a", Message: "one"})
	session.appendEvent(bilibili.DanmuEvent{Caller: "b", Message: "two"})
	session.appendEvent(bilibili.DanmuEvent{Caller: "c", Message: "three"})

	resp := session.poll(t.Context(), 1, 1, 0)
	if len(resp.Events) != 1 || resp.Events[0].Seq != 2 || !resp.HasMore || resp.NextSeq != 2 {
		t.Fatalf("unexpected poll response %#v", resp)
	}
}

func TestRemoteDanmuSessionLongPollsUntilEventOrTimeout(t *testing.T) {
	session := newRemoteDanmuSession("anchor", 1, func() {}, 10)
	go func() {
		time.Sleep(10 * time.Millisecond)
		session.appendEvent(bilibili.DanmuEvent{Caller: "a", Message: "点歌 test"})
	}()

	resp := session.poll(t.Context(), 0, 100, time.Second)
	if len(resp.Events) != 1 || resp.Events[0].Message != "点歌 test" {
		t.Fatalf("unexpected long poll response %#v", resp)
	}

	start := time.Now()
	resp = session.poll(t.Context(), resp.NextSeq, 100, 5*time.Millisecond)
	if len(resp.Events) != 0 {
		t.Fatalf("expected empty timeout response, got %#v", resp)
	}
	if time.Since(start) < 5*time.Millisecond {
		t.Fatalf("poll returned before timeout")
	}
}

func TestRemoteDanmuSessionBoundsBuffer(t *testing.T) {
	session := newRemoteDanmuSession("anchor", 1, func() {}, 2)
	session.appendEvent(bilibili.DanmuEvent{Message: "one"})
	session.appendEvent(bilibili.DanmuEvent{Message: "two"})
	session.appendEvent(bilibili.DanmuEvent{Message: "three"})

	resp := session.poll(t.Context(), 0, 100, 0)
	if len(resp.Events) != 2 || resp.Events[0].Message != "two" || resp.Events[1].Message != "three" {
		t.Fatalf("unexpected bounded events %#v", resp.Events)
	}
}
