import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ArrowRight, BookOpen, FileCode2, Plus } from 'lucide-react';
import { useState, type FormEvent } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { Badge, EmptyState, ErrorNotice, Loading, PageHeading } from '../components/ui';
import { api, date, type Product, type ProductInput } from '../lib/api';
import { isAdmin, useMe } from '../lib/auth';
import { productCardTheme } from '../lib/productCardTheme';

export function Products() {
  const admin = isAdmin(useMe());
  const query = useQuery({
    queryKey: ['products'],
    queryFn: ({ signal }) => api<Product[]>('/products', { signal }),
  });
  return (
    <>
      <PageHeading
        eyebrow="Product spaces"
        title="Understanding starts here."
        description="Turn product sources into a shared, traceable picture of what you build."
        action={
          admin && (
            <Link className="button primary" to="/products/new">
              <Plus size={17} />
              Add product
            </Link>
          )
        }
      />
      <div className="intro-strip">
        <BookOpen size={19} aria-hidden="true" />
        <p>
          A product passport connects purpose, people, domain and architecture. Every generated
          version starts as a draft.
        </p>
      </div>
      {query.isPending && <Loading />}
      {query.isError && <ErrorNotice error={query.error} />}
      {query.data?.length === 0 && !admin && (
        <EmptyState title="No product spaces yet">
          Ask an admin or a product lead to give you access to a product.
        </EmptyState>
      )}
      {query.data?.length === 0 && admin && (
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
      <div className="product-grid">
        {query.data?.map((product) => (
          <Link
            className={`product-card${product.card_theme ? ' product-branded' : ''}`}
            style={productCardTheme(product.card_theme)}
            to={`/products/${product.id}`}
            key={product.id}
          >
            <div className="card-top">
              <span className="product-initial">{product.name.slice(0, 1).toUpperCase()}</span>
              <Badge>
                {product.latest_revision_id === '' ? 'Sources connected' : 'Draft available'}
              </Badge>
            </div>
            <h2>{product.name}</h2>
            <p>
              <FileCode2 size={15} aria-hidden="true" />
              {product.sources?.length ?? 0} sources
            </p>
            <div className="card-bottom">
              <span>{date(product.created_at)}</span>
              <ArrowRight size={18} aria-hidden="true" />
            </div>
          </Link>
        ))}
      </div>
    </>
  );
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
          throw new Error('Product card brand kit files must be 100 KB or smaller.');
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
      <Link to="/" className="back-link">
        ← Product spaces
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
          <legend>Product card brand kit (optional)</legend>
          <p className="product-card-brandkit-description">
            Add brand guidelines so the LLM can style your product card with your brand colours.
          </p>
          <label>
            Product card brand kit source
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
              Product card brand kit URL
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
              Product card brand kit file
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
          <Link className="button quiet" to="/">
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
