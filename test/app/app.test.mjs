import { test } from 'node:test';
import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { createRequire } from 'node:module';
import { readdirSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import '../support/served-strings.mjs';

const APP = join(dirname(fileURLToPath(import.meta.url)), '..', '..', 'app');
globalThis.window = { marked: createRequire(import.meta.url)('../../vendor/marked-12.0.2.min.js') };
const { drawRendered } = await import('../../app/presentation/rendered-markdown.mjs');
const { safeMarkdown } = await import('../../app/lib/markdown.mjs');
const { indexPage } = await import('../../app/context/review-data.mjs');
const { breakable } = await import('../../app/lib/text.mjs');
const { english } = await import('/i18n/english.mjs');

function modules(folder) {
  return readdirSync(folder, { withFileTypes: true }).flatMap(e => (e.isDirectory() ? modules(join(folder, e.name)) : e.name.endsWith('.mjs') ? [join(folder, e.name)] : []));
}

test('every module of the app parses', () => {
  const files = modules(APP);
  assert.ok(files.length >= 12);
  for (const file of files) execFileSync(process.execPath, ['--check', file]);
});

test('the Rendered view draws the server\'s blocks, each marked and naming its source lines, and a list\'s items together', () => {
  const block = (state, markdown, start_line, extra = {}) => ({ state, markdown, side: state === 'del' ? 'LEFT' : 'RIGHT', start_line, line: start_line, text_line: start_line, left: null, list: null, ...extra });
  const html = drawRendered({ partial: false, blocks: [block('same', '# Guide', 1, { left: { start_line: 1, line: 1 } }), block('add', 'Two.', 6), block('same', '- a', 8, { list: 0 }), block('del', '- b', 7, { list: 0 })] }, 'M');
  assert.match(html, /^<div class="mdx"><p class="mdx-note">Marked:/);
  assert.match(html, /<div class="mdx-block mdx-same" data-side="RIGHT" data-from="1" data-to="1" data-text="1" data-left-from="1" data-left-to="1"><h1>Guide<\/h1>/);
  assert.match(html, /<div class="mdx-block mdx-add" title="Added" data-side="RIGHT" data-from="6" data-to="6" data-text="6"><p>Two\.<\/p>/);
  assert.match(html, /<div class="mdx-list"><div class="mdx-block mdx-same"[^>]*><ul>\n<li>a<\/li>[\s\S]*mdx-del" title="Removed" data-side="LEFT"[^>]*><ul>\n<li>b<\/li>\n<\/ul>\n<\/div><\/div>/);
  assert.match(drawRendered({ partial: false, blocks: [block('add', 'Second.', 3)] }, 'A'), /New file\.<\/p><div class="mdx-block" data-side="RIGHT" data-from="3"/, 'an added file is drawn plain');
  assert.match(drawRendered({ partial: true, blocks: [block('add', 'a', 1, { part: 0 }), block('add', 'b', 9, { part: 1 })] }, 'M'), /Only the changed parts[\s\S]*a<\/p>\n<\/div><p class="mdx-gap">/);
  assert.match(drawRendered({ partial: false, blocks: [block('same', 'a', 1)] }, 'M'), /does not alter how the file reads/);
  const leftOut = drawRendered({ partial: false, blocks: [block('del', 'Theirs.', 3), block('add', 'Ours.', 3)] }, 'M', b => b.state === 'del');
  assert.match(leftOut, /<div class="mdx-block mdx-del pr-other" title="Removed"[^>]*><p>Theirs\.<\/p>/, 'a block the filter leaves out fades, as its lines do in the diff');
  assert.match(leftOut, /<div class="mdx-block mdx-add" title="Added"[^>]*><p>Ours\.<\/p>/);
});

test('Markdown from outside the app shows raw HTML as text and keeps only safe links', () => {
  const html = safeMarkdown('<img src=x onerror=alert(1)> [go](javascript:alert(1)) [ok](https://example.com) ![pic](https://example.com/p.png)');
  assert.ok(!html.includes('<img'));
  assert.match(html, /&lt;img src=x onerror=alert\(1\)&gt;/);
  assert.ok(!html.includes('javascript:'));
  assert.match(html, /<a href="https:\/\/example\.com">ok<\/a>/);
  assert.match(html, /\[image: pic - https:\/\/example\.com\/p\.png\]/);
});

test('the page data is indexed with the totals of the change and of each component', () => {
  const data = indexPage({
    model: {
      nodes: [{ id: 'A', file: 'a.ts', extFiles: ['b.ts'] }], components: [{ id: 'one' }, { id: 'context' }], groups: [],
      files: [{ path: 'a.ts', component: 'one', adds: 3, dels: 1 }, { path: 'b.ts', component: 'one', adds: 2, dels: 0 }],
      contextFiles: [{ path: 'c.ts', component: 'context', lines: 2 }],
    },
    layout: {}, stack: null, build: 'abc',
  });
  assert.deepEqual(data.totals, { files: 2, adds: 5, dels: 1 });
  assert.deepEqual(data.compTotals, { one: { files: 2, adds: 5, dels: 1 }, context: { files: 1, adds: 0, dels: 0 } });
  assert.deepEqual(Object.keys(data.nodesByFile), ['a.ts', 'b.ts']);
  assert.equal(data.contextFiles['c.ts'].index, 'c0');
  assert.equal(data.build, 'abc');
  assert.equal(breakable('Sources/HarvestPlanner/late_frost-v2/A&B.swift'), 'Sources/<wbr>HarvestPlanner/<wbr>late_frost-v2/<wbr>A&amp;B.swift',
    'a path breaks after its slashes only, so a wrap never splits a name into what looks like two words');
  assert.equal(english.web.counts.files(1) + ', ' + english.web.counts.files(1200), '1 file, 1,200 files');
});

test('under the pull request filter, a component counts only the files and lines of that pull request', async () => {
  const { createPulls } = await import('../../app/context/pulls.mjs');
  const { countWords } = await import('../../app/presentation/diagram.mjs');
  const model = {
    components: [{ id: 'core', tag: 'Swift' }, { id: 'ledger', tag: 'Swift' }, { id: 'context', context: true }],
    files: [
      { path: 'Order.swift', component: 'core', adds: 3, dels: 1, pulls: [21, 22], byPull: { 21: [2, 0], 22: [1, 1] } },
      { path: 'Rules.swift', component: 'core', adds: 4, dels: 0, pulls: [22], byPull: { 22: [4, 0] } },
      { path: 'Book.swift', component: 'ledger', adds: 5, dels: 2, pulls: [21], byPull: { 21: [5, 2] } },
    ],
  };
  const compTotals = { core: { files: 2, adds: 7, dels: 1 }, ledger: { files: 1, adds: 5, dels: 2 }, context: { files: 1, adds: 0, dels: 0 } };
  const ctx = { data: { stack: { members: [{ pull: 21 }, { pull: 22 }] }, model, compTotals }, state: { members: [] }, filter: { pull: null } };
  const pulls = createPulls(ctx);
  assert.deepEqual(pulls.componentTotals(), compTotals);
  ctx.filter.pull = 22;
  const totals = pulls.componentTotals();
  assert.deepEqual(totals, { core: { files: 2, adds: 5, dels: 1 }, ledger: { files: 0, adds: 0, dels: 0 }, context: { files: 1, adds: 0, dels: 0 } });
  assert.deepEqual(pulls.ownLines(model.files[0]), [1, 1]);
  assert.equal(countWords(totals, model.components[0]), '2 files · +5 −1');
  assert.equal(countWords(totals, model.components[1]), '0 files');
  assert.equal(countWords(totals, model.components[1], true), '0');
  assert.equal(countWords(totals, model.components[2]), countWords(compTotals, model.components[2]));
});

test('under the pull request filter, unchanged code is never left out, and an arrow counts only the links whose ends it shows', async () => {
  const { createPulls } = await import('../../app/context/pulls.mjs');
  const { summaryLines, packageCount } = await import('../../app/presentation/diagram.mjs');
  const model = {
    components: [{ id: 'core' }, { id: 'context', context: true }],
    nodes: [
      { id: 'Order', component: 'core', pulls: [21, 22] }, { id: 'Rule', component: 'core', pulls: [22] },
      { id: 'Book', component: 'core', pulls: [21] }, { id: 'Clock', component: 'context', pulls: [] },
    ],
    groups: [{ id: 'rules', members: ['Rule'] }],
    files: [],
  };
  const comps = { core: model.components[0], context: model.components[1] };
  const ctx = { data: { stack: { members: [{ pull: 21 }, { pull: 22 }] }, model, comps, compTotals: {} }, state: { members: [] }, filter: { pull: 22 } };
  const pulls = createPulls(ctx);
  assert.deepEqual(['Order', 'Rule', 'Book', 'Clock', 'group:rules'].map(pulls.shows), [true, true, false, true, true]);
  const summary = { lines: ['→ 3 links', '← 1 link'], ways: [
    { arrow: '→', label: null, links: [['Order', 'Clock'], ['Book', 'Clock'], ['group:rules', 'Clock']] },
    { arrow: '←', label: 'ticks', links: [['Clock', 'Book']] },
  ] };
  assert.deepEqual(summary.ways.map(w => pulls.countShown(w.links)), [2, 0]);
  assert.deepEqual(summaryLines(summary, [2, 0]), ['→ 2 links', '← ticks · 0 links']);
  assert.deepEqual(summaryLines(summary, null), summary.lines);
  assert.equal(packageCount(1), '1 link');
  ctx.filter.pull = null;
  assert.equal(pulls.countShown(summary.ways[0].links), 3);
});
