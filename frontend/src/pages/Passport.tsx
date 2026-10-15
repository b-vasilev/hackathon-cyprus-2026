import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ArrowRight, Check, FileCode2, GitBranch, Play, RefreshCw } from 'lucide-react';
import { useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { EvidenceDialog, ObjectDetail, SectionContent } from '../components/PassportContent';
import { Badge, EmptyState, ErrorNotice, ExternalLink, Loading } from '../components/ui';
import {
  api,
  date,
  sourceDescription,
  type Connection,
  type FactoryStatus,
  type Product,
  type ProductObject,
  type Revision,
  type RevisionInfo,
  type Run,
} from '../lib/api';

const stages = [
  'collecting',
  'selecting_sources',
  'analyzing',
  'synthesizing',
  'composing',
  'validating',
];

function activeRun(status: string | undefined): boolean {
  return status === 'queued' || status === 'running';
}

export function PassportPage() {
  const { id = '' } = useParams();
  const queryClient = useQueryClient();
  const product = useQuery({
    queryKey: ['product', id],
    queryFn: ({ signal }) => api<Product>(`/products/${id}`, { signal }),
  });
  const setup = useQuery({
    queryKey: ['factory-status'],
    queryFn: ({ signal }) => api<FactoryStatus>('/factory/status', { signal }),
  });
  const connections = useQuery({
    queryKey: ['connections'],
    queryFn: ({ signal }) => api<Connection[]>('/llm-connections', { signal }),
    enabled: setup.data?.connections_enabled === true,
  });
  const history = useQuery({
    queryKey: ['revisions', id],
    queryFn: ({ signal }) => api<RevisionInfo[]>(`/products/${id}/passport-revisions`, { signal }),
  });
  const [connectionId, setConnectionId] = useState('');
  const [modelId, setModelId] = useState('');
  const [selectedRevision, setSelectedRevision] = useState('');
  const [sectionId, setSectionId] = useState('overview');
  const [object, setObject] = useState<ProductObject | null>(null);
  const [evidenceId, setEvidenceId] = useState<string | null>(null);
  const availableConnections = connections.data?.filter((item) => item.config.enabled) ?? [];
  const connection =
    availableConnections.find((item) => item.id === connectionId) ?? availableConnections.at(0);
  const model =
    connection?.config.models?.find((item) => item.id === modelId) ??
    connection?.config.models?.at(0);
  const runId = product.data?.latest_run_id ?? '';
  const run = useQuery({
    queryKey: ['run', runId],
    queryFn: ({ signal }) => api<Run>(`/generations/${runId}`, { signal }),
    enabled: runId !== '',
    refetchInterval: (query) => (activeRun(query.state.data?.status) ? 1500 : false),
  });
  useEffect(() => {
    if (run.data?.status === 'succeeded' || run.data?.status === 'failed') {
      void queryClient.invalidateQueries({ queryKey: ['product', id] });
      void queryClient.invalidateQueries({ queryKey: ['revisions', id] });
      void queryClient.invalidateQueries({ queryKey: ['products'] });
    }
  }, [run.data?.status, id, queryClient]);
  const revisionId =
    selectedRevision === ''
      ? (history.data?.at(0)?.id ?? product.data?.latest_revision_id ?? '')
      : selectedRevision;
  const revision = useQuery({
    queryKey: ['revision', revisionId],
    queryFn: ({ signal }) => api<Revision>(`/passport-revisions/${revisionId}`, { signal }),
    enabled: revisionId !== '',
  });
  const revisionRun = useQuery({
    queryKey: ['run', revision.data?.run_id ?? ''],
    queryFn: ({ signal }) => api<Run>(`/generations/${revision.data?.run_id ?? ''}`, { signal }),
    enabled: revision.data !== undefined,
  });
  const generate = useMutation({
    mutationFn: () =>
      api<Run>(`/products/${id}/generations`, {
        method: 'POST',
        body: JSON.stringify({ connection_id: connection?.id, model: model?.id }),
      }),
    onSuccess: async (newRun) => {
      queryClient.setQueryData(['run', newRun.id], newRun);
      setSelectedRevision('');
      await queryClient.invalidateQueries({ queryKey: ['product', id] });
    },
  });
  if (product.isPending) {
    return <Loading label="Loading product space…" />;
  }
  if (product.isError) {
    return <ErrorNotice error={product.error} />;
  }
  const section =
    revision.data?.content.views.sections?.find((item) => item.id === sectionId) ??
    revision.data?.content.views.sections?.at(0);
  const busy = activeRun(run.data?.status) || generate.isPending;
  return (
    <>
      <Link to="/" className="back-link">
        ← Product spaces
      </Link>
      <div className="passport-heading">
        <div>
          <p className="eyebrow">Product passport</p>
          <h1>{revision.data?.content.model.title ?? product.data.name}</h1>
          <p className="page-description">
            {revision.data?.content.model.tagline ??
              'Connect the sources. Generate the picture. Inspect the evidence.'}
          </p>
        </div>
        <Badge tone="warning">
          {revision.data === undefined ? 'Not generated yet' : 'Generated draft · review pending'}
        </Badge>
      </div>
      <div className="generation-controls">
        <label>
          Connection
          <select
            value={connection?.id ?? ''}
            disabled={busy}
            onChange={(event) => {
              setConnectionId(event.target.value);
              setModelId('');
            }}
          >
            <option value="" disabled>
              Select a connection
            </option>
            {availableConnections.map((item) => (
              <option key={item.id} value={item.id}>
                {item.config.name}
              </option>
            ))}
          </select>
        </label>
        <label>
          Model
          <select
            value={model?.id ?? ''}
            disabled={busy}
            onChange={(event) => setModelId(event.target.value)}
          >
            <option value="" disabled>
              Select a model
            </option>
            {connection?.config.models?.map((item) => (
              <option key={item.id} value={item.id}>
                {item.id}
              </option>
            ))}
          </select>
        </label>
        <button
          type="button"
          className="button primary"
          disabled={busy || connection === undefined || model === undefined}
          onClick={() => generate.mutate()}
        >
          {busy ? <RefreshCw size={16} className="spin" /> : <Play size={16} />}
          {busy ? 'Generating…' : 'Generate passport'}
        </button>
        {(history.data?.length ?? 0) > 0 && (
          <label className="revision-select">
            Version
            <select
              value={revisionId}
              onChange={(event) => setSelectedRevision(event.target.value)}
            >
              {history.data?.map((item) => (
                <option key={item.id} value={item.id}>
                  Version {item.version}
                </option>
              ))}
            </select>
          </label>
        )}
      </div>
      {availableConnections.length === 0 &&
        (setup.data?.connections_enabled === false || !connections.isPending) && (
          <div className="notice">
            <p>Add an enabled model connection to generate this passport.</p>
            <Link className="text-link" to="/connections">
              Set up a connection
              <ArrowRight size={15} />
            </Link>
          </div>
        )}
      {generate.isError && <ErrorNotice error={generate.error} />}
      {run.data !== undefined && activeRun(run.data.status) && (
        <GenerationProgress run={run.data} />
      )}
      {run.data?.status === 'failed' && (
        <div className="notice error" role="alert">
          <div>
            <strong>Generation stopped at {run.data.stage}.</strong>
            <p>{run.data.error_message}</p>
            {revision.data !== undefined && <p>The previous passport remains available below.</p>}
          </div>
        </div>
      )}
      {revision.isFetching && <Loading label="Loading passport revision…" />}
      {revision.isError && <ErrorNotice error={revision.error} />}
      {history.isError && <ErrorNotice error={history.error} />}
      {revision.data === undefined && !busy && !revision.isFetching && (
        <EmptyState title="Your sources are ready to explore">
          Choose a connection and model, then generate a passport. It will cover the product’s
          purpose, users, domain, scenarios, architecture and developer workflow.
        </EmptyState>
      )}
      {revision.data !== undefined && section !== undefined && (
        <>
          <div className="revision-meta">
            <span>Version {revision.data.version}</span>
            <span>{date(revision.data.created_at)}</span>
            {revisionRun.data !== undefined && (
              <span>
                {revisionRun.data.model} · connection revision{' '}
                {revisionRun.data.connection_revision}
              </span>
            )}
            <span>Expert review pending</span>
          </div>
          <div className="passport-layout">
            <nav className="section-nav" aria-label="Passport sections">
              {revision.data.content.views.sections?.map((item, index) => (
                <button
                  type="button"
                  key={item.id}
                  className={item.id === section.id ? 'active' : ''}
                  aria-current={item.id === section.id ? 'true' : undefined}
                  onClick={() => setSectionId(item.id)}
                >
                  <span>{String(index + 1).padStart(2, '0')}</span>
                  {item.title}
                </button>
              ))}
            </nav>
            <SectionContent
              key={`${revision.data.id}:${section.id}`}
              section={section}
              revision={revision.data}
              onObject={setObject}
              onEvidence={setEvidenceId}
            />
          </div>
          <details className="sources-panel">
            <summary>
              <GitBranch size={17} />
              Analyzed sources and coverage
            </summary>
            {revision.data.snapshots?.map((snapshot) => (
              <div className="source-row" key={snapshot.id}>
                <div>
                  <strong>{snapshot.locator}</strong>
                  <p className="meta">
                    {snapshot.coverage.file_count} files ·{' '}
                    {snapshot.coverage.bytes.toLocaleString()} bytes · revision{' '}
                    <code>{snapshot.revision.slice(0, 12)}</code>
                  </p>
                  <p className="field-help">{snapshot.coverage.note}</p>
                  {snapshot.coverage.selection !== '' && (
                    <p className="meta">Selection: {snapshot.coverage.selection}</p>
                  )}
                </div>
                {snapshot.locator.startsWith('https://github.com/') && (
                  <ExternalLink
                    href={`${snapshot.locator.replace(/\.git$/, '')}/tree/${snapshot.revision}`}
                  >
                    Repository
                  </ExternalLink>
                )}
              </div>
            ))}
          </details>
        </>
      )}
      <details className="sources-panel">
        <summary>
          <FileCode2 size={17} />
          Product source configuration
        </summary>
        {product.data.sources?.map((source) => (
          <div className="source-row" key={JSON.stringify(source)}>
            <strong>
              {source.kind === 'github' ? source.locator : 'Supplemental product context'}
            </strong>
            <span className="meta">{sourceDescription(source)}</span>
          </div>
        ))}
      </details>
      {object !== null && revision.data !== undefined && (
        <ObjectDetail
          object={object}
          model={revision.data.content.model}
          onClose={() => setObject(null)}
          onEvidence={setEvidenceId}
        />
      )}
      {evidenceId !== null && revisionId !== '' && (
        <EvidenceDialog
          revisionId={revisionId}
          evidenceId={evidenceId}
          onClose={() => setEvidenceId(null)}
        />
      )}
    </>
  );
}

function GenerationProgress({ run }: Readonly<{ run: Run }>) {
  const current = stages.indexOf(run.stage);
  return (
    <div className="generation-progress" role="status">
      <div className="progress-heading">
        <strong>
          {run.status === 'queued'
            ? 'Waiting for the worker'
            : `Factory is ${run.stage} your passport`}
        </strong>
        <span className="meta">
          {run.document_count} source documents ·{' '}
          {(run.input_tokens + run.output_tokens).toLocaleString()} recorded tokens
        </span>
      </div>
      <div className="stage-route">
        {stages.map((stage, index) => (
          <div className={`stage ${index === current ? 'current' : ''}`} key={stage}>
            <span className="stage-dot">{index < current ? <Check size={12} /> : index + 1}</span>
            <span>{stage}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
