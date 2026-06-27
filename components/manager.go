package components

import (
	"context"
	"errors"
	"sync"
)

type Manager struct {
	mu         sync.RWMutex
	store      *ParamStore
	components map[string]Component
}

func NewManager(paramPath string) (*Manager, error) {
	store, err := NewParamStore(paramPath)
	if err != nil {
		return nil, err
	}
	return &Manager{
		store:      store,
		components: map[string]Component{},
	}, nil
}

func (m *Manager) Register(component Component) error {
	params, err := m.store.Ensure(component.Name(), component.ParamKeys())
	if err != nil {
		return err
	}
	component.UpdateParams(params)

	m.mu.Lock()
	defer m.mu.Unlock()
	m.components[component.Name()] = component
	return nil
}

func (m *Manager) StartAll(ctx context.Context) {
	for _, component := range m.componentsList() {
		_ = component.Start(ctx)
	}
}

func (m *Manager) UpdateParams(name string, params map[string]string) (ComponentSnapshot, error) {
	component, ok := m.Component(name)
	if !ok {
		return ComponentSnapshot{}, errors.New("unknown component: " + name)
	}
	if validator, ok := component.(interface {
		ValidateParams(map[string]string) error
	}); ok {
		if err := validator.ValidateParams(params); err != nil {
			return ComponentSnapshot{}, err
		}
	}
	merged, err := m.store.Update(name, params)
	if err != nil {
		return ComponentSnapshot{}, err
	}
	component.UpdateParams(merged)
	return component.Snapshot(), nil
}

func (m *Manager) Restart(ctx context.Context, name string) (ComponentSnapshot, error) {
	component, ok := m.Component(name)
	if !ok {
		return ComponentSnapshot{}, errors.New("unknown component: " + name)
	}
	if err := component.Restart(ctx); err != nil {
		return component.Snapshot(), err
	}
	return component.Snapshot(), nil
}

func (m *Manager) Stop(ctx context.Context, name string) (ComponentSnapshot, error) {
	component, ok := m.Component(name)
	if !ok {
		return ComponentSnapshot{}, errors.New("unknown component: " + name)
	}
	if err := component.Stop(ctx); err != nil {
		return component.Snapshot(), err
	}
	return component.Snapshot(), nil
}

func (m *Manager) Component(name string) (Component, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	component, ok := m.components[name]
	return component, ok
}

func (m *Manager) Snapshots() []ComponentSnapshot {
	components := m.componentsList()
	snapshots := make([]ComponentSnapshot, 0, len(components))
	for _, component := range components {
		snapshots = append(snapshots, component.Snapshot())
	}
	return snapshots
}

func (m *Manager) componentsList() []Component {
	m.mu.RLock()
	defer m.mu.RUnlock()
	components := make([]Component, 0, len(m.components))
	for _, component := range m.components {
		components = append(components, component)
	}
	return components
}
