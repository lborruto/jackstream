package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lborruto/jackstream/internal/config"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	s := New(nil) // bt not exercised by these tests
	return httptest.NewServer(s.BuildHandler())
}

func doGet(t *testing.T, url string) (int, map[string]any) {
	t.Helper()
	res, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	var m map[string]any
	_ = json.Unmarshal(body, &m)
	return res.StatusCode, m
}

func TestManifestRoot(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()
	code, body := doGet(t, srv.URL+"/manifest.json")
	if code != 200 {
		t.Fatalf("status: %d", code)
	}
	if body["id"] != "community.jackstream" {
		t.Errorf("id: %v", body["id"])
	}
	h, _ := body["behaviorHints"].(map[string]any)
	if h["configurationRequired"] != true {
		t.Errorf("configurationRequired should be true: %v", h)
	}
}

func TestManifestConfiguredValid(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()
	cfg, _ := config.Encode(config.Config{
		JackettURL:     "http://127.0.0.1:9117",
		JackettAPIKey:  "x",
		TMDBAPIKey:     "y",
		AddonPublicURL: "http://127.0.0.1:7000",
	})
	code, body := doGet(t, srv.URL+"/"+cfg+"/manifest.json")
	if code != 200 {
		t.Fatalf("status: %d", code)
	}
	h, _ := body["behaviorHints"].(map[string]any)
	if h["configurationRequired"] != false {
		t.Errorf("configurationRequired should be false: %v", h)
	}
}

func TestManifestConfiguredBad(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()
	code, body := doGet(t, srv.URL+"/not-a-config/manifest.json")
	if code != 400 {
		t.Fatalf("status: %d", code)
	}
	if s, _ := body["error"].(string); !strings.Contains(s, "invalid_config") {
		t.Errorf("error body: %v", body)
	}
}

func TestHealth(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()
	code, body := doGet(t, srv.URL+"/health")
	if code != 200 || body["ok"] != true {
		t.Errorf("unexpected: %d %v", code, body)
	}
}
