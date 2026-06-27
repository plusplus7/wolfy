package bilibili

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type testSignatory struct{}

func (testSignatory) Sign(reqJson string) (*CommonHeader, error) {
	return &CommonHeader{
		ContentType:       JsonType,
		ContentAcceptType: JsonType,
	}, nil
}

func TestAPIRequestUsesConfiguredHTTPClientAndRejectsNon2xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			w.Header().Set("Content-Type", JsonType)
			if err := json.NewEncoder(w).Encode(BaseResp{Code: 0, Message: "ok", Data: json.RawMessage(`{}`)}); err != nil {
				t.Fatal(err)
			}
		case "/fail":
			http.Error(w, "bad gateway", http.StatusBadGateway)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	oldHost := openPlatformHTTPHost
	oldClient := openPlatformHTTPClient
	openPlatformHTTPHost = server.URL
	openPlatformHTTPClient = server.Client()
	t.Cleanup(func() {
		openPlatformHTTPHost = oldHost
		openPlatformHTTPClient = oldClient
	})

	app := NewAppService(1, "anchor", testSignatory{}, nil)
	resp, err := app.apiRequest(context.Background(), `{}`, "/ok")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Message != "ok" {
		t.Fatalf("unexpected response %#v", resp)
	}
	if _, err := app.apiRequest(context.Background(), `{}`, "/fail"); err == nil {
		t.Fatal("expected non-2xx response to fail")
	}
}
