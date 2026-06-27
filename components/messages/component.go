package messages

import (
	"context"
	"errors"
	"sync"
	"wolfy/components"
	"wolfy/model"
)

const (
	MessagesComponentName = "messages"

	messagesCheckpointPath = "./runtime/messages.checkpoint.json"
	maxMessageSize         = 20
)

type MessagesComponent struct {
	*components.BaseComponent
	manager     *model.MessageManager
	messageChan chan string
	stopChan    chan struct{}
	mu          sync.RWMutex
}

func NewMessagesComponent() *MessagesComponent {
	return &MessagesComponent{
		BaseComponent: components.NewBaseComponent(MessagesComponentName, nil),
		messageChan:   make(chan string, 128),
	}
}

func (m *MessagesComponent) Start(ctx context.Context) error {
	m.mu.Lock()
	m.manager = model.NewMessageManager(messagesCheckpointPath, maxMessageSize)
	if m.stopChan != nil {
		close(m.stopChan)
	}
	m.stopChan = make(chan struct{})
	stopChan := m.stopChan
	m.mu.Unlock()

	go m.messageRoutine(stopChan)
	m.SetStatus(components.StatusRunning, nil)
	return nil
}

func (m *MessagesComponent) Stop(ctx context.Context) error {
	m.mu.Lock()
	if m.stopChan != nil {
		close(m.stopChan)
		m.stopChan = nil
	}
	m.mu.Unlock()
	m.SetStatus(components.StatusWaiting, nil)
	return nil
}

func (m *MessagesComponent) Restart(ctx context.Context) error {
	m.SetStatus(components.StatusRestarting, nil)
	_ = m.Stop(ctx)
	return m.Start(ctx)
}

func (m *MessagesComponent) MessageChan() chan<- string {
	return m.messageChan
}

func (m *MessagesComponent) ForEachMessage(fn func(*model.Message)) error {
	m.mu.RLock()
	manager := m.manager
	m.mu.RUnlock()
	if manager == nil {
		return errors.New("messages component is not running")
	}
	manager.ForEachMessage(fn)
	return nil
}

func (m *MessagesComponent) messageRoutine(stopChan <-chan struct{}) {
	for {
		select {
		case <-stopChan:
			return
		case message := <-m.messageChan:
			m.RecordEvent(components.ComponentEventMessagesMessageReceived, "message="+message)
			m.mu.RLock()
			manager := m.manager
			m.mu.RUnlock()
			if manager == nil {
				continue
			}
			manager.Push(message)
			m.RecordEvent(components.ComponentEventMessagesMessageStored, "message="+message)
		}
	}
}
