export type RunStatus = 'queued' | 'running' | 'failed_recoverable' | 'succeeded' | 'failed' | 'cancelled';

export interface TestDefinition {
  id: string;
  targetUrl: string;
  objective: string;
  createdAt: string;
}

export interface FailureAnalysis {
  step: string;
  failedSelector: string;
  fallbackSelector: string;
  explanation: string;
  recoverable: boolean;
}

export interface RunEvent {
  seq: number;
  at: string;
  type: string;
  message: string;
  error?: string;
}

export interface Evidence {
  step: number;
  kind: string;
  label: string;
  dataUrl?: string;
  text?: string;
  capturedAt: string;
}

export interface Run {
  id: string;
  testId?: string;
  targetUrl: string;
  objective: string;
  status: RunStatus;
  currentUrl?: string;
  stepCount: number;
  maxSteps: number;
  summary?: string;
  failure?: string;
  failureAnalysis?: FailureAnalysis;
  recoveredSelector?: string;
  events: RunEvent[];
  evidence?: Evidence[];
  createdAt: string;
  updatedAt: string;
  completedAt?: string;
}
