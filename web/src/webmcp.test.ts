import { afterEach, describe, expect, it, vi } from 'vitest';
import type { CreateTestInput, FlowProofClient } from './api';
import type { FailureAnalysis, Run, TestDefinition } from './types';
import { registerFlowProofTools } from './webmcp';

type RegisteredTool = {
  name: string;
  description: string;
  inputSchema: {
    type: 'object';
    properties: Record<string, { type: string }>;
    required: string[];
    additionalProperties?: boolean;
  };
  annotations?: { readOnlyHint?: boolean; untrustedContentHint?: boolean };
  execute: (input: Record<string, string>, context: { signal: AbortSignal }) => Promise<string>;
};

type TestDocument = Document & {
  modelContext?: { registerTool: (tool: RegisteredTool) => Promise<void> | void };
};

const names = [
  'create_test',
  'start_run',
  'get_run_status',
  'inspect_failure',
  'retry_failed_step',
  'export_regression_test',
] as const;

afterEach(() => {
  delete (document as TestDocument).modelContext;
  vi.restoreAllMocks();
});

describe('registerFlowProofTools', () => {
  it('registers exactly six useful WebMCP tools with schemas and annotations', async () => {
    const registered: RegisteredTool[] = [];
    Object.defineProperty(document, 'modelContext', {
      configurable: true,
      value: { registerTool: vi.fn((tool: RegisteredTool) => registered.push(tool)) },
    });

    const client = fakeClient();
    const state = await registerFlowProofTools(client);

    expect(state.supported).toBe(true);
    expect(state.names).toEqual(names);
    expect(registered.map((tool) => tool.name)).toEqual(names);
    for (const tool of registered) {
      expect(tool.description.length).toBeGreaterThan(20);
      expect(tool.inputSchema.type).toBe('object');
      expect(tool.inputSchema.required).toHaveLength(tool.name === 'create_test' ? 2 : 1);
      expect(tool.inputSchema.additionalProperties).toBe(false);
      expect(tool.annotations?.untrustedContentHint).toBe(false);
    }

    expect(toolByName(registered, 'get_run_status').annotations?.readOnlyHint).toBe(true);
    expect(toolByName(registered, 'inspect_failure').annotations?.readOnlyHint).toBe(true);
    expect(toolByName(registered, 'export_regression_test').annotations?.readOnlyHint).toBe(true);
    expect(toolByName(registered, 'create_test').annotations?.readOnlyHint).toBe(false);
    expect(toolByName(registered, 'start_run').annotations?.readOnlyHint).toBe(false);
    expect(toolByName(registered, 'retry_failed_step').annotations?.readOnlyHint).toBe(false);

    expect(toolByName(registered, 'create_test').inputSchema.required).toEqual(['targetUrl', 'objective']);
    expect(toolByName(registered, 'start_run').inputSchema.required).toEqual(['testId']);
    expect(toolByName(registered, 'get_run_status').inputSchema.required).toEqual(['runId']);
    expect(toolByName(registered, 'inspect_failure').inputSchema.required).toEqual(['runId']);
    expect(toolByName(registered, 'retry_failed_step').inputSchema.required).toEqual(['runId']);
    expect(toolByName(registered, 'export_regression_test').inputSchema.required).toEqual(['runId']);
  });

  it('maps every execute call to the typed backend client, forwards AbortSignal, and returns JSON strings', async () => {
    const registered: RegisteredTool[] = [];
    Object.defineProperty(document, 'modelContext', {
      configurable: true,
      value: { registerTool: (tool: RegisteredTool) => registered.push(tool) },
    });
    const client = fakeClient();
    await registerFlowProofTools(client);
    const controller = new AbortController();
    const context = { signal: controller.signal };

    await expect(toolByName(registered, 'create_test').execute({ targetUrl: '/demo-store', objective: 'checkout' }, context)).resolves.toContain('test-1');
    await expect(toolByName(registered, 'start_run').execute({ testId: 'test-1' }, context)).resolves.toContain('run-1');
    await expect(toolByName(registered, 'get_run_status').execute({ runId: 'run-1' }, context)).resolves.toContain('failed_recoverable');
    await expect(toolByName(registered, 'inspect_failure').execute({ runId: 'run-1' }, context)).resolves.toContain('checkout-submit');
    await expect(toolByName(registered, 'retry_failed_step').execute({ runId: 'run-1' }, context)).resolves.toContain('succeeded');
    await expect(toolByName(registered, 'export_regression_test').execute({ runId: 'run-1' }, context)).resolves.toContain('playwright');

    expect(client.createTest).toHaveBeenCalledWith({ targetUrl: '/demo-store', objective: 'checkout' }, controller.signal);
    expect(client.startRun).toHaveBeenCalledWith('test-1', controller.signal);
    expect(client.getRun).toHaveBeenCalledWith('run-1', controller.signal);
    expect(client.inspectFailure).toHaveBeenCalledWith('run-1', controller.signal);
    expect(client.retryRun).toHaveBeenCalledWith('run-1', controller.signal);
    expect(client.exportRun).toHaveBeenCalledWith('run-1', controller.signal);
  });

  it('degrades cleanly when document.modelContext is unavailable', async () => {
    delete (document as TestDocument).modelContext;
    const client = fakeClient();

    await expect(registerFlowProofTools(client)).resolves.toEqual({ supported: false, names: [] });
    expect(client.createTest).not.toHaveBeenCalled();
  });
});

function toolByName(tools: RegisteredTool[], name: string): RegisteredTool {
  const tool = tools.find((candidate) => candidate.name === name);
  if (!tool) throw new Error(`missing tool ${name}`);
  return tool;
}

function fakeClient(): FlowProofClient & {
  createTest: ReturnType<typeof vi.fn>;
  startRun: ReturnType<typeof vi.fn>;
  getRun: ReturnType<typeof vi.fn>;
  inspectFailure: ReturnType<typeof vi.fn>;
  retryRun: ReturnType<typeof vi.fn>;
  exportRun: ReturnType<typeof vi.fn>;
} {
  const failedRun = makeRun('failed_recoverable');
  const succeededRun = makeRun('succeeded');
  return {
    createTest: vi.fn(async (_input: CreateTestInput, _signal?: AbortSignal): Promise<TestDefinition> => ({
      id: 'test-1', targetUrl: '/demo-store', objective: 'checkout', createdAt: 'now',
    })),
    startRun: vi.fn(async (_testId: string, _signal?: AbortSignal): Promise<Run> => failedRun),
    getRun: vi.fn(async (_runId: string, _signal?: AbortSignal): Promise<Run> => failedRun),
    inspectFailure: vi.fn(async (_runId: string, _signal?: AbortSignal): Promise<FailureAnalysis> => ({
      step: 'checkout',
      failedSelector: '#checkout-submit',
      fallbackSelector: '[data-testid=\"checkout-submit\"]',
      explanation: 'stable selector observed on page',
      recoverable: true,
    })),
    retryRun: vi.fn(async (_runId: string, _signal?: AbortSignal): Promise<Run> => succeededRun),
    exportRun: vi.fn(async (_runId: string, _signal?: AbortSignal) => ({ code: "import { test } from '@playwright/test';" })),
  };
}

function makeRun(status: Run['status']): Run {
  return {
    id: 'run-1',
    testId: 'test-1',
    targetUrl: '/demo-store',
    objective: 'checkout',
    status,
    stepCount: 1,
    maxSteps: 4,
    events: [],
    evidence: [],
    createdAt: 'now',
    updatedAt: 'now',
  };
}
