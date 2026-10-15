import { test } from 'node:test';
import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { createRequire } from 'node:module';
import { readdirSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

const APP = join(dirname(fileURLToPath(import.meta.url)), '..', '..', 'app');
globalThis.window = { marked: createRequire(import.meta.url)('../../vendor/marked-12.0.2.min.js') };
const { render, renderParts } = await import('../../app/lib/markdown-diff.mjs');
const { safeMarkdown } = await import('../../app/lib/markdown.mjs');
const { indexPage } = await import('../../app/review/review-data.mjs');
const { breakable, plural } = await import('../../app/lib/text.mjs');

function modules(folder) {
  return readdirSync(folder, { withFileTypes: true }).flatMap(e => (e.isDirectory() ? modules(join(folder, e.name)) : e.name.endsWith('.mjs') ? [join(folder, e.name)] : []));
}

test('every module of the app parses', () => {
  const files = modules(APP);
  assert.ok(files.length >= 12);
  for (const file of files) execFileSync(process.execPath, ['--check', file]);
});

test('the Rendered view marks each added or removed block, and a changed list item by item', () => {
  const html = render('# Guide\n\nOne.\n\n- a\n- b\n', '# Guide\n\nOne.\n\nTwo.\n\n- a\n- c\n');
  assert.match(html, /<div class="mdx-block mdx-add" title="Added"><p>Two\.<\/p>/);
  assert.match(html, /mdx-del" title="Removed"><ul>\n<li>b<\/li>/);
  assert.match(html, /mdx-add" title="Added"><ul>\n<li>c<\/li>/);
  assert.match(render(null, '# New'), /New file\./);
  assert.match(renderParts([{ before: 'a', after: 'b' }]), /Only the changed parts/);
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
      contextFiles: [{ path: 'c.ts', component: 'context' }],
    },
    layout: {}, diff: {}, sources: {}, context: { 'c.ts': ['x', 'y'] }, stack: null,
  });
  assert.deepEqual(data.totals, { files: 2, adds: 5, dels: 1 });
  assert.deepEqual(data.compTotals, { one: { files: 2, adds: 5, dels: 1 }, context: { files: 1, adds: 0, dels: 0 } });
  assert.deepEqual(Object.keys(data.nodesByFile), ['a.ts', 'b.ts']);
  assert.equal(data.contextFiles['c.ts'].lines, 2);
  assert.equal(breakable('Sources/A_b.swift'), 'Sources/<wbr>A_<wbr>b.swift');
  assert.equal(plural(1, 'file') + ', ' + plural(1200, 'file'), '1 file, 1,200 files');
});
