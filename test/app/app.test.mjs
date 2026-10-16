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
  assert.match(html, /<div class="mdx-block mdx-add" title="Added"[^>]*><p>Two\.<\/p>/);
  assert.match(html, /mdx-del" title="Removed"[^>]*><ul>\n<li>b<\/li>/);
  assert.match(html, /mdx-add" title="Added"[^>]*><ul>\n<li>c<\/li>/);
  assert.match(render(null, '# New'), /New file\./);
  assert.match(renderParts([{ before: 'a', after: 'b', oldStart: 1, newStart: 1 }]), /Only the changed parts/);
});

test('each block and list item of the Rendered view names the source lines it was read from', () => {
  const before = '# Guide\n\nOne\nline.\n\n- a\n- b\n\n```\ncode\n```\n';
  const after = '# Guide\n\nOne\nline.\n\nTwo.\n\n- a\n- c\n\n```\ncode\n```\n';
  const html = render(before, after);
  const lines = text => (new RegExp('<div class="mdx-block[^"]*"(?: title="[^"]*")? ([^>]*)>(?:<[^>]+>\\n?)*' + text).exec(html) || [])[1];
  assert.equal(lines('Guide'), 'data-side="RIGHT" data-from="1" data-to="1" data-text="1" data-left-from="1" data-left-to="1"');
  assert.equal(lines('One\nline'), 'data-side="RIGHT" data-from="3" data-to="4" data-text="3" data-left-from="3" data-left-to="4"', 'a paragraph keeps its line breaks, so a sentence in it maps to its own line');
  assert.equal(lines('Two'), 'data-side="RIGHT" data-from="6" data-to="6" data-text="6"');
  assert.equal(lines('a<'), 'data-side="RIGHT" data-from="8" data-to="8" data-text="8" data-left-from="6" data-left-to="6"');
  assert.equal(lines('b<'), 'data-side="LEFT" data-from="7" data-to="7" data-text="7"', 'a removed block names its lines before the change');
  assert.equal(lines('c<'), 'data-side="RIGHT" data-from="9" data-to="9" data-text="9"');
  assert.equal(lines('code'), 'data-side="RIGHT" data-from="11" data-to="13" data-text="12" data-left-from="9" data-left-to="11"', 'a fenced block\'s text starts after its fence');
  assert.match(render('[x]: https://example.com\n\nOld.\n', '[x]: https://example.com\n\nNew.\n'), /data-side="RIGHT" data-from="3" data-to="3" data-text="3"><p>New\./, 'a link definition, which draws nothing, still counts its lines');
  assert.match(renderParts([{ before: 'Old.', after: 'New.', oldStart: 40, newStart: 42 }]), /data-side="LEFT" data-from="40"[\s\S]*data-side="RIGHT" data-from="42"/);
  assert.match(render(null, 'First.\n\nSecond.\n'), /<div class="mdx-block" data-side="RIGHT" data-from="3" data-to="3" data-text="3"><p>Second\./);
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
