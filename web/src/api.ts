import type { FailureAnalysis, Run, TestDefinition } from './types';

export interface CreateTestInput {
  targetUrl: string;
  objective: string;
}

export interface ExportResult {
  code: string;
}

export interface FlowProofClient {
  createTest(input: CreateTestInput, signal?: AbortSignal): Promise<TestDefinition>;
  startRun(testId: string, signal?: AbortSignal): Promise<Run>;
  getRun(runId: string, signal?: AbortSignal): Promise<Run>;
  inspectFailure(runId: string, signal?: AbortSignal): Promise<FailureAnalysis>;
  retryRun(runId: string, signal?: AbortSignal): Promise<Run>;
  exportRun(runId: string, signal?: AbortSignal): Promise<ExportResult>;
}

interface APIErrorPayload {
  error?: { code?: string; message?: string };
}

export class HTTPFlowProofClient implements FlowProofClient {
  async createTest(input: CreateTestInput, signal?: AbortSignal): Promise<TestDefinition> {
    let targetUrl = input.targetUrl;
    try {
      new URL(targetUrl);
    } catch {
      targetUrl = new URL(targetUrl, window.location.origin).toString();
    }

    return requestJSON<TestDefinition>('/api/tests', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ...input, targetUrl }),
      signal,
    });
  }

  async startRun(testId: string, signal?: AbortSignal): Promise<Run> {
    return requestJSON<Run>(`/api/tests/${encodeURIComponent(testId)}/runs`, { method: 'POST', signal });
  }

  async getRun(runId: string, signal?: AbortSignal): Promise<Run> {
    return requestJSON<Run>(`/api/runs/${encodeURIComponent(runId)}`, { signal });
  }

  async inspectFailure(runId: string, signal?: AbortSignal): Promise<FailureAnalysis> {
    return requestJSON<FailureAnalysis>(`/api/runs/${encodeURIComponent(runId)}/failure`, { signal });
  }

  async retryRun(runId: string, signal?: AbortSignal): Promise<Run> {
    return requestJSON<Run>(`/api/runs/${encodeURIComponent(runId)}/retry`, { method: 'POST', signal });
  }

  async exportRun(runId: string, signal?: AbortSignal): Promise<ExportResult> {
    return requestJSON<ExportResult>(`/api/runs/${encodeURIComponent(runId)}/export`, { signal });
  }
}

export const flowProofClient: FlowProofClient = new HTTPFlowProofClient();

async function requestJSON<T>(url: string, init: RequestInit): Promise<T> {
  const response = await fetch(url, init);
  if (response.ok) return (await response.json()) as T;

  let message = `${response.status} ${response.statusText}`.trim();
  try {
    const payload = (await response.json()) as APIErrorPayload;
    if (payload.error?.message) message = payload.error.message;
    if (payload.error?.code) message = `${payload.error.code}: ${message}`;
  } catch {
    // Keep the HTTP status when the response body is not JSON.
  }
  throw new Error(message);
}
