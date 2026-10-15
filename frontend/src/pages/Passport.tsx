import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ArrowRight, FileCode2, GitBranch, Play, RefreshCw, Square, Trash2 } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { DeleteConfirmation } from '../components/DeleteConfirmation';
import { DocumentContent } from '../components/DocumentContent';
import { GenerationBadges, GenerationProgress } from '../components/GenerationProgress';
import { PartialResearchNotice } from '../components/PartialResearchNotice';
import { EvidenceDialog, ObjectDetail, SectionContent } from '../components/PassportContent';
import { ProductAccess } from '../components/ProductAccess';
import { ProductBrandSources, ProductIdentity } from '../components/ProductIdentity';
import { SourceCoverage } from '../components/SourceCoverage';
import { DisclosureHint, EmptyState, ErrorNotice, ExternalLink, Loading } from '../components/ui';
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
import { canManage, isAdmin, useMe } from '../lib/auth';
import { activeRun, cancelledRun, mergeRun, stageLabels } from '../lib/generation';
import { generationMessage } from '../lib/generationMessages';
import { documentChapters, legacySections } from '../lib/passportDocument';
import { sectionTitles } from '../lib/passportSections';
import { productCardTheme } from '../lib/productCardTheme';
import { useGeneration } from '../lib/useGeneration';

export function PassportPage() {
  const { id = '' } = useParams();
  const navigate = useNavigate();
  const [confirmDelete, setConfirmDelete] = useState(false);
  const queryClient = useQueryClient();
  const me = useMe();
  const manage = canManage(me, id);
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
    enabled: manage && setup.data?.connections_enabled === true,
  });
  const history = useQuery({
    queryKey: ['revisions', id],
    queryFn: ({ signal }) => api<RevisionInfo[]>(`/products/${id}/passport-revisions`, { signal }),
  });
  const [connectionId, setConnectionId] = useState('');
  const [modelId, setModelId] = useState('');
  const [selectedRevision, setSelectedRevision] = useState('');
  const [sectionId, setSectionId] = useState('');
  const readingPane = useRef<HTMLDivElement>(null);
  function selectSection(nextId: string) {
    setSectionId(nextId);
    requestAnimationFrame(() => {
      readingPane.current?.scrollIntoView({ block: 'start' });
      readingPane.current?.querySelector('h2')?.focus({ preventScroll: true });
    });
  }
  const [object, setObject] = useState<ProductObject | null>(null);
  const [evidenceId, setEvidenceId] = useState<string | null>(null);
  const availableConnections = connections.data?.filter((item) => item.config.enabled) ?? [];
  const runId = product.data?.latest_run_id ?? '';
  const { run, streamState } = useGeneration(runId);
  const connection =
    availableConnections.find(
      (item) => item.id === (connectionId === '' ? run.data?.connection_id : connectionId)
    ) ?? availableConnections.at(0);
  const model =
    connection?.config.models?.find(
      (item) => item.id === (modelId === '' ? run.data?.model : modelId)
    ) ?? connection?.config.models?.at(0);
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
      queryClient.setQueryData<Product>(['product', id], (previous) =>
        previous === undefined
          ? previous
          : { ...previous, latest_run_id: newRun.id, latest_run_status: newRun.status }
      );
      setSelectedRevision('');
      await queryClient.invalidateQueries({ queryKey: ['product', id] });
    },
  });
  const cancelGeneration = useMutation({
    mutationFn: () => api<Run>(`/generations/${runId}/cancel`, { method: 'POST' }),
    onSuccess: (cancelled) => {
      queryClient.setQueryData<Run>(['run', cancelled.id], (previous) =>
        mergeRun(previous, cancelled)
      );
    },
  });
  const deletePassport = useMutation({
    mutationFn: () => api<Product>(`/products/${id}`, { method: 'DELETE' }),
    onSuccess: () => {
      void navigate('/');
      queryClient.removeQueries({ queryKey: ['product', id] });
      queryClient.removeQueries({ queryKey: ['revisions', id] });
      for (const item of history.data ?? []) {
        queryClient.removeQueries({ queryKey: ['revision', item.id] });
      }
      void queryClient.invalidateQueries({ queryKey: ['products'] });
    },
  });
  if (product.isPending) {
    return <Loading label="Loading product space…" />;
  }
  if (product.isError) {
    return <ErrorNotice error={product.error} />;
  }
  const views = revision.data?.content.views;
  const chapters = views === undefined ? undefined : documentChapters(views);
  const sections = views === undefined ? [] : legacySections(views);
  const entries = chapters ?? sections;
  const selected = entries.find((item) => item.id === sectionId) ?? entries.at(0);
  const chapter = chapters?.find((item) => item.id === selected?.id);
  const section =
    chapters === undefined ? sections.find((item) => item.id === selected?.id) : undefined;
  const position = entries.findIndex((item) => item.id === selected?.id);
  const previous = entries.at(position - 1);
  const next = entries.at(position + 1);
  const documentRun =
    revisionRun.data ?? (run.data?.id === revision.data?.run_id ? run.data : undefined);
  const generating = activeRun(run.data?.status ?? product.data.latest_run_status);
  const busy = generating || generate.isPending || deletePassport.isPending;
  const generateLabel = run.data?.status === 'failed' ? 'Generate again' : 'Generate passport';
  return (
    <>
      <Link to="/" className="back-link">
        ← Product spaces
      </Link>
      <div
        className={`passport-heading${revision.data?.content.views.card_theme ? ' product-branded' : ''}`}
        style={productCardTheme(revision.data?.content.views.card_theme)}
      >
        <div>
          <ProductIdentity
            name={product.data.name}
            theme={revision.data?.content.views.card_theme}
          />
          <p className="eyebrow">Product passport</p>
          <h1>{revision.data?.content.model.title ?? product.data.name}</h1>
          <p className="page-description">
            {revision.data?.content.model.tagline ??
              'Connect the sources. Generate the picture. Inspect the evidence.'}
          </p>
          {product.data.card_brandkit && (
            <p className="brandkit-note">
              Product brand kit:{' '}
              {product.data.card_brandkit.file_name === ''
                ? product.data.card_brandkit.url
                : product.data.card_brandkit.file_name}
            </p>
          )}
        </div>
        <GenerationBadges
          run={run.data}
          pending={generate.isPending}
          version={revision.data?.version}
          hasDraft={product.data.latest_revision_id !== '' || revision.data !== undefined}
        />
      </div>
      {isAdmin(me) && (
        <button
          type="button"
          className="text-button destructive"
          disabled={busy}
          onClick={() => {
            deletePassport.reset();
            setConfirmDelete(true);
          }}
        >
          <Trash2 size={16} aria-hidden="true" />
          Delete passport
        </button>
      )}
      {confirmDelete && (
        <DeleteConfirmation
          title="Delete product passport?"
          label="Delete passport"
          pending={deletePassport.isPending}
          disabled={busy}
          error={deletePassport.error}
          onClose={() => setConfirmDelete(false)}
          onConfirm={() => deletePassport.mutate()}
        >
          Delete “{product.data.name}”, all passport versions, connected source settings and
          generation data? This cannot be undone. Original repositories stay unchanged.
        </DeleteConfirmation>
      )}
      {(manage || (history.data?.length ?? 0) > 0) && (
        <div className="generation-controls">
          {manage && (
            <>
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
                {busy ? 'Generating…' : generateLabel}
              </button>
              <CancelGenerationButton
                status={run.data?.status}
                pending={cancelGeneration.isPending}
                onCancel={() => cancelGeneration.mutate()}
              />
            </>
          )}
          <VersionSelect history={history.data} value={revisionId} onChange={setSelectedRevision} />
        </div>
      )}
      <MissingConnectionNotice
        manage={manage}
        admin={isAdmin(me)}
        available={availableConnections.length}
        setupEnabled={setup.data?.connections_enabled}
        loading={connections.isPending}
      />
      {generate.isError && <ErrorNotice error={generate.error} />}
      {cancelGeneration.isError && <ErrorNotice error={cancelGeneration.error} />}
      {run.isError && <ErrorNotice error={run.error} />}
      <GenerationOutcome run={run.data} hasRevision={revision.data !== undefined} />
      {run.data !== undefined && (
        <GenerationProgress key={run.data.id} run={run.data} streamState={streamState} />
      )}
      {run.data !== undefined && run.data.id !== revision.data?.run_id && (
        <PartialResearchNotice
          events={run.data.events}
          label="This generation has incomplete research"
        />
      )}
      {revision.isFetching && <Loading label="Loading passport revision…" />}
      {revision.isError && <ErrorNotice error={revision.error} />}
      {history.isError && <ErrorNotice error={history.error} />}
      {revision.data === undefined &&
        !busy &&
        !revision.isFetching &&
        run.data?.status !== 'failed' && (
          <EmptyState title="Your sources are ready to explore">
            {emptyPassportMessage(manage)}
          </EmptyState>
        )}
      {revision.data !== undefined && selected !== undefined && (
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
          <PartialResearchNotice
            events={documentRun?.events}
            label="This passport has incomplete research"
          />
          {chapters === undefined && !sections.some((item) => item.id === 'repositories') && (
            <p className="notice">
              This draft uses an earlier structure. Generate a new passport to add repository
              connections, technologies and deeper architecture research.
            </p>
          )}
          <div
            className={`passport-layout${revision.data.content.views.card_theme ? ' product-passport' : ''}`}
            style={productCardTheme(revision.data.content.views.card_theme)}
          >
            <nav className="section-nav" aria-label="Passport sections">
              {entries.map((item, index) => (
                <button
                  type="button"
                  key={item.id}
                  className={item.id === selected.id ? 'active' : ''}
                  aria-current={item.id === selected.id ? 'true' : undefined}
                  onClick={() => selectSection(item.id)}
                >
                  <span>{String(index + 1).padStart(2, '0')}</span>
                  {chapters === undefined ? (sectionTitles[item.id] ?? item.title) : item.title}
                </button>
              ))}
            </nav>
            <div className="passport-reading-pane" ref={readingPane}>
              {chapter !== undefined && (
                <DocumentContent
                  key={`${revision.data.id}:${chapter.id}`}
                  chapter={chapter}
                  revision={revision.data}
                  onObject={setObject}
                  onEvidence={setEvidenceId}
                />
              )}
              {section !== undefined && (
                <SectionContent
                  key={`${revision.data.id}:${section.id}`}
                  section={section}
                  revision={revision.data}
                  onObject={setObject}
                  onEvidence={setEvidenceId}
                />
              )}
              {chapters !== undefined && (
                <nav className="chapter-navigation" aria-label="Continue reading">
                  {position > 0 && previous !== undefined && (
                    <button
                      type="button"
                      className="button quiet"
                      onClick={() => selectSection(previous.id)}
                    >
                      <span>Previous chapter</span>
                      {previous.title}
                    </button>
                  )}
                  {next !== undefined && (
                    <button
                      type="button"
                      className="button quiet next-chapter"
                      onClick={() => selectSection(next.id)}
                    >
                      <span>Next chapter</span>
                      {next.title}
                      <ArrowRight size={16} aria-hidden="true" />
                    </button>
                  )}
                </nav>
              )}
            </div>
          </div>
          <ProductBrandSources theme={revision.data.content.views.card_theme} />
          <details className="sources-panel">
            <summary>
              <GitBranch size={17} />
              Analyzed sources and coverage
              <DisclosureHint show="Show sources" hide="Hide sources" />
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
                  <SourceCoverage snapshot={snapshot} />
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
          <DisclosureHint show="Show configuration" hide="Hide configuration" />
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
      {manage && <ProductAccess productId={id} />}
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

function CancelGenerationButton({
  status,
  pending,
  onCancel,
}: Readonly<{
  status: string | undefined;
  pending: boolean;
  onCancel: () => void;
}>) {
  if (!activeRun(status)) {
    return null;
  }
  return (
    <button type="button" className="button quiet" disabled={pending} onClick={onCancel}>
      <Square size={16} aria-hidden="true" />
      {pending ? 'Cancelling…' : 'Cancel generation'}
    </button>
  );
}

function GenerationOutcome({
  run,
  hasRevision,
}: Readonly<{ run: Run | undefined; hasRevision: boolean }>) {
  if (run?.status !== 'failed') {
    return null;
  }
  const cancelled = cancelledRun(run);
  return (
    <div className={cancelled ? 'notice' : 'notice error'} role={cancelled ? 'status' : 'alert'}>
      <div>
        <strong>
          {cancelled
            ? 'Generation cancelled.'
            : generationMessage(`Generation stopped: ${stageLabels[run.stage] ?? run.stage}.`)}
        </strong>
        <p>
          {cancelled
            ? 'You can generate the passport again.'
            : generationMessage(run.error_message)}
        </p>
        {hasRevision && <p>The previous passport remains available below.</p>}
      </div>
    </div>
  );
}

function VersionSelect({
  history,
  value,
  onChange,
}: Readonly<{ history?: RevisionInfo[]; value: string; onChange: (id: string) => void }>) {
  if (history === undefined || history.length === 0) {
    return null;
  }
  return (
    <label className="revision-select">
      Version
      <select value={value} onChange={(event) => onChange(event.target.value)}>
        {history.map((item) => (
          <option key={item.id} value={item.id}>
            Version {item.version}
          </option>
        ))}
      </select>
    </label>
  );
}

function MissingConnectionNotice({
  manage,
  admin,
  available,
  setupEnabled,
  loading,
}: Readonly<{
  manage: boolean;
  admin: boolean;
  available: number;
  setupEnabled?: boolean;
  loading: boolean;
}>) {
  // Connections are only fetched for people who can generate, once setup allows them.
  if (!manage || available > 0 || (setupEnabled !== false && loading)) {
    return null;
  }
  return (
    <div className="notice">
      <p>Add an enabled model connection to generate this passport.</p>
      {admin ? (
        <Link className="text-link" to="/connections">
          Set up a connection
          <ArrowRight size={15} />
        </Link>
      ) : (
        <p>Ask an admin to set one up.</p>
      )}
    </div>
  );
}

function emptyPassportMessage(manage: boolean): string {
  return manage
    ? 'Choose a connection and model, then generate a passport. It will cover the product’s purpose, users, domain, scenarios, architecture and developer workflow.'
    : 'No passport has been generated yet. A product lead can generate it.';
}
