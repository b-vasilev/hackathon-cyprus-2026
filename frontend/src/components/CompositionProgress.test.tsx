import assert from 'node:assert/strict';
import test from 'node:test';
import { renderToStaticMarkup } from 'react-dom/server';
import { CompositionProgress } from './CompositionProgress';

function renderProgress(maxAttempts?: number) {
  return renderToStaticMarkup(
    <CompositionProgress
      composition={{
        action: '',
        parallelism: 1,
        tasks: [
          {
            id: 'architecture-part',
            chapter_id: 'architecture',
            title: 'Whole-system structure',
            kind: 'markdown',
            status: 'failed',
            attempt: 6,
            max_attempts: maxAttempts,
            started_at: '',
            finished_at: '',
          },
        ],
      }}
      now={0}
      running={false}
    />
  );
}

test('composition uses the attempt limit supplied by the server', () => {
  assert.match(renderProgress(11), /Failed · attempt 6\/11/);
  assert.match(renderProgress(8), /Failed · attempt 6\/8/);
});

test('saved progress without an attempt limit does not invent one', () => {
  const html = renderProgress();
  assert.match(html, /Failed · attempt 6/);
  assert.doesNotMatch(html, /attempt 6\//);
});
