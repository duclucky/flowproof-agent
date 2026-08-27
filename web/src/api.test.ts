import { afterEach, describe, expect, it, vi } from 'vitest';
import { HTTPFlowProofClient } from './api';

afterEach(() => {
  vi.restoreAllMocks();
});

describe('HTTPFlowProofClient.createTest', () => {
  it('resolves a relative target against the current page origin before sending it', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(responseForTest());
    const client = new HTTPFlowProofClient();

    await client.createTest({ targetUrl: '/demo-store', objective: 'Verify checkout recovery' });

    const [, init] = fetchMock.mock.calls[0];
    const body = JSON.parse(String(init?.body));
    expect(body.targetUrl).toBe(new URL('/demo-store', window.location.origin).toString());
  });

  it('preserves an already-absolute target URL', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(responseForTest());
    const client = new HTTPFlowProofClient();
    const targetUrl = 'https://example.com/demo-store?case=absolute#checkout';

    await client.createTest({ targetUrl, objective: 'Verify checkout recovery' });

    const [, init] = fetchMock.mock.calls[0];
    const body = JSON.parse(String(init?.body));
    expect(body.targetUrl).toBe(targetUrl);
  });
});

function responseForTest(): Response {
  return new Response(JSON.stringify({
    id: 'test-1',
    targetUrl: 'https://example.com/demo-store',
    objective: 'Verify checkout recovery',
    createdAt: 'now',
  }), { status: 201, headers: { 'Content-Type': 'application/json' } });
}
