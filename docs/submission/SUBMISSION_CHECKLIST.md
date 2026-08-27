# FlowProof — Submission Checklist

## Product

- [x] Local implementation complete.
- [x] Six WebMCP tools implemented with `document.modelContext.registerTool(...)`.
- [x] Deterministic stale-selector failure and evidence-backed retry implemented.
- [x] Playwright regression export implemented.
- [x] Go/frontend verification gates passing.
- [x] Real Chromium local acceptance passing.
- [x] Public GitHub repository available.
- [x] MIT license present at repository root.

## Publication

- [x] Push current `main` to GitHub.
- [x] Deploy a public working URL: `https://flowproof-production-e9fd.up.railway.app`.
- [x] Smoke-test public `/`, `/healthz`, `/demo-store`, and full fail → inspect → retry → success → export lifecycle.
- [x] Verify public URL remains available without authentication.

Production acceptance on 2026-08-27 used run `run-cd0b5bef534f70d8`: the real Chromium container reached `failed_recoverable` on stale `#checkout-submit`, inspection returned the DOM-proven `[data-testid="checkout-submit"]` fallback, retry reached `succeeded`, and `/export` returned Playwright code containing the recovered selector and `Order confirmed` assertion.

## WebMCP browser proof

- [x] Open the public app in Chrome 151.0.7922.174 with WebMCP testing enabled and verify `document.modelContext` is exposed.
- [x] Verify exactly six FlowProof tools register.
- [x] Exercise the complete six-tool workflow through the native WebMCP testing surface.
- [x] Capture clean production evidence for the demo video.

Native production proof on 2026-08-27 used run `run-3b79d6882ff40c92`: Chrome enumerated exactly six FlowProof tools, `start_run` produced the intended `failed_recoverable` state, `get_run_status` returned the persisted timeline/evidence, `inspect_failure` returned the DOM-proven fallback, `retry_failed_step` reached `succeeded`, and `export_regression_test` returned Playwright code containing the recovered selector and `Order confirmed` assertion.

## Video

- [x] Final script drafted in `docs/submission/VIDEO_SCRIPT.md`.
- [ ] Record final demo under 3 minutes.
- [ ] Confirm no secrets/private identifiers appear in the recording.
- [ ] Publish public YouTube video.

## Devpost

- [x] Submission copy drafted in `docs/submission/DEVPOST.md`.
- [x] Replace `Live app: TBD` with deployed URL.
- [ ] Replace `Demo video: TBD` with public YouTube URL.
- [ ] Enter project description and links in Devpost.
- [ ] Final rules/eligibility/deadline check immediately before submission.
- [ ] Submit entry.
- [ ] Keep app live through judging.

## Deployment status

Railway production runs the public GHCR image `ghcr.io/duclucky/flowproof-agent:latest`. GitHub Actions run `33064211227` published the WebMCP compatibility fix from commit `1d325ee`, and Railway deployment `d12d04fe-796b-4884-a0ca-f448dcd28b86` reached Online. The public deployment then passed the full six-tool native WebMCP lifecycle in Chrome 151.
