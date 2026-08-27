# FlowProof — Public Demo Script (<3 minutes)

Target length: 2:35–2:50.

## 0:00–0:18 — Problem

Show the FlowProof dashboard.

Voiceover:
“Browser agents are powerful until a selector changes. FlowProof gives them a structured QA control plane: run the workflow, capture evidence, diagnose the failure, retry only when the page proves a safe fallback, and turn the recovery into a regression test.”

## 0:18–0:38 — WebMCP surface

Show the six registered tool names in source or browser tooling:
`create_test`, `start_run`, `get_run_status`, `inspect_failure`, `retry_failed_step`, `export_regression_test`.

Voiceover:
“FlowProof exposes exactly six narrow tools with the current `document.modelContext.registerTool` API. The external browser agent is the agent; FlowProof stays deterministic and requires no model API key.”

## 0:38–1:02 — Create and start

Create a test against the built-in `/demo-store` with the checkout objective, then start it.

Voiceover:
“The fixture is deterministic. It adds the item, applies coupon SAVE20, and then intentionally tries a stale checkout selector.”

## 1:02–1:28 — Failure and evidence

Show `failed_recoverable`, failed selector `#checkout-submit`, evidence/timeline, and the verified fallback `[data-testid="checkout-submit"]`.

Voiceover:
“The stale selector fails for real. FlowProof captures text and screenshot evidence. It does not invent a repair: it exposes the stable fallback only because the current DOM proves that selector exists.”

## 1:28–1:55 — Inspect and retry

Call `inspect_failure`, then `retry_failed_step`.

Voiceover:
“The agent can inspect a structured diagnosis, then explicitly retry the failed step. FlowProof replays setup in a fresh Chromium session and uses only the observed safe selector.”

## 1:55–2:14 — Success

Show succeeded state and order confirmation.

Voiceover:
“The retry succeeds and the browser verifies the order confirmation.”

## 2:14–2:36 — Export regression

Call `export_regression_test` and show the generated Playwright TypeScript containing `[data-testid="checkout-submit"]` and the `Order confirmed` assertion.

Voiceover:
“The recovery becomes a reusable Playwright regression test, so the fix has a life after the agent session.”

## 2:36–2:48 — Close

Show dashboard overview.

Voiceover:
“FlowProof makes browser QA agent-native, evidence-backed, and deterministic: structured WebMCP tools in, verified regression test out.”

## Recording checklist

- Keep browser zoom at 100% and text readable at 1080p or higher.
- Do not show local paths, tokens, email addresses, cloud console credentials, or terminal history containing secrets.
- Keep the six WebMCP names visible long enough to read.
- Show the actual stale selector, actual fallback, success state, and exported test.
- Final exported video must remain below 3:00.
