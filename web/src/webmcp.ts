import type { FlowProofClient } from './api';

export const FLOWPROOF_TOOL_NAMES = [
  'create_test',
  'start_run',
  'get_run_status',
  'inspect_failure',
  'retry_failed_step',
  'export_regression_test',
] as const;

type ToolName = (typeof FLOWPROOF_TOOL_NAMES)[number];

interface ToolSchema {
  type: 'object';
  properties: Record<string, { type: 'string'; description: string }>;
  required: string[];
  additionalProperties: false;
}

interface ToolDefinition {
  name: ToolName;
  description: string;
  inputSchema: ToolSchema;
  annotations: { readOnlyHint: boolean; untrustedContentHint: false };
  execute: (input: Record<string, string>, context?: { signal: AbortSignal }) => Promise<string>;
}

interface ModelContextLike {
  registerTool(tool: ToolDefinition): Promise<void> | void;
}

declare global {
  interface Document {
    modelContext?: ModelContextLike;
  }
}

export interface WebMCPRegistrationState {
  supported: boolean;
  names: ToolName[];
}

export async function registerFlowProofTools(client: FlowProofClient): Promise<WebMCPRegistrationState> {
  const modelContext = document.modelContext;
  if (!modelContext) return { supported: false, names: [] };

  const tools = buildTools(client);
  for (const tool of tools) await modelContext.registerTool(tool);
  return { supported: true, names: tools.map((tool) => tool.name) };
}

function buildTools(client: FlowProofClient): ToolDefinition[] {
  return [
    tool(
      'create_test',
      'Create a deterministic FlowProof browser QA test for a validated target URL and objective.',
      schema(
        {
          targetUrl: { type: 'string', description: 'Same-origin demo URL or explicitly allowed target URL.' },
          objective: { type: 'string', description: 'Human-readable browser QA objective.' },
        },
        ['targetUrl', 'objective'],
      ),
      false,
      async (input, signal) => client.createTest({ targetUrl: input.targetUrl, objective: input.objective }, signal),
    ),
    tool(
      'start_run',
      'Start the deterministic browser QA run for a previously created FlowProof test definition.',
      schema({ testId: { type: 'string', description: 'FlowProof test identifier returned by create_test.' } }, ['testId']),
      false,
      async (input, signal) => client.startRun(input.testId, signal),
    ),
    tool(
      'get_run_status',
      'Read the current FlowProof run state, event timeline, evidence summary, and recovery status.',
      schema({ runId: { type: 'string', description: 'FlowProof run identifier.' } }, ['runId']),
      true,
      async (input, signal) => client.getRun(input.runId, signal),
    ),
    tool(
      'inspect_failure',
      'Inspect structured evidence and the verified safe fallback for a recoverable browser QA failure.',
      schema({ runId: { type: 'string', description: 'Recoverable FlowProof run identifier.' } }, ['runId']),
      true,
      async (input, signal) => client.inspectFailure(input.runId, signal),
    ),
    tool(
      'retry_failed_step',
      'Retry the failed deterministic browser step using only the fallback selector verified from page evidence.',
      schema({ runId: { type: 'string', description: 'Recoverable FlowProof run identifier.' } }, ['runId']),
      false,
      async (input, signal) => client.retryRun(input.runId, signal),
    ),
    tool(
      'export_regression_test',
      'Export the succeeded FlowProof run as a complete Playwright TypeScript regression test.',
      schema({ runId: { type: 'string', description: 'Succeeded FlowProof run identifier.' } }, ['runId']),
      true,
      async (input, signal) => client.exportRun(input.runId, signal),
    ),
  ];
}

function tool(
  name: ToolName,
  description: string,
  inputSchema: ToolSchema,
  readOnlyHint: boolean,
  execute: (input: Record<string, string>, signal?: AbortSignal) => Promise<unknown>,
): ToolDefinition {
  return {
    name,
    description,
    inputSchema,
    annotations: { readOnlyHint, untrustedContentHint: false },
    execute: async (input, context) => JSON.stringify(await execute(input, context?.signal)),
  };
}

function schema(
  properties: ToolSchema['properties'],
  required: string[],
): ToolSchema {
  return { type: 'object', properties, required, additionalProperties: false };
}
