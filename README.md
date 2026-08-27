# FlowProof

Agent-native deterministic browser QA for the OpenAI WebMCP Challenge. FlowProof exposes six structured tools through `document.modelContext.registerTool(...)`, captures evidence for a stale-selector failure, performs only an evidence-backed recovery, and exports a Playwright TypeScript regression test. No external model API key is required in the contest path.

## WebMCP tools

- `create_test` — define a validated target URL and objective.
- `start_run` — start deterministic browser QA.
- `get_run_status` — read run state, timeline, and evidence.
- `inspect_failure` — read structured diagnosis and verified fallback.
- `retry_failed_step` — retry only with the observed safe selector.
- `export_regression_test` — export a succeeded run as Playwright TypeScript.

Read-only annotations are used for status, inspection, and export. WebMCP AbortSignal is forwarded to backend fetches. If `document.modelContext` is unavailable, the dashboard reports manual mode and keeps the same operations available to a human. The deprecated `navigator.modelContext` API is not used.

## Demo

The built-in `/demo-store` intentionally lacks stale `#checkout-submit` and exposes stable `[data-testid="checkout-submit"]`. Create a test, start it, observe `failed_recoverable`, inspect evidence, retry the verified fallback, confirm `Order confirmed · FP-2048`, then export the recovered Playwright regression. The fallback is exposed only when actual page inventory proves it exists.

Production demo: `https://flowproof-production-e9fd.up.railway.app`

## Architecture

Go owns configuration, target safety, JSON persistence, chromedp execution, run orchestration, HTTP API, demo target, and static SPA serving. React/TypeScript owns the operations dashboard, typed same-origin API client, and WebMCP registration.

The production Linux container is built by GitHub Actions, published as `ghcr.io/duclucky/flowproof-agent:latest`, and deployed on Railway.

## Safety

FlowProof is not a general browser agent. Only HTTP/HTTPS targets are accepted. The built-in same-origin demo is allowed by default; external targets require exact membership in `FLOWPROOF_ALLOWED_HOSTS`. URL credentials, unsafe private/loopback/link-local external targets, and implicit subdomains are rejected. The workflow has no payment, credential, file-transfer, destructive-account, or arbitrary-script capability.

## Local run

Prerequisites: Go 1.26+, Node.js 24+, npm, and Chromium/Chrome.

```powershell
cd web
npm ci
npm run build
cd ..
go run ./cmd/server
```

Open `http://localhost:8080`. Set `FLOWPROOF_CHROME_PATH` if the browser executable is not auto-discovered.

Verification:

```powershell
go test ./... -count=1
go vet ./...
go build ./cmd/server
cd web
npm test -- --run
npm run typecheck
npm run lint
npm run build
```

## Configuration

- `PORT` default `8080`
- `FLOWPROOF_STATE_PATH` default `data/runs.json`
- `FLOWPROOF_CHROME_PATH` optional browser executable
- `FLOWPROOF_WEB_DIR` default `web/dist`
- `FLOWPROOF_ALLOWED_HOSTS` comma-separated exact external hosts
- `FLOWPROOF_STEP_TIMEOUT` default `4s`
- `FLOWPROOF_RUN_TIMEOUT` default `45s`

## HTTP API

`GET /healthz`, `GET /demo-store`, `POST /api/tests`, `POST /api/tests/{testId}/runs`, `GET /api/runs/{runId}`, `GET /api/runs/{runId}/failure`, `POST /api/runs/{runId}/retry`, `GET /api/runs/{runId}/export`.

## Docker

```powershell
docker build -t flowproof .
docker run --rm -p 8080:8080 flowproof
```

The multi-stage image builds Vite and Go, installs Chromium, serves `web/dist`, and runs as an unprivileged user.

## Under-three-minute video script

- 0:00-0:20: problem and FlowProof dashboard.
- 0:20-0:45: show the six WebMCP tools and narrow schemas.
- 0:45-1:15: create/start; stale selector fails recoverably.
- 1:15-1:45: show timeline/evidence and call `inspect_failure`.
- 1:45-2:10: call `retry_failed_step`; show `Order confirmed · FP-2048`.
- 2:10-2:35: export Playwright regression using the recovered selector.
- 2:35-2:55: close on deterministic safety, human/agent shared state, and reusable regression value.

## Submission status

The public repository, MIT license, production Railway URL, GHCR image, full public create → fail → inspect → retry → success → export acceptance, and native six-tool WebMCP proof in Chrome 151 are complete. Remaining submission work is the public YouTube demo under three minutes and the final Devpost entry. Keep the production app available through judging.

## License

MIT. See `LICENSE`.
