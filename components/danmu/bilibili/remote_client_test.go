package bilibili

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRemoteDanmuClientStartsGame(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/openapi/games" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get(ContentTypeHeader); got != JsonType {
			t.Fatalf("unexpected content type %q", got)
		}
		var req StartGameRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req.AppID != 123 || req.AnchorCode != "anchor" {
			t.Fatalf("unexpected start request %#v", req)
		}
		if err := json.NewEncoder(w).Encode(StartGameResponse{Data: GameSession{
			AnchorCode: "anchor",
			AppID:      123,
			GameID:     "game",
			Status:     "running",
			LastSeq:    7,
		}}); err != nil {
			t.Fatal(err)
		}
	}))
	defer server.Close()

	oldClient := remoteDanmuHTTPClient
	remoteDanmuHTTPClient = server.Client()
	t.Cleanup(func() {
		remoteDanmuHTTPClient = oldClient
	})

	resp, err := NewRemoteDanmuClient(server.URL).StartGame(t.Context(), StartGameRequest{AppID: 123, AnchorCode: "anchor"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Data.GameID != "game" || resp.Data.LastSeq != 7 {
		t.Fatalf("unexpected response %#v", resp)
	}
}

func TestRemoteDanmuClientRejectsNon2xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad gateway", http.StatusBadGateway)
	}))
	defer server.Close()

	oldClient := remoteDanmuHTTPClient
	remoteDanmuHTTPClient = server.Client()
	t.Cleanup(func() {
		remoteDanmuHTTPClient = oldClient
	})

	if _, err := NewRemoteDanmuClient(server.URL).GetGame(t.Context(), "anchor"); err == nil {
		t.Fatal("expected non-2xx response to fail")
	}
}
