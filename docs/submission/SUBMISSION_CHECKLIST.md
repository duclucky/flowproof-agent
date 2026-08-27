# FlowProof ΓÇö Submission Checklist

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

- [ ] Open the public app in a Chrome build/profile exposing `document.modelContext`.
- [ ] Verify exactly six FlowProof tools register.
- [ ] Exercise at least one tool from the browser agent surface.
- [ ] Capture clean evidence for the demo video.

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

Railway production is live from the public GHCR image `ghcr.io/duclucky/flowproof-agent:latest`. Railway's source/snapshot build path failed before Dockerfile execution at the Metal builder scheduling stage, so the build was moved to GitHub Actions and the resulting image was deployed directly. GitHub Actions run `33057258565` succeeded, and Railway deployment `20500250-1b7f-42bb-963f-5ce7781b641a` reached `SUCCESS` with image digest `sha256:e8f8fd4b060fdee45b9c44e0c0ae83851b31cca4b8f2d4c6f675c1b68f01b781`.
