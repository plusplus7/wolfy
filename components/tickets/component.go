package tickets

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"sync"
	"wolfy/components"
	"wolfy/components/songs"
	"wolfy/model"
)

const (
	TicketsComponentName = "tickets"

	ticketsCheckpointPath = "./runtime/tickets.checkpoint.json"
	maxTicketSize         = 12
)

type TicketsComponent struct {
	*components.BaseComponent
	songs        *songs.SongsComponent
	ticketMaster model.ITicketMaster
	taskChan     chan *model.Task
	messageSink  chan<- string
	stopChan     chan struct{}
	mu           sync.RWMutex
}

func NewTicketsComponent(songs *songs.SongsComponent, messageSink chan<- string) *TicketsComponent {
	return &TicketsComponent{
		BaseComponent: components.NewBaseComponent(TicketsComponentName, nil),
		songs:         songs,
		taskChan:      make(chan *model.Task, 128),
		messageSink:   messageSink,
	}
}

func (t *TicketsComponent) Start(ctx context.Context) error {
	storage := t.songs.Storage()
	if storage == nil {
		err := errors.New("songs component is not running")
		t.SetStatus(components.StatusWaiting, err)
		return nil
	}

	t.mu.Lock()
	t.ticketMaster = NewMaimaiTicketMasterWithStorage(storage, ticketsCheckpointPath, maxTicketSize)
	if t.stopChan != nil {
		close(t.stopChan)
	}
	t.stopChan = make(chan struct{})
	stopChan := t.stopChan
	t.mu.Unlock()

	go t.taskRoutine(stopChan)
	t.SetStatus(components.StatusRunning, nil)
	return nil
}

func (t *TicketsComponent) Stop(ctx context.Context) error {
	t.mu.Lock()
	if t.stopChan != nil {
		close(t.stopChan)
		t.stopChan = nil
	}
	t.mu.Unlock()
	t.SetStatus(components.StatusWaiting, nil)
	return nil
}

func (t *TicketsComponent) Restart(ctx context.Context) error {
	t.SetStatus(components.StatusRestarting, nil)
	_ = t.Stop(ctx)
	return t.Start(ctx)
}

func (t *TicketsComponent) TaskChan() chan<- *model.Task {
	return t.taskChan
}

func (t *TicketsComponent) HandleTask(task *model.Task) (string, error) {
	if task == nil {
		err := errors.New("task is nil")
		t.RecordEvent(components.ComponentEventTicketsTaskFailed, err.Error())
		return "", err
	}

	t.mu.RLock()
	ticketMaster := t.ticketMaster
	messageSink := t.messageSink
	t.mu.RUnlock()

	if ticketMaster == nil {
		err := errors.New("tickets component is not running")
		t.RecordEvent(components.ComponentEventTicketsTaskFailed, describeTaskEvent(task)+" error="+err.Error())
		return "", err
	}

	var msg string
	var err error
	cmd := task.Command
	caller := task.Caller
	content := task.Content
	index := task.Index
	taskMessage := describeTaskEvent(task)
	t.RecordEvent(components.ComponentEventTicketsTaskReceived, taskMessage)

	if cmd == model.CommandPick {
		msg, err = ticketMaster.AddTicket(caller, content)
	} else {
		switch cmd {
		case model.CommandFinish:
			msg, err = ticketMaster.FinishTicket(caller, index)
		case model.CommandNextRank:
			msg, err = ticketMaster.NextRank(caller, index)
		case model.CommandNextLevel:
			msg, err = ticketMaster.NextLevel(caller, index)
		}
	}
	if err != nil {
		pushMessage(messageSink, "err "+caller+" "+err.Error())
		t.RecordEvent(components.ComponentEventTicketsTaskFailed, taskMessage+" error="+err.Error())
	} else {
		pushMessage(messageSink, "inf "+caller+" "+msg)
		t.RecordEvent(components.ComponentEventTicketsTaskCompleted, taskMessage+" result="+msg)
	}
	return msg, err
}

func (t *TicketsComponent) ForEachTicket(fn func(model.ITicket)) error {
	t.mu.RLock()
	ticketMaster := t.ticketMaster
	t.mu.RUnlock()
	if ticketMaster == nil {
		return errors.New("tickets component is not running")
	}
	ticketMaster.ForEachTicket(fn)
	return nil
}

func (t *TicketsComponent) taskRoutine(stopChan <-chan struct{}) {
	for {
		select {
		case <-stopChan:
			return
		case task := <-t.taskChan:
			if task == nil {
				continue
			}
			msg, err := t.HandleTask(task)
			if err != nil {
				log.Printf("task failed: %v", err)
			} else {
				log.Println(msg)
			}
		}
	}
}

func pushMessage(messageSink chan<- string, message string) {
	if messageSink == nil {
		return
	}
	messageSink <- message
}

func TaskFromFrontend(caller, event, content string) *model.Task {
	index, err := strconv.ParseInt(content, 10, 64)
	if err != nil {
		index = -1
	}
	return &model.Task{
		Command: FrontendEventCommand(event),
		Caller:  caller,
		Content: content,
		Index:   index,
	}
}

const (
	FrontendEventClickCoverInfo = "click_cover_info"
	FrontendEventClickGenreInfo = "click_genre_info"
	FrontendEventClickSongInfo  = "click_song_info"
	FrontendEventClickCreator   = "click_creator"
	FrontendEventPick           = "pick"
)

func FrontendEventCommand(event string) string {
	switch event {
	case FrontendEventClickCoverInfo:
		return model.CommandFinish
	case FrontendEventClickGenreInfo:
		return model.CommandNextRank
	case FrontendEventClickSongInfo:
		return model.CommandNextLevel
	case FrontendEventPick, FrontendEventClickCreator:
		return model.CommandPick
	default:
		return ""
	}
}

func describeTaskEvent(task *model.Task) string {
	return fmt.Sprintf("command=%s caller=%s content=%s index=%d", task.Command, task.Caller, task.Content, task.Index)
}
