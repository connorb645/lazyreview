import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readdirSync, readFileSync } from 'node:fs';
import { join, dirname, relative, sep } from 'node:path';
import { fileURLToPath } from 'node:url';
import { english } from '../../src/i18n/english.mjs';

const SRC = join(dirname(fileURLToPath(import.meta.url)), '..', '..', 'src');

function modules(folder) {
  return readdirSync(folder, { withFileTypes: true }).flatMap(entry => {
    const path = join(folder, entry.name);
    if (entry.isDirectory()) return modules(path);
    return entry.name.endsWith('.mjs') ? [path] : [];
  });
}

function frozenThrough(value, path = 'english') {
  if (!value || typeof value !== 'object') return [];
  return [...(Object.isFrozen(value) ? [] : [path]), ...Object.entries(value).flatMap(([key, inner]) => frozenThrough(inner, `${path}.${key}`))];
}

test('no error in src/ is given written words: its sentence comes from the strings', () => {
  const written = modules(SRC)
    .filter(path => relative(SRC, path).split(sep)[0] !== 'i18n')
    .flatMap(path => readFileSync(path, 'utf8').split('\n')
      .map((line, i) => (/new Error\(\s*['"`]/.test(line) ? `${relative(SRC, path)}:${i + 1}: ${line.trim()}` : null))
      .filter(Boolean));
  assert.deepEqual(written, []);
});

test('the strings are frozen, every area of them', () => {
  assert.deepEqual(frozenThrough(english), []);
});
