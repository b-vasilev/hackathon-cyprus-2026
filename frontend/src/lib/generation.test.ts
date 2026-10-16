import assert from 'node:assert/strict';
import test from 'node:test';
import { cancelledRun, duration, mergeRun, stageProgress } from './generation.ts';
import type { Run } from './api';

const timestamp = (second: number) => new Date(Date.UTC(2026, 9, 15, 12, 0, second)).toISOString();

function event(id: number, type: string, stage: string, second: number): Run['events'][number] {
  return { id, type, stage, created_at: timestamp(second), message: type };
}

function run(overrides: Partial<Run> = {}): Run {
  return {
    id: 'run-one',
    product_id: 'product-one',
    connection_id: 'connection-one',
    connection_revision: 1,
    model: 'fixture',
    request_timeout_seconds: 60,
    status: 'queued',
    stage: 'queued',
    error_code: '',
    error_message: '',
    created_at: timestamp(0),
    started_at: '',
    stage_started_at: '',
    finished_at: '',
    document_count: 0,
    input_tokens: 0,
    output_tokens: 0,
    events: [],
    ...overrides,
  };
}

test('reconnect replay merges events once and ignores older HTTP snapshots', () => {
  const first = event(1, 'run_queued', 'queued', 0);
  const second = event(2, 'stage_started', 'collecting', 1);
  const previous = run({ events: [first, second], status: 'running', stage: 'collecting' });
  const third = event(3, 'stage_completed', 'collecting', 3);
  const incoming = run({ events: [second, third], status: 'running', stage: 'analyzing' });
  const merged = mergeRun(previous, incoming);
  assert.deepEqual(
    merged.events.map((item) => item.id),
    [1, 2, 3]
  );
  assert.deepEqual(mergeRun(merged, incoming), merged);
  assert.equal(mergeRun(merged, previous), merged);
  assert.equal(mergeRun(merged, run()), merged);
});

test('terminal runs cannot be replaced by a delayed active snapshot', () => {
  const finished = run({ status: 'failed', finished_at: timestamp(3) });
  assert.equal(mergeRun(finished, run({ status: 'running' })), finished);
  assert.equal(mergeRun(finished, run({ id: 'run-two' })).id, 'run-two');
});

test('a resumed run with a newer event replaces its failed state', () => {
  const failedEvent = event(5, 'run_failed', 'analyzing', 5);
  const failed = run({ status: 'failed', stage: 'analyzing', events: [failedEvent] });
  const resumed = run({
    status: 'queued',
    stage: 'queued',
    events: [failedEvent, event(6, 'run_resumed', 'queued', 9)],
  });
  const merged = mergeRun(failed, resumed);
  assert.equal(merged.status, 'queued');
  assert.deepEqual(
    merged.events.map((item) => item.id),
    [5, 6]
  );
  // An active snapshot that is not newer still cannot revive the failed run.
  assert.equal(mergeRun(failed, run({ status: 'running', events: [failedEvent] })), failed);
});

test('cancellation stops stage timing and remains terminal after delayed updates', () => {
  const cancelled = run({
    status: 'failed',
    error_code: 'cancelled',
    stage: 'analyzing',
    finished_at: timestamp(5),
    events: [event(1, 'stage_started', 'analyzing', 1), event(2, 'run_cancelled', 'analyzing', 5)],
  });
  assert.equal(cancelledRun(cancelled), true);
  assert.equal(cancelledRun(run({ status: 'failed', error_code: 'provider_error' })), false);
  assert.deepEqual(stageProgress(cancelled, 'analyzing', Date.parse(timestamp(20))), {
    status: 'Cancelled',
    timing: '4s',
  });
  assert.equal(mergeRun(cancelled, run({ status: 'running' })), cancelled);
});

test('queued and skipped stages do not imply completed work', () => {
  assert.deepEqual(stageProgress(run(), 'analyzing', Date.parse(timestamp(3))), {
    status: 'Pending',
    timing: '',
  });
  const skipped = run({ events: [event(1, 'stage_skipped', 'selecting_sources', 1)] });
  assert.equal(
    stageProgress(skipped, 'selecting_sources', Date.parse(timestamp(3))).status,
    'Skipped'
  );
});

test('validation timing includes repairs and repeated validation', () => {
  const repairing = run({
    status: 'running',
    stage: 'repairing',
    events: [
      event(1, 'stage_started', 'validating', 1),
      event(2, 'validation_failed', 'validating', 2),
      event(3, 'stage_started', 'repairing', 2),
    ],
  });
  assert.deepEqual(stageProgress(repairing, 'validating', Date.parse(timestamp(5))), {
    status: 'Repairing',
    timing: '4s',
  });
  const finished = run({
    status: 'succeeded',
    stage: 'complete',
    events: [
      ...repairing.events,
      event(4, 'stage_completed', 'repairing', 5),
      event(5, 'stage_started', 'validating', 5),
      event(6, 'stage_completed', 'validating', 6),
    ],
  });
  assert.deepEqual(stageProgress(finished, 'validating', Date.parse(timestamp(30))), {
    status: 'Completed',
    timing: '5s',
  });
});

test('failed stage timing stops at the failure event', () => {
  const failed = run({
    status: 'failed',
    stage: 'analyzing',
    events: [event(1, 'stage_started', 'analyzing', 1), event(2, 'run_failed', 'analyzing', 4)],
  });
  assert.deepEqual(stageProgress(failed, 'analyzing', Date.parse(timestamp(30))), {
    status: 'Failed',
    timing: '3s',
  });
  assert.equal(duration('', Date.parse(timestamp(30))), '—');
  assert.equal(stageProgress(failed, 'composing', Date.parse(timestamp(30))).status, 'Not started');
  assert.equal(
    stageProgress(run({ status: 'succeeded' }), 'analyzing', Date.parse(timestamp(30))).status,
    'Not recorded'
  );
});
