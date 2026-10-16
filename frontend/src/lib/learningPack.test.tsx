import assert from 'node:assert/strict';
import test from 'node:test';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { renderToStaticMarkup } from 'react-dom/server';
import { packKey, usePublishedPack } from './learningPack';
import type { LearningPack } from './api';

function PublishedPractice({ revisionId }: Readonly<{ revisionId: string }>) {
  const { data } = usePublishedPack('product', revisionId);
  return <span>{data?.revision_id ?? 'no practice'}</span>;
}

test('published practice cache is isolated by the displayed passport revision', () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const pack: LearningPack = {
    id: 'pack-1',
    product_id: 'product',
    revision_id: 'revision-1',
    revision_version: 1,
    status: 'ready',
    error: '',
    updated_at: '',
    published: true,
    published_at: '2026-10-16T10:00:00Z',
    unpublished_changes: false,
  };
  const render = (revisionId: string) =>
    renderToStaticMarkup(
      <QueryClientProvider client={client}>
        <PublishedPractice revisionId={revisionId} />
      </QueryClientProvider>
    );
  try {
    client.setQueryData([packKey, 'published', 'product', 'revision-1'], pack);
    assert.equal(render('revision-1'), '<span>revision-1</span>');
    assert.equal(render('revision-2'), '<span>no practice</span>');

    // The server may switch publication between loading the passport and the practice.
    client.setQueryData([packKey, 'published', 'product', 'revision-2'], pack);
    assert.equal(render('revision-2'), '<span>no practice</span>');
    assert.equal(render(''), '<span>no practice</span>');
  } finally {
    client.clear();
  }
});
