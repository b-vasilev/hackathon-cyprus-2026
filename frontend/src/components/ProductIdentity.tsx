import { useMutation, useQueryClient } from '@tanstack/react-query';
import { ImageUp, X } from 'lucide-react';
import { useId, useState } from 'react';
import { api, type Product } from '../lib/api';
import { ErrorNotice } from './ui';

const maxIconBytes = 256_000;

/** A custom icon takes priority over the logo discovered for the passport. */
export function ProductIdentity({
  name,
  theme,
  icon = '',
}: Readonly<{ name: string; theme: Product['card_theme']; icon?: string }>) {
  const [failedURL, setFailedURL] = useState('');
  const logo = icon === '' ? (theme?.logo_url ?? '') : icon;
  const safeLogo = /^data:image\/(?:svg\+xml|png|jpeg|gif);base64,[\da-z+/=]+$/i.test(logo);
  if (!safeLogo || failedURL === logo) {
    return (
      <span className="product-initial" aria-hidden="true">
        {name.slice(0, 1).toUpperCase()}
      </span>
    );
  }
  return (
    <span className="product-logo">
      <img src={logo} alt="" width={128} height={44} onError={() => setFailedURL(logo)} />
    </span>
  );
}

export function ProductBrandSources({ theme }: Readonly<{ theme: Product['card_theme'] }>) {
  return (
    <details className="sources-panel brand-sources">
      <summary>Product brand sources</summary>
      {(theme?.sources?.length ?? 0) === 0 ? (
        <p className="meta">
          No verified product brand sources in this revision. Wayfinder defaults fill missing
          values.
        </p>
      ) : (
        <>
          <p className="meta">
            Original product assets and colours. Wayfinder defaults fill missing values.
          </p>
          <ul>
            {theme?.sources?.map((source) => (
              <li key={`${source.url}:${source.label}`}>
                {source.url.startsWith('https://') ? (
                  <a href={source.url} target="_blank" rel="noreferrer">
                    {source.label}
                  </a>
                ) : (
                  source.label
                )}
              </li>
            ))}
          </ul>
        </>
      )}
    </details>
  );
}

/** Lets admins and product leads upload or remove the product's custom icon. */
export function ProductIconControl({ product }: Readonly<{ product: Product }>) {
  const queryClient = useQueryClient();
  const inputId = useId();
  const [tooLarge, setTooLarge] = useState(false);
  const update = useMutation({
    mutationFn: async (file: File | null) => {
      const path = `/products/${product.id}/icon`;
      if (file === null) {
        return api<Product>(path, { method: 'DELETE' });
      }
      const bytes = new Uint8Array(await file.arrayBuffer());
      const data = btoa(Array.from(bytes, (byte) => String.fromCodePoint(byte)).join(''));
      return api<Product>(path, { method: 'PUT', body: JSON.stringify({ data }) });
    },
    onSuccess: async (saved) => {
      queryClient.setQueryData(['product', saved.id], saved);
      await queryClient.invalidateQueries({ queryKey: ['products'] });
    },
  });
  return (
    <div className="product-icon-control">
      <input
        id={inputId}
        className="visually-hidden"
        type="file"
        accept=".svg,.png,.jpg,.jpeg,.gif,image/svg+xml,image/png,image/jpeg,image/gif"
        disabled={update.isPending}
        onChange={(event) => {
          const file = event.target.files?.[0];
          event.target.value = '';
          if (file === undefined) {
            return;
          }
          setTooLarge(file.size > maxIconBytes);
          if (file.size <= maxIconBytes) {
            update.mutate(file);
          }
        }}
      />
      <label htmlFor={inputId} className="text-button" aria-disabled={update.isPending}>
        <ImageUp size={16} aria-hidden="true" />
        {product.icon_url === '' ? 'Upload icon' : 'Change icon'}
      </label>
      {product.icon_url !== '' && (
        <button
          type="button"
          className="text-button"
          disabled={update.isPending}
          onClick={() => update.mutate(null)}
        >
          <X size={16} aria-hidden="true" />
          Remove icon
        </button>
      )}
      <span className="meta">SVG, PNG, JPEG or GIF, up to 256 KB.</span>
      {tooLarge && <ErrorNotice error={new Error('Choose an image of 256 KB or smaller.')} />}
      {update.isError && <ErrorNotice error={update.error} />}
    </div>
  );
}
