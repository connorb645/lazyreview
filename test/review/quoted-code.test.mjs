import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readHunks } from '../../src/review/diff-hunks.mjs';
import { quotedCode } from '../../src/review/quoted-code.mjs';

const DIFF = 'diff --git a/a.ts b/a.ts\n--- a/a.ts\n+++ b/a.ts\n@@ -1,4 +1,4 @@ top\n one\n-two\n+TWO\n three\n\n@@ -9 +9,2 @@\n-x\n+y\n+z\n\\ No newline at end of file\n';

test('a file\'s diff is read hunk by hunk, row by row, with each row\'s numbers on each side', () => {
  const [first, second] = readHunks(DIFF);
  assert.deepEqual({ ...first, rows: undefined }, { header: '@@ -1,4 +1,4 @@ top', old_start: 1, old_count: 4, new_start: 1, new_count: 4, rows: undefined });
  assert.deepEqual(first.rows.map(r => [r.kind, r.old, r.new, r.text]), [
    ['same', 1, 1, 'one'], ['del', 2, null, 'two'], ['add', null, 2, 'TWO'], ['same', 3, 3, 'three'], ['same', 4, 4, ''],
  ], 'an empty row inside a hunk is an empty unchanged line');
  assert.deepEqual(second.rows.map(r => r.kind), ['del', 'add', 'add', 'note'], 'the empty row after the last hunk is no line');
  assert.equal(second.old_count, 1);
  assert.deepEqual(readHunks(undefined), []);
});

test('a comment quotes the code of its lines from the page: the rows of its side, or every row of a range across sides', () => {
  const page = {
    model: { files: [{ path: 'a.ts', status: 'M' }, { path: 'new.ts', status: 'A' }] },
    diff: { 'a.ts': DIFF, 'new.ts': '@@ -0,0 +1,2 @@\n+first\n+second\n' },
    sources: { 'a.ts': ['one', 'TWO', 'three', '', 'five', 'six', 'seven', 'eight', 'y', 'z'] },
    context: { 'c.ts': ['class C {}', '}'] },
  };
  const quote = (path, side, line, start_line, start_side) => quotedCode(page, { path, side, line, start_line, start_side });
  assert.equal(quote('a.ts', 'RIGHT', 3, 1), 'one\nTWO\nthree', 'the removed row between is not on the new side');
  assert.equal(quote('a.ts', 'LEFT', 2), 'two');
  assert.equal(quote('a.ts', 'RIGHT', 2, 2, 'LEFT'), 'two\nTWO', 'from a removed line to an added one, both');
  assert.equal(quote('a.ts', 'RIGHT', 6, 5), 'five\nsix', 'a line outside the hunks comes from the whole text');
  assert.equal(quote('a.ts', 'LEFT', 6), '', 'the page holds no whole text of the old side');
  assert.equal(quote('new.ts', 'RIGHT', 2), 'second');
  assert.equal(quote('c.ts', 'RIGHT', 1), 'class C {}', 'an unchanged file of the context quotes its own code');
});
