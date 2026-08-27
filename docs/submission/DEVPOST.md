# FlowProof ΓÇö WebMCP Challenge Submission Draft

## Project title

FlowProof ΓÇö Agent-native browser QA

## One-line pitch

FlowProof turns a brittle browser failure into an evidence-backed recovery and a reusable Playwright regression test through six narrow WebMCP tools.

## Description

Browser agents are useful until a selector changes. FlowProof gives an agent a structured QA control plane instead of asking it to improvise against a broken page.

The FlowProof page registers exactly six tools with `document.modelContext.registerTool(...)`:

- `create_test`
- `start_run`
- `get_run_status`
- `inspect_failure`
- `retry_failed_step`
- `export_regression_test`

The built-in deterministic `/demo-store` proves the full workflow. Setup succeeds, then the stale selector `#checkout-submit` intentionally fails. FlowProof captures browser evidence and exposes `[data-testid="checkout-submit"]` only because the current DOM proves that fallback exists. The agent can inspect the structured failure, retry the failed step with the observed stable selector, verify the order confirmation, and export a Playwright TypeScript regression test.

FlowProof intentionally does not put another LLM inside the application. The external browser/ChatGPT agent is the agent; FlowProof is the deterministic execution, evidence, safety, and regression layer. That keeps the demo focused on WebMCP and removes model/API-key failure modes from the product path.

## How WebMCP is used

FlowProof uses the current imperative WebMCP surface, `document.modelContext.registerTool(...)`. Each tool has a narrow JSON schema and maps to a concrete backend operation. Read-only annotations are used for status, failure inspection, and regression export. The `AbortSignal` supplied by WebMCP is forwarded to backend requests.

If the browser does not expose `document.modelContext`, the same dashboard remains usable in manual mode so the product is still inspectable without hiding the WebMCP dependency.

## Technical architecture

- Go backend for target validation, orchestration, persistence, HTTP API, and static serving.
- chromedp + Chromium for deterministic browser execution and evidence capture.
- React + TypeScript + Vite for the operations dashboard and WebMCP registration.
- JSON persisted run state for the contest demo.
- Docker packaging with Chromium installed and an unprivileged runtime user.
- GitHub Actions publishes the production container to GHCR; Railway runs the published image.

## Safety model

FlowProof is not a general-purpose remote browser executor. HTTP/HTTPS targets only. The built-in same-origin fixture is allowed by default; external targets require exact allowlisting. URL credentials and private/loopback/link-local external targets are rejected. Recovery selectors are never invented: a fallback is offered only when current browser evidence proves it exists.

## Demo sequence

1. Create a test against `/demo-store`.
2. Start the run.
3. Observe the intentional `#checkout-submit` failure as `failed_recoverable`.
4. Inspect the structured failure and captured evidence.
5. Retry with the observed `[data-testid="checkout-submit"]` selector.
6. Verify successful order confirmation.
7. Export a Playwright regression test using the recovered selector.

## Links

- Source repository: https://github.com/duclucky/flowproof-agent
- Live app: https://flowproof-production-e9fd.up.railway.app
- Demo video: TBD

## License

MIT
