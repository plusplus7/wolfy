package danmu

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"
	"wolfy/components"
	"wolfy/components/danmu/bilibili"
	"wolfy/model"
)

const (
	DanmuComponentName = "danmu"
	ParamRemoteBaseURL = "remote_base_url"
	ParamAppID         = "app_id"
	ParamAnchorCode    = "anchor_code"

	defaultRemoteDanmuServer = "https://plusplus7.com:42376"
	defaultPollLimit         = 100
	defaultPollWaitMS        = 1000
)

type RemoteDanmuClient interface {
	StartGame(ctx context.Context, req bilibili.StartGameRequest) (*bilibili.StartGameResponse, error)
	GetGame(ctx context.Context, anchorCode string) (*bilibili.StartGameResponse, error)
	StopGame(ctx context.Context, anchorCode string, req bilibili.StopGameRequest) (*bilibili.StopGameResponse, error)
	PullDanmu(ctx context.Context, anchorCode string, afterSeq int64, limit int, waitMS int) (*bilibili.PullDanmuResponse, error)
}

type RemoteDanmuClientFactory func(baseURL string) RemoteDanmuClient

type DanmuComponent struct {
	*components.BaseComponent
	taskSink      chan<- *model.Task
	messageSink   chan<- string
	cancel        context.CancelFunc
	client        RemoteDanmuClient
	clientFactory RemoteDanmuClientFactory
	lastSeq       int64
	mu            sync.Mutex
}

func NewDanmuComponent(taskSink chan<- *model.Task, messageSink chan<- string) *DanmuComponent {
	return &DanmuComponent{
		BaseComponent: components.NewBaseComponent(DanmuComponentName, []string{
			ParamRemoteBaseURL,
			ParamAppID,
			ParamAnchorCode,
		}),
		taskSink:      taskSink,
		messageSink:   messageSink,
		clientFactory: func(baseURL string) RemoteDanmuClient { return bilibili.NewRemoteDanmuClient(baseURL) },
	}
}

func (d *DanmuComponent) SetRemoteClientFactory(factory RemoteDanmuClientFactory) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if factory == nil {
		d.clientFactory = func(baseURL string) RemoteDanmuClient { return bilibili.NewRemoteDanmuClient(baseURL) }
		return
	}
	d.clientFactory = factory
}

func (d *DanmuComponent) Start(ctx context.Context) error {
	params := d.Params()
	remoteBaseURL := params[ParamRemoteBaseURL]
	appIDText := params[ParamAppID]
	anchorCode := params[ParamAnchorCode]
	if remoteBaseURL == "" {
		remoteBaseURL = defaultRemoteDanmuServer
	}
	if appIDText == "" {
		err := errors.New("app_id is empty")
		d.SetStatus(components.StatusWaiting, err)
		return nil
	}
	if anchorCode == "" {
		err := errors.New("anchor_code is empty")
		d.SetStatus(components.StatusWaiting, err)
		return nil
	}
	appID, err := strconv.ParseInt(appIDText, 10, 64)
	if err != nil {
		d.SetStatus(components.StatusError, err)
		return err
	}

	d.mu.Lock()
	if d.cancel != nil {
		d.cancel()
	}
	runCtx, cancel := context.WithCancel(ctx)
	client := d.clientFactory(remoteBaseURL)
	d.cancel = cancel
	d.client = client
	d.mu.Unlock()

	d.SetStatus(components.StatusRestarting, nil)
	startResp, err := client.StartGame(runCtx, bilibili.StartGameRequest{
		AppID:      appID,
		AnchorCode: anchorCode,
	})
	if err != nil {
		cancel()
		d.SetStatus(components.StatusError, err)
		d.RecordEvent(components.ComponentEventDanmuListenerFailed, err.Error())
		return err
	}
	lastSeq := int64(0)
	if startResp != nil {
		lastSeq = startResp.Data.LastSeq
	}
	d.setLastSeq(lastSeq)
	d.RecordEvent(components.ComponentEventDanmuListenerStarted, "remote danmu listener started")
	d.SetStatus(components.StatusRunning, nil)
	go d.pullLoop(runCtx, client, anchorCode)
	return nil
}

func (d *DanmuComponent) Stop(ctx context.Context) error {
	d.mu.Lock()
	cancel := d.cancel
	client := d.client
	anchorCode := d.Params()[ParamAnchorCode]
	d.cancel = nil
	d.client = nil
	d.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if client != nil && anchorCode != "" {
		_, err := client.StopGame(ctx, anchorCode, bilibili.StopGameRequest{Reason: "local stop"})
		if err != nil && ctx.Err() == nil {
			d.SetStatus(components.StatusError, err)
			return err
		}
	}
	d.SetStatus(components.StatusWaiting, nil)
	return nil
}

func (d *DanmuComponent) Restart(ctx context.Context) error {
	d.SetStatus(components.StatusRestarting, nil)
	_ = d.Stop(ctx)
	return d.Start(ctx)
}

func (d *DanmuComponent) LastSeq() int64 {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.lastSeq
}

func (d *DanmuComponent) Config() (remoteBaseURL string, appID int64, anchorCode string) {
	params := d.Params()
	remoteBaseURL = params[ParamRemoteBaseURL]
	if remoteBaseURL == "" {
		remoteBaseURL = defaultRemoteDanmuServer
	}
	appID, _ = strconv.ParseInt(params[ParamAppID], 10, 64)
	return remoteBaseURL, appID, params[ParamAnchorCode]
}

func (d *DanmuComponent) pullLoop(ctx context.Context, client RemoteDanmuClient, anchorCode string) {
	for {
		afterSeq := d.LastSeq()
		resp, err := client.PullDanmu(ctx, anchorCode, afterSeq, defaultPollLimit, defaultPollWaitMS)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			d.RecordEvent(components.ComponentEventDanmuListenerFailed, err.Error())
			d.SetStatus(components.StatusError, err)
			if !sleepWithContext(ctx, time.Second) {
				return
			}
			continue
		}
		if resp == nil {
			continue
		}
		if err := d.handleEvents(ctx, resp.Events); err != nil {
			if ctx.Err() != nil {
				return
			}
			d.RecordEvent(components.ComponentEventDanmuListenerFailed, err.Error())
			d.SetStatus(components.StatusError, err)
			return
		}
		d.setLastSeq(resp.NextSeq)
		if resp.HasMore {
			continue
		}
	}
}

func (d *DanmuComponent) handleEvents(ctx context.Context, events []bilibili.DanmuEvent) error {
	for _, event := range events {
		d.RecordEvent(components.ComponentEventDanmuDanmuReceived, fmt.Sprintf("caller=%s message=%s", event.Caller, event.Message))
		if d.messageSink != nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case d.messageSink <- formatDanmuMessage(event.Caller, event.Message):
			}
		}
		if event.Task == nil {
			continue
		}
		if d.taskSink == nil {
			continue
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case d.taskSink <- event.Task:
			d.RecordEvent(components.ComponentEventDanmuTaskDelivered, describeTaskEvent(event.Task))
		}
	}
	return nil
}

func (d *DanmuComponent) setLastSeq(lastSeq int64) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.lastSeq = lastSeq
}

func sleepWithContext(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func formatDanmuMessage(caller, message string) string {
	return "inf " + caller + " " + message
}

func describeTaskEvent(task *model.Task) string {
	return fmt.Sprintf("command=%s caller=%s content=%s index=%d", task.Command, task.Caller, task.Content, task.Index)
}
