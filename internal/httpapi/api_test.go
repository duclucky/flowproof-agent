package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ducky/flowproof-agent/internal/config"
	"github.com/ducky/flowproof-agent/internal/model"
	"github.com/ducky/flowproof-agent/internal/orchestrator"
	"github.com/ducky/flowproof-agent/internal/store"
)

const apiStableSelector = `[data-testid="checkout-submit"]`

type apiFakeDriver struct{}

func (apiFakeDriver) RunInitial(_ context.Context, tc model.TestDefinition) (model.BrowserObservation, error) {
	return model.BrowserObservation{
			CurrentURL: tc.TargetURL, VisibleText: "Checkout ready", Screenshot: []byte("failure-png"),
			AttemptedSelector: "#checkout-submit", FallbackSelector: apiStableSelector,
		}, &model.RecoverableStepError{
			Step: "submit_checkout", Selector: "#checkout-submit", FallbackSelector: apiStableSelector, Message: "selector not found",
		}
}

func (apiFakeDriver) Retry(_ context.Context, tc model.TestDefinition, selector string) (model.BrowserObservation, error) {
	if selector != apiStableSelector {
		return model.BrowserObservation{}, errors.New("unexpected selector")
	}
	return model.BrowserObservation{CurrentURL: tc.TargetURL, VisibleText: "Order confirmed FP-2048", Screenshot: []byte("success-png")}, nil
}

func newAPIServer(t *testing.T) *httptest.Server {
	t.Helper()
	st, err := store.NewFileStore(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	svc := orchestrator.New(st, apiFakeDriver{})
	h := New(config.Config{RunTimeout: 5e9}, svc)
	server := httptest.NewServer(h)
	t.Cleanup(server.Close)
	return server
}

func TestHealthAndDemoTarget(t *testing.T) {
	server := newAPIServer(t)

	resp, err := http.Get(server.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("health status = %d", resp.StatusCode)
	}

	resp, err = http.Get(server.URL + "/demo-store")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	html := string(body)
	for _, want := range []string{`data-testid="add-to-cart"`, `name="coupon"`, `data-testid="apply-coupon"`, `data-testid="checkout-submit"`, "SAVE20"} {
		if !strings.Contains(html, want) {
			t.Fatalf("demo target missing %q", want)
		}
	}
	if strings.Contains(html, ` id="checkout-submit"`) {
		t.Fatal("demo target must not contain the intentionally stale selector")
	}
}

func TestFullHTTPRecoveryWorkflow(t *testing.T) {
	server := newAPIServer(t)

	testCase := postJSON[model.TestDefinition](t, server.URL+"/api/tests", map[string]any{
		"targetUrl": server.URL + "/demo-store",
		"objective": "Complete checkout and verify the order confirmation",
	}, http.StatusCreated)
	if testCase.ID == "" {
		t.Fatal("missing test id")
	}

	run := postJSON[model.Run](t, server.URL+"/api/tests/"+testCase.ID+"/runs", map[string]any{}, http.StatusCreated)
	if run.Status != model.RunFailedRecoverable || run.FailureAnalysis == nil {
		t.Fatalf("start run = %#v", run)
	}

	got := getJSON[model.Run](t, server.URL+"/api/runs/"+run.ID, http.StatusOK)
	if got.ID != run.ID || len(got.Events) < 3 || len(got.Evidence) == 0 {
		t.Fatalf("run status response missing timeline/evidence: %#v", got)
	}

	analysis := getJSON[model.FailureAnalysis](t, server.URL+"/api/runs/"+run.ID+"/failure", http.StatusOK)
	if analysis.FallbackSelector != apiStableSelector || !analysis.Recoverable {
		t.Fatalf("analysis = %#v", analysis)
	}

	run = postJSON[model.Run](t, server.URL+"/api/runs/"+run.ID+"/retry", map[string]any{}, http.StatusOK)
	if run.Status != model.RunSucceeded || run.RecoveredSelector != apiStableSelector {
		t.Fatalf("retry run = %#v", run)
	}

	exported := getJSON[struct {
		Code string `json:"code"`
	}](t, server.URL+"/api/runs/"+run.ID+"/export", http.StatusOK)
	if !strings.Contains(exported.Code, "@playwright/test") || !strings.Contains(exported.Code, apiStableSelector) {
		t.Fatalf("unexpected export: %s", exported.Code)
	}

	resp := doRequest(t, http.MethodPost, server.URL+"/api/runs/"+run.ID+"/retry", []byte(`{}`), "application/json")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("second retry status = %d, want 409", resp.StatusCode)
	}
}

func TestAPIRejectsMalformedOversizedAndUnsafeCreateRequests(t *testing.T) {
	server := newAPIServer(t)

	resp := doRequest(t, http.MethodPost, server.URL+"/api/tests", []byte(`{`), "application/json")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("malformed status = %d", resp.StatusCode)
	}
	assertErrorShape(t, resp)
	validThenGarbage := append(mustJSON(t, map[string]any{
		"targetUrl": server.URL + "/demo-store",
		"objective": "verify parser strictness",
	}), []byte(" trailing-garbage")...)
	resp = doRequest(t, http.MethodPost, server.URL+"/api/tests", validThenGarbage, "application/json")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("trailing garbage status = %d, want 400", resp.StatusCode)
	}
	assertErrorShape(t, resp)

	oversized := mustJSON(t, map[string]any{
		"targetUrl": server.URL + "/demo-store",
		"objective": strings.Repeat("x", 70*1024),
	})
	resp = doRequest(t, http.MethodPost, server.URL+"/api/tests", oversized, "application/json")
	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized status = %d, want 413", resp.StatusCode)
	}
	assertErrorShape(t, resp)

	resp = doRequest(t, http.MethodPost, server.URL+"/api/tests", mustJSON(t, map[string]any{
		"targetUrl": "http://169.254.169.254/latest/meta-data", "objective": "read metadata",
	}), "application/json")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("unsafe target status = %d, want 400", resp.StatusCode)
	}
	assertErrorShape(t, resp)
}

func TestAPINotFoundUsesJSONError(t *testing.T) {
	server := newAPIServer(t)
	resp, err := http.Get(server.URL + "/api/runs/missing")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
	assertErrorShape(t, resp)
}

func postJSON[T any](t *testing.T, rawURL string, payload any, wantStatus int) T {
	t.Helper()
	resp := doRequest(t, http.MethodPost, rawURL, mustJSON(t, payload), "application/json")
	defer resp.Body.Close()
	if resp.StatusCode != wantStatus {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("POST %s status=%d want=%d body=%s", rawURL, resp.StatusCode, wantStatus, body)
	}
	var out T
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out
}

func getJSON[T any](t *testing.T, rawURL string, wantStatus int) T {
	t.Helper()
	resp, err := http.Get(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != wantStatus {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("GET %s status=%d want=%d body=%s", rawURL, resp.StatusCode, wantStatus, body)
	}
	var out T
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out
}

func doRequest(t *testing.T, method, rawURL string, body []byte, contentType string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, rawURL, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	b, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func assertErrorShape(t *testing.T, resp *http.Response) {
	t.Helper()
	defer resp.Body.Close()
	var payload struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if payload.Error.Code == "" || payload.Error.Message == "" {
		t.Fatalf("invalid error payload: %#v", payload)
	}
}
