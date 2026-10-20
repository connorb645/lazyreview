import { test } from 'node:test';
import assert from 'node:assert/strict';
import { writeFileSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { createRequire } from 'node:module';
import { layoutModel } from '../../src/build/layout/layout.mjs';

// The arrows the layout draws between packages and between components for made-up models, with their words and the
// links they count, which the Go layout is tested against. Places are left out: the Go layout places parts its own way.
const RECORD = join(dirname(fileURLToPath(import.meta.url)), '..', '..', 'terminal', 'testdata', 'layouts.json');
const ELK = createRequire(import.meta.url)('../../vendor/elk-0.9.3.bundled.js');

const node = (id, component, pkg, extra = {}) => ({
  id, name: id, kind: 'class', stereotype: '', file: `${pkg}/${id}.swift`, component, package: pkg, state: 'new', group: null, test: false, ...extra,
});
const pkg = (id, title, path, extra = {}) => ({ id, title, path, description: '', bare: false, ...extra });
const edge = (source, target, kind, extra = {}) => ({ source, target, kind, label: null, key: false, ...extra });

const MODELS = [
  {
    name: 'areas and named links',
    model: {
      components: [
        { id: 'app', title: 'App', tag: 'App/', color: 'blue', area: null, context: false, packages: [pkg('ui', 'Screens', 'App/ui/'), pkg('data', 'Data', 'App/data/')] },
        { id: 'sdk', title: 'SDK', tag: 'Sources/', color: 'teal', area: 'lib', context: false, packages: [pkg('core', 'Core', 'Sources/core/')] },
      ],
      areas: [{ id: 'lib', title: 'The library', description: '', components: ['sdk'] }],
      files: [], contextFiles: [],
      nodes: [node('Home', 'app', 'ui'), node('Detail', 'app', 'ui'), node('Store', 'app', 'data'), node('Client', 'sdk', 'core'), node('Cache', 'sdk', 'core'), node('Token', 'sdk', 'core')],
      edges: [
        edge('Home', 'Detail', 'has', { key: true, label: 'opens' }),
        edge('Detail', 'Store', 'use', { key: true, label: 'reads' }),
        edge('Store', 'Client', 'use'),
        edge('Client', 'Cache', 'has'),
      ],
      groups: [],
      summary: [{ source: 'app', target: 'sdk', count: 1, label: 'calls', links: [['Store', 'Client']] }],
    },
  },
  {
    name: 'labels, tests, groups and arrows both ways',
    model: {
      components: [
        { id: 'shop', title: 'Shop', tag: 'Shop/', color: 'blue', area: null, context: false,
          packages: [pkg('ui', 'Screens', 'Shop/ui/'), pkg('data', 'Data', 'Shop/data/'), pkg('checks', 'Checks', 'Shop/checks/'), pkg('tests', 'Tests', 'Tests/')] },
        { id: 'core', title: 'Core', tag: 'Core/', color: 'teal', area: null, context: false,
          packages: [pkg('model', 'Model', 'Core/model/', { description: 'What a basket holds' })] },
        { id: 'context', title: 'Unchanged code', tag: 'Base/', color: 'grey', area: null, context: true, packages: [pkg('top', 'Base', 'Base/', { bare: true })] },
      ],
      areas: [],
      files: [
        { path: 'Shop/ui/Home.swift', status: 'M', adds: 1234, dels: 5678, component: 'shop', package: 'ui' },
        { path: 'Core/model/Kind.swift', status: 'A', adds: 12, dels: 0, component: 'core', package: 'model' },
      ],
      contextFiles: [{ path: 'Base/Clock.swift', component: 'context', package: 'top' }],
      nodes: [
        node('Home', 'shop', 'ui'), node('List', 'shop', 'ui'), node('Store', 'shop', 'data'), node('Cache', 'shop', 'data'),
        node('Fixtures', 'shop', 'checks'), node('StoreChecks', 'shop', 'checks', { test: true }),
        node('HomeTests', 'shop', 'tests', { test: true }), node('ListTests', 'shop', 'tests', { test: true }),
        node('Kind', 'core', 'model', { kind: 'protocol', stereotype: '«protocol»' }),
        node('Apple', 'core', 'model', { kind: 'struct', group: 'fruit' }), node('Pear', 'core', 'model', { kind: 'struct', group: 'fruit' }),
        node('Basket', 'core', 'model'),
        node('Clock', 'context', 'top', { state: 'context' }),
      ],
      edges: [
        edge('Home', 'Store', 'use', { key: true, label: 'reads' }),
        edge('List', 'Store', 'use', { key: true, label: 'loads' }),
        edge('List', 'Cache', 'use', { key: true, label: 'reads' }),
        edge('Home', 'Cache', 'use', { key: true, label: 'saves' }),
        edge('StoreChecks', 'Store', 'test'),
        edge('StoreChecks', 'Cache', 'test'),
        edge('HomeTests', 'Home', 'test'),
        edge('ListTests', 'List', 'test'),
        edge('group:fruit', 'Kind', 'realize', { label: 'each conforms to' }),
        edge('Basket', 'Apple', 'has'),
        edge('Store', 'Basket', 'use'),
        edge('Cache', 'Kind', 'use'),
        edge('Basket', 'Store', 'use', { key: true, label: 'feeds' }),
        edge('Kind', 'List', 'use'),
        edge('Basket', 'Clock', 'use'),
      ],
      groups: [{ id: 'fruit', title: '2 types conforming to Kind', description: 'Fruit', members: ['Apple', 'Pear'], target: 'Kind' }],
      summary: [
        { source: 'core', target: 'context', count: 1, label: null, links: [['Basket', 'Clock']] },
        { source: 'core', target: 'shop', count: 2, label: 'feeds', links: [['Basket', 'Store'], ['Kind', 'List']] },
        { source: 'shop', target: 'core', count: 2, label: null, links: [['Store', 'Basket'], ['Cache', 'Kind']] },
      ],
    },
  },
  {
    name: 'files with nothing linking them',
    model: {
      components: [{ id: 'docs', title: 'docs', tag: 'docs/', color: 'blue', area: null, context: false, packages: [pkg('top', 'docs', 'docs/', { bare: true })] }],
      areas: [], files: [], contextFiles: [], edges: [], groups: [], summary: [],
      nodes: [node('file:docs/a.md', 'docs', 'top', { name: 'a.md', kind: 'artifact', stereotype: '«file»', file: 'docs/a.md' }),
        node('file:docs/b.md', 'docs', 'top', { name: 'b.md', kind: 'artifact', stereotype: '«file»', file: 'docs/b.md' })],
    },
  },
];

test('record the arrows the layout draws for made-up models', async () => {
  const record = [];
  for (const { name, model } of MODELS) {
    const layout = await layoutModel(model, new ELK());
    record.push({
      name,
      model,
      components: Object.keys(layout.components).sort(),
      boxes: Object.keys(layout.boxes).sort(),
      pkgArrows: layout.pkgArrows.map(({ comp, from, to, count, words, countText, links }) => ({ comp, from, to, count, words, countText, links })),
      summaries: layout.summaries.map(({ source, target, both, lines, ways }) => ({ source, target, both, lines, ways })),
    });
  }
  assert.ok(record[1].pkgArrows.some(a => a.words === 'reads, loads'), 'an arrow names two of its links\' words');
  assert.ok(record[1].summaries.some(s => s.both), 'an arrow between components can run both ways');
  writeFileSync(RECORD, JSON.stringify(record, null, 2) + '\n');
});
