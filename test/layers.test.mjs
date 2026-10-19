import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readdirSync, readFileSync } from 'node:fs';
import { join, dirname, relative, resolve, sep } from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = join(dirname(fileURLToPath(import.meta.url)), '..');
const SRC = join(ROOT, 'src');
const LOGIC = ['ask', 'build', 'config', 'home', 'loader', 'queue', 'review'];

function modules(folder) {
  return readdirSync(folder, { withFileTypes: true }).flatMap(entry => {
    const path = join(folder, entry.name);
    if (entry.isDirectory()) return modules(path);
    return entry.name.endsWith('.mjs') ? [path] : [];
  });
}

function layered(root) {
  return modules(root).map(path => {
    const text = readFileSync(path, 'utf8');
    const imports = [...text.matchAll(/^(?:import|export)\b[^;]*?\bfrom '([^']+)';$/gm)].map(([, spec]) => {
      if (spec.startsWith('node:')) return { spec, layer: 'node:' };
      const target = relative(root, resolve(dirname(path), spec));
      return { spec, layer: target.startsWith('..') ? target : target.split(sep)[0] };
    });
    return { name: relative(root, path), layer: relative(root, path).split(sep)[0], text, imports };
  });
}

function importsBreaking(root, layers, allowed) {
  return layered(root)
    .filter(m => layers.includes(m.layer))
    .flatMap(m => m.imports.filter(i => !allowed(i.layer, m.layer)).map(i => `${m.name} imports ${i.spec}`));
}

function naming(root, layers, pattern) {
  return layered(root).filter(m => !layers.includes(m.layer) && pattern.test(m.text)).map(m => m.name);
}

test('commands import only node\'s own modules, the models and the strings', () => {
  assert.deepEqual(importsBreaking(SRC, ['commands'], (to, from) => ['node:', 'models', 'i18n', from].includes(to)), []);
});

test('models import only node\'s own modules and the strings', () => {
  assert.deepEqual(importsBreaking(SRC, ['models'], (to, from) => ['node:', 'i18n', from].includes(to)), []);
});

test('the logic and the store never import what the user sees, nor the startup', () => {
  assert.deepEqual(importsBreaking(SRC, [...LOGIC, 'store'], to => !['server', 'cli', 'app'].includes(to)), []);
});

test('the server never imports the command line', () => {
  assert.deepEqual(importsBreaking(SRC, ['server'], to => to !== 'cli'), []);
});

test('the strings import nothing', () => {
  assert.deepEqual(importsBreaking(SRC, ['i18n'], () => false), []);
});

test('only the commands run other programs', () => {
  assert.deepEqual(naming(SRC, ['commands'], /\b(execFile|execFileSync|spawn)\b|node:child_process/), []);
});

test('every folder of src/ has a layer', () => {
  const folders = readdirSync(SRC, { withFileTypes: true }).filter(e => e.isDirectory()).map(e => e.name);
  assert.deepEqual(folders.filter(f => ![...LOGIC, 'commands', 'models', 'i18n', 'store', 'server', 'cli', 'app'].includes(f)), []);
});
