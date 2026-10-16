import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ArrowRight, Check, Plus, Trash2 } from 'lucide-react';
import { useState, type FormEvent } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { DeleteConfirmation } from '../components/DeleteConfirmation';
import { ProductIdentity } from '../components/ProductIdentity';
import { Badge, EmptyState, ErrorNotice, Loading, PageHeading } from '../components/ui';
import { api, type Product, type ProductInput } from '../lib/api';
import { canManage, isAdmin, useMe } from '../lib/auth';
import { activeRun } from '../lib/generation';
import { productCardTheme } from '../lib/productCardTheme';

export function Products() {
  const me = useMe();
  const admin = isAdmin(me);
  const client = useQueryClient();
  const [deleting, setDeleting] = useState<Product | null>(null);
  const remove = useMutation({
    mutationFn: (id: string) => api<Product>(`/products/${id}`, { method: 'DELETE' }),
    onSuccess: (_, id) => {
      client.setQueryData<Product[]>(['products'], (products) =>
        products?.filter((product) => product.id !== id)
      );
      client.removeQueries({ queryKey: ['product', id] });
      client.removeQueries({ queryKey: ['revisions', id] });
      setDeleting(null);
      void client.invalidateQueries({ queryKey: ['products'] });
    },
  });
  const query = useQuery({
    queryKey: ['products'],
    queryFn: ({ signal }) => api<Product[]>('/products', { signal }),
    // Poll only while a generation is running.
    refetchInterval: (current) =>
      current.state.data?.some((product) => activeRun(product.latest_run_status)) === true
        ? 5000
        : false,
  });
  const products = query.data?.filter((product) => canManage(me, product.id));
  return (
    <>
      <PageHeading
        eyebrow="Product passports"
        title="Studio"
        description="Create, review and publish product passports from your sources."
        action={
          admin && (
            <Link className="button primary" to="/products/new">
              <Plus size={17} />
              Add product
            </Link>
          )
        }
      />
      {query.isPending && <Loading />}
      {query.isError && <ErrorNotice error={query.error} />}
      {products?.length === 0 && !admin && (
        <EmptyState title="No products yet">
          Ask an admin or a product lead to give you access to a product.
        </EmptyState>
      )}
      {products?.length === 0 && admin && (
        <EmptyState
          title="Create your first product passport"
          action={
            <Link className="button primary" to="/products/new">
              Connect product sources
              <ArrowRight size={16} />
            </Link>
          }
        >
          Add a repository and any useful product context. Wayfinder will generate the content and
          diagrams.
        </EmptyState>
      )}
      <div className="published-grid">
        {products?.map((product) => (
          <article
            className="studio-tile"
            style={productCardTheme(product.card_theme)}
            key={product.id}
          >
            <Link className="published-tile" to={`/products/${product.id}`}>
              <span className="published-tile-icon">
                <ProductIdentity
                  name={product.name}
                  theme={product.card_theme}
                  icon={product.icon_url}
                />
              </span>
              <h2>{product.name}</h2>
              <div className="studio-card-statuses">
                <span className="studio-card-status">
                  {product.published_revision_id !== '' && <Check size={14} aria-hidden="true" />}
                  {productStatus(product)}
                </span>
                {product.latest_run_status === 'queued' && (
                  <Badge tone="active">Generation queued</Badge>
                )}
                {product.latest_run_status === 'running' && (
                  <Badge tone="active">
                    {product.latest_revision_id === '' ? 'Generating…' : 'Regenerating…'}
                  </Badge>
                )}
                {product.latest_run_status === 'failed' && (
                  <Badge tone="error">Generation stopped</Badge>
                )}
                {product.latest_run_status === 'succeeded' &&
                  product.published_revision_id !== '' &&
                  product.latest_revision_id !== product.published_revision_id && (
                    <Badge tone="active">New draft ready</Badge>
                  )}
              </div>
            </Link>
            {admin && (
              <button
                type="button"
                className="studio-tile-delete"
                disabled={activeRun(product.latest_run_status) || remove.isPending}
                aria-label={`Delete passport for ${product.name}`}
                title="Delete passport"
                onClick={() => {
                  remove.reset();
                  setDeleting(product);
                }}
              >
                <span className="studio-tile-delete-icon">
                  <Trash2 size={14} aria-hidden="true" />
                </span>
              </button>
            )}
          </article>
        ))}
      </div>
      {deleting !== null && (
        <DeleteConfirmation
          title="Delete product passport?"
          label="Delete passport"
          pending={remove.isPending}
          error={remove.error}
          onClose={() => setDeleting(null)}
          onConfirm={() => remove.mutate(deleting.id)}
        >
          Delete “{deleting.name}”, all passport versions, connected source settings and generation
          data? This cannot be undone. Original repositories stay unchanged.
        </DeleteConfirmation>
      )}
    </>
  );
}

function productStatus(product: Product): string {
  if (product.published_revision_id !== '') {
    return 'Published';
  }
  return product.latest_revision_id === '' ? 'Sources connected' : 'Draft available';
}

export function NewProduct() {
  const navigate = useNavigate();
  const client = useQueryClient();
  const [name, setName] = useState('');
  const [repositories, setRepositories] = useState('');
  const [ref, setRef] = useState('');
  const [include, setInclude] = useState('');
  const [context, setContext] = useState('');
  const [cardBrandkitMode, setCardBrandkitMode] = useState('url');
  const [cardBrandkitUrl, setCardBrandkitUrl] = useState('');
  const [cardBrandkitFile, setCardBrandkitFile] = useState<File | null>(null);
  const create = useMutation({
    mutationFn: async (input: ProductInput) => {
      let cardBrandkit: ProductInput['card_brandkit'];
      if (cardBrandkitMode === 'url' && cardBrandkitUrl.trim() !== '') {
        cardBrandkit = { url: cardBrandkitUrl.trim(), file_name: '', text: '' };
      } else if (cardBrandkitMode === 'file' && cardBrandkitFile !== null) {
        if (!/\.(txt|md|json|css|html)$/i.test(cardBrandkitFile.name)) {
          throw new Error('Choose a TXT, Markdown, JSON, CSS or HTML file.');
        }
        if (cardBrandkitFile.size > 100_000) {
          throw new Error('Product brand kit files must be 100 KB or smaller.');
        }
        const text = new TextDecoder('utf-8', { fatal: true }).decode(
          await cardBrandkitFile.arrayBuffer()
        );
        if (text.trim() === '') {
          throw new Error('The brand kit file is empty.');
        }
        cardBrandkit = { url: '', file_name: cardBrandkitFile.name, text };
      }
      return api<Product>('/products', {
        method: 'POST',
        body: JSON.stringify({ ...input, card_brandkit: cardBrandkit }),
      });
    },
    onSuccess: async (product) => {
      await client.invalidateQueries({ queryKey: ['products'] });
      void navigate(`/products/${product.id}`);
    },
  });
  function submit(event: FormEvent) {
    event.preventDefault();
    const sources: NonNullable<ProductInput['sources']> = repositories
      .split('\n')
      .map((url) => url.trim())
      .filter((url) => url !== '')
      .map((url) => ({ kind: 'github', locator: url, ref, include, text: '' }));
    if (context.trim() !== '') {
      sources.push({ kind: 'text', locator: '', ref: '', include: '', text: context });
    }
    create.mutate({ name, sources });
  }
  return (
    <>
      <Link to="/products" className="back-link">
        ← Studio
      </Link>
      <PageHeading
        eyebrow="New product"
        title="Give Wayfinder its sources."
        description="The passport is generated from your materials. You do not need to write it first."
      />
      <form className="form-panel narrow" onSubmit={submit}>
        <label>
          Product name
          <input
            required
            maxLength={100}
            value={name}
            onChange={(event) => setName(event.target.value)}
            placeholder="A name for this product space"
          />
        </label>
        <label>
          GitHub repositories
          <textarea
            rows={3}
            value={repositories}
            onChange={(event) => setRepositories(event.target.value)}
            placeholder={'https://github.com/organization/product\nOne repository per line'}
          />
        </label>
        <p className="field-help">
          Private repositories use the server’s GITHUB_TOKEN. Each repository is analyzed at a fixed
          commit.
        </p>
        <div className="form-grid">
          <label>
            Branch, tag or commit
            <input
              value={ref}
              onChange={(event) => setRef(event.target.value)}
              placeholder="Default branch"
            />
          </label>
          <label>
            Include patterns
            <input
              value={include}
              onChange={(event) => setInclude(event.target.value)}
              placeholder="Optional, e.g. backend/**,docs/**"
            />
          </label>
        </div>
        <label>
          Additional source context
          <textarea
            rows={6}
            value={context}
            onChange={(event) => setContext(event.target.value)}
            placeholder="Optional product notes, business context or existing documentation."
          />
        </label>
        <p className="field-help">
          Repository files and these notes are source material. Missing information will appear as
          open questions.
        </p>
        <fieldset className="product-card-brandkit-fields" disabled={create.isPending}>
          <legend>Product brand kit (optional)</legend>
          <p className="product-card-brandkit-description">
            Branding is discovered from product sources. Add guidelines here to override the colours
            and logo used on the product card and inside its passport.
          </p>
          <label>
            Product brand kit source
            <select
              value={cardBrandkitMode}
              onChange={(event) => {
                setCardBrandkitMode(event.target.value);
                setCardBrandkitFile(null);
              }}
            >
              <option value="url">Link</option>
              <option value="file">Upload file</option>
            </select>
          </label>
          {cardBrandkitMode === 'url' ? (
            <label>
              Product brand kit URL
              <input
                type="url"
                maxLength={2048}
                value={cardBrandkitUrl}
                onChange={(event) => setCardBrandkitUrl(event.target.value)}
                placeholder="https://example.com/brand/guidelines"
                aria-describedby="product-card-brandkit-help"
              />
            </label>
          ) : (
            <label>
              Product brand kit file
              <input
                type="file"
                accept=".txt,.md,.json,.css,.html"
                onChange={(event) => setCardBrandkitFile(event.target.files?.item(0) ?? null)}
                aria-describedby="product-card-brandkit-help"
              />
            </label>
          )}
          <p id="product-card-brandkit-help" className="field-help">
            {cardBrandkitMode === 'url'
              ? 'Use a direct public HTTPS link to text, HTML, CSS or JSON. No login or redirects. Up to 100 KB.'
              : 'TXT, Markdown, JSON, CSS or HTML in UTF-8. Up to 100 KB.'}{' '}
            The LLM reads the guidelines when you generate the passport.
          </p>
        </fieldset>
        {create.isError && <ErrorNotice error={create.error} />}
        <div className="form-actions">
          <Link className="button quiet" to="/products">
            Cancel
          </Link>
          <button
            className="button primary"
            disabled={create.isPending || (repositories.trim() === '' && context.trim() === '')}
          >
            {create.isPending ? 'Creating…' : 'Create product space'}
            <ArrowRight size={17} />
          </button>
        </div>
      </form>
    </>
  );
}
