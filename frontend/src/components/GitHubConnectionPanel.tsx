import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { KeyRound, Unplug } from 'lucide-react';
import { useState, type FormEvent } from 'react';
import { useSearchParams } from 'react-router-dom';
import githubMark from '../assets/github-mark.svg';
import { api, type GitHubConnection } from '../lib/api';
import { Badge, Dialog, ErrorNotice, Loading } from './ui';

const connectionKey = 'github-connection';
const connectionPath = '/git-connection';

export function GitHubConnectionPanel() {
  const client = useQueryClient();
  const [params] = useSearchParams();
  const [editing, setEditing] = useState(false);
  const connection = useQuery({
    queryKey: [connectionKey],
    queryFn: ({ signal }) => api<GitHubConnection>(connectionPath, { signal }),
  });
  const disconnect = useMutation({
    mutationFn: () => api<GitHubConnection>(connectionPath, { method: 'DELETE' }),
    onSuccess: async () => {
      await client.invalidateQueries({ queryKey: [connectionKey] });
    },
  });
  const enabled = connection.data?.enabled === true;
  const busy = disconnect.isPending;
  return (
    <section aria-label="GitHub connection" className="connection-list github-connection-section">
      <header className="connections-section-heading">
        <div>
          <h2>GitHub connection</h2>
          <p>Repository access for source collection.</p>
        </div>
      </header>
      <article className="connection-card github-connection-card">
        <div className="connection-heading">
          <div className="connection-title">
            <span className="small-icon">
              <img src={githubMark} width={22} height={22} alt="" aria-hidden="true" />
            </span>
            <div>
              <h3>GitHub</h3>
              <p>
                {connection.data?.connected
                  ? `Connected as ${connection.data.login}`
                  : 'Access private repositories for passport generation.'}
              </p>
            </div>
          </div>
          <Badge>{connection.data?.connected ? 'Connected' : 'Not connected'}</Badge>
        </div>
        <p className="github-connection-description">
          This is a shared source connection for this portal. Public repositories work without
          login.
          {connection.data?.connected &&
            ` Connected with ${connection.data.auth_method === 'oauth' ? 'GitHub sign-in' : 'a personal access token'}.`}
        </p>
        {connection.isPending && <Loading />}
        {connection.isError && <ErrorNotice error={connection.error} />}
        {connection.data?.enabled === false && (
          <p className="github-connection-description">
            Ask the server administrator to enable encrypted connection storage.
          </p>
        )}
        <div className="github-connection-actions">
          <button
            type="button"
            className="button primary"
            disabled={!enabled || busy}
            onClick={() => setEditing(true)}
          >
            <KeyRound size={17} />
            {connection.data?.connected ? 'Replace with token' : 'Connect with token'}
          </button>
          {connection.data?.connected && (
            <button
              type="button"
              className="button quiet"
              disabled={busy}
              onClick={() => disconnect.mutate()}
            >
              <Unplug size={17} />
              {disconnect.isPending ? 'Disconnecting…' : 'Disconnect'}
            </button>
          )}
        </div>
        {params.get('github') === 'connected' && connection.data?.connected && (
          <p role="status">GitHub sign-in completed.</p>
        )}
        {params.get('github') === 'error' && (
          <ErrorNotice
            error={new Error('GitHub sign-in was cancelled, expired or failed. Please try again.')}
          />
        )}
        {disconnect.isError && <ErrorNotice error={disconnect.error} />}
      </article>
      {editing && (
        <TokenEditor
          onClose={() => setEditing(false)}
          onSaved={async () => {
            await client.invalidateQueries({ queryKey: [connectionKey] });
            setEditing(false);
          }}
        />
      )}
    </section>
  );
}

function TokenEditor({
  onClose,
  onSaved,
}: Readonly<{ onClose: () => void; onSaved: () => Promise<void> }>) {
  const [token, setToken] = useState('');
  const save = useMutation({
    mutationFn: () =>
      api<GitHubConnection>(connectionPath, { method: 'PUT', body: JSON.stringify({ token }) }),
    onSuccess: async () => {
      setToken('');
      await onSaved();
    },
  });
  function submit(event: FormEvent) {
    event.preventDefault();
    save.mutate();
  }
  return (
    <Dialog title="Connect GitHub with a token" onClose={onClose}>
      <form onSubmit={submit} className="connection-form">
        <p>
          Use a fine-grained personal access token with read access to Contents for the repositories
          you want to analyze.
        </p>
        <label>
          Personal access token
          <input
            type="password"
            required
            autoComplete="off"
            maxLength={8192}
            value={token}
            onChange={(event) => setToken(event.target.value)}
          />
        </label>
        <p className="field-help">
          We check your GitHub account before saving. The token is encrypted on the server and never
          returned to the browser.
        </p>
        {save.isError && <ErrorNotice error={save.error} />}
        <div className="form-actions">
          <button type="button" className="button quiet" onClick={onClose}>
            Cancel
          </button>
          <button className="button primary" disabled={save.isPending}>
            {save.isPending ? 'Connecting…' : 'Connect GitHub'}
          </button>
        </div>
      </form>
    </Dialog>
  );
}
