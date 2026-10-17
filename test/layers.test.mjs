import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readdirSync, readFileSync } from 'node:fs';
import { join, dirname, relative, resolve, sep } from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = join(dirname(fileURLToPath(import.meta.url)), '..');
const SRC = join(ROOT, 'src');
const LOGIC = ['ask', 'build', 'home', 'loader', 'queue', 'review', 'settings'];

function modules(folder) {
  return readdirSync(folder, { withFileTypes: true }).flatMap(entry => {
    const path = join(folder, entry.name);
    if (entry.isDirectory()) return modules(path);
    return entry.name.endsWith('.mjs') ? [path] : [];
  });
}

function layered() {
  return modules(SRC).map(path => {
    const text = readFileSync(path, 'utf8');
    const imports = [...text.matchAll(/^(?:import|export)\b[^;]*?\bfrom '([^']+)';$/gm)].map(([, spec]) => {
      if (spec.startsWith('node:')) return { spec, layer: 'node:' };
      const target = relative(SRC, resolve(dirname(path), spec));
      return { spec, layer: target.startsWith('..') ? target : target.split(sep)[0] };
    });
    return { name: relative(SRC, path), layer: relative(SRC, path).split(sep)[0], text, imports };
  });
}

function importsBreaking(layers, allowed) {
  return layered()
    .filter(m => layers.includes(m.layer))
    .flatMap(m => m.imports.filter(i => !allowed(i.layer, m.layer)).map(i => `${m.name} imports ${i.spec}`));
}

test('commands import only node\'s own modules, the models and the strings', () => {
  assert.deepEqual(importsBreaking(['commands'], (to, from) => ['node:', 'models', 'i18n', from].includes(to)), []);
});

test('models import only node\'s own modules and the strings', () => {
  assert.deepEqual(importsBreaking(['models'], (to, from) => ['node:', 'i18n', from].includes(to)), []);
});

test('the logic and the store never import what the user sees, nor the startup', () => {
  assert.deepEqual(importsBreaking([...LOGIC, 'store'], to => !['server', 'cli', 'app'].includes(to)), []);
});

test('the server never imports the command line', () => {
  assert.deepEqual(importsBreaking(['server'], to => to !== 'cli'), []);
});

test('only the commands run other programs', () => {
  const running = layered().filter(m => m.layer !== 'commands' && /\b(execFile|execFileSync|spawn)\b|node:child_process/.test(m.text)).map(m => m.name);
  assert.deepEqual(running, []);
});

test('every folder of src/ has a layer', () => {
  const folders = readdirSync(SRC, { withFileTypes: true }).filter(e => e.isDirectory()).map(e => e.name);
  assert.deepEqual(folders.filter(f => ![...LOGIC, 'commands', 'models', 'i18n', 'store', 'server', 'cli', 'app'].includes(f)), []);
});
