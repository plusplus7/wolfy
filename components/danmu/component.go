package danmu

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"wolfy/components"
	"wolfy/components/danmu/bilibili"
	servercomponent "wolfy/components/server"
	"wolfy/model"
)

const (
	DanmuComponentName = "danmu"
	ParamAppID         = "app_id"
	ParamAnchorCode    = "anchor_code"
	ParamBilibiliAKID  = "bilibili_ak_id"
	ParamBilibiliAKSec = "bilibili_ak_secret"

	remoteSignatoryServer = "https://plusplus7.com:42376"
)

type DanmuComponent struct {
	*components.BaseComponent
	taskSink    chan<- *model.Task
	messageSink chan<- string
	cancel      context.CancelFunc
	source      func() string
	mu          sync.Mutex
}

func NewDanmuComponent(taskSink chan<- *model.Task, messageSink chan<- string) *DanmuComponent {
	return &DanmuComponent{
		BaseComponent: components.NewBaseComponent(DanmuComponentName, []string{
			ParamAppID,
			ParamAnchorCode,
			ParamBilibiliAKID,
			ParamBilibiliAKSec,
		}),
		taskSink:    taskSink,
		messageSink: messageSink,
	}
}

func (d *DanmuComponent) SetDanmuSourceFunc(source func() string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.source = source
}

func (d *DanmuComponent) Start(ctx context.Context) error {
	if !d.active() {
		_ = d.Stop(ctx)
		d.SetStatus(components.StatusWaiting, errors.New("danmu source is not danmu"))
		return nil
	}
	params := d.Params()
	appIDText := params[ParamAppID]
	anchorCode := params[ParamAnchorCode]
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
	appID, err := strconv.Atoi(appIDText)
	if err != nil {
		d.SetStatus(components.StatusError, err)
		return err
	}

	d.mu.Lock()
	if d.cancel != nil {
		d.cancel()
	}
	runCtx, cancel := context.WithCancel(ctx)
	d.cancel = cancel
	d.mu.Unlock()

	akID := params[ParamBilibiliAKID]
	akSecret := params[ParamBilibiliAKSec]
	var signatory bilibili.ISignatory
	if akID != "" && akSecret != "" {
		signatory = bilibili.NewLocalSignatory(akID, akSecret)
	} else {
		signatory = bilibili.NewRemoteSignatory(remoteSignatoryServer, anchorCode)
	}
	app := bilibili.NewAppService(int64(appID), anchorCode, signatory, d.messageSink)
	app.SetEventRecorder(func(eventType components.ComponentEventType, codeLocation, message string) {
		d.RecordEventAt(eventType, codeLocation, message)
	})
	d.SetStatus(components.StatusRestarting, nil)
	go d.forwardTasks(runCtx, app)
	return nil
}

func (d *DanmuComponent) active() bool {
	d.mu.Lock()
	source := d.source
	d.mu.Unlock()
	if source == nil {
		return true
	}
	return source() == servercomponent.DanmuSourceDanmu
}

func (d *DanmuComponent) Stop(ctx context.Context) error {
	d.mu.Lock()
	if d.cancel != nil {
		d.cancel()
		d.cancel = nil
	}
	d.mu.Unlock()
	d.SetStatus(components.StatusWaiting, nil)
	return nil
}

func (d *DanmuComponent) Restart(ctx context.Context) error {
	d.SetStatus(components.StatusRestarting, nil)
	_ = d.Stop(ctx)
	return d.Start(ctx)
}

func (d *DanmuComponent) forwardTasks(ctx context.Context, app *bilibili.AppService) {
	taskChan := app.Spin(ctx)
	if taskChan == nil {
		if ctx.Err() == nil {
			d.RecordEvent(components.ComponentEventDanmuListenerFailed, "danmu listener failed to start")
			d.SetStatus(components.StatusError, errors.New("danmu listener failed to start"))
		}
		return
	}
	d.RecordEvent(components.ComponentEventDanmuListenerStarted, "danmu listener started")
	d.SetStatus(components.StatusRunning, nil)
	for {
		select {
		case <-ctx.Done():
			return
		case task := <-taskChan:
			if task == nil {
				continue
			}
			select {
			case <-ctx.Done():
				return
			case d.taskSink <- task:
				d.RecordEvent(components.ComponentEventDanmuTaskDelivered, describeTaskEvent(task))
			}
		}
	}
}

func describeTaskEvent(task *model.Task) string {
	return fmt.Sprintf("command=%s caller=%s content=%s index=%d", task.Command, task.Caller, task.Content, task.Index)
}
