package httpapi

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ducky/flowproof-agent/internal/config"
	"github.com/ducky/flowproof-agent/internal/orchestrator"
	"github.com/ducky/flowproof-agent/internal/store"
)

func TestStaticFrontendServingAndSPAFallback(t *testing.T) {
	webDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(webDir, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(webDir, "index.html"), []byte("<!doctype html><div>flowproof-dashboard</div>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(webDir, "assets", "app.js"), []byte("console.log('flowproof-asset')"), 0o644); err != nil {
		t.Fatal(err)
	}

	st, err := store.NewFileStore(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	svc := orchestrator.New(st, apiFakeDriver{})
	server := httptest.NewServer(New(config.Config{RunTimeout: 5e9, WebDir: webDir}, svc))
	t.Cleanup(server.Close)

	for _, path := range []string{"/", "/runs/demo"} {
		resp, err := http.Get(server.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "flowproof-dashboard") {
			t.Fatalf("GET %s status=%d body=%q, want SPA index", path, resp.StatusCode, body)
		}
	}

	resp, err := http.Get(server.URL + "/assets/app.js")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "flowproof-asset") {
		t.Fatalf("asset status=%d body=%q", resp.StatusCode, body)
	}

	resp, err = http.Get(server.URL + "/assets/missing.js")
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("missing asset status=%d, want 404", resp.StatusCode)
	}
	if strings.Contains(string(body), "flowproof-dashboard") {
		t.Fatal("missing asset must not fall back to index.html")
	}

	resp, err = http.Get(server.URL + "/api/runs/missing")
	if err != nil {
		t.Fatal(err)
	}
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("API missing status=%d, want 404", resp.StatusCode)
	}
	if !strings.Contains(resp.Header.Get("Content-Type"), "application/json") || !strings.Contains(string(body), `"code":"not_found"`) {
		t.Fatalf("API 404 lost JSON routing: content-type=%q body=%q", resp.Header.Get("Content-Type"), body)
	}
}
