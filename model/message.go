package model

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"sync"
	"wolfy/internal/fileutil"
)

type Message struct {
	Content string `json:"content"`
}

type MessageManager struct {
	messages       []*Message
	maxSize        int
	lock           *sync.Mutex
	checkPointPath string
}

func NewMessageManager(checkPointPath string, maxSize int) *MessageManager {
	m := &MessageManager{
		maxSize:        maxSize,
		lock:           &sync.Mutex{},
		checkPointPath: checkPointPath,
	}

	if err := m.loadCheckPoint(); err != nil {
		m.messages = make([]*Message, 0)
		if errors.Is(err, os.ErrNotExist) {
			if err := m.saveCheckPoint(); err != nil {
				log.Printf("failed to initialize message checkpoint: %v", err)
			}
		} else {
			log.Printf("failed to load message checkpoint: %v", err)
		}
	}
	return m
}

func (m *MessageManager) loadCheckPoint() error {
	if m.checkPointPath == "" {
		return nil
	}

	file, err := os.ReadFile(m.checkPointPath)
	if err != nil {
		return err
	}
	var messages []*Message
	err = json.Unmarshal(file, &messages)
	if err != nil {
		return err
	}
	if m.maxSize <= 0 {
		messages = nil
	} else if len(messages) > m.maxSize {
		messages = messages[:m.maxSize]
	}
	m.messages = messages
	return nil
}

func (m *MessageManager) saveCheckPoint() error {
	if m.checkPointPath == "" {
		return nil
	}

	result, err := json.Marshal(m.messages)
	if err != nil {
		return err
	}
	return fileutil.AtomicWriteFile(m.checkPointPath, result, 0644)
}

func (m *MessageManager) Push(message string) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if m.maxSize <= 0 {
		m.messages = nil
		if err := m.saveCheckPoint(); err != nil {
			log.Printf("failed to save check point %v", err)
		}
		return
	}

	if len(m.messages) >= m.maxSize {
		m.messages = m.messages[:len(m.messages)-1]
	}

	m.messages = append([]*Message{{
		Content: message,
	}}, m.messages...)

	err := m.saveCheckPoint()
	if err != nil {
		log.Printf("failed to save check point %v", err)
	}
}

func (m *MessageManager) ForEachMessage(fn func(message *Message)) {
	m.lock.Lock()
	messages := make([]*Message, 0, len(m.messages))
	for _, message := range m.messages {
		messages = append(messages, message)
	}
	m.lock.Unlock()

	for _, message := range messages {
		fn(message)
	}
}
