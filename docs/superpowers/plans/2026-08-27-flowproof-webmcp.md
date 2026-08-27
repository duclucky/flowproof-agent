# FlowProof WebMCP Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: use `executing-plans` and `test-driven-development` task-by-task. The user explicitly forbids commits for this run; do not execute any commit step.

**Goal:** Deliver a locally verified WebMCP Challenge submission where an external agent can create, run, diagnose, recover, and export a deterministic browser QA workflow through six structured WebMCP tools.

**Architecture:** Go owns durable domain state, target validation, browser execution, HTTP API, demo target, and static serving. React/TypeScript owns the operations dashboard and WebMCP registration. The contest path is deterministic and does not depend on Gemini or another external model.

**Tech Stack:** Go 1.26, `chromedp`, standard `net/http`, React, TypeScript, Vite, Vitest.

## Global constraints

- Use current `document.modelContext.registerTool(...)`, never deprecated `navigator.modelContext`.
- Required tools: `create_test`, `start_run`, `get_run_status`, `inspect_failure`, `retry_failed_step`, `export_regression_test`.
- No external AI/API key required for demo.
- No arbitrary browser-agent scope, payments, credentials, uploads/downloads, destructive actions, or unrestricted target URLs.
- No commit, push, publish, or external deploy.
- Preserve unrelated user work.
- Every production behavior change must follow failing-test-first TDD.

---

### Task 1: Domain model, store, and run-state orchestrator

**Files:**
- Modify: `internal/model/types.go`
- Modify: `internal/store/store.go`
- Create: `internal/orchestrator/orchestrator_test.go`
- Create: `internal/orchestrator/orchestrator.go`
- Create: `internal/browser/driver.go`

**Interfaces:**
- `browser.Driver` provides bounded deterministic actions needed by the demo and returns structured observations/evidence.
- `orchestrator.Service.CreateTest`, `StartRun`, `GetRun`, `InspectFailure`, `RetryFailedStep`, and `ExportRegressionTest` are the only application use-cases consumed by HTTP.
- Runs support `queued`, `running`, `failed_recoverable`, `succeeded`, and `failed` states.

- [ ] Write focused tests proving create/start state transitions, stale-selector recoverable failure, invalid retry rejection, successful safe retry, event/evidence ordering, and Playwright export selector.
- [ ] Run `go test ./internal/orchestrator -count=1`; verify RED is caused by missing interfaces/behavior.
- [ ] Implement the minimum domain types, fake-testable driver boundary, service, and persistence updates.
- [ ] Re-run focused tests until GREEN, then `go test ./internal/store ./internal/orchestrator -count=1`.

### Task 2: Chromedp driver and target safety

**Files:**
- Modify: `internal/browser/browser.go`
- Create: `internal/browser/browser_test.go`
- Modify: `internal/config/config.go`
- Create: `internal/config/config_test.go`

**Interfaces:**
- Production driver implements the Task 1 `browser.Driver`.
- Target policy exposes `ValidateTarget(rawURL, serverOrigin string) error` or equivalent and exact external-host allowlisting.

- [ ] Add failing tests for allowed built-in demo URL, rejected non-http schemes, credentials, private/link-local external targets, and exact host allowlist behavior.
- [ ] Add driver tests for selector/error normalization without requiring a live browser where possible.
- [ ] Run focused packages and confirm RED.
- [ ] Remove mandatory Gemini/Vertex configuration from the contest path; keep only server/browser/store/allowed-host/timeouts required by FlowProof.
- [ ] Implement chromedp driver actions/snapshot/screenshot and target guardrails.
- [ ] Run `go test ./internal/browser ./internal/config -count=1` until GREEN.

### Task 3: HTTP API and deterministic demo target

**Files:**
- Create: `internal/httpapi/api_test.go`
- Create: `internal/httpapi/api.go`
- Create: `internal/httpapi/demo.go`
- Create: `cmd/server/main.go`

**Interfaces:**
- Endpoints: `POST /api/tests`, `POST /api/tests/{testId}/runs`, `GET /api/runs/{runId}`, `GET /api/runs/{runId}/failure`, `POST /api/runs/{runId}/retry`, `GET /api/runs/{runId}/export`, `GET /healthz`, `GET /demo-store`.
- Errors: `{"error":{"code":string,"message":string}}`.

- [ ] Write httptest RED cases for valid flow, malformed JSON, oversized input, missing IDs, invalid state, and deterministic demo markup containing stable `[data-testid="checkout-submit"]` but not stale `#checkout-submit`.
- [ ] Run `go test ./internal/httpapi -count=1` and confirm expected failures.
- [ ] Implement handlers with body limits, method/path routing, bounded contexts, consistent JSON responses, and same-origin demo URL resolution.
- [ ] Wire `cmd/server` without external model credentials.
- [ ] Re-run focused tests and then `go test ./... -count=1`.

### Task 4: Frontend API and six WebMCP tools

**Files:**
- Create: `web/package.json`, `web/tsconfig.json`, `web/vite.config.ts`, `web/vitest.config.ts`, `web/index.html`
- Create: `web/src/types.ts`
- Create: `web/src/api.ts`
- Create: `web/src/webmcp.ts`
- Create: `web/src/webmcp.test.ts`

**Interfaces:**
- `registerFlowProofTools(client)` returns registration state/cleanup and registers exactly the six required names.
- API client methods mirror backend use-cases and accept optional `AbortSignal`.

- [ ] Create frontend manifests/config and a Vitest test that fails because the WebMCP registration module does not exist.
- [ ] Test all six names/descriptions/schemas, backend argument mapping, JSON result strings, read-only annotations where appropriate, `AbortSignal` forwarding, and unsupported `document.modelContext`.
- [ ] Run `npm test -- --run` in `web`; confirm RED for missing implementation, not setup errors.
- [ ] Implement the minimal typed API client and `document.modelContext.registerTool` integration.
- [ ] Re-run frontend tests and `npm run typecheck`.

### Task 5: Operations dashboard UI

**Files:**
- Create: `web/src/App.tsx`, `web/src/App.test.tsx`, `web/src/main.tsx`, `web/src/styles.css`

**Interfaces:**
- UI consumes Task 4 client/tool registration and displays one selected test/run.
- Manual controls invoke the same backend use-cases as WebMCP.

- [ ] Write failing UI tests for WebMCP support badge, create/start controls, recoverable failure panel, retry action, timeline/evidence rendering, and exported Playwright panel.
- [ ] Run focused Vitest tests and confirm RED.
- [ ] Implement the dense dark operations dashboard using the approved design tokens, accessible labels/focus states, reduced motion, and responsive layout.
- [ ] Implement bounded polling only while a run is non-terminal/recoverable and cleanup timers on unmount.
- [ ] Run `npm test -- --run`, `npm run typecheck`, `npm run lint`, and `npm run build`.

### Task 6: Static serving and packaging

**Files:**
- Modify: `internal/httpapi/api.go`
- Modify/Create tests in `internal/httpapi/api_test.go`
- Create: `Dockerfile`
- Create: `.dockerignore`
- Create: `.gitignore`
- Create: `README.md`
- Create/Update: `.supercode/PROJECT.md`, `.supercode/ARCHITECTURE.md`, `.supercode/CURRENT_STATE.md`, `.supercode/DECISIONS.md`, `.supercode/TODO.md`

- [ ] Add RED handler tests for SPA/static fallback and missing asset behavior.
- [ ] Serve `web/dist` from configurable filesystem path without making Go tests depend on a pre-existing build artifact.
- [ ] Add multi-stage Docker build that builds frontend, Go server, and supplies Chromium in runtime image.
- [ ] Document local prerequisites, commands, WebMCP Chrome/origin-trial setup, architecture, safety boundary, API, and exact 3-minute demo script.
- [ ] Build frontend then Go binary; run targeted HTTP tests GREEN.

### Task 7: Full verification and runtime acceptance

**Files:** no planned source changes unless a failing gate reveals a defect.

- [ ] Read `skill://verification-before-completion` and `skill://engineering-quality`.
- [ ] Run `gofmt` over Go files and confirm no formatting diff remains.
- [ ] Run `go vet ./...`, `go test ./... -count=1`, frontend lint/typecheck/tests/build, and fresh Go production build.
- [ ] Start the local server in deterministic demo mode with a bounded process.
- [ ] Exercise create -> start -> recoverable failure -> inspect -> retry -> succeeded -> export through real HTTP and verify returned states/evidence/test code.
- [ ] Open the dashboard with browser tooling, verify visible WebMCP fallback/support state, timeline, failure/recovery UI, responsive layout, and no console errors.
- [ ] Stop all local test processes.
- [ ] Review final Git diff/status for secrets, placeholders, unrelated edits, generated junk, and scope drift.
- [ ] Report exact checks and any residual WebMCP limitation that cannot be locally proven without a WebMCP-enabled browser.
