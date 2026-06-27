package blivedm

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"wolfy/components"
	"wolfy/components/danmaku"
	servercomponent "wolfy/components/server"
	"wolfy/model"

	blivedmclient "github.com/Akegarasu/blivedm-go/client"
	"github.com/Akegarasu/blivedm-go/message"
)

const (
	BlivedmComponentName = "blivedm"
	ParamRoomID          = "room_id"
	ParamCookie          = "cookie"
)

type dmClient interface {
	SetCookie(cookie string)
	OnDanmaku(func(*message.Danmaku))
	Start() error
	Stop()
}

type clientFactory func(roomID int) dmClient

type BlivedmComponent struct {
	*components.BaseComponent
	taskSink    chan<- *model.Task
	messageSink chan<- string
	cancel      context.CancelFunc
	client      dmClient
	source      func() string
	factory     clientFactory
	mu          sync.Mutex
}

func NewBlivedmComponent(taskSink chan<- *model.Task, messageSink chan<- string) *BlivedmComponent {
	return &BlivedmComponent{
		BaseComponent: components.NewBaseComponent(BlivedmComponentName, []string{
			ParamRoomID,
			ParamCookie,
		}),
		taskSink:    taskSink,
		messageSink: messageSink,
		factory: func(roomID int) dmClient {
			return blivedmclient.NewClient(roomID)
		},
	}
}

func (b *BlivedmComponent) SetDanmuSourceFunc(source func() string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.source = source
}

func (b *BlivedmComponent) Start(ctx context.Context) error {
	if !b.active() {
		_ = b.Stop(ctx)
		b.SetStatus(components.StatusWaiting, errors.New("danmu source is not blivedm"))
		return nil
	}
	params := b.Params()
	roomIDText := params[ParamRoomID]
	if roomIDText == "" {
		err := errors.New("room_id is empty")
		b.SetStatus(components.StatusWaiting, err)
		return nil
	}
	roomID, err := strconv.Atoi(roomIDText)
	if err != nil {
		b.SetStatus(components.StatusError, err)
		return err
	}
	if roomID <= 0 {
		err := errors.New("room_id must be greater than 0")
		b.SetStatus(components.StatusError, err)
		return err
	}

	b.mu.Lock()
	if b.cancel != nil {
		b.cancel()
	}
	if b.client != nil {
		b.client.Stop()
	}
	runCtx, cancel := context.WithCancel(ctx)
	b.cancel = cancel
	factory := b.factory
	b.mu.Unlock()

	client := factory(roomID)
	if cookie := params[ParamCookie]; cookie != "" {
		client.SetCookie(cookie)
	}
	client.OnDanmaku(func(msg *message.Danmaku) {
		if msg == nil || msg.Sender == nil {
			return
		}
		eventMessage := fmt.Sprintf("caller=%s message=%s", msg.Sender.Uname, msg.Content)
		b.RecordEvent(components.ComponentEventBlivedmDanmuReceived, eventMessage)
		if b.messageSink != nil {
			select {
			case <-runCtx.Done():
				return
			case b.messageSink <- formatDanmuMessage(msg.Sender.Uname, msg.Content):
			}
		}
		task := danmaku.ParseTask(msg.Sender.Uname, msg.Content)
		if task == nil {
			return
		}
		select {
		case <-runCtx.Done():
			return
		case b.taskSink <- task:
			b.RecordEvent(components.ComponentEventBlivedmTaskDelivered, describeTaskEvent(task))
		}
	})

	b.SetStatus(components.StatusRestarting, nil)
	if err := client.Start(); err != nil {
		cancel()
		b.RecordEvent(components.ComponentEventBlivedmClientFailed, "room_id="+roomIDText+" error="+err.Error())
		b.SetStatus(components.StatusError, err)
		return err
	}

	b.mu.Lock()
	b.client = client
	b.mu.Unlock()
	b.RecordEvent(components.ComponentEventBlivedmClientStarted, "room_id="+roomIDText)
	b.SetStatus(components.StatusRunning, nil)
	return nil
}

func (b *BlivedmComponent) Stop(ctx context.Context) error {
	b.mu.Lock()
	if b.cancel != nil {
		b.cancel()
		b.cancel = nil
	}
	if b.client != nil {
		b.client.Stop()
		b.client = nil
	}
	b.mu.Unlock()
	b.SetStatus(components.StatusWaiting, nil)
	return nil
}

func (b *BlivedmComponent) Restart(ctx context.Context) error {
	b.SetStatus(components.StatusRestarting, nil)
	_ = b.Stop(ctx)
	return b.Start(ctx)
}

func (b *BlivedmComponent) active() bool {
	b.mu.Lock()
	source := b.source
	b.mu.Unlock()
	if source == nil {
		return false
	}
	return source() == servercomponent.DanmuSourceBlivedm
}

func formatDanmuMessage(caller, message string) string {
	return "inf " + caller + " " + message
}

func describeTaskEvent(task *model.Task) string {
	return fmt.Sprintf("command=%s caller=%s content=%s index=%d", task.Command, task.Caller, task.Content, task.Index)
}
