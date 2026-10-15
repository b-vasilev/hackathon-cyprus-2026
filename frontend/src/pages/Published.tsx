import { useQuery } from '@tanstack/react-query';
import { ArrowUpRight } from 'lucide-react';
import { Link } from 'react-router-dom';
import { ProductIdentity } from '../components/ProductIdentity';
import { EmptyState, ErrorNotice, Loading, PageHeading } from '../components/ui';
import { api, type Product } from '../lib/api';
import { productCardTheme } from '../lib/productCardTheme';

export function PublishedPassports() {
  const products = useQuery({
    queryKey: ['products'],
    queryFn: ({ signal }) => api<Product[]>('/products', { signal }),
    select: (items) => items.filter((item) => item.published_revision_id !== ''),
  });
  return (
    <>
      <PageHeading
        eyebrow="Product passports"
        title="Published passports"
        description="Explore approved product passports."
      />
      {products.isPending && <Loading />}
      {products.isError && <ErrorNotice error={products.error} />}
      {products.data?.length === 0 && (
        <EmptyState title="No published passports yet">
          Published passports for your products will appear here.
        </EmptyState>
      )}
      <div className="product-grid">
        {products.data?.map((product) => (
          <article
            className={`product-card published-card${product.card_theme ? ' product-branded' : ''}`}
            style={productCardTheme(product.card_theme)}
            key={product.id}
          >
            <Link className="product-card-content" to={`/published/${product.id}`}>
              <ProductIdentity name={product.name} theme={product.card_theme} />
              <h2>{product.name}</h2>
              <ArrowUpRight size={20} aria-hidden="true" />
            </Link>
          </article>
        ))}
      </div>
    </>
  );
}
