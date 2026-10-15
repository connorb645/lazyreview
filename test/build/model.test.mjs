import { test } from 'node:test';
import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { mkdtempSync, writeFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { splitDiff } from '../../src/build/diff/diff-parser.mjs';
import { readChangedTypes, isTestPath } from '../../src/build/model/changed-types.mjs';
import { findRelations } from '../../src/build/model/relations.mjs';
import { chooseContext } from '../../src/build/model/context-selection.mjs';
import { buildModel } from '../../src/build/model/model.mjs';
import { readConfig } from '../../src/build/config/config.mjs';

// The diff git itself makes between two versions of a file.
function changed(path, baseText, text) {
  const folder = mkdtempSync(join(tmpdir(), 'review-desk-test-'));
  const a = join(folder, 'a'), b = join(folder, 'b');
  writeFileSync(a, baseText ?? '');
  writeFileSync(b, text ?? '');
  const run = spawnSync('git', ['diff', '--no-index', '--no-color', '-U3', a, b], { encoding: 'utf8' });
  rmSync(folder, { recursive: true });
  const status = baseText == null ? 'A' : text == null ? 'D' : 'M';
  const body = run.stdout.split('\n').filter(l => /^[@+\- ]|^\\|^$/.test(l) && !/^(---|\+\+\+) /.test(l)).join('\n');
  const header = `diff --git a/${path} b/${path}\n${status === 'A' ? 'new file mode 100644\n' : status === 'D' ? 'deleted file mode 100644\n' : ''}`;
  const [chunk] = splitDiff(header + body);
  return { ...chunk, text, baseText, adds: 0, dels: 0 };
}

const BASE = `protocol Store {}

final class Cache: Store {
    let size: Int
}

final class Untouched {
    let cache: Cache
}
`;
const HEAD = `protocol Store {}

final class Cache: Store {
    let size: Int
    let clock: Clock
}

final class Untouched {
    let cache: Cache
}

struct Fresh {
    let cache: Cache
}
`;

test('changed types: new, changed, untouched, and a file box for code with no types', () => {
  const result = readChangedTypes([
    changed('Sources/Cache.swift', BASE, HEAD),
    changed('Sources/Clock.swift', null, 'protocol Clock {}'),
    changed('README.md', 'old', 'new'),
  ]);
  assert.deepEqual(result.nodes.map(n => [n.name, n.kind, n.state]), [
    ['Cache', 'class', 'changed'], ['Fresh', 'struct', 'new'], ['Clock', 'protocol', 'new'], ['README.md', 'artifact', 'changed'],
  ]);
  assert.deepEqual(result.untouched.map(n => [n.name, n.state]), [['Store', 'same'], ['Untouched', 'same']]);
});

test('an extension joins the changed type it extends, or stands for its own file', () => {
  const result = readChangedTypes([
    changed('Sources/Cache.swift', null, 'final class Cache {}'),
    changed('Sources/Cache+Store.swift', null, 'extension Cache: Store {\n    func keep() {}\n}'),
    changed('Sources/String+Trim.swift', null, 'extension String {\n    var trimmed: String { self }\n}'),
  ]);
  const cache = result.nodes.find(n => n.name === 'Cache');
  assert.deepEqual(cache.inherits, ['Store']);
  assert.deepEqual(cache.extFiles, ['Sources/Cache+Store.swift']);
  assert.deepEqual(result.nodes.map(n => [n.name, n.kind]), [['Cache', 'class'], ['String+Trim', 'extension']]);
});

test('test code is known by its path', () => {
  assert.equal(isTestPath('Tests/CoreTests/CacheTests.swift'), true);
  assert.equal(isTestPath('app/src/test/java/x/CacheTest.java'), true);
  assert.equal(isTestPath('pkg/test_cache.py'), true);
  assert.equal(isTestPath('web/cache.spec.ts'), true);
  assert.equal(isTestPath('Sources/Core/Cache.swift'), false);
});

function node(id, fields) {
  return { id, name: id, kind: 'class', file: `Sources/${id}.swift`, family: 'swift', inherits: [], members: [], made: [],
    extends: [], view: false, test: false, code: '', ...fields };
}

test('relations: conformance, inheritance, holding, use, tests and the nearest type of a name', () => {
  const nodes = [
    node('Store', { kind: 'protocol' }),
    node('Base'),
    node('Cache', { inherits: ['Store', 'Base'], members: ['[Entry]'], code: 'class Cache { func f() { Logger.log() } }' }),
    node('Entry', { kind: 'struct' }),
    node('Logger'),
    node('Logger@Other', { name: 'Logger', file: 'Other/Logger.swift' }),
    node('CacheTests', { test: true, code: 'Cache()' }),
    node('Main', { family: 'jvm', code: 'Cache()' }),
  ];
  const edges = findRelations(nodes);
  const kinds = Object.fromEntries(edges.map(e => [`${e.source}>${e.target}`, e.kind]));
  assert.deepEqual(kinds, {
    'Cache>Store': 'realize', 'Cache>Base': 'inherit', 'Cache>Entry': 'has', 'Cache>Logger': 'use', 'CacheTests>Cache': 'test',
  });
});

test('relations: imports link script and Python files that have no types', () => {
  const nodes = [
    node('file:web/app.ts', { name: 'app.ts', kind: 'artifact', file: 'web/app.ts', family: 'js' }),
    node('file:web/util/format.ts', { name: 'format.ts', kind: 'artifact', file: 'web/util/format.ts', family: 'js' }),
    node('file:pkg/run.py', { name: 'run.py', kind: 'artifact', file: 'pkg/run.py', family: 'python' }),
    node('Parser', { file: 'pkg/parse/core.py', family: 'python' }),
  ];
  const imports = new Map([
    ['web/app.ts', [{ spec: './util/format', names: ['format'] }]],
    ['pkg/run.py', [{ spec: '.parse.core', names: ['Parser'], python: true }]],
  ]);
  const edges = findRelations(nodes, imports).map(e => `${e.source}>${e.target}`);
  assert.deepEqual(edges.sort(), ['file:pkg/run.py>Parser', 'file:web/app.ts>file:web/util/format.ts']);
});

test('context: without the option only structural links to untouched types count', () => {
  const drawn = [node('Cache', { state: 'changed', inherits: ['Store'], code: 'Helper()' })];
  const untouched = [node('Store', { kind: 'protocol', state: 'same' }), node('Helper', { state: 'same' })];
  const candidates = [node('Monitor', { state: 'context', code: 'let c = Cache()' })];
  const off = chooseContext({ drawn, untouched, candidates, options: { enabled: false, max: 40, include: [], exclude: [] } });
  assert.deepEqual(off.same.map(n => n.name), ['Store']);
  assert.deepEqual(off.context, []);
  const on = chooseContext({ drawn, untouched, candidates, options: { enabled: true, max: 40, include: [], exclude: [] } });
  assert.deepEqual(on.same.map(n => n.name).sort(), ['Helper', 'Store']);
  assert.deepEqual(on.context.map(n => n.name), ['Monitor']);
  const capped = chooseContext({ drawn, untouched, candidates, options: { enabled: true, max: 1, include: ['Monitor'], exclude: ['Helper'] } });
  assert.deepEqual([...capped.same, ...capped.context].map(n => n.name), ['Monitor']);
});

test('the model: ids, placement, named links, groups and arrows between components', () => {
  const payloads = [...Array(6)].map((_, i) => node(`Payload${i}`, { kind: 'struct', state: 'new', inherits: ['Payload'], file: 'Sources/Format/Payloads.swift' }));
  const nodes = [
    node('Payload', { kind: 'protocol', state: 'new', file: 'Sources/Format/Payload.swift' }),
    ...payloads,
    node('Reader', { state: 'new', file: 'Sources/Format/Reader.swift', code: 'Payload0 Payload1' }),
    node('Screen', { state: 'new', file: 'App/Screen.swift', code: 'Reader() Reader' }),
    node('Panel', { state: 'new', file: 'App/Panel.swift', code: 'Reader()' }),
    node('Monitor', { state: 'context', file: 'Sources/Monitor.swift', code: 'Reader()' }),
  ];
  const files = ['Sources/Format/Payload.swift', 'Sources/Format/Payloads.swift', 'Sources/Format/Reader.swift', 'App/Screen.swift', 'App/Panel.swift']
    .map(path => ({ path, oldPath: null, status: 'A', adds: 3, dels: 0 }));
  const config = readConfig({
    components: [
      { id: 'sdk', title: 'SDK', paths: ['Sources/'] },
      { id: 'app', title: 'App', color: 'rose', paths: ['App/'] },
    ],
    links: [{ from: 'Screen', to: 'Reader', label: 'reads with' }],
    summaries: [{ from: 'app', to: 'sdk', label: 'shows what the SDK reads' }],
  });
  const model = buildModel({ files, nodes, imports: new Map(), config });
  assert.deepEqual(model.components.map(c => [c.id, c.color]), [['sdk', 'blue'], ['app', 'rose'], ['context', 'grey']]);
  assert.equal(model.groups.length, 1);
  assert.equal(model.groups[0].title, '6 types conforming to Payload');
  assert.ok(model.edges.some(e => e.source === `group:${model.groups[0].id}` && e.target === 'Payload' && e.label === 'each conforms to'));
  assert.ok(!model.edges.some(e => e.source === 'Payload0' && e.target === 'Payload'));
  const named = model.edges.find(e => e.source === 'Screen' && e.target === 'Reader');
  assert.deepEqual([named.label, named.key], ['reads with', true]);
  assert.deepEqual(model.summary.map(s => [s.source, s.target, s.count, s.label]),
    [['app', 'sdk', 2, 'shows what the SDK reads'], ['context', 'sdk', 1, null]]);
  assert.equal(model.nodes.find(n => n.id === 'Payload').stereotype, '«protocol»');
  assert.deepEqual(model.contextFiles, [{ path: 'Sources/Monitor.swift', component: 'context', package: 'top' }]);
  assert.deepEqual(model.components.map(c => c.packages.map(p => p.bare)), [[true], [true], [true]]);
});
