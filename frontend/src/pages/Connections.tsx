import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Check, KeyRound, Pencil, Plug, Plus, ShieldCheck, Trash2 } from 'lucide-react';
import { useState, type FormEvent } from 'react';
import { DeleteConfirmation } from '../components/DeleteConfirmation';
import { GitHubConnectionPanel } from '../components/GitHubConnectionPanel';
import { Badge, Dialog, EmptyState, ErrorNotice, Loading, PageHeading } from '../components/ui';
import {
  api,
  message,
  protocolLabels,
  type AvailableModel,
  type Connection,
  type ConnectionConfig,
  type FactoryStatus,
  type ModelConfig,
  type Parameter,
  type ProbeResult,
} from '../lib/api';

function defaultConfig(): ConnectionConfig {
  return {
    name: '',
    enabled: true,
    protocol: 'openai_responses',
    endpoint_url: 'https://api.openai.com/v1',
    endpoint_mode: 'base',
    auth_mode: 'header',
    auth_header: 'Authorization',
    auth_prefix: 'Bearer ',
    secret: '',
    headers: [],
    query_params: [],
    models: [
      { id: '', output_mode: 'json_schema', max_output_tokens: 64_000, reasoning_effort: '' },
    ],
    timeout_seconds: 180,
    max_concurrent: 2,
    quota_group: '',
  };
}

export function Connections() {
  const queryClient = useQueryClient();
  const status = useQuery({
    queryKey: ['factory-status'],
    queryFn: ({ signal }) => api<FactoryStatus>('/factory/status', { signal }),
  });
  const connections = useQuery({
    queryKey: ['connections'],
    queryFn: ({ signal }) => api<Connection[]>('/llm-connections', { signal }),
    enabled: status.data?.connections_enabled === true,
  });
  const [editing, setEditing] = useState<Connection | 'new' | null>(null);
  const [deleting, setDeleting] = useState<Connection | null>(null);
  const deleteConnection = useMutation({
    mutationFn: (id: string) => api<Connection>(`/llm-connections/${id}`, { method: 'DELETE' }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['connections'] });
      setDeleting(null);
    },
  });
  const [checks, setChecks] = useState<Record<string, string | undefined>>({});
  const probe = useMutation({
    mutationFn: ({ id, model }: Readonly<{ id: string; model: string }>) =>
      api<ProbeResult>(`/llm-connections/${id}/test`, {
        method: 'POST',
        body: JSON.stringify({ model }),
      }),
    onSuccess: (_, input) =>
      setChecks((old) => ({
        ...old,
        [`${input.id}:${input.model}`]: 'Structured output check passed',
      })),
    onError: (error, input) =>
      setChecks((old) => ({ ...old, [`${input.id}:${input.model}`]: message(error) })),
  });
  return (
    <>
      <PageHeading
        eyebrow="Wayfinder settings"
        title="Your connections"
        description="Connect GitHub for source access and choose your model endpoints."
      />
      <GitHubConnectionPanel />
      <section aria-labelledby="llm-connections-heading" className="connections-section">
        <header className="connections-section-heading">
          <div>
            <h2 id="llm-connections-heading">LLM connections</h2>
            <p>Model providers and gateways for passport generation.</p>
          </div>
          <button
            type="button"
            className="button primary"
            onClick={() => setEditing('new')}
            disabled={status.data?.connections_enabled !== true}
          >
            <Plus size={17} />
            Add LLM connection
          </button>
        </header>
        <div className="intro-strip">
          <ShieldCheck size={19} aria-hidden="true" />
          <p>
            Keys are encrypted on the server. Each passport run uses a fixed connection revision and
            model.
          </p>
        </div>
        {status.data?.connections_enabled === false && (
          <div className="notice warning">
            <KeyRound size={19} aria-hidden="true" />
            <div>
              <strong>Connection storage needs a master key.</strong>
              <p>
                Set CONNECTION_MASTER_KEY to a stable secret of at least 32 characters and restart
                the API. Keep it safe: it decrypts your saved connections.
              </p>
            </div>
          </div>
        )}
        {(status.isPending || connections.isFetching) && <Loading />}
        {(status.isError || connections.isError) && (
          <ErrorNotice error={status.error ?? connections.error} />
        )}
        {connections.data?.length === 0 && (
          <EmptyState title="Connect a model endpoint">
            Use your own URL, credentials and model names. Multiple connections can expose the same
            model.
          </EmptyState>
        )}
        <div className="connection-list">
          {connections.data?.map((connection) => (
            <article className="connection-card" key={connection.id}>
              <div className="connection-heading">
                <div className="connection-title">
                  <span className="small-icon">
                    <Plug size={20} aria-hidden="true" />
                  </span>
                  <div>
                    <h2>{connection.config.name}</h2>
                    <p>{protocolLabels[connection.config.protocol]}</p>
                  </div>
                </div>
                <div className="row-actions">
                  <Badge>{connection.config.enabled ? 'Enabled' : 'Disabled'}</Badge>
                  <button
                    type="button"
                    className="button quiet"
                    onClick={() => setEditing(connection)}
                  >
                    <Pencil size={15} />
                    Edit
                  </button>
                  <button
                    type="button"
                    className="text-button destructive"
                    disabled={probe.isPending || deleteConnection.isPending}
                    onClick={() => {
                      deleteConnection.reset();
                      setDeleting(connection);
                    }}
                    aria-label={`Delete ${connection.config.name}`}
                  >
                    <Trash2 size={15} aria-hidden="true" />
                    Delete
                  </button>
                </div>
              </div>
              <p className="endpoint">{connection.config.endpoint_url}</p>
              <div className="connection-models">
                {connection.config.models?.map((model) => (
                  <div className="model-row" key={model.id}>
                    <div>
                      <code>{model.id}</code>
                      <span className="meta">
                        {model.output_mode.replaceAll('_', ' ')} ·{' '}
                        {model.max_output_tokens.toLocaleString()} output tokens
                      </span>
                      {checks[`${connection.id}:${model.id}`] !== undefined && (
                        <p className="check-result" role="status">
                          {checks[`${connection.id}:${model.id}`]}
                        </p>
                      )}
                    </div>
                    <button
                      type="button"
                      className="button secondary"
                      disabled={!connection.config.enabled || probe.isPending}
                      onClick={() => probe.mutate({ id: connection.id, model: model.id })}
                    >
                      {probe.isPending &&
                      probe.variables.id === connection.id &&
                      probe.variables.model === model.id
                        ? 'Checking…'
                        : 'Check connection'}
                    </button>
                  </div>
                ))}
              </div>
              <div className="connection-foot">
                <KeyRound size={14} aria-hidden="true" />
                {connection.has_secret ? 'Key saved' : 'No authentication'}
                <span>Revision {connection.revision}</span>
              </div>
            </article>
          ))}
        </div>
      </section>
      {deleting !== null && (
        <DeleteConfirmation
          title="Delete connection?"
          label="Delete connection"
          pending={deleteConnection.isPending}
          error={deleteConnection.error}
          onClose={() => setDeleting(null)}
          onConfirm={() => deleteConnection.mutate(deleting.id)}
        >
          Delete “{deleting.config.name}” and its saved credentials? Existing passports and
          generation history will remain. This cannot be undone.
        </DeleteConfirmation>
      )}
      {editing !== null && (
        <ConnectionEditor
          connection={editing === 'new' ? null : editing}
          onClose={() => setEditing(null)}
          onSaved={async () => {
            await queryClient.invalidateQueries({ queryKey: ['connections'] });
            setEditing(null);
          }}
        />
      )}
    </>
  );
}

function ConnectionEditor({
  connection,
  onClose,
  onSaved,
}: Readonly<{
  connection: Connection | null;
  onClose: () => void;
  onSaved: () => Promise<void>;
}>) {
  const [config, setConfig] = useState<ConnectionConfig>(() =>
    connection === null ? defaultConfig() : structuredClone(connection.config)
  );
  const [modelKeys, setModelKeys] = useState(() =>
    (config.models ?? []).map(() => crypto.randomUUID())
  );
  const [headers, setHeaders] = useState(() =>
    JSON.stringify(connection?.config.headers ?? [], null, 2)
  );
  const [query, setQuery] = useState(() =>
    JSON.stringify(connection?.config.query_params ?? [], null, 2)
  );
  const [formError, setFormError] = useState<string | null>(null);
  const save = useMutation({
    mutationFn: (input: ConnectionConfig) =>
      api<Connection>(
        connection === null ? '/llm-connections' : `/llm-connections/${connection.id}`,
        { method: connection === null ? 'POST' : 'PATCH', body: JSON.stringify(input) }
      ),
    onSuccess: onSaved,
  });
  const availableModels = useMutation({
    mutationFn: (input: ConnectionConfig) =>
      api<AvailableModel[]>('/llm-connections/models', {
        method: 'POST',
        body: JSON.stringify({ connection_id: connection?.id ?? '', config: input }),
      }),
  });
  function formConfig(): ConnectionConfig {
    const parsedHeaders = JSON.parse(headers) as Parameter[];
    const parsedQuery = JSON.parse(query) as Parameter[];
    if (!Array.isArray(parsedHeaders) || !Array.isArray(parsedQuery)) {
      throw new TypeError('Headers and query parameters must be JSON arrays.');
    }
    return { ...config, headers: parsedHeaders, query_params: parsedQuery };
  }
  function fetchModels() {
    try {
      const input = formConfig();
      setFormError(null);
      availableModels.mutate(input);
    } catch {
      setFormError('Check the JSON arrays in the additional HTTP settings.');
    }
  }
  function set<K extends keyof ConnectionConfig>(key: K, value: ConnectionConfig[K]) {
    if (key !== 'models') {
      availableModels.reset();
    }
    setConfig((old) => ({ ...old, [key]: value }));
  }
  function model(index: number, update: Partial<ModelConfig>) {
    set(
      'models',
      (config.models ?? []).map((item, i) => (i === index ? { ...item, ...update } : item))
    );
  }
  function addModel() {
    set('models', [
      ...(config.models ?? []),
      { id: '', output_mode: 'json_schema', max_output_tokens: 64_000, reasoning_effort: '' },
    ]);
    setModelKeys((keys) => [...keys, crypto.randomUUID()]);
  }
  function removeModel(index: number) {
    set('models', config.models?.filter((_, i) => i !== index) ?? []);
    setModelKeys((keys) => keys.filter((_, i) => i !== index));
  }
  function preset(protocol: string) {
    availableModels.reset();
    const anthropic = protocol === 'anthropic_messages';
    setConfig((old) => ({
      ...old,
      protocol,
      endpoint_url: anthropic ? 'https://api.anthropic.com' : 'https://api.openai.com/v1',
      endpoint_mode: 'base',
      auth_mode: 'header',
      auth_header: anthropic ? 'X-Api-Key' : 'Authorization',
      auth_prefix: anthropic ? '' : 'Bearer ',
    }));
  }
  function submit(event: FormEvent) {
    event.preventDefault();
    try {
      const input = formConfig();
      setFormError(null);
      save.mutate(input);
    } catch {
      setFormError('Check the JSON arrays in the additional HTTP settings.');
    }
  }
  return (
    <Dialog
      wide
      title={connection === null ? 'Add LLM connection' : 'Edit LLM connection'}
      onClose={onClose}
    >
      <form onSubmit={submit} className="connection-form">
        <div className="preset-row">
          {Object.entries(protocolLabels).map(([key, label]) => (
            <button
              type="button"
              key={key}
              className={`preset ${config.protocol === key ? 'selected' : ''}`}
              onClick={() => preset(key)}
            >
              {label}
            </button>
          ))}
        </div>
        <label>
          Connection name
          <input
            required
            value={config.name}
            maxLength={100}
            onChange={(event) => set('name', event.target.value)}
            placeholder="e.g. Company gateway"
          />
        </label>
        <div className="form-grid">
          <label>
            API protocol
            <select
              value={config.protocol}
              onChange={(event) => set('protocol', event.target.value)}
            >
              {Object.entries(protocolLabels).map(([value, label]) => (
                <option key={value} value={value}>
                  {label}
                </option>
              ))}
            </select>
          </label>
          <label>
            URL type
            <select
              value={config.endpoint_mode}
              onChange={(event) => set('endpoint_mode', event.target.value)}
            >
              <option value="base">Base URL</option>
              <option value="full">Full endpoint URL</option>
            </select>
          </label>
        </div>
        <label>
          Endpoint URL
          <input
            required
            type="url"
            value={config.endpoint_url}
            onChange={(event) => set('endpoint_url', event.target.value)}
          />
        </label>
        <p className="field-help">
          Local endpoints are allowed, including localhost and private networks. Other HTTP
          endpoints need an exact origin in the server’s LLM_ALLOWED_PRIVATE_ORIGINS setting.
        </p>
        <label>
          Authentication
          <select
            value={config.auth_mode}
            onChange={(event) => set('auth_mode', event.target.value)}
          >
            <option value="header">API key in a header</option>
            <option value="none">No authentication</option>
          </select>
        </label>
        {config.auth_mode === 'header' && (
          <>
            <div className="form-grid">
              <label>
                Header name
                <input
                  required
                  value={config.auth_header}
                  onChange={(event) => set('auth_header', event.target.value)}
                />
              </label>
              <label>
                Header prefix
                <input
                  value={config.auth_prefix}
                  onChange={(event) => set('auth_prefix', event.target.value)}
                  placeholder="Empty, or Bearer followed by a space"
                />
              </label>
            </div>
            <label>
              API key
              <input
                type="password"
                autoComplete="off"
                value={config.secret}
                required={!connection?.has_secret}
                onChange={(event) => set('secret', event.target.value)}
                placeholder={
                  connection?.has_secret === true
                    ? 'Leave empty to keep the saved key'
                    : 'Provider or gateway key'
                }
              />
            </label>
          </>
        )}
        <div className="subheading">
          <h3>Models</h3>
          <button
            type="button"
            className="text-button"
            onClick={fetchModels}
            disabled={availableModels.isPending}
          >
            {availableModels.isPending ? 'Fetching models…' : 'Fetch models'}
          </button>
          <button type="button" className="text-button" onClick={addModel}>
            <Plus size={14} />
            Add model
          </button>
        </div>
        {availableModels.isError && <ErrorNotice error={availableModels.error} />}
        {availableModels.isSuccess && (
          <p className="field-help" role="status">
            {availableModels.data.length === 0
              ? 'The endpoint returned no models.'
              : `${availableModels.data.length} models available. Choose a model ID below.`}
          </p>
        )}
        <datalist id="connection-model-options">
          {availableModels.data?.map((item) => (
            <option key={item.id} value={item.id} />
          ))}
        </datalist>
        {config.models?.map((item, index) => (
          <div className="model-form" key={modelKeys.at(index)}>
            <label>
              Model ID
              <input
                required
                value={item.id}
                list="connection-model-options"
                onChange={(event) => model(index, { id: event.target.value })}
                placeholder="The exact model name or deployment alias"
              />
            </label>
            <div className="form-grid">
              <label>
                Output mode
                <select
                  value={item.output_mode}
                  onChange={(event) => model(index, { output_mode: event.target.value })}
                >
                  <option value="json_schema">Native JSON Schema</option>
                  <option value="tool_call">Tool call</option>
                  <option value="prompt_json">JSON by instruction</option>
                </select>
              </label>
              <label>
                Output token limit
                <input
                  required
                  type="number"
                  min={256}
                  max={64_000}
                  value={item.max_output_tokens}
                  onChange={(event) =>
                    model(index, { max_output_tokens: Number(event.target.value) })
                  }
                />
              </label>
            </div>
            {config.protocol !== 'anthropic_messages' && (
              <label>
                Reasoning effort
                <select
                  value={item.reasoning_effort}
                  onChange={(event) => model(index, { reasoning_effort: event.target.value })}
                >
                  <option value="">Provider default</option>
                  <option value="low">Low (faster)</option>
                  <option value="medium">Medium</option>
                  <option value="high">High</option>
                </select>
              </label>
            )}
            {(config.models?.length ?? 0) > 1 && (
              <button
                type="button"
                className="text-button destructive"
                onClick={() => removeModel(index)}
              >
                Remove model
              </button>
            )}
          </div>
        ))}
        <details className="advanced">
          <summary>Additional HTTP settings and limits</summary>
          <p className="field-help">
            Use arrays of {JSON.stringify({ name: 'X-Organization', value: '', secret: true })}.
            Empty secret values keep their saved value when editing.
          </p>
          <label>
            Additional headers
            <textarea
              rows={4}
              className="code-input"
              value={headers}
              onChange={(event) => {
                setHeaders(event.target.value);
                availableModels.reset();
              }}
            />
          </label>
          <label>
            Query parameters
            <textarea
              rows={4}
              className="code-input"
              value={query}
              onChange={(event) => {
                setQuery(event.target.value);
                availableModels.reset();
              }}
            />
          </label>
          <div className="form-grid">
            <label>
              Timeout in seconds
              <input
                type="number"
                min={10}
                max={600}
                value={config.timeout_seconds}
                onChange={(event) => set('timeout_seconds', Number(event.target.value))}
              />
            </label>
            <label>
              Concurrent requests
              <input
                type="number"
                min={1}
                max={16}
                value={config.max_concurrent}
                onChange={(event) => set('max_concurrent', Number(event.target.value))}
              />
            </label>
          </div>
          <label>
            Shared quota group
            <input
              value={config.quota_group}
              onChange={(event) => set('quota_group', event.target.value)}
              placeholder="Optional"
            />
          </label>
        </details>
        <label className="checkbox-label">
          <input
            type="checkbox"
            checked={config.enabled}
            onChange={(event) => set('enabled', event.target.checked)}
          />
          Enable this connection
        </label>
        {formError !== null && <ErrorNotice error={new Error(formError)} />}
        {save.isError && <ErrorNotice error={save.error} />}
        <div className="form-actions">
          <button type="button" className="button quiet" onClick={onClose}>
            Cancel
          </button>
          <button className="button primary" disabled={save.isPending}>
            <Check size={17} />
            {save.isPending ? 'Saving…' : 'Save connection'}
          </button>
        </div>
      </form>
    </Dialog>
  );
}
