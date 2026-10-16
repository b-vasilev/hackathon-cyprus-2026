import assert from 'node:assert/strict';
import test from 'node:test';
import { renderToStaticMarkup } from 'react-dom/server';
import { PracticeLearner } from './LearningPack';
import type { LearningPack } from '../lib/api';

function publishedPack(overrides: Partial<LearningPack> = {}): LearningPack {
  const question = {
    id: 'q1',
    chapter_id: 'overview',
    kind: 'single',
    prompt: 'Which component handles the request?',
    options: ['API', 'Worker'],
    correct: [0],
    explanation: 'The API receives the request.',
  };
  return {
    id: 'pack-1',
    product_id: 'product',
    revision_id: 'revision-1',
    revision_version: 1,
    status: 'ready',
    error: '',
    updated_at: '2026-10-16T10:00:00Z',
    published: true,
    published_at: '2026-10-16T10:00:00Z',
    unpublished_changes: false,
    stale: false,
    content: {
      quiz: [question],
      mission: {
        title: 'Study the request flow',
        intro: 'Follow a request through the product.',
        steps: [
          {
            id: 's1',
            title: 'Read the handler',
            body: 'Find where the request is handled.',
            chapter_id: 'overview',
            commands: [],
            done_when: 'You found the handler.',
          },
        ],
        check: [{ ...question, id: 'check-1' }],
        finale: 'You are ready to go!',
      },
    },
    ...overrides,
  };
}

test('practice progress belongs to one published pack', async (t) => {
  const originalStorage = Object.getOwnPropertyDescriptor(globalThis, 'localStorage');
  const storage = new Map<string, string>();
  Object.defineProperty(globalThis, 'localStorage', {
    configurable: true,
    value: { getItem: (key: string) => storage.get(key) ?? null },
  });
  t.after(() => {
    if (originalStorage) {
      Object.defineProperty(globalThis, 'localStorage', originalStorage);
    } else {
      Reflect.deleteProperty(globalThis, 'localStorage');
    }
  });

  const first = publishedPack();
  function complete(prefix = 'practice:pack-1:2026-10-16T10:00:00Z') {
    storage.set(
      'of.foundations.progress',
      JSON.stringify({
        [`${prefix}:quiz`]: { quizPassed: true },
        [`${prefix}:task`]: { quizPassed: true, lessons: ['s1'] },
      })
    );
  }
  function render(pack: LearningPack) {
    return renderToStaticMarkup(<PracticeLearner pack={pack} chapters={[]} onChapter={() => {}} />);
  }
  function assertFresh(pack: LearningPack) {
    const html = render(pack);
    assert.ok(html.includes('Which component handles the request?'));
    assert.ok(!html.includes('>Passed</'));
    assert.ok(!html.includes('Mission complete'));
  }

  await t.test('legacy product progress does not pass a new practice', () => {
    complete('product:product');
    assertFresh(first);
  });
  await t.test('reopening the same publication keeps both quizzes and steps complete', () => {
    complete();
    const html = render(first);
    assert.equal((html.match(/>Passed</g) ?? []).length, 2);
    assert.ok(html.includes('Mission complete'));
  });
  await t.test('another passport revision starts fresh even with the same step IDs', () => {
    complete();
    assertFresh(publishedPack({ id: 'pack-2', revision_id: 'revision-2', revision_version: 2 }));
  });
  await t.test('new practice for the same passport starts fresh', () => {
    complete();
    assertFresh(publishedPack({ published_at: '2026-10-16T11:00:00Z' }));
  });
  await t.test('an unpublished regeneration keeps the published practice progress', () => {
    complete();
    const html = render(
      publishedPack({ updated_at: '2026-10-16T12:00:00Z', unpublished_changes: true })
    );
    assert.equal((html.match(/>Passed</g) ?? []).length, 2);
    assert.ok(html.includes('Mission complete'));
  });
});
