import type { Product } from './api';
import type { CSSProperties } from 'react';

const hexColour = /^#[\da-f]{6}$/i;

function foreground(background: string): string {
  const channels = [1, 3, 5].map((offset) => {
    const value = Number.parseInt(background.slice(offset, offset + 2), 16) / 255;
    return value <= 0.040_45 ? value / 12.92 : ((value + 0.055) / 1.055) ** 2.4;
  });
  const luminance = 0.2126 * channels[0] + 0.7152 * channels[1] + 0.0722 * channels[2];
  // Black or white always provides at least 4.5:1 contrast for a solid background.
  return luminance > 0.179 ? '#000000' : '#FFFFFF';
}

export function productCardTheme(theme: Product['card_theme']): CSSProperties | undefined {
  if (!theme || !hexColour.test(theme.accent_color) || !hexColour.test(theme.background_color)) {
    return undefined;
  }
  return {
    '--product-card-accent': theme.accent_color,
    '--product-card-on-accent': foreground(theme.accent_color),
    '--product-card-background': theme.background_color,
    '--product-card-ink': foreground(theme.background_color),
  } as CSSProperties;
}
