import { Check, CircleAlert, Clock3, LoaderCircle, Square } from 'lucide-react';
import { useEffect, useState } from 'react';
import {
  activeRun,
  cancelledRun,
  duration,
  stageLabels,
  stageProgress,
  stages,
} from '../lib/generation';
import { generationMessage } from '../lib/generationMessages';
import { AnalysisProgress } from './AnalysisProgress';
import { CompositionProgress } from './CompositionProgress';
import { ResearchProgress } from './ResearchProgress';
import { Badge } from './ui';
import type { Run } from '../lib/api';

export function GenerationBadges({
  run,
  pending,
  version,
  hasDraft,
}: Readonly<{
  run: Run | undefined;
  pending: boolean;
  version: number | undefined;
  hasDraft: boolean;
}>) {
  const busy = pending || activeRun(run?.status);
  let runLabel = 'Generating first draft';
  if (hasDraft) {
    runLabel = 'Generating a new draft';
  }
  if (run?.status === 'queued') {
    runLabel = 'Queued for generation';
  }
  return (
    <div className="generation-badges" aria-live="polite">
      {hasDraft && (
        <Badge tone="warning">
          {version === undefined ? 'Draft available' : `Draft v${version}`} · review pending
        </Badge>
      )}
      {busy && (
        <Badge tone="active">
          <Clock3 size={14} aria-hidden="true" />
          {runLabel}
        </Badge>
      )}
      {cancelledRun(run) && !busy && (
        <Badge>
          <Square size={14} aria-hidden="true" />
          Generation cancelled
        </Badge>
      )}
      {run?.status === 'failed' && !cancelledRun(run) && !busy && (
        <Badge tone="error">
          <CircleAlert size={14} aria-hidden="true" />
          Generation failed
        </Badge>
      )}
      {!hasDraft && !busy && run?.status !== 'failed' && (
        <Badge>
          {run?.status === 'succeeded' ? 'Draft ready · review pending' : 'Not generated yet'}
        </Badge>
      )}
    </div>
  );
}

function progressTitle(run: Run): string {
  if (cancelledRun(run)) {
    return 'Generation cancelled';
  }
  if (run.status === 'failed') {
    return 'Generation stopped';
  }
  return stageLabels[run.stage] ?? 'Generating the passport';
}

function connectionLabel(state: string): string {
  if (state === 'live') {
    return 'Live updates connected';
  }
  if (state === 'reconnecting') {
    return 'Reconnecting to live updates · checking every 5s';
  }
  return 'Connecting to live updates';
}

export function GenerationProgress({
  run,
  streamState,
}: Readonly<{ run: Run; streamState: string }>) {
  const [now, setNow] = useState(Date.now);
  const busy = activeRun(run.status);
  useEffect(() => {
    if (!busy) {
      return;
    }
    const timer = globalThis.setInterval(() => setNow(Date.now()), 1000);
    return () => globalThis.clearInterval(timer);
  }, [busy]);
  const end = run.finished_at === '' ? now : Date.parse(run.finished_at);
  const latest = run.events.at(-1);
  const analysis = run.analysis ?? undefined;
  return (
    <section className="generation-progress" aria-label="Generation progress">
      <div className="progress-heading">
        <strong role="status">{progressTitle(run)}</strong>
        <span className="meta">
          {run.document_count} source documents ·{' '}
          {(run.input_tokens + run.output_tokens).toLocaleString()} recorded tokens
        </span>
      </div>
      <div className="progress-timing">
        <span>Total elapsed {duration(run.created_at, end)}</span>
        {busy && run.stage_started_at !== '' && (
          <span>Current stage {duration(run.stage_started_at, end)}</span>
        )}
        {run.status === 'queued' && <span>Waiting for an available worker</span>}
      </div>
      <ol className="stage-route" aria-label="Generation stages">
        {stages.map((stage, index) => {
          const { status, timing } = stageProgress(run, stage, end);
          const current = status === 'Running' || status === 'Repairing';
          return (
            <li
              className={`stage ${current ? 'current' : ''}`}
              key={stage}
              aria-current={current ? 'step' : undefined}
            >
              <span className="stage-dot" aria-hidden="true">
                {status === 'Completed' ? <Check size={12} /> : index + 1}
              </span>
              <span className="stage-description">
                <span>{stageLabels[stage]}</span>
                <span className="stage-state">
                  {status}
                  {timing === '' ? '' : ` · ${timing}`}
                </span>
              </span>
            </li>
          );
        })}
      </ol>
      {run.research !== undefined && (
        <ResearchProgress progress={run.research} running={busy && run.stage === 'researching'} />
      )}
      {analysis !== undefined && (
        <AnalysisProgress
          analysis={analysis}
          events={run.events}
          stage={run.stage}
          now={end}
          running={busy && run.stage === 'analyzing'}
        />
      )}
      {run.composition !== undefined && run.composition !== null && (
        <CompositionProgress
          composition={run.composition}
          now={end}
          running={busy && run.stage === 'composing'}
        />
      )}
      {latest !== undefined &&
        run.stage !== 'composing' &&
        (run.research === undefined || run.stage !== 'researching') && (
          <p className="progress-latest">{generationMessage(latest.message)}</p>
        )}
      <div className="progress-connection">
        {busy && <LoaderCircle size={14} aria-hidden="true" />}
        <span role="status">
          {busy ? connectionLabel(streamState) : 'Run finished · activity saved'}
        </span>
        {busy && latest !== undefined && (
          <span>Last worker activity {duration(latest.created_at, now)} ago</span>
        )}
      </div>
      <details className="generation-activity">
        <summary>Activity ({run.events.length})</summary>
        {run.events.length === 0 && <p className="meta">No saved activity for this run.</p>}
        <ol aria-label="Worker activity, newest first">
          {run.events.toReversed().map((event) => (
            <li key={event.id}>
              <time dateTime={event.created_at}>
                {new Date(event.created_at).toLocaleTimeString('en-GB')}
              </time>
              <div>
                <span className="activity-stage">
                  {generationMessage(stageLabels[event.stage] ?? event.stage)}
                </span>
                <p>{generationMessage(event.message)}</p>
              </div>
            </li>
          ))}
        </ol>
      </details>
    </section>
  );
}
