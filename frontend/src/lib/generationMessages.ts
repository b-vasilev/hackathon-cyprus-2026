// Translate internal tooling names only at the log/error presentation boundary.
// Source evidence and the original API data keep their exact text.
export function generationMessage(value: string): string {
  if (!/repomix|\bmcp\b|\bmcp__/i.test(value)) {
    return value;
  }
  return value
    .replaceAll(/\bthrough[ \t]+repomix(?:[ \t]+mcp)?\b/gi, '')
    .replaceAll(/\brepomix(?:[ \t]+mcp)?[ \t]+reads\b/gi, 'source reads')
    .replaceAll(/\bmcp__\w+/gi, 'source tool')
    .replaceAll(/\brepomix(?:[._]\w+)+/gi, 'source reader')
    .replaceAll(/repomix(?:[ \t]+mcp)?/gi, 'source reader')
    .replaceAll(/\bmcp\b/gi, 'source tools')
    .replaceAll(/[ \t]{2,}/g, ' ')
    .replaceAll(/[ \t]([.,;:!?])/g, '$1')
    .trim();
}
