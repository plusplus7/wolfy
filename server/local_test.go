package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"testing"
	"time"
	"wolfy/components"
	danmucomponent "wolfy/components/danmu"
	"wolfy/components/danmu/bilibili"
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

func TestRestartComponentEndpointRestartsTarget(t *testing.T) {
	manager, err := components.NewManager(filepath.Join(t.TempDir(), "params.json"))
	if err != nil {
		t.Fatal(err)
	}
	danmu := newLifecycleComponent(danmucomponent.DanmuComponentName)
	if err := manager.Register(danmu); err != nil {
		t.Fatal(err)
	}
	local := NewLocalServer(manager, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/components/danmu/restart", nil)
	rec := httptest.NewRecorder()

	local.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status %d: %s", rec.Code, rec.Body.String())
	}
	if danmu.starts != 1 {
		t.Fatalf("expected danmu to restart, starts=%d", danmu.starts)
	}
}

func TestStopComponentEndpointStopsTargetOnly(t *testing.T) {
	manager, err := components.NewManager(filepath.Join(t.TempDir(), "params.json"))
	if err != nil {
		t.Fatal(err)
	}
	danmu := newLifecycleComponent(danmucomponent.DanmuComponentName)
	if err := manager.Register(danmu); err != nil {
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

type localHTTPFakeRemoteClient struct {
	startCalls  int
	stopCalls   int
	pullStarted chan struct{}
}

func (f *localHTTPFakeRemoteClient) StartGame(ctx context.Context, req bilibili.StartGameRequest) (*bilibili.StartGameResponse, error) {
	f.startCalls++
	return &bilibili.StartGameResponse{Data: bilibili.GameSession{Status: "running", LastSeq: 9}}, nil
}

func (f *localHTTPFakeRemoteClient) GetGame(ctx context.Context, anchorCode string) (*bilibili.StartGameResponse, error) {
	return &bilibili.StartGameResponse{Data: bilibili.GameSession{Status: "running", LastSeq: 9}}, nil
}

func (f *localHTTPFakeRemoteClient) StopGame(ctx context.Context, anchorCode string, req bilibili.StopGameRequest) (*bilibili.StopGameResponse, error) {
	f.stopCalls++
	return &bilibili.StopGameResponse{Data: bilibili.GameSession{Status: "stopped"}}, nil
}

func (f *localHTTPFakeRemoteClient) PullDanmu(ctx context.Context, anchorCode string, afterSeq int64, limit int, waitMS int) (*bilibili.PullDanmuResponse, error) {
	select {
	case f.pullStarted <- struct{}{}:
	default:
	}
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestDanmuEndpointsManageRemoteBridge(t *testing.T) {
	manager, err := components.NewManager(filepath.Join(t.TempDir(), "params.json"))
	if err != nil {
		t.Fatal(err)
	}
	danmu := danmucomponent.NewDanmuComponent(make(chan *model.Task, 1), make(chan string, 1))
	fakeClient := &localHTTPFakeRemoteClient{pullStarted: make(chan struct{}, 1)}
	danmu.SetRemoteClientFactory(func(baseURL string) danmucomponent.RemoteDanmuClient { return fakeClient })
	if err := manager.Register(danmu); err != nil {
		t.Fatal(err)
	}
	local := NewLocalServer(manager, nil, nil, danmu)

	req := httptest.NewRequest(
		http.MethodPatch,
		"/api/danmu",
		bytes.NewBufferString(`{"config":{"remote_base_url":"http://remote.test","app_id":42,"anchor_code":"anchor"}}`),
	)
	rec := httptest.NewRecorder()
	local.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch status %d: %s", rec.Code, rec.Body.String())
	}
	var patchBody struct {
		Data LocalDanmuStatus `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &patchBody); err != nil {
		t.Fatal(err)
	}
	if patchBody.Data.Config.RemoteBaseURL != "http://remote.test" || patchBody.Data.Config.AppID != 42 || patchBody.Data.Config.AnchorCode != "anchor" {
		t.Fatalf("unexpected patch body %#v", patchBody)
	}

	rec = httptest.NewRecorder()
	local.router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/danmu/start", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("start status %d: %s", rec.Code, rec.Body.String())
	}
	if fakeClient.startCalls != 1 {
		t.Fatalf("expected start call, got %d", fakeClient.startCalls)
	}
	select {
	case <-fakeClient.pullStarted:
	case <-time.After(time.Second):
		t.Fatal("pull loop did not start")
	}

	rec = httptest.NewRecorder()
	local.router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/danmu", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("get status %d: %s", rec.Code, rec.Body.String())
	}
	var statusBody struct {
		Data LocalDanmuStatus `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &statusBody); err != nil {
		t.Fatal(err)
	}
	if statusBody.Data.Status != components.StatusRunning || statusBody.Data.LastSeq != 9 {
		t.Fatalf("unexpected status body %#v", statusBody)
	}

	rec = httptest.NewRecorder()
	local.router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/danmu/stop", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("stop status %d: %s", rec.Code, rec.Body.String())
	}
	if fakeClient.stopCalls != 1 {
		t.Fatalf("expected stop call, got %d", fakeClient.stopCalls)
	}
}

func TestLocalDanmuContractJSONTags(t *testing.T) {
	assertLocalJSONTag(t, reflect.TypeOf(RemoteDanmuConfig{}), "RemoteBaseURL", "remote_base_url")
	assertLocalJSONTag(t, reflect.TypeOf(RemoteDanmuConfig{}), "AppID", "app_id")
	assertLocalJSONTag(t, reflect.TypeOf(RemoteDanmuConfig{}), "AnchorCode", "anchor_code")
	assertLocalJSONTag(t, reflect.TypeOf(UpdateRemoteDanmuConfigRequest{}), "Config", "config")
	assertLocalJSONTag(t, reflect.TypeOf(LocalDanmuStatus{}), "LastSeq", "last_seq")
}

func assertLocalJSONTag(t *testing.T, typ reflect.Type, fieldName string, want string) {
	t.Helper()
	field, ok := typ.FieldByName(fieldName)
	if !ok {
		t.Fatalf("missing field %s on %s", fieldName, typ.Name())
	}
	if got := field.Tag.Get("json"); got != want {
		t.Fatalf("%s.%s json tag = %q, want %q", typ.Name(), fieldName, got, want)
	}
}
