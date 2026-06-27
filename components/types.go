package components

import (
	"context"
	"errors"
	"sync"
)

type Status string

const (
	StatusWaiting    Status = "waiting"
	StatusRunning    Status = "running"
	StatusError      Status = "error"
	StatusRestarting Status = "restarting"
)

type ComponentSnapshot struct {
	Name   string            `json:"name"`
	Status Status            `json:"status"`
	Error  string            `json:"error"`
	Params map[string]string `json:"params"`
	Events []ComponentEvent  `json:"events"`
}

type Component interface {
	Name() string
	ParamKeys() []string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Restart(ctx context.Context) error
	UpdateParams(params map[string]string)
	Snapshot() ComponentSnapshot
}

type BaseComponent struct {
	mu       sync.RWMutex
	name     string
	status   Status
	lastErr  string
	params   map[string]string
	paramSet map[string]struct{}
	events   []ComponentEvent
}

func NewBaseComponent(name string, keys []string) *BaseComponent {
	params := make(map[string]string, len(keys))
	paramSet := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		params[key] = ""
		paramSet[key] = struct{}{}
	}
	return &BaseComponent{
		name:     name,
		status:   StatusWaiting,
		params:   params,
		paramSet: paramSet,
	}
}

func (b *BaseComponent) Name() string {
	return b.name
}

func (b *BaseComponent) ParamKeys() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	keys := make([]string, 0, len(b.params))
	for key := range b.params {
		keys = append(keys, key)
	}
	return keys
}

func (b *BaseComponent) UpdateParams(params map[string]string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	changed := changedParamKeys(b.params, params)
	for key := range b.params {
		b.params[key] = params[key]
	}
	if len(changed) > 0 {
		b.appendEventLocked(ComponentEventParamsUpdated, CallerLocation(1), describeParamUpdate(changed))
	}
}

func (b *BaseComponent) Params() map[string]string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return cloneMap(b.params)
}

func (b *BaseComponent) SetStatus(status Status, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	oldStatus := b.status
	oldErr := b.lastErr
	b.status = status
	if err != nil {
		b.lastErr = err.Error()
	} else {
		b.lastErr = ""
	}
	if b.status != oldStatus || b.lastErr != oldErr {
		b.appendEventLocked(ComponentEventStatusChanged, CallerLocation(1), describeStatus(b.status, b.lastErr))
	}
}

func (b *BaseComponent) Snapshot() ComponentSnapshot {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return ComponentSnapshot{
		Name:   b.name,
		Status: b.status,
		Error:  b.lastErr,
		Params: cloneMap(b.params),
		Events: cloneEvents(b.events),
	}
}

func (b *BaseComponent) ValidateParams(params map[string]string) error {
	for key := range params {
		if _, ok := b.paramSet[key]; !ok {
			return errors.New("unknown component parameter: " + key)
		}
	}
	return nil
}

func cloneMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
