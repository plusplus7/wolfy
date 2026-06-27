package components

import (
	"context"
	"path/filepath"
	"testing"
)

type testComponent struct {
	*BaseComponent
	starts   int
	stops    int
	restarts int
}

func newTestComponent(name string, keys []string) *testComponent {
	return &testComponent{BaseComponent: NewBaseComponent(name, keys)}
}

func (t *testComponent) Start(ctx context.Context) error {
	t.starts++
	t.SetStatus(StatusRunning, nil)
	return nil
}

func (t *testComponent) Stop(ctx context.Context) error {
	t.stops++
	t.SetStatus(StatusWaiting, nil)
	return nil
}

func (t *testComponent) Restart(ctx context.Context) error {
	t.restarts++
	t.SetStatus(StatusRestarting, nil)
	return t.Start(ctx)
}

func TestManagerInitializesEmptyParams(t *testing.T) {
	manager, err := NewManager(filepath.Join(t.TempDir(), "params.json"))
	if err != nil {
		t.Fatal(err)
	}
	component := newTestComponent("demo", []string{"token", "path"})
	if err := manager.Register(component); err != nil {
		t.Fatal(err)
	}
	snapshot := component.Snapshot()
	if snapshot.Params["token"] != "" || snapshot.Params["path"] != "" {
		t.Fatalf("expected empty default params, got %#v", snapshot.Params)
	}
}

func TestManagerUpdateParamsDoesNotRestart(t *testing.T) {
	manager, err := NewManager(filepath.Join(t.TempDir(), "params.json"))
	if err != nil {
		t.Fatal(err)
	}
	component := newTestComponent("demo", []string{"token"})
	if err := manager.Register(component); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.UpdateParams("demo", map[string]string{"token": "abc"}); err != nil {
		t.Fatal(err)
	}
	if component.starts != 0 || component.restarts != 0 {
		t.Fatalf("expected update to avoid lifecycle calls, starts=%d restarts=%d", component.starts, component.restarts)
	}
	if got := component.Snapshot().Params["token"]; got != "abc" {
		t.Fatalf("unexpected param value %q", got)
	}
}

func TestManagerRestartTargetsOnlyOneComponent(t *testing.T) {
	manager, err := NewManager(filepath.Join(t.TempDir(), "params.json"))
	if err != nil {
		t.Fatal(err)
	}
	first := newTestComponent("first", nil)
	second := newTestComponent("second", nil)
	if err := manager.Register(first); err != nil {
		t.Fatal(err)
	}
	if err := manager.Register(second); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Restart(context.Background(), "second"); err != nil {
		t.Fatal(err)
	}
	if first.restarts != 0 || first.starts != 0 {
		t.Fatalf("first component should not restart, starts=%d restarts=%d", first.starts, first.restarts)
	}
	if second.restarts != 1 || second.starts != 1 {
		t.Fatalf("second component restart mismatch, starts=%d restarts=%d", second.starts, second.restarts)
	}
}

func TestManagerStopTargetsOnlyOneComponent(t *testing.T) {
	manager, err := NewManager(filepath.Join(t.TempDir(), "params.json"))
	if err != nil {
		t.Fatal(err)
	}
	first := newTestComponent("first", nil)
	second := newTestComponent("second", nil)
	if err := manager.Register(first); err != nil {
		t.Fatal(err)
	}
	if err := manager.Register(second); err != nil {
		t.Fatal(err)
	}
	snapshot, err := manager.Stop(context.Background(), "second")
	if err != nil {
		t.Fatal(err)
	}
	if first.stops != 0 {
		t.Fatalf("first component should not stop, stops=%d", first.stops)
	}
	if second.stops != 1 {
		t.Fatalf("expected second component to stop once, stops=%d", second.stops)
	}
	if snapshot.Name != "second" || snapshot.Status != StatusWaiting {
		t.Fatalf("unexpected stop snapshot %#v", snapshot)
	}
}

func TestManagerStopUnknownComponent(t *testing.T) {
	manager, err := NewManager(filepath.Join(t.TempDir(), "params.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Stop(context.Background(), "missing"); err == nil {
		t.Fatal("expected unknown component error")
	}
}
