import { test } from 'node:test';
import assert from 'node:assert/strict';
import { createRequire } from 'node:module';
import { shelfPack, route } from '../../src/build/layout/shelf.mjs';
import { layoutModel } from '../../src/build/layout/layout.mjs';
import { nodeSize, measure, farTitleRoom } from '../../src/build/layout/sizes.mjs';

const require = createRequire(import.meta.url);
const ELK = require('../../vendor/elk-0.9.3.bundled.js');

const overlaps = (a, b) => a.x < b.x + b.w && b.x < a.x + a.w && a.y < b.y + b.h && b.y < a.y + a.h;

test('a dominant box gets the top row to itself; the rest share rows', () => {
  const packed = shelfPack([{ id: 'big', w: 4000, h: 3000 }, { id: 'a', w: 1500, h: 1000 }, { id: 'b', w: 1500, h: 900 }],
    { top: 200, side: 90, bottom: 90 }, 1.2);
  assert.equal(packed.at.big.y, 200);
  assert.equal(packed.at.a.y, packed.at.b.y);
  assert.ok(packed.at.a.y > packed.at.big.y + 3000);
  const boxes = Object.values(packed.at);
  for (const a of boxes) for (const b of boxes) if (a !== b) assert.ok(!overlaps(a, b));
});

test('two similar boxes sit side by side when the target is wide', () => {
  const packed = shelfPack([{ id: 'a', w: 1000, h: 400 }, { id: 'b', w: 900, h: 380 }], { top: 0, side: 0, bottom: 0 }, 3);
  assert.equal(packed.at.a.y, packed.at.b.y);
});

test('arrows run straight between facing frames and stop short of them', () => {
  const across = route({ x: 0, y: 0, w: 100, h: 400 }, { x: 600, y: 100, w: 100, h: 400 }, { width: 50, height: 30 });
  assert.equal(across.points.length, 2);
  assert.equal(across.points[0][1], across.points[1][1]);
  assert.ok(across.points[0][0] > 100 && across.points[1][0] < 600);
  const down = route({ x: 0, y: 0, w: 800, h: 300 }, { x: 100, y: 700, w: 400, h: 300 }, { width: 50, height: 30 });
  assert.equal(down.points[0][0], down.points[1][0]);
});

test('box sizes leave room for the longest of name, file and stereotype', () => {
  const n = { name: 'CheckoutPaymentSources', kind: 'enum', file: 'Sources/CheckoutPaymentSources.swift', stereotype: '«enum»' };
  assert.ok(nodeSize(n).width >= measure('CheckoutPaymentSources', 'name').width + 28);
  assert.equal(nodeSize({ ...n, kind: 'artifact' }).height, 46);
});

test('the layout gives every type, package and component a place, and nothing overlaps', async () => {
  const types = (component, pkg, names) => names.map(name => ({ id: name, name, kind: 'class', stereotype: '', file: `${pkg}/${name}.swift`,
    component, package: pkg, state: 'new', group: null }));
  const model = {
    components: [
      { id: 'app', title: 'App', tag: 'App/', context: false, packages: [{ id: 'ui', title: 'Screens', path: 'App/ui/' }, { id: 'data', title: 'Data', path: 'App/data/' }] },
      { id: 'sdk', title: 'SDK', tag: 'Sources/', context: false, packages: [{ id: 'core', title: 'Core', path: 'Sources/core/' }] },
    ],
    areas: [{ id: 'lib', title: 'The library', description: '', components: ['sdk'] }],
    files: [],
    contextFiles: [],
    nodes: [...types('app', 'ui', ['Home', 'Detail']), ...types('app', 'data', ['Store']), ...types('sdk', 'core', ['Client', 'Cache', 'Token'])],
    edges: [
      { source: 'Home', target: 'Detail', kind: 'has', key: true, label: 'opens' },
      { source: 'Detail', target: 'Store', kind: 'use', key: true, label: 'reads' },
      { source: 'Store', target: 'Client', kind: 'use', key: false, label: null },
      { source: 'Client', target: 'Cache', kind: 'has', key: false, label: null },
    ],
    groups: [],
    summary: [{ source: 'app', target: 'sdk', count: 1, label: 'calls', links: [['Store', 'Client']] }],
  };
  const layout = await layoutModel(model, new ELK());
  for (const n of model.nodes) assert.ok(layout.boxes[n.id], `no box for ${n.id}`);
  for (const p of ['pkg:app:ui', 'pkg:app:data', 'pkg:sdk:core']) assert.ok(layout.boxes[p], `no frame for ${p}`);
  assert.ok(overlaps(layout.areas.lib, layout.components.sdk));
  assert.ok(!overlaps(layout.components.app, layout.components.sdk));
  const inside = (b, f) => b.x >= f.x && b.y >= f.y && b.x + b.w <= f.x + f.w && b.y + b.h <= f.y + f.h;
  for (const n of model.nodes) assert.ok(inside(layout.boxes[n.id], layout.components[n.component]), `${n.id} sits outside its component`);
  assert.equal(layout.summaries.length, 1);
  assert.ok(layout.summaries[0].points.length >= 2);
  assert.ok(layout.routes.some(r => r.label === 'opens' && r.labelAt), 'a named link inside a package carries its label');
  assert.ok(layout.pkgArrows.some(a => a.words === 'reads' && a.from === 'ui' && a.to === 'data'), 'a named link between packages names the arrow');
  assert.deepEqual(layout.summaries[0].ways, [{ arrow: '→', label: 'calls', links: [['Store', 'Client']] }], 'an arrow between components names the links it counts, each way');
  assert.deepEqual(layout.pkgArrows.find(a => a.from === 'ui').links, [['Detail', 'Store']], 'an arrow between packages names the links it counts');
  assert.ok(layout.w > 0 && layout.h > 0);
  for (const c of model.components) assert.ok(layout.components[c.id].w >= farTitleRoom(c.title), `${c.id} is too narrow for its far title`);
  assert.deepEqual(layout.farTitle, { size: 150, overhang: 200 });
});

test('a lone component is not widened for its far title, and a bare package has no header', async () => {
  const model = {
    components: [{ id: 'only', title: 'A Rather Long Component Title', tag: 'Tests/', context: false,
      packages: [{ id: 'p', title: 'A Rather Long Component Title', path: 'Tests/', bare: true }] }],
    areas: [], files: [], contextFiles: [], edges: [], groups: [], summary: [],
    nodes: [{ id: 'T', name: 'T', kind: 'class', stereotype: '', file: 'Tests/T.swift', component: 'only', package: 'p', state: 'new', group: null }],
  };
  const layout = await layoutModel(model, new ELK());
  assert.ok(layout.components.only.w < farTitleRoom('A Rather Long Component Title'));
  assert.ok(layout.boxes.T.y - layout.boxes['pkg:only:p'].y < 40, 'a bare package starts its types at its top');
});
