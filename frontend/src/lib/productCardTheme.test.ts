import assert from 'node:assert/strict';
import test from 'node:test';
import { productCardTheme } from './productCardTheme.ts';

function luminance(hex: string): number {
  const values = [1, 3, 5].map((offset) => {
    const value = Number.parseInt(hex.slice(offset, offset + 2), 16) / 255;
    return value <= 0.040_45 ? value / 12.92 : ((value + 0.055) / 1.055) ** 2.4;
  });
  return values[0] * 0.2126 + values[1] * 0.7152 + values[2] * 0.0722;
}

function contrast(first: string, second: string): number {
  const values = [luminance(first), luminance(second)].sort((a, b) => b - a);
  return (values[0] + 0.05) / (values[1] + 0.05);
}

test('product accents stay readable for light, dark and saturated brand palettes', () => {
  for (const accent of ['#FFFFFF', '#FFFF00', '#000000', '#2563EB', '#006F60']) {
    for (const background of ['#FFFFFF', '#000000', '#808080']) {
      const style = productCardTheme({
        accent_color: accent,
        background_color: background,
        logo_url: '',
        sources: [],
      }) as Record<string, string>;
      assert.ok(contrast(style['--product-card-on-accent'], accent) >= 4.5);
      assert.ok(contrast(style['--product-card-ink'], background) >= 4.5);
      assert.ok(contrast(style['--product-card-link'], background) >= 4.5);
      assert.ok(contrast(style['--product-content-accent'], '#FFFFFF') >= 4.5);
      assert.ok(
        contrast(style['--product-content-accent'], style['--product-content-soft']) >= 4.5
      );
      assert.equal(style['--of-brand'], undefined, 'product theme must not replace portal tokens');
    }
  }
});

test('missing or invalid themes use portal defaults', () => {
  assert.equal(productCardTheme(undefined), undefined);
  assert.equal(
    productCardTheme({
      accent_color: 'red',
      background_color: '#FFFFFF',
      logo_url: '',
      sources: [],
    }),
    undefined
  );
});
