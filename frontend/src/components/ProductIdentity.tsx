import { useState } from 'react';
import type { Product } from '../lib/api';

export function ProductIdentity({
  name,
  theme,
}: Readonly<{ name: string; theme: Product['card_theme'] }>) {
  const [failedURL, setFailedURL] = useState('');
  const logo = theme?.logo_url ?? '';
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
