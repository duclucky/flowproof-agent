# FlowProof WebMCP Design

**Date:** 2026-08-27
**Status:** Approved for implementation

## Goal

Build a small, reliable WebMCP Challenge submission that turns browser QA into an agent-native workflow. A human and an external browser agent share one FlowProof dashboard. The page exposes structured QA capabilities through the current WebMCP imperative API, `document.modelContext.registerTool(...)`; the Go backend executes and records a deterministic browser test, exposes recoverable failure evidence, completes a safe retry, and exports a Playwright regression test.

## Product scope

The contest demo contains exactly one polished operations dashboard, one same-project demo target, one deterministic stale-selector failure, one recovery path, six WebMCP tools, evidence/timeline UI, and regression export. It is not a general web agent and does not require an external LLM/API key.

Required tools:

- `create_test` — define the target URL and objective.
- `start_run` — start the deterministic browser QA workflow.
- `get_run_status` — return status, timeline, evidence summary, and failure state.
- `inspect_failure` — return structured diagnosis for a recoverable failed step.
- `retry_failed_step` — execute the approved safe fallback selector and finish the flow.
- `export_regression_test` — return a complete Playwright TypeScript regression test using the recovered selector.

All tools use real JSON Schema, useful descriptions, the current `document.modelContext` API, and same-origin backend calls. Long-running fetches receive the WebMCP execution `AbortSignal`. Ordinary browsers get a graceful “WebMCP unavailable” indicator while manual dashboard controls continue to work.

## Demo workflow

1. Agent calls `create_test` for the built-in checkout demo.
2. Agent calls `start_run`. FlowProof drives Chromium through add-to-cart/checkout setup and intentionally attempts the stale selector `#checkout-submit`.
3. The run becomes `failed_recoverable`; the timeline records the attempted selector, browser error, current URL, visible-text snapshot, and screenshot evidence.
4. Agent calls `inspect_failure`. FlowProof reports that the selector is stale and identifies `[data-testid="checkout-submit"]` as the deterministic safe fallback observed on the page.
5. Agent calls `retry_failed_step`. FlowProof uses that fallback, verifies the success confirmation, records recovery evidence, and marks the run `succeeded`.
6. Agent calls `export_regression_test`. FlowProof returns a runnable Playwright test that uses the recovered stable selector.

## Architecture

### Go backend

Use standard `net/http`, the existing JSON store pattern, and a narrow orchestrator. The contest path has no Gemini dependency. Existing Gemini planner code may be removed if it complicates compilation; the product behavior must remain deterministic and locally runnable.

Core boundaries:

- `internal/model`: test definitions, run state, events, evidence, failure analysis.
- `internal/store`: persistence interface plus JSON file implementation.
- `internal/browser`: `Driver` abstraction; production `ChromedpDriver`, deterministic fake driver for tests.
- `internal/orchestrator`: valid run-state transitions and initial/retry/export workflows.
- `internal/httpapi`: request validation, JSON API, built-in demo target, and static frontend serving.
- `cmd/server`: configuration/wiring only.

Run transitions are bounded and explicit: `queued -> running -> failed_recoverable -> running -> succeeded`; terminal/invalid transitions are rejected.

### HTTP API

- `POST /api/tests`
- `POST /api/tests/{testId}/runs`
- `GET /api/runs/{runId}`
- `GET /api/runs/{runId}/failure`
- `POST /api/runs/{runId}/retry`
- `GET /api/runs/{runId}/export`
- `GET /healthz`
- `GET /demo-store` for the deterministic target

JSON errors use a consistent `{"error":{"code","message"}}` shape. Body sizes and operation timeouts are bounded.

### Target safety

Only `http`/`https` targets are accepted. By default, the built-in same-origin demo target is allowed. External hosts require exact membership in `FLOWPROOF_ALLOWED_HOSTS`. Reject credentials in URLs, unsupported schemes, malformed hostnames, loopback/private/link-local targets unless they are the server’s own explicit demo origin, and cross-origin redirects outside the allowlist. No payment, credential, file upload/download, destructive account, or arbitrary script action exists in the workflow.

### Frontend

Use React + TypeScript + Vite. Keep state simple: API client, WebMCP registration module, and one dashboard page. The browser agent and human see the same run status. Poll a running/recoverable run at a bounded interval and stop polling in terminal state. Generated Playwright code is displayed in a read-only code panel and can be copied manually.

The visual system is a dense technical operations dashboard: dark background `#020617`, foreground `#F8FAFC`, surfaces around `#0F172A`/`#1E293B`, one green accent `#16A34A`, red only for failure, high-contrast borders, Fira Sans/Fira Code fallback stack, subtle 150–300ms transitions, visible keyboard focus, reduced-motion support, responsive 375/768/1024/1440 layouts, and SVG icons rather than emoji.

## Testing

All new behavior follows RED/GREEN/REFACTOR. Backend tests use a fake browser driver and real HTTP handlers/store behavior. Frontend tests fake `document.modelContext` and `fetch` to prove all six tool registrations, argument forwarding, AbortSignal propagation, and graceful unsupported-browser behavior. Runtime acceptance uses the real local Go server and Chromium against `/demo-store`.

## Acceptance criteria

- Six non-trivial WebMCP tools register through `document.modelContext.registerTool`.
- Full create/start/fail/inspect/retry/succeed/export flow works through HTTP with no external AI/API key.
- Browser failure and recovery are visible in dashboard timeline/evidence.
- Exported Playwright test uses the recovered stable selector.
- Go format/vet/tests, frontend lint/typecheck/tests/build, production build, local HTTP workflow, and browser dashboard smoke all pass.
- Dockerfile and README are sufficient to reproduce the demo.
- No secrets read or committed; no external deployment/publish/commit performed.
