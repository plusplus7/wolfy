package components

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

const maxComponentEventSize = 100

type ComponentEventType string

const (
	ComponentEventStatusChanged ComponentEventType = "component.status_changed"
	ComponentEventParamsUpdated ComponentEventType = "component.params_updated"

	ComponentEventSongsStorageLoaded     ComponentEventType = "songs.storage_loaded"
	ComponentEventSongsStorageLoadFailed ComponentEventType = "songs.storage_load_failed"

	ComponentEventTicketsTaskReceived  ComponentEventType = "tickets.task_received"
	ComponentEventTicketsTaskCompleted ComponentEventType = "tickets.task_completed"
	ComponentEventTicketsTaskFailed    ComponentEventType = "tickets.task_failed"

	ComponentEventMessagesMessageReceived ComponentEventType = "messages.message_received"
	ComponentEventMessagesMessageStored   ComponentEventType = "messages.message_stored"

	ComponentEventDanmuListenerStarted ComponentEventType = "danmu.listener_started"
	ComponentEventDanmuListenerFailed  ComponentEventType = "danmu.listener_failed"
	ComponentEventDanmuDanmuReceived   ComponentEventType = "danmu.danmu_received"
	ComponentEventDanmuTaskDelivered   ComponentEventType = "danmu.task_delivered"
)

type ComponentEvent struct {
	Time         string             `json:"time"`
	Component    string             `json:"component"`
	Type         ComponentEventType `json:"type"`
	CodeLocation string             `json:"code_location"`
	Message      string             `json:"message"`
}

type ComponentEventTypeInfo struct {
	Type        ComponentEventType `json:"type"`
	Description string             `json:"description"`
}

var componentEventCatalog = []ComponentEventTypeInfo{
	{Type: ComponentEventStatusChanged, Description: "component status changed"},
	{Type: ComponentEventParamsUpdated, Description: "component params were updated"},
	{Type: ComponentEventSongsStorageLoaded, Description: "songs storage loaded successfully"},
	{Type: ComponentEventSongsStorageLoadFailed, Description: "songs storage failed to load"},
	{Type: ComponentEventTicketsTaskReceived, Description: "tickets received a task"},
	{Type: ComponentEventTicketsTaskCompleted, Description: "tickets completed a task"},
	{Type: ComponentEventTicketsTaskFailed, Description: "tickets failed to handle a task"},
	{Type: ComponentEventMessagesMessageReceived, Description: "messages received a message"},
	{Type: ComponentEventMessagesMessageStored, Description: "messages stored a message"},
	{Type: ComponentEventDanmuListenerStarted, Description: "open-platform danmu listener started"},
	{Type: ComponentEventDanmuListenerFailed, Description: "open-platform danmu listener failed"},
	{Type: ComponentEventDanmuDanmuReceived, Description: "open-platform danmu received a danmu"},
	{Type: ComponentEventDanmuTaskDelivered, Description: "open-platform danmu delivered a task to tickets"},
}

var (
	repoRootOnce sync.Once
	repoRoot     string
)

func ComponentEventTypes() []ComponentEventTypeInfo {
	out := make([]ComponentEventTypeInfo, len(componentEventCatalog))
	copy(out, componentEventCatalog)
	return out
}

func CallerLocation(skip int) string {
	_, file, line, ok := runtime.Caller(skip + 1)
	if !ok {
		return "unknown"
	}
	if root := repoRootPath(); root != "" {
		if rel, err := filepath.Rel(root, file); err == nil && !strings.HasPrefix(rel, "..") {
			file = rel
		}
	}
	return fmt.Sprintf("%s:%d", filepath.ToSlash(file), line)
}

func (b *BaseComponent) RecordEvent(eventType ComponentEventType, message string) {
	b.RecordEventAt(eventType, CallerLocation(1), message)
}

func (b *BaseComponent) RecordEventAt(eventType ComponentEventType, codeLocation, message string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.appendEventLocked(eventType, codeLocation, message)
}

func (b *BaseComponent) appendEventLocked(eventType ComponentEventType, codeLocation, message string) {
	event := ComponentEvent{
		Time:         time.Now().Format(time.RFC3339Nano),
		Component:    b.name,
		Type:         eventType,
		CodeLocation: codeLocation,
		Message:      message,
	}
	if len(b.events) >= maxComponentEventSize {
		b.events = append(b.events[1:], event)
		return
	}
	b.events = append(b.events, event)
}

func changedParamKeys(current, next map[string]string) []string {
	keys := make([]string, 0, len(current))
	for key, value := range current {
		if value != next[key] {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

func describeParamUpdate(keys []string) string {
	return "updated params: " + strings.Join(keys, ",")
}

func describeStatus(status Status, errText string) string {
	if errText == "" {
		return "status=" + string(status)
	}
	return "status=" + string(status) + " error=" + errText
}

func cloneEvents(in []ComponentEvent) []ComponentEvent {
	out := make([]ComponentEvent, len(in))
	copy(out, in)
	return out
}

func repoRootPath() string {
	repoRootOnce.Do(func() {
		dir, err := os.Getwd()
		if err != nil {
			return
		}
		for {
			if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
				repoRoot = dir
				return
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				repoRoot = dir
				return
			}
			dir = parent
		}
	})
	return repoRoot
}
