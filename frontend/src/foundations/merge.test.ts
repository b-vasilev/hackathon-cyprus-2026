import assert from 'node:assert/strict';
import test from 'node:test';
import { mergeStates, type LearningState } from './merge.ts';

const module = (
  overrides: Partial<LearningState['modules'][string]> = {}
): LearningState['modules'][string] => ({
  lessons: [],
  quiz_passed: false,
  visited: false,
  best_score: 0,
  failed_quiz: false,
  comeback: false,
  ...overrides,
});

test('merging keeps everything either copy has learned', () => {
  const account: LearningState = {
    modules: { git: module({ lessons: ['commits'], best_score: 0.6, failed_quiz: true }) },
    flags: ['curious'],
    unlocked: { 'first-step': '2026-10-15T10:00:00Z' },
  };
  const browser: LearningState = {
    modules: {
      git: module({
        lessons: ['commits', 'rebase'],
        quiz_passed: true,
        best_score: 1,
        comeback: true,
      }),
      http: module({ visited: true }),
    },
    flags: ['curious', 'off-route'],
    unlocked: { 'first-step': '2026-10-15T09:00:00Z', perfect: '2026-10-15T11:00:00Z' },
  };
  const merged = mergeStates(account, browser);
  assert.deepEqual(merged.modules.git, {
    lessons: ['commits', 'rebase'],
    quiz_passed: true,
    visited: false,
    best_score: 1,
    failed_quiz: true,
    comeback: true,
  });
  assert.deepEqual(merged.modules.http, module({ visited: true }));
  assert.deepEqual(merged.flags, ['curious', 'off-route']);
  // The earliest unlock date wins.
  assert.deepEqual(merged.unlocked, {
    'first-step': '2026-10-15T09:00:00Z',
    perfect: '2026-10-15T11:00:00Z',
  });
});

test('merging tolerates empty lists sent as null', () => {
  const merged = mergeStates(
    { modules: { git: module({ lessons: null }) }, flags: null, unlocked: {} },
    { modules: { git: module({ lessons: ['commits'] }) }, flags: null, unlocked: {} }
  );
  assert.deepEqual(merged.modules.git.lessons, ['commits']);
  assert.deepEqual(merged.flags, []);
});
