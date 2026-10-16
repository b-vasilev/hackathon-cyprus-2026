import assert from 'node:assert/strict';
import test from 'node:test';
import { QueryClient, QueryObserver } from '@tanstack/react-query';
import { setSignedInUser } from './auth';
import type { Me } from './api';

const me: Me = {
  user: { id: 'alice', email: 'alice@example.com', role: 'user', created_at: '2026-10-16' },
  memberships: [],
};

test('sign-in updates the active session observer without a page reload and clears previous account data', async () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const observer = new QueryObserver<Me>(client, {
    queryKey: ['me'],
    queryFn: () => Promise.reject(new Error('Not signed in')),
  });
  const unsubscribe = observer.subscribe(() => {});
  try {
    await observer.refetch();
    assert.equal(observer.getCurrentResult().status, 'error');
    const sessionQuery = client.getQueryCache().find({ queryKey: ['me'] });
    client.setQueryData(['connections'], [{ id: 'previous-user-connection' }]);
    client.setQueryData(['github-connection'], { login: 'previous-user' });

    await setSignedInUser(client, me);

    assert.equal(client.getQueryCache().find({ queryKey: ['me'] }), sessionQuery);
    assert.equal(observer.getCurrentResult().status, 'success');
    assert.deepEqual(observer.getCurrentResult().data, me);
    assert.equal(client.getQueryData(['connections']), undefined);
    assert.equal(client.getQueryData(['github-connection']), undefined);
  } finally {
    unsubscribe();
    client.clear();
  }
});

test('an in-flight session check cannot replace successful sign-in with an unauthorized response', async () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const observer = new QueryObserver<Me>(client, {
    queryKey: ['me'],
    queryFn: ({ signal }) =>
      new Promise((_resolve, reject) => {
        signal.addEventListener('abort', () => reject(new Error('Session check cancelled')));
      }),
  });
  const unsubscribe = observer.subscribe(() => {});
  try {
    assert.equal(observer.getCurrentResult().fetchStatus, 'fetching');
    await setSignedInUser(client, me);
    assert.equal(observer.getCurrentResult().status, 'success');
    assert.equal(observer.getCurrentResult().fetchStatus, 'idle');
    assert.deepEqual(observer.getCurrentResult().data, me);
  } finally {
    unsubscribe();
    client.clear();
  }
});
