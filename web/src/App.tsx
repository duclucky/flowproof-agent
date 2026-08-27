import { useEffect, useMemo, useState } from 'react';
import type { FormEvent } from 'react';
import { flowProofClient } from './api';
import type { FlowProofClient } from './api';
import type { FailureAnalysis, Run, TestDefinition } from './types';
import { registerFlowProofTools } from './webmcp';
import './styles.css';

const DEFAULT_OBJECTIVE = 'Verify checkout recovers from a stale selector and confirms the order';
const ACTIVE_STATUSES: Run['status'][] = ['queued', 'running'];

export interface AppProps {
  client?: FlowProofClient;
  pollIntervalMs?: number;
}

export function App({ client = flowProofClient, pollIntervalMs = 1500 }: AppProps) {
  const [targetUrl, setTargetUrl] = useState('/demo-store');
  const [objective, setObjective] = useState(DEFAULT_OBJECTIVE);
  const [testDefinition, setTestDefinition] = useState<TestDefinition | null>(null);
  const [run, setRun] = useState<Run | null>(null);
  const [analysis, setAnalysis] = useState<FailureAnalysis | null>(null);
  const [exportCode, setExportCode] = useState('');
  const [webMCP, setWebMCP] = useState<'checking' | 'available' | 'unavailable'>('checking');
  const [busy, setBusy] = useState('');
  const [error, setError] = useState('');

  useEffect(() => {
    let alive = true;
    void registerFlowProofTools(client)
      .then((state) => { if (alive) setWebMCP(state.supported ? 'available' : 'unavailable'); })
      .catch(() => { if (alive) setWebMCP('unavailable'); });
    return () => { alive = false; };
  }, [client]);

  useEffect(() => {
    if (!run || !ACTIVE_STATUSES.includes(run.status)) return;
    const controller = new AbortController();
    const timer = window.setTimeout(async () => {
      try {
        const next = await client.getRun(run.id, controller.signal);
        if (!controller.signal.aborted) setRun(next);
      } catch (cause) {
        if (!controller.signal.aborted) setError(messageOf(cause));
      }
    }, pollIntervalMs);
    return () => {
      controller.abort();
      window.clearTimeout(timer);
    };
  }, [client, pollIntervalMs, run]);

  const statusLabel = useMemo(() => run ? run.status.replaceAll('_', ' ') : 'not started', [run]);

  async function createTest(event?: FormEvent) {
    event?.preventDefault();
    await perform('Creating test', async (signal) => {
      const created = await client.createTest({ targetUrl, objective }, signal);
      setTestDefinition(created);
      setRun(null);
      setAnalysis(null);
      setExportCode('');
    });
  }

  async function startRun() {
    if (!testDefinition) return;
    await perform('Starting run', async (signal) => {
      const started = await client.startRun(testDefinition.id, signal);
      setRun(started);
      setAnalysis(null);
      setExportCode('');
      if (started.status === 'failed_recoverable') {
        setAnalysis(await client.inspectFailure(started.id, signal));
      }
    });
  }

  async function retryFailedStep() {
    if (!run) return;
    await perform('Retrying verified fallback', async (signal) => {
      const recovered = await client.retryRun(run.id, signal);
      setRun(recovered);
      setAnalysis(null);
    });
  }

  async function exportRegression() {
    if (!run) return;
    await perform('Exporting regression', async (signal) => {
      const result = await client.exportRun(run.id, signal);
      setExportCode(result.code);
    });
  }

  async function perform(label: string, operation: (signal: AbortSignal) => Promise<void>) {
    const controller = new AbortController();
    setBusy(label);
    setError('');
    try {
      await operation(controller.signal);
    } catch (cause) {
      setError(messageOf(cause));
    } finally {
      setBusy('');
    }
  }

  return (
    <main className="shell">
      <header className="topbar">
        <div className="brand-lockup">
          <Logo />
          <div><p className="eyebrow">Agent-native browser QA</p><h1>FlowProof</h1></div>
        </div>
        <div className={`support-badge support-${webMCP}`} role="status">
          <span className="support-dot" aria-hidden="true" />
          {webMCP === 'available' ? 'WebMCP available' : webMCP === 'checking' ? 'Checking WebMCP' : 'WebMCP unavailable · manual mode'}
        </div>
      </header>

      <section className="hero-grid">
        <div className="hero-copy">
          <p className="kicker">STRUCTURED QA · EVIDENCE · RECOVERY</p>
          <h2>Turn a brittle browser failure into a verified regression test.</h2>
          <p>FlowProof gives browser agents six narrow WebMCP tools to create, run, diagnose, safely retry, and export deterministic QA.</p>
        </div>
        <div className="run-meter" aria-label="Run state">
          <span>RUN STATE</span><strong>{statusLabel}</strong>
          <div className="meter-track"><i style={{ width: `${progressFor(run)}%` }} /></div>
        </div>
      </section>

      {error && <div className="alert" role="alert"><strong>Operation failed</strong><span>{error}</span></div>}

      <section className="workspace-grid">
        <article className="panel control-panel">
          <div className="panel-heading"><div><p className="section-index">01 / DEFINE</p><h3>Test contract</h3></div><span className="mono">same-origin demo</span></div>
          <form onSubmit={createTest} className="test-form">
            <label>Target URL<input value={targetUrl} onChange={(event) => setTargetUrl(event.target.value)} aria-label="Target URL" /></label>
            <label>Objective<textarea value={objective} onChange={(event) => setObjective(event.target.value)} aria-label="Objective" rows={4} /></label>
            <button className="primary" type="submit" disabled={Boolean(busy)}>Create test</button>
          </form>
          <div className="identity-strip"><span>TEST ID</span><code>{testDefinition?.id ?? '—'}</code></div>
          <button className="secondary wide" type="button" disabled={!testDefinition || Boolean(busy)} onClick={() => void startRun()}>Start run</button>
          {busy && <p className="busy" role="status"><span className="spinner" />{busy}</p>}
        </article>

        <article className="panel execution-panel">
          <div className="panel-heading"><div><p className="section-index">02 / EXECUTE</p><h3>Live run</h3></div>{run && <StatusPill status={run.status} />}</div>
          {!run ? <Empty title="No run yet" body="Create a test contract and start the controlled checkout workflow." /> : (
            <>
              <div className="metrics">
                <Metric label="Run ID" value={run.id} mono />
                <Metric label="Steps" value={`${run.stepCount}/${run.maxSteps}`} />
                <Metric label="Current URL" value={run.currentUrl ?? run.targetUrl} mono />
              </div>
              {run.status === 'failed_recoverable' && (
                <section className="failure-card" aria-label="Recoverable failure">
                  <div className="failure-title"><FailureIcon /><div><p>RECOVERABLE FAILURE</p><h4>Selector stale — evidence preserved</h4></div></div>
                  <dl>
                    <div><dt>Attempted</dt><dd><code>{analysis?.failedSelector ?? run.failureAnalysis?.failedSelector ?? '#checkout-submit'}</code></dd></div>
                    <div><dt>Verified fallback</dt><dd><code>{analysis?.fallbackSelector ?? run.failureAnalysis?.fallbackSelector ?? '—'}</code></dd></div>
                  </dl>
                  <p>{analysis?.explanation ?? run.failureAnalysis?.explanation ?? run.failure}</p>
                  <button className="danger-action" type="button" disabled={Boolean(busy)} onClick={() => void retryFailedStep()}>Retry failed step</button>
                </section>
              )}
              {run.status === 'succeeded' && (
                <section className="success-card"><SuccessIcon /><div><p>RUN SUCCEEDED</p><h4>{run.summary ?? 'Order confirmed · FP-2048'}</h4><code>{run.recoveredSelector}</code></div></section>
              )}
            </>
          )}
        </article>
      </section>

      <section className="lower-grid">
        <article className="panel timeline-panel">
          <div className="panel-heading"><div><p className="section-index">03 / EVIDENCE</p><h3>Execution timeline</h3></div><span className="mono">{run?.events.length ?? 0} events</span></div>
          {!run?.events.length ? <Empty title="Timeline waiting" body="Events appear here as the browser workflow advances." /> : (
            <ol className="timeline">{run.events.map((event) => <li key={`${event.seq}-${event.at}`}><span>{String(event.seq).padStart(2, '0')}</span><div><strong>{event.type.replaceAll('_', ' ')}</strong><p>{event.message}</p>{event.error && <code>{event.error}</code>}</div><time>{timeLabel(event.at)}</time></li>)}</ol>
          )}
        </article>
        <article className="panel evidence-panel">
          <div className="panel-heading"><div><p className="section-index">04 / ARTIFACTS</p><h3>Captured evidence</h3></div><span className="mono">{run?.evidence?.length ?? 0} items</span></div>
          {!run?.evidence?.length ? <Empty title="No evidence yet" body="DOM/text snapshots and screenshots remain attached to the run." /> : (
            <div className="evidence-list">{run.evidence.map((item, index) => <div className="evidence-item" key={`${item.step}-${item.label}-${index}`}><div><span>{item.kind}</span><strong>{item.label}</strong></div>{item.dataUrl ? <img src={item.dataUrl} alt={`${item.label} evidence`} /> : <pre>{item.text ?? 'Evidence captured'}</pre>}</div>)}</div>
          )}
        </article>
      </section>

      <section className="panel export-panel">
        <div className="panel-heading"><div><p className="section-index">05 / REGRESSION</p><h3>Playwright export</h3></div><button className="secondary" type="button" disabled={run?.status !== 'succeeded' || Boolean(busy)} onClick={() => void exportRegression()}>Export regression test</button></div>
        {exportCode ? <pre className="code-panel"><code>{exportCode}</code></pre> : <Empty title="Export locked" body="Recover the failed run first. The generated test will use the verified stable selector." />}
      </section>

      <footer><span>FLOWPROOF / WEBMCP CHALLENGE</span><span>Deterministic demo · no external model key</span></footer>
    </main>
  );
}

function Metric({ label, value, mono = false }: { label: string; value: string; mono?: boolean }) { return <div className="metric"><span>{label}</span><strong className={mono ? 'mono' : ''}>{value}</strong></div>; }
function Empty({ title, body }: { title: string; body: string }) { return <div className="empty"><span className="empty-mark" /><div><strong>{title}</strong><p>{body}</p></div></div>; }
function StatusPill({ status }: { status: Run['status'] }) { return <span className={`status-pill status-${status}`}>{status.replaceAll('_', ' ')}</span>; }
function progressFor(run: Run | null) { if (!run) return 0; if (run.status === 'succeeded') return 100; if (run.status === 'failed_recoverable') return 72; if (run.status === 'running') return 46; if (run.status === 'queued') return 18; return 100; }
function timeLabel(raw: string) { const value = new Date(raw); return Number.isNaN(value.getTime()) ? raw : value.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' }); }
function messageOf(cause: unknown) { return cause instanceof Error ? cause.message : String(cause); }
function Logo() { return <svg className="logo" viewBox="0 0 40 40" aria-hidden="true"><path d="M7 11.5 20 4l13 7.5v17L20 36 7 28.5z" /><path d="m13 20 4.5 4.5L28 14" /></svg>; }
function FailureIcon() { return <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 3 2.8 19h18.4L12 3Z" /><path d="M12 9v4M12 16.5v.1" /></svg>; }
function SuccessIcon() { return <svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="12" r="9" /><path d="m8 12 2.5 2.5L16 9" /></svg>; }
