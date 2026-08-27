import '@testing-library/jest-dom/vitest';
import { act, cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import type { CreateTestInput, FlowProofClient } from './api';
import type { FailureAnalysis, Run, TestDefinition } from './types';
import { App } from './App';

afterEach(() => {
  cleanup();
  vi.useRealTimers();
  delete (document as Document & { modelContext?: unknown }).modelContext;
});

describe('FlowProof dashboard', () => {
  it('shows a graceful WebMCP unavailable badge while manual controls remain usable', async () => {
    render(<App client={fakeClient()} />);
    expect(await screen.findByText(/WebMCP unavailable/i)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /create test/i })).toBeEnabled();
  });

  it('creates a test, runs it into recoverable failure, and renders diagnosis, timeline, and evidence', async () => {
    const client = fakeClient();
    render(<App client={client} />);

    fireEvent.click(screen.getByRole('button', { name: /create test/i }));
    expect(await screen.findByText(/test-1/i)).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /start run/i }));

    expect(await screen.findByText(/recoverable failure/i)).toBeInTheDocument();
    expect(screen.getByText('#checkout-submit')).toBeInTheDocument();
    expect(screen.getByText('[data-testid="checkout-submit"]')).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: /selector stale/i })).toBeInTheDocument();
    expect(screen.getByText(/checkout attempt failed/i)).toBeInTheDocument();
    expect(screen.getByText(/DOM snapshot/i)).toBeInTheDocument();
    expect(client.inspectFailure).toHaveBeenCalledWith('run-1', expect.any(AbortSignal));
  });

  it('retries the verified fallback and exports the recovered Playwright regression', async () => {
    const client = fakeClient();
    render(<App client={client} />);
    fireEvent.click(screen.getByRole('button', { name: /create test/i }));
    await screen.findByText(/test-1/i);
    fireEvent.click(screen.getByRole('button', { name: /start run/i }));
    await screen.findByText(/recoverable failure/i);

    fireEvent.click(screen.getByRole('button', { name: /retry failed step/i }));
    expect(await screen.findByText(/run succeeded/i)).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: /export regression test/i }));
    const exportedCode = await screen.findByText((content) => content.includes('@playwright/test'));
    expect(exportedCode).toHaveTextContent('data-testid="checkout-submit"');
  });

  it('polls a running run and stops after a terminal state', async () => {
    vi.useFakeTimers();
    const client = fakeClient();
    client.startRun = vi.fn(async () => makeRun('running'));
    client.getRun = vi.fn(async () => makeRun('succeeded'));
    render(<App client={client} pollIntervalMs={1000} />);

    fireEvent.click(screen.getByRole('button', { name: /create test/i }));
    await act(async () => Promise.resolve());
    fireEvent.click(screen.getByRole('button', { name: /start run/i }));
    await act(async () => Promise.resolve());
    await act(async () => { vi.advanceTimersByTime(1000); await Promise.resolve(); });
    expect(client.getRun).toHaveBeenCalledTimes(1);
    await act(async () => { vi.advanceTimersByTime(3000); await Promise.resolve(); });
    expect(client.getRun).toHaveBeenCalledTimes(1);
  });
});

function fakeClient() {
  const failed = makeRun('failed_recoverable');
  const succeeded = makeRun('succeeded');
  return {
    createTest: vi.fn(async (_input: CreateTestInput, _signal?: AbortSignal): Promise<TestDefinition> => ({ id: 'test-1', targetUrl: '/demo-store', objective: 'Verify checkout recovery', createdAt: 'now' })),
    startRun: vi.fn(async (_id: string, _signal?: AbortSignal): Promise<Run> => failed),
    getRun: vi.fn(async (_id: string, _signal?: AbortSignal): Promise<Run> => failed),
    inspectFailure: vi.fn(async (_id: string, _signal?: AbortSignal): Promise<FailureAnalysis> => ({ step: 'checkout', failedSelector: '#checkout-submit', fallbackSelector: '[data-testid="checkout-submit"]', explanation: 'selector stale; stable selector observed in page inventory', recoverable: true })),
    retryRun: vi.fn(async (_id: string, _signal?: AbortSignal): Promise<Run> => succeeded),
    exportRun: vi.fn(async (_id: string, _signal?: AbortSignal) => ({ code: "import { test } from '@playwright/test';\nawait page.locator('[data-testid=\"checkout-submit\"]').click();" })),
  } satisfies FlowProofClient;
}

function makeRun(status: Run['status']): Run {
  return {
    id: 'run-1', testId: 'test-1', targetUrl: '/demo-store', objective: 'Verify checkout recovery', status, stepCount: 3, maxSteps: 4,
    summary: status === 'succeeded' ? 'Order confirmed · FP-2048' : undefined,
    failure: status === 'failed_recoverable' ? 'checkout attempt failed' : undefined,
    failureAnalysis: status === 'failed_recoverable' ? { step: 'checkout', failedSelector: '#checkout-submit', fallbackSelector: '[data-testid="checkout-submit"]', explanation: 'selector stale; stable selector observed in page inventory', recoverable: true } : undefined,
    recoveredSelector: status === 'succeeded' ? '[data-testid="checkout-submit"]' : undefined,
    events: [{ seq: 1, at: 'now', type: 'step_failed', message: 'checkout attempt failed' }],
    evidence: [{ step: 3, kind: 'text', label: 'DOM snapshot', text: 'Complete checkout', capturedAt: 'now' }],
    createdAt: 'now', updatedAt: 'now', completedAt: status === 'succeeded' ? 'now' : undefined,
  };
}
