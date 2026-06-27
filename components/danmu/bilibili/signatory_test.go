package bilibili

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRemoteSignatorySignsThroughRemoteServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method %s", r.Method)
		}
		if got := r.Header.Get(ContentTypeHeader); got != JsonType {
			t.Fatalf("unexpected content type %q", got)
		}
		var req RemoteSignRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		if req.AnchorCode != "anchor" || req.ReqJson != `{}` {
			t.Fatalf("unexpected sign request %#v", req)
		}
		if err := json.NewEncoder(w).Encode(RemoteSignResponse{Header: CommonHeader{AccessKeyId: "key"}}); err != nil {
			t.Fatal(err)
		}
	}))
	defer server.Close()

	oldClient := remoteSignHTTPClient
	remoteSignHTTPClient = server.Client()
	t.Cleanup(func() {
		remoteSignHTTPClient = oldClient
	})

	header, err := NewRemoteSignatory(server.URL, "anchor").Sign(`{}`)
	if err != nil {
		t.Fatal(err)
	}
	if header.AccessKeyId != "key" {
		t.Fatalf("unexpected header %#v", header)
	}
}

func TestRemoteSignatoryRejectsNon2xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad gateway", http.StatusBadGateway)
	}))
	defer server.Close()

	oldClient := remoteSignHTTPClient
	remoteSignHTTPClient = server.Client()
	t.Cleanup(func() {
		remoteSignHTTPClient = oldClient
	})

	if _, err := NewRemoteSignatory(server.URL, "anchor").Sign(`{}`); err == nil {
		t.Fatal("expected non-2xx response to fail")
	}
}
