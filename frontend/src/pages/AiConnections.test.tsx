import assert from 'node:assert/strict';
import test from 'node:test';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { renderToStaticMarkup } from 'react-dom/server';
import { MemoryRouter } from 'react-router-dom';
import { AIConnections, AIConsent } from './AiConnections';

test('account setup shows browser sign-in and account app disconnect controls', () => {
  const client = new QueryClient({
    defaultOptions: { queries: { staleTime: Number.POSITIVE_INFINITY } },
  });
  client.setQueryData(['ai-connections'], {
    url: 'https://portal.example/mcp',
    connections: [
      {
        id: 'grant',
        name: 'Example AI',
        created_at: '2026-10-16T12:00:00Z',
        expires_at: '2026-11-15T12:00:00Z',
      },
    ],
  });
  const html = renderToStaticMarkup(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <AIConnections />
      </MemoryRouter>
    </QueryClientProvider>
  );
  assert.ok(html.includes('https://portal.example/mcp'));
  assert.ok(html.includes('You can ask your AI to add this connection for you'));
  assert.ok(html.includes('Copy setup prompt'));
  assert.ok(html.includes('Restart your AI app or CLI session, then start a new chat'));
  for (const name of [
    'Claude Code CLI',
    'Codex CLI',
    'Claude Desktop',
    'Codex desktop',
    'Pi',
    'OpenCode',
    'Cursor',
    'VS Code / GitHub Copilot',
  ]) {
    assert.ok(html.includes(`<summary>${name}</summary>`), `Missing setup guide: ${name}`);
  }
  assert.ok(html.includes('claude mcp add --transport http --scope user wayfinder'));
  assert.ok(html.includes('codex mcp login wayfinder'));
  assert.ok(html.includes('pi mcp login wayfinder'));
  assert.ok(html.includes('opencode mcp auth wayfinder'));
  assert.ok(html.includes('localhost will not work'));
  assert.ok(html.includes('No token needs copying'));
  assert.ok(html.includes('Product access changes apply automatically'));
  assert.ok(html.includes('Disconnect Example AI'));
  assert.ok(!html.includes('Create access token'));
  client.clear();
});

test('consent asks the user to allow or cancel and displays the app callback', () => {
  const client = new QueryClient({
    defaultOptions: { queries: { staleTime: Number.POSITIVE_INFINITY } },
  });
  const search = '?client_id=client&redirect_uri=https%3A%2F%2Fapp.example%2Fcallback';
  client.setQueryData(['ai-consent', search], {
    client_name: 'Example AI',
    redirect_uris: ['https://app.example/callback'],
  });
  const html = renderToStaticMarkup(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[`/ai-connections/authorize${search}`]}>
        <AIConsent />
      </MemoryRouter>
    </QueryClientProvider>
  );
  assert.ok(html.includes('Example AI'));
  assert.ok(html.includes('https://app.example/callback'));
  assert.ok(html.includes('Allow read access'));
  assert.ok(html.includes('Cancel'));
  assert.ok(html.includes('published passports'));
  client.clear();
});
