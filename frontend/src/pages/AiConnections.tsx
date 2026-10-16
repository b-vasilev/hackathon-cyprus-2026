import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Check, Copy, Plug, Unplug } from 'lucide-react';
import { useState } from 'react';
import { useLocation } from 'react-router-dom';
import { McpSetupGuide } from '../components/McpSetupGuide';
import { EmptyState, ErrorNotice, Loading, PageHeading } from '../components/ui';
import { api, type MCPClient, type MCPSettings } from '../lib/api';

export function AIConnections() {
  const queryClient = useQueryClient();
  const [copied, setCopied] = useState<'url' | 'prompt'>();
  const [copyError, setCopyError] = useState<unknown>();
  const settings = useQuery({
    queryKey: ['ai-connections'],
    queryFn: () => api<MCPSettings>('/ai-connections'),
  });
  const disconnect = useMutation({
    mutationFn: (id: string) => api<undefined>(`/ai-connections/${id}`, { method: 'DELETE' }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['ai-connections'] }),
  });
  async function copySetup(target: 'url' | 'prompt') {
    if (!settings.data) {
      return;
    }
    try {
      await navigator.clipboard.writeText(
        target === 'url'
          ? settings.data.url
          : `Add Wayfinder as a remote Streamable HTTP MCP server in my AI app using this URL: ${settings.data.url}. Use OAuth browser sign-in and help me complete it. Keep my existing MCP servers. Remind me to restart the app or CLI session and start a new chat after setup.`
      );
      setCopied(target);
      setCopyError(undefined);
    } catch (error) {
      setCopyError(error);
    }
  }
  return (
    <div className="ai-connections">
      <PageHeading
        eyebrow="Account"
        title="Connect AI"
        description="Let your AI app read the published passports available to your account."
      />
      {settings.isPending && <Loading label="Loading AI connections" />}
      {settings.error && <ErrorNotice error={settings.error} />}
      {settings.data && (
        <>
          <section className="form-panel" aria-labelledby="ai-setup-title">
            <h2 id="ai-setup-title">Connect your app</h2>
            <p>
              You can ask your AI to add this connection for you. Give it the server URL below, or
              copy the setup prompt and send it to an AI that can change your app settings.
            </p>
            <ol>
              <li>Ask your AI to add Wayfinder, or follow the steps for your app below.</li>
              <li>Choose Connect, sign in to Wayfinder, and approve read access.</li>
              <li>Restart your AI app or CLI session, then start a new chat.</li>
              <li>Ask your AI to list products or export their full passports as Markdown.</li>
            </ol>
            <label>
              MCP server URL
              <input
                readOnly
                value={settings.data.url}
                onFocus={(event) => event.target.select()}
              />
            </label>
            <div className="form-actions">
              <button
                className="button primary"
                type="button"
                onClick={() => void copySetup('prompt')}
              >
                {copied === 'prompt' ? (
                  <Check size={16} aria-hidden="true" />
                ) : (
                  <Copy size={16} aria-hidden="true" />
                )}
                {copied === 'prompt' ? 'Prompt copied' : 'Copy setup prompt'}
              </button>
              <button
                className="button secondary"
                type="button"
                onClick={() => void copySetup('url')}
              >
                {copied === 'url' ? (
                  <Check size={16} aria-hidden="true" />
                ) : (
                  <Copy size={16} aria-hidden="true" />
                )}
                {copied === 'url' ? 'URL copied' : 'Copy server URL'}
              </button>
            </div>
            <p className="field-help">
              Your app must support remote MCP with browser sign-in. No token needs copying. Product
              access changes apply automatically. Drafts and unpublished edits stay private.
            </p>
            {copyError !== undefined && <ErrorNotice error={copyError} />}
          </section>
          <McpSetupGuide url={settings.data.url} />
          <section aria-labelledby="ai-apps-title">
            <h2 id="ai-apps-title">Connected apps</h2>
            {disconnect.error && <ErrorNotice error={disconnect.error} />}
            {settings.data.connections.length === 0 ? (
              <EmptyState title="No apps connected">
                Add the server URL in your AI app to start.
              </EmptyState>
            ) : (
              <ul className="ai-app-list">
                {settings.data.connections.map((connection) => (
                  <li key={connection.id}>
                    <Plug size={20} aria-hidden="true" />
                    <div>
                      <strong>{connection.name}</strong>
                      <p className="field-help">
                        Read-only · Connected {new Date(connection.created_at).toLocaleDateString()}
                        {' · '}Expires {new Date(connection.expires_at).toLocaleDateString()}
                      </p>
                    </div>
                    <button
                      className="button secondary"
                      type="button"
                      disabled={disconnect.isPending}
                      onClick={() => disconnect.mutate(connection.id)}
                      aria-label={`Disconnect ${connection.name}`}
                    >
                      <Unplug size={16} aria-hidden="true" /> Disconnect
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </section>
        </>
      )}
    </div>
  );
}

export function AIConsent() {
  const location = useLocation();
  const client = useQuery({
    queryKey: ['ai-consent', location.search],
    queryFn: () => api<MCPClient>(`/ai-connections/authorize${location.search}`),
    retry: false,
  });
  const consent = useMutation({
    mutationFn: (allow: boolean) =>
      api<{ redirect_url: string }>('/ai-connections/authorize', {
        method: 'POST',
        body: JSON.stringify({
          authorization: Object.fromEntries(
            [...new URLSearchParams(location.search)].filter(([key]) =>
              [
                'client_id',
                'redirect_uri',
                'response_type',
                'code_challenge',
                'code_challenge_method',
                'resource',
                'scope',
                'state',
              ].includes(key)
            )
          ),
          allow,
        }),
      }),
    onSuccess: (result) => globalThis.location.assign(result.redirect_url),
  });
  const callback = new URLSearchParams(location.search).get('redirect_uri');
  return (
    <div className="ai-connections">
      <PageHeading
        eyebrow="AI connection"
        title="Allow read access?"
        description="Connect an AI app to your Wayfinder account."
      />
      {client.isPending && <Loading label="Checking connection request" />}
      {client.error && <ErrorNotice error={client.error} />}
      {client.data && (
        <section className="form-panel" aria-label="Approve AI connection">
          <h2>{client.data.client_name}</h2>
          <p>This app will be able to list your products and read their published passports.</p>
          <p>Access follows your account permissions. You can disconnect the app at any time.</p>
          <p className="field-help">
            App names are supplied by the app. Approve only if you started this connection.
          </p>
          <p className="field-help">
            Return to: <code>{callback}</code>
          </p>
          {consent.error && <ErrorNotice error={consent.error} />}
          <div className="form-actions">
            <button
              className="button secondary"
              type="button"
              disabled={consent.isPending}
              onClick={() => consent.mutate(false)}
            >
              Cancel
            </button>
            <button
              className="button primary"
              type="button"
              disabled={consent.isPending}
              onClick={() => consent.mutate(true)}
            >
              Allow read access
            </button>
          </div>
        </section>
      )}
    </div>
  );
}
