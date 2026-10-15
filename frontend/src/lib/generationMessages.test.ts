import assert from 'node:assert/strict';
import test from 'node:test';
import { generationMessage } from './generationMessages.ts';

test('historical source-reading progress uses plain English', () => {
  assert.equal(
    generationMessage('Reading source code through Repomix MCP (3 ranges).'),
    'Reading source code (3 ranges).'
  );
  assert.equal(generationMessage('39 Repomix reads'), '39 source reads');
  assert.equal(generationMessage('start Repomix MCP'), 'start source reader');
  assert.equal(
    generationMessage('Research completed: 39 Repomix reads; 12 source files read.'),
    'Research completed: 39 source reads; 12 source files read.'
  );
});

test('historical tool errors hide internal names and preserve the failure reason', () => {
  assert.equal(
    generationMessage('start Repomix MCP: connection refused'),
    'start source reader: connection refused'
  );
  assert.equal(generationMessage('MCP request timed out'), 'source tools request timed out');
  assert.equal(generationMessage('mcp__repomix__read_file: failed'), 'source tool: failed');
  assert.equal(generationMessage('repomix.read_file: unavailable'), 'source reader: unavailable');
});

test('generic generation messages keep their original text', () => {
  const value = 'Reading source code (3 ranges).\nChecking the result.';
  assert.equal(generationMessage(value), value);
  assert.equal(generationMessage(''), '');
});
