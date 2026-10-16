import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readThreads, placeThreads } from '../../src/review/github-threads.mjs';

const comment = (id, login, extra = {}) => ({
  databaseId: id, author: login === null ? null : { login, __typename: 'User' }, body: `Comment ${id}`, createdAt: '2026-10-15T10:00:00Z',
  url: `https://github.com/octo/widgets/pull/30#discussion_r${id}`, diffHunk: '@@ -1,5 +1,5 @@\n a\n b\n-c\n+C\n d\n e', ...extra,
});
const node = (extra = {}) => ({
  id: 'T1', isResolved: false, isOutdated: false, path: 'docs/guide.md', line: 3, startLine: null, originalLine: 3, originalStartLine: null,
  diffSide: 'RIGHT', startDiffSide: null, subjectType: 'LINE', resolvedBy: null,
  comments: { totalCount: 1, nodes: [comment(11, 'alex')] }, ...extra,
});

test('a thread from GitHub is read with its state, its comments in order, and who wrote each', () => {
  const [t] = readThreads(30, [node({
    isResolved: true, resolvedBy: { login: 'sam' }, startLine: 2, startDiffSide: 'LEFT',
    comments: { totalCount: 104, nodes: [comment(11, 'alex'), comment(12, 'lint-helper', { author: { login: 'lint-helper', __typename: 'Bot' } }), comment(13, null)] },
  })]);
  assert.deepEqual({ ...t, comments: undefined }, {
    id: 'T1', pull: 30, path: 'docs/guide.md', side: 'RIGHT', line: 3, start_line: 2, start_side: 'LEFT', original_line: 3,
    resolved: true, resolved_by: 'sam', outdated: false, file: false, comments: undefined, more: 101, hunk: '-c\n+C\n d\n e',
  });
  assert.deepEqual(t.comments.map(c => [c.id, c.author, c.bot]), [[11, 'alex', false], [12, 'lint-helper', true], [13, 'ghost', false]]);
  assert.deepEqual(t.comments[0], { id: 11, author: 'alex', bot: false, body: 'Comment 11', at: '2026-10-15T10:00:00Z', url: 'https://github.com/octo/widgets/pull/30#discussion_r11' });
  const [outdated, whole] = readThreads(30, [node({ isOutdated: true, line: null }), node({ subjectType: 'FILE', line: null })]);
  assert.deepEqual([outdated.outdated, outdated.line, whole.file], [true, null, true]);
});

test('on a single pull request a thread sits on its own line; outdated and whole-file threads sit on none', () => {
  const threads = readThreads(30, [node(), node({ id: 'T2', startLine: 1 }), node({ id: 'T3', isOutdated: true, line: null }), node({ id: 'T4', subjectType: 'FILE', line: null })]);
  assert.deepEqual(placeThreads(threads, null).map(t => t.at), [
    { path: 'docs/guide.md', side: 'RIGHT', line: 3 },
    { path: 'docs/guide.md', side: 'RIGHT', line: 3, start_line: 1, start_side: 'RIGHT' },
    null,
    null,
  ]);
});

test('on a stack a thread sits on the combined diff\'s line that shows its pull request\'s own line', () => {
  const stack = {
    lines: {
      'web/cart.ts': {
        RIGHT: { 4: [21, 2], 5: [21, 3], 9: { c: [[21, 7], [22, 8]] }, 12: [22, 12] },
        LEFT: { 6: [21, 6] },
      },
      'web/total.ts': { RIGHT: { 3: [22, 3, 'web/sum.ts'] }, LEFT: {} },
    },
  };
  const on = (pull, path, line, extra = {}) => readThreads(pull, [node({ path, line, ...extra })])[0];
  const placed = placeThreads([
    on(21, 'web/cart.ts', 3, { startLine: 2 }),
    on(22, 'web/cart.ts', 8),
    on(21, 'web/cart.ts', 6, { diffSide: 'LEFT' }),
    on(22, 'web/sum.ts', 3),
    on(21, 'web/cart.ts', 12),
  ], stack);
  assert.deepEqual(placed.map(t => t.at), [
    { path: 'web/cart.ts', side: 'RIGHT', line: 5, start_line: 4, start_side: 'RIGHT' },
    { path: 'web/cart.ts', side: 'RIGHT', line: 9 },
    { path: 'web/cart.ts', side: 'LEFT', line: 6 },
    { path: 'web/total.ts', side: 'RIGHT', line: 3 },
    null,
  ], 'a line #22 changed again after #21 is not shown as #21 had it, so its thread sits on no line');
});
