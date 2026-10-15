import assert from 'node:assert/strict';
import test from 'node:test';
import { renderToStaticMarkup } from 'react-dom/server';
import { orderedResearchAreas, researchCounts } from '../lib/researchProgress';
import { ResearchProgress } from './ResearchProgress';
import type { ResearchProgress as ResearchState } from '../lib/researchProgress';

const progress: ResearchState = {
  status: 'in_progress',
  focus: {
    area: 'Payments',
    question: 'How are repeated callbacks handled?',
    reason: 'Check whether a repeated request can charge twice.',
  },
  activity: 'Reading source code (2 ranges).',
  last_result: 'The callback handler rejects repeated payment identifiers.',
  budget: {
    used: 46,
    limit: 50,
    hard_limit: 120,
    deadline: '2026-10-15T12:30:00Z',
    decisions: [
      {
        turn: 40,
        limit: 50,
        kind: 'extended',
        reason: 'Added 10 turns: new source evidence was verified; 2 reachable questions remain.',
      },
    ],
  },
  areas: [
    {
      id: 'responsibility/orders',
      name: 'Orders',
      repository: 'product',
      status: 'checked',
      questions: [
        {
          id: 'orders/implementation',
          question: 'How are orders stored?',
          status: 'answered',
          result: 'Orders are saved before processing.',
          evidence: [{ document_id: 'doc', path: 'orders.go', start_line: 20, end_line: 35 }],
        },
      ],
    },
    {
      id: 'responsibility/payments',
      name: 'Payments',
      repository: 'product',
      status: 'investigating',
      questions: [
        {
          id: 'payments/validation',
          question: 'Does a retry send another notification?',
          status: 'pending',
          result: '',
          evidence: [],
        },
      ],
    },
    {
      id: 'scope/vendor',
      name: 'Vendor',
      repository: 'product',
      status: 'excluded',
      questions: [
        {
          id: 'vendor',
          question: 'Is this product code?',
          status: 'excluded',
          result: 'Third-party code, verified in its package manifest.',
          evidence: [],
        },
      ],
    },
  ],
};

test('current plan counts exclude discarded scopes and prioritize the active area', () => {
  const counts = researchCounts(progress.areas);
  assert.equal(counts.checked, 1);
  assert.equal(counts.questions, 2);
  assert.equal(counts.answered, 1);
  assert.equal(counts.excluded, 1);
  assert.equal(orderedResearchAreas(progress)[0]?.name, 'Payments');
});

test('one compact panel contains focus, results, expandable areas and budget reasons', () => {
  const html = renderToStaticMarkup(<ResearchProgress progress={progress} running />);
  assert.match(html, /aria-label="Research plan"/);
  assert.match(html, /How are repeated callbacks handled\?/);
  assert.match(html, /The callback handler rejects repeated payment identifiers/);
  assert.match(html, /class="research-area-list"/);
  assert.match(html, /orders.go:L20–L35/);
  assert.match(html, /Why research continues/);
  assert.match(html, /allowance extended to 50/);
  assert.match(html, /Hard cap 120 turns · 30 min/);
  assert.doesNotMatch(html, /<details[^>]*open/);
  assert.doesNotMatch(html, /role="progressbar"/);
});

test('a stopped run shows the reason and does not claim active research', () => {
  const stopped: ResearchState = {
    ...progress,
    status: 'partial',
    budget: {
      ...progress.budget,
      decisions: [
        { turn: 50, limit: 50, kind: 'partial', reason: 'No new evidence after recovery.' },
      ],
    },
  };
  const html = renderToStaticMarkup(<ResearchProgress progress={stopped} running={false} />);
  assert.match(html, /Why research stopped/);
  assert.match(html, /No new evidence after recovery/);
  assert.doesNotMatch(html, /How are repeated callbacks handled\?/);
  assert.doesNotMatch(html, /Next budget review/);
  assert.match(html, /Partly checked/);
});

test('an interrupted run does not present an old budget decision as ongoing work', () => {
  const html = renderToStaticMarkup(<ResearchProgress progress={progress} running={false} />);
  assert.match(html, /Why research stopped/);
  assert.match(html, /ended before research coverage was accepted/);
  assert.doesNotMatch(html, /Next budget review/);
});
