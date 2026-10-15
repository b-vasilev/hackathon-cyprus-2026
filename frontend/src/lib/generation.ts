import type { Run } from './api';

export const stages = [
  'collecting',
  'selecting_sources',
  'researching',
  'analyzing',
  'synthesizing',
  'composing',
  'validating',
];

export const stageLabels: Record<string, string> = {
  queued: 'Waiting for the worker',
  collecting: 'Collecting source files',
  selecting_sources: 'Selecting source files',
  researching: 'Exploring repository code',
  analyzing: 'Analyzing evidence',
  synthesizing: 'Reconciling the product model',
  composing: 'Preparing passport sections',
  validating: 'Checking the passport',
  repairing: 'Repairing the passport',
  complete: 'Passport draft saved',
};

export function activeRun(status: string | undefined): boolean {
  return status === 'queued' || status === 'running';
}

export function cancelledRun(run: Run | undefined): boolean {
  return run?.status === 'failed' && run.error_code === 'cancelled';
}

// Both HTTP snapshots and reconnect replay can overlap with events already seen.
export function mergeRun(previous: Run | undefined, incoming: Run): Run {
  if (previous?.id !== incoming.id) {
    return incoming;
  }
  const previousId = previous.events.at(-1)?.id ?? 0;
  const incomingId = incoming.events.at(-1)?.id ?? 0;
  if (previousId > incomingId || (!activeRun(previous.status) && activeRun(incoming.status))) {
    return previous;
  }
  const events = new Map(previous.events.map((event) => [event.id, event]));
  for (const event of incoming.events) {
    events.set(event.id, event);
  }
  return { ...incoming, events: [...events.values()].sort((left, right) => left.id - right.id) };
}

export function duration(start: string, end: number): string {
  if (start === '') {
    return '—';
  }
  return elapsed(Math.max(0, end - Date.parse(start)));
}

export function elapsed(milliseconds: number): string {
  const seconds = Math.floor(milliseconds / 1000);
  if (seconds < 60) {
    return `${seconds}s`;
  }
  return `${Math.floor(seconds / 60)}m ${seconds % 60}s`;
}

function routeStage(stage: string): string {
  return stage === 'repairing' ? 'validating' : stage;
}

const eventStates: Record<string, string | undefined> = {
  stage_started: 'Running',
  stage_completed: 'Completed',
  stage_skipped: 'Skipped',
  validation_failed: 'Needs repair',
  run_failed: 'Failed',
  run_cancelled: 'Cancelled',
};

function stageState(run: Run, stage: string, events: Run['events']): string {
  if (routeStage(run.stage) === stage) {
    if (run.status === 'failed') {
      return cancelledRun(run) ? 'Cancelled' : 'Failed';
    }
    if (run.status === 'running') {
      return run.stage === 'repairing' ? 'Repairing' : 'Running';
    }
  }
  const last = events.findLast((event) => eventStates[event.type] !== undefined);
  if (last !== undefined) {
    return eventStates[last.type] ?? 'Pending';
  }
  if (activeRun(run.status)) {
    return 'Pending';
  }
  return run.events.length === 0 ? 'Not recorded' : 'Not started';
}

function stageTiming(events: Run['events'], now: number): string {
  let started: number | undefined;
  let milliseconds = 0;
  let hasTiming = false;
  for (const event of events) {
    if (event.type === 'stage_started') {
      started = Date.parse(event.created_at);
      hasTiming = true;
    }
    if (
      ['stage_completed', 'validation_failed', 'run_failed', 'run_cancelled'].includes(
        event.type
      ) &&
      started !== undefined
    ) {
      milliseconds += Math.max(0, Date.parse(event.created_at) - started);
      started = undefined;
    }
  }
  if (started !== undefined) {
    milliseconds += Math.max(0, now - started);
  }
  return hasTiming ? elapsed(milliseconds) : '';
}

export function stageProgress(run: Run, stage: string, now: number) {
  const events = run.events.filter((event) => routeStage(event.stage) === stage);
  return { status: stageState(run, stage, events), timing: stageTiming(events, now) };
}
