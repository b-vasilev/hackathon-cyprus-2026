import { useQuery } from '@tanstack/react-query';
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
        title="Products"
        description="Explore approved product passports."
      />
      {products.isPending && <Loading />}
      {products.isError && <ErrorNotice error={products.error} />}
      {products.data?.length === 0 && (
        <EmptyState title="No published passports yet">
          Published passports for your products will appear here.
        </EmptyState>
      )}
      <div className="published-grid">
        {products.data?.map((product) => (
          <Link
            className="published-tile"
            style={productCardTheme(product.card_theme)}
            key={product.id}
            to={`/published/${product.id}`}
          >
            <span className="published-tile-icon">
              <ProductIdentity
                name={product.name}
                theme={product.card_theme}
                icon={product.icon_url}
              />
            </span>
            <h2>{product.name}</h2>
          </Link>
        ))}
      </div>
    </>
  );
}
