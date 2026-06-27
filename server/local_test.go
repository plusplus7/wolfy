package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"wolfy/components"
	blivedmcomponent "wolfy/components/blivedm"
	danmucomponent "wolfy/components/danmu"
	servercomponent "wolfy/components/server"
	"wolfy/model"
)

type lifecycleComponent struct {
	*components.BaseComponent
	starts  int
	stops   int
	stopErr error
}

func newLifecycleComponent(name string) *lifecycleComponent {
	return &lifecycleComponent{BaseComponent: components.NewBaseComponent(name, nil)}
}

func (l *lifecycleComponent) Start(ctx context.Context) error {
	l.starts++
	l.SetStatus(components.StatusRunning, nil)
	return nil
}

func (l *lifecycleComponent) Stop(ctx context.Context) error {
	l.stops++
	if l.stopErr != nil {
		l.SetStatus(components.StatusError, l.stopErr)
		return l.stopErr
	}
	l.SetStatus(components.StatusWaiting, nil)
	return nil
}

func (l *lifecycleComponent) Restart(ctx context.Context) error {
	return l.Start(ctx)
}

func TestSysInfoListsComponents(t *testing.T) {
	manager, err := components.NewManager(filepath.Join(t.TempDir(), "params.json"))
	if err != nil {
		t.Fatal(err)
	}
	serverComponent := servercomponent.NewServerComponent()
	if err := manager.Register(serverComponent); err != nil {
		t.Fatal(err)
	}
	local := NewLocalServer(manager, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/sysinfo", nil)
	rec := httptest.NewRecorder()

	local.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status %d: %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Data []components.ComponentSnapshot `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Data) != 1 || body.Data[0].Name != "server" {
		t.Fatalf("unexpected sysinfo response %#v", body.Data)
	}
	if body.Data[0].Events == nil {
		t.Fatalf("expected events to be present in sysinfo response")
	}
}

func TestComponentEventTypesEndpoint(t *testing.T) {
	manager, err := components.NewManager(filepath.Join(t.TempDir(), "params.json"))
	if err != nil {
		t.Fatal(err)
	}
	local := NewLocalServer(manager, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/component-event-types", nil)
	rec := httptest.NewRecorder()

	local.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status %d: %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Data struct {
			Types []components.ComponentEventTypeInfo `json:"types"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Data.Types) == 0 {
		t.Fatalf("expected event types, got %#v", body.Data)
	}
	if body.Data.Types[0].Type == "" || body.Data.Types[0].Description == "" {
		t.Fatalf("expected event type and description, got %#v", body.Data.Types[0])
	}
}

func TestRestartDanmuComponentStopsPeer(t *testing.T) {
	manager, err := components.NewManager(filepath.Join(t.TempDir(), "params.json"))
	if err != nil {
		t.Fatal(err)
	}
	danmu := newLifecycleComponent(danmucomponent.DanmuComponentName)
	blivedm := newLifecycleComponent(blivedmcomponent.BlivedmComponentName)
	serverComponent := servercomponent.NewServerComponent()
	if err := manager.Register(serverComponent); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.UpdateParams("server", map[string]string{servercomponent.ParamDanmuSource: servercomponent.DanmuSourceBlivedm}); err != nil {
		t.Fatal(err)
	}
	if err := manager.Register(danmu); err != nil {
		t.Fatal(err)
	}
	if err := manager.Register(blivedm); err != nil {
		t.Fatal(err)
	}
	local := NewLocalServer(manager, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/components/blivedm/restart", nil)
	rec := httptest.NewRecorder()

	local.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status %d: %s", rec.Code, rec.Body.String())
	}
	if danmu.stops != 1 {
		t.Fatalf("expected danmu peer to stop, stops=%d", danmu.stops)
	}
	if blivedm.starts != 1 {
		t.Fatalf("expected blivedm to restart, starts=%d", blivedm.starts)
	}
}

func TestRestartInactiveDanmuComponentDoesNotStopActivePeer(t *testing.T) {
	manager, err := components.NewManager(filepath.Join(t.TempDir(), "params.json"))
	if err != nil {
		t.Fatal(err)
	}
	danmu := newLifecycleComponent(danmucomponent.DanmuComponentName)
	blivedm := newLifecycleComponent(blivedmcomponent.BlivedmComponentName)
	serverComponent := servercomponent.NewServerComponent()
	if err := manager.Register(serverComponent); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.UpdateParams("server", map[string]string{servercomponent.ParamDanmuSource: servercomponent.DanmuSourceBlivedm}); err != nil {
		t.Fatal(err)
	}
	if err := manager.Register(danmu); err != nil {
		t.Fatal(err)
	}
	if err := manager.Register(blivedm); err != nil {
		t.Fatal(err)
	}
	local := NewLocalServer(manager, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/components/danmu/restart", nil)
	rec := httptest.NewRecorder()

	local.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status %d: %s", rec.Code, rec.Body.String())
	}
	if blivedm.stops != 0 {
		t.Fatalf("inactive restart should not stop active peer, stops=%d", blivedm.stops)
	}
	if danmu.starts != 1 {
		t.Fatalf("expected danmu restart attempt, starts=%d", danmu.starts)
	}
}

func TestStopComponentEndpointStopsTargetOnly(t *testing.T) {
	manager, err := components.NewManager(filepath.Join(t.TempDir(), "params.json"))
	if err != nil {
		t.Fatal(err)
	}
	danmu := newLifecycleComponent(danmucomponent.DanmuComponentName)
	blivedm := newLifecycleComponent(blivedmcomponent.BlivedmComponentName)
	if err := manager.Register(danmu); err != nil {
		t.Fatal(err)
	}
	if err := manager.Register(blivedm); err != nil {
		t.Fatal(err)
	}
	local := NewLocalServer(manager, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/components/danmu/stop", nil)
	rec := httptest.NewRecorder()

	local.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status %d: %s", rec.Code, rec.Body.String())
	}
	if danmu.stops != 1 {
		t.Fatalf("expected danmu to stop once, stops=%d", danmu.stops)
	}
	if blivedm.stops != 0 {
		t.Fatalf("stop should not stop peer, blivedm stops=%d", blivedm.stops)
	}
	var body struct {
		Data components.ComponentSnapshot `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Data.Name != danmucomponent.DanmuComponentName || body.Data.Status != components.StatusWaiting {
		t.Fatalf("unexpected stop response %#v", body.Data)
	}
}

func TestStopServerComponentReturnsErrorWithoutStopping(t *testing.T) {
	manager, err := components.NewManager(filepath.Join(t.TempDir(), "params.json"))
	if err != nil {
		t.Fatal(err)
	}
	serverComponent := servercomponent.NewServerComponent()
	if err := manager.Register(serverComponent); err != nil {
		t.Fatal(err)
	}
	if err := serverComponent.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	local := NewLocalServer(manager, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/components/server/stop", nil)
	rec := httptest.NewRecorder()

	local.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status %d: %s", rec.Code, rec.Body.String())
	}
	if serverComponent.Snapshot().Status != components.StatusRunning {
		t.Fatalf("server component should remain running, got %#v", serverComponent.Snapshot())
	}
	var body struct {
		Msg  string                       `json:"msg"`
		Data components.ComponentSnapshot `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Msg != "server stop is not supported from the HTTP API" || body.Data.Name != "server" {
		t.Fatalf("unexpected stop server response %#v", body)
	}
}

func TestStopComponentEndpointReturnsError(t *testing.T) {
	manager, err := components.NewManager(filepath.Join(t.TempDir(), "params.json"))
	if err != nil {
		t.Fatal(err)
	}
	component := newLifecycleComponent("demo")
	component.stopErr = errors.New("stop failed")
	if err := manager.Register(component); err != nil {
		t.Fatal(err)
	}
	local := NewLocalServer(manager, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/components/demo/stop", nil)
	rec := httptest.NewRecorder()

	local.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status %d: %s", rec.Code, rec.Body.String())
	}
	if component.stops != 1 {
		t.Fatalf("expected stop attempt, stops=%d", component.stops)
	}
	var body struct {
		Msg  string                       `json:"msg"`
		Data components.ComponentSnapshot `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Msg != "stop failed" || body.Data.Status != components.StatusError {
		t.Fatalf("unexpected stop error response %#v", body)
	}
}

func TestStopUnknownComponentEndpointReturnsError(t *testing.T) {
	manager, err := components.NewManager(filepath.Join(t.TempDir(), "params.json"))
	if err != nil {
		t.Fatal(err)
	}
	local := NewLocalServer(manager, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/components/missing/stop", nil)
	rec := httptest.NewRecorder()

	local.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status %d: %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Msg string `json:"msg"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Msg != "unknown component: missing" {
		t.Fatalf("unexpected stop unknown response %#v", body)
	}
}

func TestUpdateComponentParamsEndpoint(t *testing.T) {
	manager, err := components.NewManager(filepath.Join(t.TempDir(), "params.json"))
	if err != nil {
		t.Fatal(err)
	}
	danmu := danmucomponent.NewDanmuComponent(make(chan *model.Task, 1), nil)
	if err := manager.Register(danmu); err != nil {
		t.Fatal(err)
	}
	local := NewLocalServer(manager, nil, nil)
	req := httptest.NewRequest(
		http.MethodPatch,
		"/api/components/danmu/params",
		bytes.NewBufferString(`{"params":{"app_id":"123"}}`),
	)
	rec := httptest.NewRecorder()

	local.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status %d: %s", rec.Code, rec.Body.String())
	}
	if got := danmu.Snapshot().Params[danmucomponent.ParamAppID]; got != "123" {
		t.Fatalf("expected app_id to update, got %q", got)
	}
	if danmu.Snapshot().Status != components.StatusWaiting {
		t.Fatalf("expected update not to start/restart danmu, got %s", danmu.Snapshot().Status)
	}
}

func TestTicketsEndpointsReturn503WhenTicketsComponentIsMissing(t *testing.T) {
	manager, err := components.NewManager(filepath.Join(t.TempDir(), "params.json"))
	if err != nil {
		t.Fatal(err)
	}
	local := NewLocalServer(manager, nil, nil)
	for _, path := range []string{"/api/messages", "/api/tickets"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()

		local.router.ServeHTTP(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("%s: expected 503, got %d: %s", path, rec.Code, rec.Body.String())
		}
	}
}

func TestLocalServerCORSRejectsUnexpectedOrigins(t *testing.T) {
	manager, err := components.NewManager(filepath.Join(t.TempDir(), "params.json"))
	if err != nil {
		t.Fatal(err)
	}
	local := NewLocalServer(manager, nil, nil)
	req := httptest.NewRequest(http.MethodOptions, "/api/sysinfo", nil)
	req.Header.Set("Origin", "https://evil.example")
	req.Header.Set("Access-Control-Request-Method", http.MethodGet)
	rec := httptest.NewRecorder()

	local.router.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("unexpected CORS origin %q", got)
	}
}

func TestLocalServerCORSAllowsLocalOrigins(t *testing.T) {
	manager, err := components.NewManager(filepath.Join(t.TempDir(), "params.json"))
	if err != nil {
		t.Fatal(err)
	}
	local := NewLocalServer(manager, nil, nil)
	req := httptest.NewRequest(http.MethodOptions, "/api/sysinfo", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", http.MethodGet)
	rec := httptest.NewRecorder()

	local.router.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("unexpected CORS origin %q", got)
	}
}
