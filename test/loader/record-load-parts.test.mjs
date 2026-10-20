import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync, writeFileSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { readThreads, placeThreads } from '../../src/review/github-threads.mjs';
import { regroup } from '../../src/queue/regroup.mjs';

// How the review threads read from GitHub are kept and placed, as github.json holds them, and how a load takes a
// review's queue to the pull requests it names, which the Go loader is tested against.
const RECORD = join(dirname(fileURLToPath(import.meta.url)), '..', '..', 'terminal', 'testdata', 'load-parts.json');

const hunk = '@@ -1,6 +1,6 @@\n a\n b\n-c\n+C\n d\n e';
const remark = (id, login, extra = {}) => ({
  databaseId: id, author: login === null ? null : { login, __typename: 'User' }, body: `Comment ${id}`, createdAt: '2026-10-15T10:00:00Z',
  url: `https://github.com/octo/widgets/pull/30#discussion_r${id}`, diffHunk: hunk, ...extra,
});
const node = (extra = {}) => ({
  id: 'T1', isResolved: false, isOutdated: false, path: 'docs/guide.md', line: 3, startLine: null, originalLine: 3, originalStartLine: null,
  diffSide: 'RIGHT', startDiffSide: null, subjectType: 'LINE', resolvedBy: null,
  comments: { totalCount: 1, nodes: [remark(11, 'alex')] }, ...extra,
});

const STACK = {
  members: [],
  warnings: [],
  lines: {
    'web/cart.ts': {
      RIGHT: { 4: [21, 2], 5: [21, 3], 9: { c: [[21, 7], [22, 8]] }, 10: { c: [] }, 12: [22, 12], 14: { x: 'no pull request\'s own diff adds this line as it reads here' } },
      LEFT: { 6: [21, 6], 7: [21, 7] },
    },
    'web/total.ts': { RIGHT: { 3: [22, 3, 'web/sum.ts'], 4: [22, 4, 'web/sum.ts'] }, LEFT: {} },
  },
};

const THREADS = [
  { pull: 30, stack: null, nodes: [
    node(),
    node({ id: 'T2', isResolved: true, resolvedBy: { login: 'sam' }, startLine: 2, startDiffSide: 'LEFT', comments: { totalCount: 104, nodes: [
      remark(11, 'alex'), remark(12, 'lint-helper', { author: { login: 'lint-helper', __typename: 'Bot' } }), remark(13, null),
      remark(14, '', { body: 'Use <b>bold</b> & "quotes"\nand a second line' }),
    ] } }),
    node({ id: 'T3', isOutdated: true, line: null, originalLine: 9 }),
    node({ id: 'T4', subjectType: 'FILE', line: null, originalLine: null }),
    node({ id: 'T5', startLine: 1 }),
    node({ id: 'T6', comments: { totalCount: 0, nodes: [] } }),
    node({ id: 'T7', comments: { totalCount: 1, nodes: [remark(15, 'alex', { diffHunk: '' })] } }),
    node({ id: 'T8', diffSide: 'LEFT', line: 5, comments: { totalCount: 2, nodes: [remark(16, 'alex', { diffHunk: '@@ -5 +5 @@\n-x' })] } }),
  ] },
  { pull: 21, stack: STACK, nodes: [
    node({ path: 'web/cart.ts', line: 3, startLine: 2 }),
    node({ id: 'T2', path: 'web/cart.ts', line: 7 }),
    node({ id: 'T3', path: 'web/cart.ts', line: 6, diffSide: 'LEFT' }),
    node({ id: 'T4', path: 'web/cart.ts', line: 7, diffSide: 'LEFT', startLine: 6 }),
    node({ id: 'T5', path: 'web/cart.ts', line: 12 }),
    node({ id: 'T6', path: 'web/cart.ts', line: 3, startLine: 1 }),
  ] },
  { pull: 22, stack: STACK, nodes: [
    node({ path: 'web/cart.ts', line: 8 }),
    node({ id: 'T2', path: 'web/sum.ts', line: 4, startLine: 3 }),
    node({ id: 'T3', path: 'web/sum.ts', line: 4, startLine: 8 }),
    node({ id: 'T4', path: 'web/cart.ts', line: 12, isOutdated: true }),
  ] },
];

const member = (pull, extra = {}) => ({
  pull, head: `${pull}`.repeat(40).slice(0, 40), start: `s${pull}`.repeat(40).slice(0, 40), title: `Pull request ${pull}`,
  url: `https://github.com/octo/widgets/pull/${pull}`, author: 'author', base: 'main', description: '', state: 'open', ended_at: null,
  color: 0, comments: [], posted: [], ...extra,
});
const fresh = (pull, extra = {}) => {
  const { color, comments, posted, ...rest } = member(pull, extra);
  return rest;
};
const comment = (id, extra = {}) => ({ id, path: 'a.ts', line: 3, side: 'RIGHT', body: `Comment ${id}`, code: '', ...extra });
const posted = (review, comments = []) => ({ set: 'set000000000', action: 'comment', review, at: '2026-10-15T11:00:00Z', comments });
const review = (id, members, extra = {}) => ({
  id, host: 'github.com', repo: 'octo/widgets', account: null, me: 'reviewer', title: 'A review', base: { ref: 'main', sha: 'b'.repeat(40) },
  context: false, loaded_at: '2026-10-15T10:00:00.000Z', notes: [], moved: null, members, earlier: [], ...extra,
});

const REGROUPS = [
  { previous: null, fresh: [fresh(1), fresh(2)], leaving: {}, absorbed: [], anchor: 'github.com/octo/widgets/1', now: 'now' },
  {
    previous: review('github.com/octo/widgets/1', [
      member(1, { color: 3, comments: [comment(1)] }),
      member(2, { color: 1, comments: [comment(2, { side: 'LEFT' }), comment(3), comment(4, { start_line: 1, start_side: 'LEFT' }), comment(5, { stale: 'already stale' })] }),
    ]),
    fresh: [fresh(1, { head: 'moved1head'.repeat(4) }), fresh(2, { start: 'start111'.repeat(5) }), fresh(4)],
    leaving: {}, absorbed: [], anchor: 'github.com/octo/widgets/1', now: 'now',
  },
  {
    previous: review('github.com/octo/widgets/1', [member(1, { color: 1, comments: [comment(4)], posted: [posted('https://github.com/r/1', [comment(7)])] })]),
    fresh: [fresh(1), fresh(2)], leaving: {},
    absorbed: [{ ref: 'github.com/octo/widgets/2', review: review('github.com/octo/widgets/2', [
      member(2, { color: 1, head: 'other2head'.repeat(4), comments: [comment(1), comment(2, { side: 'LEFT' })], posted: [posted('https://github.com/r/2'), posted('https://github.com/r/2')] }),
    ]) }],
    anchor: 'github.com/octo/widgets/1', now: '2026-10-15T12:00:00.000Z',
  },
  {
    previous: review('github.com/octo/widgets/1', [member(1, { color: 1 }), member(2, { color: 2, comments: [comment(1), comment(2)] })]),
    fresh: [fresh(1)], leaving: { 2: { state: 'open', ended_at: null } }, absorbed: [], anchor: 'a', now: 'n',
  },
  {
    previous: review('github.com/octo/widgets/1', [member(1, { color: 1 }), member(2, { color: 2, comments: [comment(1)] })]),
    fresh: [fresh(1)], leaving: { 2: { state: 'closed', ended_at: 'then' } }, absorbed: [], anchor: 'a', now: 'n',
  },
  {
    previous: review('github.com/octo/widgets/1', [member(1, { color: 1 }), member(2, { color: 2, state: 'merged', comments: [comment(1)] }), member(3, { color: 3, state: '' })],
      { earlier: [member(9, { color: 5, state: 'closed', ended_at: 'then' }), member(4, { color: 6, state: 'merged', comments: [comment(8)] })] }),
    fresh: [fresh(1), fresh(4)], leaving: {}, absorbed: [], anchor: 'a', now: 'n',
  },
  {
    previous: review('github.com/octo/widgets/1', [member(1, { color: 1, comments: [comment(1)] })], { moved: { to: 'github.com/octo/widgets/5', stack: [5, 1], at: 'then' } }),
    fresh: [fresh(1)], leaving: {}, absorbed: [], anchor: 'a', now: 'n',
  },
  {
    previous: review('github.com/octo/widgets/1', [1, 2, 3, 4, 5, 6, 7, 8].map(n => member(n, { color: n }))),
    fresh: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10].map(n => fresh(n)), leaving: {}, absorbed: [], anchor: 'a', now: 'n',
  },
  {
    previous: review('github.com/octo/widgets/1', [member(1, { color: 1 }), member(2, { color: 2, state: '', comments: [comment(1)] })]),
    fresh: [fresh(1)], leaving: {}, absorbed: [], anchor: 'a', now: 'n',
  },
];

function recorded() {
  const threads = THREADS.map(({ pull, stack, nodes }) => {
    const read = readThreads(pull, nodes);
    return { pull, stack, nodes, github: JSON.stringify(placeThreads(read, stack), null, 2) + '\n' };
  });
  const regroups = REGROUPS.map(given => {
    try {
      const out = regroup({ ...given, absorbed: given.absorbed.map(a => ({ ref: { id: a.ref }, review: a.review })) });
      return { given, out: { ...out, moved: out.moved.map(m => ({ ref: m.ref.id, review: m.review })) } };
    } catch (error) {
      return { given, error: error.message };
    }
  });
  return { threads, regroups };
}

test('the threads and regrouped queues the Go loader is tested against are the ones Node gives now', () => {
  const now = recorded();
  if (process.env.LAZYREVIEW_RECORD) {
    writeFileSync(RECORD, JSON.stringify(now, null, 1) + '\n');
    return;
  }
  assert.deepEqual(JSON.parse(readFileSync(RECORD, 'utf8')), now, `${RECORD} is not what Node gives now; write it again with LAZYREVIEW_RECORD=1 npm test, then run the Go tests`);
});
