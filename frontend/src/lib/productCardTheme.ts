import type { Product } from './api';
import type { CSSProperties } from 'react';

const hexColour = /^#[\da-f]{6}$/i;

function luminance(background: string): number {
  const channels = [1, 3, 5].map((offset) => {
    const value = Number.parseInt(background.slice(offset, offset + 2), 16) / 255;
    return value <= 0.040_45 ? value / 12.92 : ((value + 0.055) / 1.055) ** 2.4;
  });
  return 0.2126 * channels[0] + 0.7152 * channels[1] + 0.0722 * channels[2];
}

function foreground(background: string): string {
  // Black or white always provides at least 4.5:1 contrast for a solid background.
  return luminance(background) > 0.179 ? '#000000' : '#FFFFFF';
}

function mix(first: string, second: string, weight: number): string {
  return `#${[1, 3, 5]
    .map((offset) => {
      const a = Number.parseInt(first.slice(offset, offset + 2), 16);
      const b = Number.parseInt(second.slice(offset, offset + 2), 16);
      return Math.round(a * weight + b * (1 - weight))
        .toString(16)
        .padStart(2, '0');
    })
    .join('')}`;
}

function readableAccent(accent: string, background: string): string {
  const surface = luminance(background);
  const target = foreground(background);
  for (let step = 0; step <= 20; step += 1) {
    const colour = mix(accent, target, 1 - step / 20);
    const value = luminance(colour);
    if ((Math.max(value, surface) + 0.05) / (Math.min(value, surface) + 0.05) >= 4.5) {
      return colour;
    }
  }
  return target;
}

export function productCardTheme(theme: Product['card_theme']): CSSProperties | undefined {
  if (!theme || !hexColour.test(theme.accent_color) || !hexColour.test(theme.background_color)) {
    return undefined;
  }
  const soft = mix(theme.accent_color, '#FFFFFF', 0.08);
  return {
    '--product-card-accent': theme.accent_color,
    '--product-card-on-accent': foreground(theme.accent_color),
    '--product-card-background': theme.background_color,
    '--product-card-ink': foreground(theme.background_color),
    '--product-card-link': readableAccent(theme.accent_color, theme.background_color),
    '--product-content-accent': readableAccent(theme.accent_color, soft),
    '--product-content-soft': soft,
  } as CSSProperties;
}
