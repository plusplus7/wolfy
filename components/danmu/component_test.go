package danmu

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"
	"wolfy/components"
	"wolfy/components/danmu/bilibili"
	"wolfy/model"
)

type fakeRemoteDanmuClient struct {
	mu          sync.Mutex
	startResp   *bilibili.StartGameResponse
	pullResp    []*bilibili.PullDanmuResponse
	afterSeqs   []int64
	stopCalls   int
	pullEntered chan struct{}
}

func (f *fakeRemoteDanmuClient) StartGame(ctx context.Context, req bilibili.StartGameRequest) (*bilibili.StartGameResponse, error) {
	if f.startResp != nil {
		return f.startResp, nil
	}
	return &bilibili.StartGameResponse{Data: bilibili.GameSession{Status: "running"}}, nil
}

func (f *fakeRemoteDanmuClient) GetGame(ctx context.Context, anchorCode string) (*bilibili.StartGameResponse, error) {
	return f.startResp, nil
}

func (f *fakeRemoteDanmuClient) StopGame(ctx context.Context, anchorCode string, req bilibili.StopGameRequest) (*bilibili.StopGameResponse, error) {
	f.mu.Lock()
	f.stopCalls++
	f.mu.Unlock()
	return &bilibili.StopGameResponse{Data: bilibili.GameSession{Status: "stopped"}}, nil
}

func (f *fakeRemoteDanmuClient) PullDanmu(ctx context.Context, anchorCode string, afterSeq int64, limit int, waitMS int) (*bilibili.PullDanmuResponse, error) {
	f.mu.Lock()
	f.afterSeqs = append(f.afterSeqs, afterSeq)
	callIndex := len(f.afterSeqs) - 1
	if f.pullEntered != nil {
		select {
		case f.pullEntered <- struct{}{}:
		default:
		}
	}
	if callIndex < len(f.pullResp) {
		resp := f.pullResp[callIndex]
		f.mu.Unlock()
		return resp, nil
	}
	f.mu.Unlock()
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestRemoteBridgeDeliversMessagesAndTasksOncePerCursor(t *testing.T) {
	taskSink := make(chan *model.Task, 1)
	messageSink := make(chan string, 2)
	client := &fakeRemoteDanmuClient{
		startResp: &bilibili.StartGameResponse{Data: bilibili.GameSession{Status: "running", LastSeq: 0}},
		pullResp: []*bilibili.PullDanmuResponse{
			{
				Events: []bilibili.DanmuEvent{
					{Seq: 1, Caller: "alice", Message: "hello"},
					{Seq: 2, Caller: "bob", Message: "点歌 sky", Task: &model.Task{Command: model.CommandPick, Caller: "bob", Content: "sky", Index: -1}},
				},
				NextSeq: 2,
			},
		},
		pullEntered: make(chan struct{}, 8),
	}
	component := NewDanmuComponent(taskSink, messageSink)
	component.SetRemoteClientFactory(func(baseURL string) RemoteDanmuClient { return client })
	component.UpdateParams(map[string]string{
		ParamRemoteBaseURL: "http://remote.test",
		ParamAppID:         "42",
		ParamAnchorCode:    "anchor",
	})

	if err := component.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = component.Stop(context.Background()) })

	if got := <-messageSink; got != "inf alice hello" {
		t.Fatalf("unexpected first message %q", got)
	}
	if got := <-messageSink; got != "inf bob 点歌 sky" {
		t.Fatalf("unexpected second message %q", got)
	}
	task := <-taskSink
	if task.Command != model.CommandPick || task.Caller != "bob" || task.Content != "sky" {
		t.Fatalf("unexpected task %#v", task)
	}
	waitFor(t, func() bool { return component.LastSeq() == 2 })
	waitFor(t, func() bool {
		client.mu.Lock()
		defer client.mu.Unlock()
		return reflect.DeepEqual(client.afterSeqs, []int64{0, 2})
	})
}

func TestDanmuComponentRegistersRemoteOnlyParamsWithLegacySavedParams(t *testing.T) {
	dir := t.TempDir()
	paramPath := filepath.Join(dir, "params.json")
	if err := os.WriteFile(paramPath, []byte(`{
  "danmu": {
    "remote_base_url": "http://remote.test",
    "app_id": "42",
    "anchor_code": "anchor",
    "bilibili_ak_id": "legacy",
    "bilibili_ak_secret": "legacy"
  }
}`), 0644); err != nil {
		t.Fatal(err)
	}
	manager, err := components.NewManager(paramPath)
	if err != nil {
		t.Fatal(err)
	}
	component := NewDanmuComponent(nil, nil)
	if err := manager.Register(component); err != nil {
		t.Fatal(err)
	}
	params := component.Snapshot().Params
	if _, ok := params["bilibili_ak_id"]; ok {
		t.Fatalf("legacy ak id should not be exposed in snapshot params: %#v", params)
	}
	if params[ParamRemoteBaseURL] != "http://remote.test" || params[ParamAppID] != "42" || params[ParamAnchorCode] != "anchor" {
		t.Fatalf("unexpected params %#v", params)
	}
}

func waitFor(t *testing.T, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition was not met before timeout")
}
