package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/ducky/flowproof-agent/internal/config"
	"github.com/ducky/flowproof-agent/internal/model"
	"github.com/ducky/flowproof-agent/internal/orchestrator"
	"github.com/ducky/flowproof-agent/internal/store"
)

const maxJSONBodyBytes int64 = 64 << 10

type API struct {
	cfg config.Config
	svc *orchestrator.Service
}

func New(cfg config.Config, svc *orchestrator.Service) http.Handler {
	api := &API{cfg: cfg, svc: svc}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", api.health)
	mux.HandleFunc("GET /demo-store", demoStore)
	mux.HandleFunc("POST /api/tests", api.createTest)
	mux.HandleFunc("POST /api/tests/{testID}/runs", api.startRun)
	mux.HandleFunc("GET /api/runs/{runID}", api.getRun)
	mux.HandleFunc("GET /api/runs/{runID}/failure", api.inspectFailure)
	mux.HandleFunc("POST /api/runs/{runID}/retry", api.retryRun)
	mux.HandleFunc("GET /api/runs/{runID}/export", api.exportRun)
	mux.Handle("/", staticFrontend(cfg.WebDir))
	return apiSecurityHeaders(mux)
}

func (a *API) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *API) createTest(w http.ResponseWriter, r *http.Request) {
	var req model.CreateTestRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}
	if err := config.ValidateTarget(req.TargetURL, requestOrigin(r), a.cfg.AllowedHosts); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_target", err.Error())
		return
	}
	tc, err := a.svc.CreateTest(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_test", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, tc)
}

func (a *API) startRun(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := contextWithConfiguredTimeout(r, a.cfg.RunTimeout)
	defer cancel()
	run, err := a.svc.StartRun(ctx, r.PathValue("testID"))
	if err != nil {
		a.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, run)
}

func (a *API) getRun(w http.ResponseWriter, r *http.Request) {
	run, err := a.svc.GetRun(r.PathValue("runID"))
	if err != nil {
		a.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (a *API) inspectFailure(w http.ResponseWriter, r *http.Request) {
	analysis, err := a.svc.InspectFailure(r.PathValue("runID"))
	if err != nil {
		a.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, analysis)
}

func (a *API) retryRun(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := contextWithConfiguredTimeout(r, a.cfg.RunTimeout)
	defer cancel()
	run, err := a.svc.RetryFailedStep(ctx, r.PathValue("runID"))
	if err != nil {
		a.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (a *API) exportRun(w http.ResponseWriter, r *http.Request) {
	code, err := a.svc.ExportRegressionTest(r.PathValue("runID"))
	if err != nil {
		a.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"code": code})
}

func (a *API) writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound), errors.Is(err, orchestrator.ErrTestNotFound):
		writeError(w, http.StatusNotFound, "not_found", "resource not found")
	case errors.Is(err, orchestrator.ErrInvalidRunState):
		writeError(w, http.StatusConflict, "invalid_run_state", err.Error())
	case errors.Is(err, http.ErrHandlerTimeout):
		writeError(w, http.StatusGatewayTimeout, "timeout", "operation timed out")
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "operation failed")
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	var extra any
	err := decoder.Decode(&extra)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return fmt.Errorf("request body must contain one JSON object")
	}
	return fmt.Errorf("invalid trailing JSON content: %w", err)
}

func writeDecodeError(w http.ResponseWriter, err error) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		writeError(w, http.StatusRequestEntityTooLarge, "request_too_large", "JSON request body exceeds 64 KiB")
		return
	}
	writeError(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}

func requestOrigin(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwarded := strings.ToLower(strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Proto"), ",")[0])); forwarded == "https" {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

func contextWithConfiguredTimeout(r *http.Request, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		timeout = 45 * time.Second
	}
	return context.WithTimeout(r.Context(), timeout)
}

func staticFrontend(webDir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.NotFound(w, r)
			return
		}
		if strings.TrimSpace(webDir) == "" {
			http.NotFound(w, r)
			return
		}

		cleanPath := path.Clean("/" + r.URL.Path)
		rel := strings.TrimPrefix(cleanPath, "/")
		if rel == "" {
			rel = "index.html"
		}
		candidate := filepath.Join(webDir, filepath.FromSlash(rel))
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			http.ServeFile(w, r, candidate)
			return
		}

		if filepath.Ext(rel) != "" {
			http.NotFound(w, r)
			return
		}
		indexPath := filepath.Join(webDir, "index.html")
		if info, err := os.Stat(indexPath); err != nil || info.IsDir() {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, indexPath)
	})
}

func apiSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		next.ServeHTTP(w, r)
	})
}
