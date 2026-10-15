import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ArrowRight, FileCode2, GitBranch, Play, RefreshCw, Square, Trash2 } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { DeleteConfirmation } from '../components/DeleteConfirmation';
import { DocumentContent } from '../components/DocumentContent';
import { GenerationBadges, GenerationProgress } from '../components/GenerationProgress';
import { PartialResearchNotice } from '../components/PartialResearchNotice';
import { EvidenceDialog, ObjectDetail, SectionContent } from '../components/PassportContent';
import { PassportPublication } from '../components/PassportPublication';
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

export function PassportPage({ publishedView = false }: Readonly<{ publishedView?: boolean }>) {
  const { id = '' } = useParams();
  const navigate = useNavigate();
  const [confirmDelete, setConfirmDelete] = useState(false);
  const queryClient = useQueryClient();
  const me = useMe();
  const manage = !publishedView && canManage(me, id);
  const product = useQuery({
    queryKey: ['product', id],
    queryFn: ({ signal }) => api<Product>(`/products/${id}`, { signal }),
  });
  const setup = useQuery({
    queryKey: ['factory-status'],
    queryFn: ({ signal }) => api<FactoryStatus>('/factory/status', { signal }),
    enabled: !publishedView,
  });
  const connections = useQuery({
    queryKey: ['connections'],
    queryFn: ({ signal }) => api<Connection[]>('/llm-connections', { signal }),
    enabled: manage && setup.data?.connections_enabled === true,
  });
  const history = useQuery({
    queryKey: ['revisions', id],
    queryFn: ({ signal }) => api<RevisionInfo[]>(`/products/${id}/passport-revisions`, { signal }),
    enabled: !publishedView,
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
  const runId = publishedView ? '' : (product.data?.latest_run_id ?? '');
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
  const { drafts, revisionId, revision, revisionRun } = usePassportRevision({
    id,
    selectedRevision,
    publishedView,
    product: product.data,
    history: history.data,
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
      void navigate('/products');
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
  const generating =
    !publishedView && activeRun(run.data?.status ?? product.data.latest_run_status);
  const busy = generating || generate.isPending || deletePassport.isPending;
  const generateLabel = run.data?.status === 'failed' ? 'Generate again' : 'Generate passport';
  return (
    <>
      <PassportHeader
        product={product.data}
        revision={revision.data}
        run={run.data}
        pending={generate.isPending}
        publishedView={publishedView}
        hasRevision={revisionId !== ''}
      />
      {!publishedView && (
        <nav className="passport-tabs" aria-label="Passport views">
          <Link to={`/products/${id}`} className="active" aria-current="page">
            Drafts
          </Link>
          <Link to={`/published/${id}`}>Published</Link>
        </nav>
      )}
      {manage && isAdmin(me) && (
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
      {manage && (
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
            {busy ? 'Generating…' : generateLabel}
          </button>
          <CancelGenerationButton
            status={run.data?.status}
            pending={cancelGeneration.isPending}
            onCancel={() => cancelGeneration.mutate()}
          />
          <VersionSelect history={drafts} value={revisionId} onChange={setSelectedRevision} />
        </div>
      )}
      {!publishedView && (
        <MissingConnectionNotice
          manage={manage}
          admin={isAdmin(me)}
          available={availableConnections.length}
          setupEnabled={setup.data?.connections_enabled}
          loading={connections.isPending}
        />
      )}
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
      {manage && revision.data !== undefined && (
        <PassportPublication
          key={revision.data.id}
          revision={revision.data}
          history={history.data ?? []}
        />
      )}
      {publishedView && revisionId === '' && !product.isFetching && (
        <EmptyState title="No published passport yet">
          This product does not have a published passport yet.
        </EmptyState>
      )}
      {!publishedView &&
        revision.data === undefined &&
        !busy &&
        !revision.isFetching &&
        run.data?.status !== 'failed' && (
          <EmptyDraft hasPublished={product.data.published_revision_id !== ''} />
        )}
      {revision.data !== undefined && selected !== undefined && (
        <>
          {!publishedView && (
            <div className="revision-meta">
              <span>Version {revision.data.version}</span>
              <span>{date(revision.data.created_at)}</span>
              {revision.data.published && <span>Published {date(revision.data.published_at)}</span>}
              {revisionRun.data !== undefined && (
                <span>
                  {revisionRun.data.model} · connection revision{' '}
                  {revisionRun.data.connection_revision}
                </span>
              )}
              <span>
                {revision.data.approved_at === ''
                  ? 'Expert review pending'
                  : `Approved ${date(revision.data.approved_at)}`}
              </span>
            </div>
          )}
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
          {!publishedView && <ProductBrandSources theme={revision.data.content.views.card_theme} />}
          <details className="sources-panel analyzed-sources">
            <summary>
              <GitBranch size={17} />
              Analyzed sources and coverage
              <DisclosureHint show="Show sources" hide="Hide sources" />
            </summary>
            {revision.data.snapshots?.map((snapshot) => (
              <div className="source-snapshot" key={snapshot.id}>
                <div className="source-snapshot-heading">
                  <div className="source-snapshot-identity">
                    <strong>
                      {snapshot.locator
                        .replace(/^https:\/\/github\.com\//, '')
                        .replace(/\.git$/, '')}
                    </strong>
                    <div className="source-snapshot-meta">
                      <span>{snapshot.coverage.file_count.toLocaleString()} files</span>
                      <span>
                        {new Intl.NumberFormat('en', { maximumFractionDigits: 1 }).format(
                          snapshot.coverage.bytes / 1024 / 1024
                        )}{' '}
                        MB
                      </span>
                      <span>
                        Commit <code>{snapshot.revision.slice(0, 12)}</code>
                      </span>
                    </div>
                  </div>
                  {snapshot.locator.startsWith('https://github.com/') && (
                    <ExternalLink
                      href={`${snapshot.locator.replace(/\.git$/, '')}/tree/${snapshot.revision}`}
                    >
                      Repository
                    </ExternalLink>
                  )}
                </div>
                <SourceCoverage snapshot={snapshot} />
              </div>
            ))}
          </details>
        </>
      )}
      {!publishedView && (
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
      )}
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
            Version {item.version} · {revisionStatus(item)}
          </option>
        ))}
      </select>
    </label>
  );
}

function revisionStatus(revision: RevisionInfo): string {
  if (revision.published) {
    return 'Published';
  }
  return revision.approved_at === '' ? 'Draft' : 'Approved';
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

function EmptyDraft({ hasPublished }: Readonly<{ hasPublished: boolean }>) {
  return (
    <EmptyState
      title={hasPublished ? 'No unpublished drafts' : 'Your sources are ready to explore'}
    >
      {hasPublished
        ? 'Generate a new version to update the published passport.'
        : 'Choose a connection and model, then generate a passport.'}
    </EmptyState>
  );
}

function selectedDraftId(
  drafts: RevisionInfo[] | undefined,
  product: Product | undefined,
  selected: string
): string {
  if (drafts !== undefined) {
    return drafts.find((item) => item.id === selected)?.id ?? drafts.at(0)?.id ?? '';
  }
  return product?.latest_revision_id === product?.published_revision_id
    ? ''
    : (product?.latest_revision_id ?? '');
}

function PassportHeader({
  product,
  revision,
  run,
  pending,
  publishedView,
  hasRevision,
}: Readonly<{
  product: Product;
  revision: Revision | undefined;
  run: Run | undefined;
  pending: boolean;
  publishedView: boolean;
  hasRevision: boolean;
}>) {
  return (
    <>
      <Link to={publishedView ? '/published' : '/products'} className="back-link">
        ← {publishedView ? 'Published passports' : 'Product studio'}
      </Link>
      <div
        className={`passport-heading${revision?.content.views.card_theme ? ' product-branded' : ''}`}
        style={productCardTheme(revision?.content.views.card_theme)}
      >
        <div className="passport-heading-content">
          <p className="eyebrow">Product passport</p>
          <div className="passport-identity">
            <div className="passport-product-mark">
              <ProductIdentity name={product.name} theme={revision?.content.views.card_theme} />
            </div>
            <h1>{revision?.content.model.title ?? product.name}</h1>
          </div>
          <p className="page-description">
            {revision?.content.model.tagline ??
              'Connect the sources. Generate the picture. Inspect the evidence.'}
          </p>
          {!publishedView && product.card_brandkit && (
            <p className="brandkit-note">
              Product brand kit:{' '}
              {product.card_brandkit.file_name === ''
                ? product.card_brandkit.url
                : product.card_brandkit.file_name}
            </p>
          )}
        </div>
        {!publishedView && (
          <GenerationBadges
            run={run}
            pending={pending}
            version={revision?.version}
            hasDraft={hasRevision}
            approvedAt={revision?.approved_at}
            emptyLabel="No drafts yet"
            published={revision?.published}
          />
        )}
      </div>
    </>
  );
}

function usePassportRevision({
  id,
  selectedRevision,
  publishedView,
  product,
  history,
}: Readonly<{
  id: string;
  selectedRevision: string;
  publishedView: boolean;
  product: Product | undefined;
  history: RevisionInfo[] | undefined;
}>) {
  const drafts = history?.filter((item) => !item.published);
  const draftRevisionId = selectedDraftId(drafts, product, selectedRevision);
  const revisionId = publishedView ? (product?.published_revision_id ?? '') : draftRevisionId;
  const revision = useQuery({
    queryKey: [publishedView ? 'published-revision' : 'revision', revisionId],
    queryFn: ({ signal }) =>
      api<Revision>(
        publishedView ? `/products/${id}/passport` : `/passport-revisions/${revisionId}`,
        { signal }
      ),
    enabled: revisionId !== '',
  });
  const revisionRun = useQuery({
    queryKey: ['run', revision.data?.run_id ?? ''],
    queryFn: ({ signal }) => api<Run>(`/generations/${revision.data?.run_id ?? ''}`, { signal }),
    enabled: !publishedView && revision.data !== undefined,
  });
  return { drafts, revisionId, revision, revisionRun };
}
