import { test } from 'node:test';
import assert from 'node:assert/strict';
import { workbench } from '../support/workbench.mjs';
import { PullRef } from '../../src/models/pull-ref.mjs';
import { fileView } from '../../src/review/file-view.mjs';
import { pageView } from '../../src/review/page-view.mjs';

const ref = n => new PullRef('github.com', 'octo/widgets', n);

test('a file of a stack is drawn row by row, each row with the pull request that changed it and where a comment on it goes', async () => {
  const w = workbench();
  try {
    await w.loader.load({ links: [w.link(21), w.link(22)] });
    const { page, build } = w.store.readPage(ref(21));
    const basket = fileView(page, 'Sources/Basket/Basket.swift');
    const rows = basket.hunks.flatMap(h => h.rows);
    assert.deepEqual(rows.find(r => r.new === 3), { kind: 'add', old: null, new: 3, text: '    let search: Search', owner: 22, words: 'Changed in #22. A comment here goes on #22, line 3.' });
    assert.deepEqual(rows.find(r => r.new === 1), { kind: 'same', old: 1, new: 1, text: 'final class Basket {', owner: null, words: 'Unchanged. A comment here goes on #22, line 1.' });
    assert.equal(rows.find(r => r.kind === 'del').owner, 22);
    assert.equal(basket.whole.length, 13);

    const catalog = fileView(page, 'Sources/Catalog/MemoryCatalog.swift');
    assert.deepEqual(catalog.gaps, [{ before: 0, from: 1, to: 3, delta: 0 }], 'the unchanged lines before the first hunk are a run of their own');
    assert.equal(fileView(page, 'Sources/Catalog/Search.swift').gaps.length, 0, 'an added file has no runs between hunks');
    assert.equal(fileView(page, 'nowhere.swift'), null);

    const view = pageView(build, page);
    assert.deepEqual(Object.keys(view).sort(), ['build', 'layout', 'model', 'stack']);
    assert.deepEqual(Object.keys(view.stack).sort(), ['members', 'warnings'], 'the stack comes without its line map');
  } finally {
    w.done();
  }
});

test('a file of a single pull request has rows with no owner, and a page is parsed once per build and shared frozen', async () => {
  const w = workbench();
  try {
    await w.loader.load({ links: [w.link(30)] });
    const read = w.store.readPage(ref(30));
    assert.equal(w.store.readPage(ref(30)), read, 'the same build is read once');
    assert.ok(Object.isFrozen(read.page.model.files[0]));
    assert.ok(fileView(read.page, 'docs/guide.md').hunks[0].rows.every(r => r.owner === null && r.words === null));
    const { rendered } = fileView(read.page, 'docs/guide.md');
    assert.deepEqual(rendered.blocks.filter(b => b.state !== 'same').map(b => [b.state, b.side, b.start_line, b.markdown]),
      [['del', 'LEFT', 3, 'The catalog lists items.'], ['add', 'RIGHT', 3, 'The catalog lists every item.']], 'a Markdown file comes with its Rendered view');
  } finally {
    w.done();
  }
});
