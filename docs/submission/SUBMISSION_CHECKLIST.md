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
- [ ] Deploy a public working URL.
- [ ] Smoke-test public `/`, `/healthz`, `/demo-store`, and full fail→inspect→retry→success→export lifecycle.
- [ ] Verify public URL remains available without authentication.

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
- [ ] Replace `Live app: TBD` with deployed URL.
- [ ] Replace `Demo video: TBD` with public YouTube URL.
- [ ] Enter project description and links in Devpost.
- [ ] Final rules/eligibility/deadline check immediately before submission.
- [ ] Submit entry.
- [ ] Keep app live through judging.

## Current deployment blocker

Cloud Run source deployment was attempted on 2026-08-27 but the configured Google Cloud project has no billing-enabled account, so Cloud Build/Artifact Registry/Cloud Run APIs cannot be activated. No billing change was made automatically. Railway CLI is available through `npx` but the machine is not authenticated to Railway. Vercel is authenticated, but the current Go + Chromium workload is not a suitable Vercel serverless deployment without changing the architecture.
