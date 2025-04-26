package model

import (
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"
)

type Message struct {
	Content    string `json:"content"`
	ExpireTime int64  `json:"expire_time"`
}

type MessageManager struct {
	messages       []*Message
	maxSize        int
	lifeTime       time.Duration
	lock           *sync.Mutex
	checkPointPath string
}

func NewMessageManager(checkPointPath string, maxSize int, leftTime time.Duration) *MessageManager {
	m := &MessageManager{
		maxSize:        maxSize,
		lock:           &sync.Mutex{},
		checkPointPath: checkPointPath,
		lifeTime:       leftTime,
	}

	if ok := m.loadCheckPoint(); ok != nil {
		m.messages = make([]*Message, 0)
		err := m.saveCheckPoint()
		if err != nil {
			panic(err)
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
	err = os.WriteFile(m.checkPointPath, result, 0644)
	if err != nil {
		return err
	}
	return nil
}

func (m *MessageManager) Push(message string) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if len(m.messages) >= m.maxSize {
		m.messages = m.messages[:len(m.messages)-1]
	}

	m.messages = append([]*Message{{
		Content:    message,
		ExpireTime: time.Now().Add(m.lifeTime).Unix(),
	}}, m.messages...)

	err := m.saveCheckPoint()
	if err != nil {
		log.Fatalf("failed to save check point %v", err)
	}
}

func (m *MessageManager) ForEachMessage(fn func(message *Message)) {
	m.lock.Lock()
	defer m.lock.Unlock()

	for _, message := range m.messages {
		if message.ExpireTime > time.Now().Unix() {
			fn(message)
		}
	}
}
