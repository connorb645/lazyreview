import { test } from 'node:test';
import assert from 'node:assert/strict';
import { setId, offered, linesIn, queueView, liveMember } from '../../src/queue/queue-rules.mjs';
import { regroup } from '../../src/queue/regroup.mjs';
import { commentTarget } from '../../src/queue/comment-target.mjs';

const member = (pull, extra = {}) => ({ pull, head: `${pull}`.repeat(10).slice(0, 10), start: 'start000', author: 'author', state: 'open', color: null, comments: [], posted: [], ...extra });
const comment = (id, extra = {}) => ({ id, path: 'a.ts', line: 3, side: 'RIGHT', body: `Comment ${id}`, ...extra });

test('a set id names one action on one head with one set of comments, and nothing else', () => {
  const review = { repo: 'octo/widgets', me: 'reviewer', members: [], earlier: [] };
  const m = member(12, { comments: [comment(1)] });
  const id = setId(review, m, 'comment');
  assert.match(id, /^[0-9a-f]{12}$/);
  assert.equal(setId(review, { ...m }, 'comment'), id, 'the same queue gives the same set');
  assert.notEqual(setId(review, m, 'approve-with-comments'), id);
  assert.notEqual(setId(review, { ...m, head: 'other' }, 'comment'), id);
  assert.notEqual(setId(review, { ...m, comments: [comment(1, { body: 'Changed' })] }, 'comment'), id);
  assert.equal(setId(review, { ...m, comments: [comment(1, { code: 'shown only' })] }, 'comment'), id, 'what the page shows beside a comment is not part of what is posted');
  assert.equal(setId(review, { ...m, comments: [] }, 'approve'), setId(review, m, 'approve'), 'Approve sends no comments, so they do not change its set');
});

test('the actions offered leave out a stale comment and one\'s own approval', () => {
  const review = { repo: 'o/r', me: 'reviewer' };
  assert.deepEqual(offered(review, member(1)), ['approve']);
  assert.deepEqual(offered(review, member(1, { comments: [comment(1)] })), ['comment', 'approve-with-comments', 'approve']);
  assert.deepEqual(offered(review, member(1, { comments: [comment(1), comment(2, { stale: 'moved' })] })), ['approve']);
  assert.deepEqual(offered(review, member(1, { author: 'reviewer', comments: [comment(1)] })), ['comment']);
});

test('the lines of a patch are read on both sides', () => {
  const lines = linesIn('@@ -3,3 +3,4 @@\n one\n-two\n+TWO\n+three\n four\n\\ No newline at end of file');
  assert.deepEqual([...lines.RIGHT], [3, 4, 5, 6]);
  assert.deepEqual([...lines.LEFT], [3, 4, 5]);
  assert.deepEqual(linesIn(undefined), { RIGHT: new Set(), LEFT: new Set() }, 'a file GitHub shows no patch for, such as a binary one, has no lines');
  assert.deepEqual([...linesIn('@@ -1 +1 @@\n-a\n+b\n').RIGHT], [1], 'a trailing newline adds no line');
});

test('the queue view offers no button for an ended pull request, one that left, or a moved review', () => {
  const review = { repo: 'o/r', me: 'me', moved: null, members: [member(1, { comments: [comment(1)] }), member(2, { state: 'merged' })], earlier: [member(3)] };
  const view = queueView(review);
  assert.deepEqual(view.members.map(m => Object.keys(m.sets)), [['comment', 'approve-with-comments', 'approve'], []]);
  assert.deepEqual(Object.keys(view.earlier[0].sets), []);
  assert.deepEqual(queueView({ ...review, moved: { to: 'x' } }).members.map(m => Object.keys(m.sets)), [[], []]);
  assert.throws(() => liveMember(review, 3), /#3 is not in this stack any more/);
  assert.throws(() => liveMember({ ...review, moved: { to: 'github.com/o/r/9' } }, 1), /this review moved to github\.com\/o\/r\/9/);
});

test('regrouping keeps each member\'s comments and colour, and marks stale what may have moved', () => {
  const previous = {
    moved: null,
    members: [member(1, { color: 3, comments: [comment(1)] }), member(2, { color: 1, comments: [comment(2, { side: 'LEFT' }), comment(3)] })],
    earlier: [],
  };
  const fresh = [
    { ...member(1), comments: undefined, posted: undefined },
    { ...member(2), start: 'start111', comments: undefined, posted: undefined },
    { ...member(4), comments: undefined, posted: undefined },
  ];
  fresh[0].head = 'moved1head';
  const out = regroup({ previous, fresh, anchor: 'github.com/o/r/1', now: 't' });
  assert.deepEqual(out.members.map(m => m.color), [3, 1, 2], 'a new member takes the first free colour');
  assert.match(out.members[0].comments[0].stale, /^queued at 11111111, and #1 is now at moved1he/);
  assert.match(out.members[1].comments[0].stale, /queued on a removed line, and #2's diff now starts from start111/);
  assert.equal(out.members[1].comments[1].stale, undefined, 'a comment on the new side stays when only the start moved');
  assert.equal(out.notes.length, 2);
});

test('regrouping moves another review\'s queue in, renumbered, and refuses to drop an open member with comments', () => {
  const previous = { moved: null, members: [member(1, { color: 1, comments: [comment(4)] })], earlier: [] };
  const other = { id: 'github.com/o/r/2', moved: null, members: [member(2, { color: 1, comments: [comment(1), comment(2)] })], earlier: [] };
  const out = regroup({ previous, fresh: [member(1), member(2)], absorbed: [{ ref: 'two', review: other }], anchor: 'github.com/o/r/1', now: 'now' });
  assert.deepEqual(out.members[1].comments.map(c => c.id), [5, 6]);
  assert.deepEqual(out.moved[0].review.moved, { to: 'github.com/o/r/1', stack: [1, 2], at: 'now' });
  assert.deepEqual(out.moved[0].review.members[0].comments, []);
  const stacked = { moved: null, members: out.members, earlier: out.earlier };
  assert.throws(() => regroup({ previous: stacked, fresh: [member(1)], leaving: { 2: { state: 'open' } }, anchor: 'a', now: 'n' }), /#2 is still open with 2 queued comments, so it cannot leave the stack/);
  const left = regroup({ previous: { ...stacked, members: [out.members[0], { ...out.members[1], comments: [] }] }, fresh: [member(1)], leaving: { 2: { state: 'closed', ended_at: 'then' } }, anchor: 'a', now: 'n' });
  assert.deepEqual(left.earlier.map(m => [m.pull, m.state, m.ended_at]), [[2, 'closed', 'then']]);
});

test('a comment on a single pull request goes where it was made', () => {
  const review = { members: [member(7)] };
  assert.deepEqual(commentTarget(review, null, { path: 'a.ts', line: 3, side: 'RIGHT', start_line: 2 }),
    { pull: 7, target: { path: 'a.ts', line: 3, side: 'RIGHT', start_line: 2 }, shown: null });
  assert.throws(() => commentTarget(review, null, { path: 'a.ts', line: 3, side: 'RIGHT', pull: 8 }), /this review is #7's, not #8's/);
  const stack = { members: [{ pull: 7, head: 'other' }], lines: {} };
  assert.throws(() => commentTarget(review, stack, { path: 'a.ts', line: 3, side: 'RIGHT' }), /the page shows other heads of #7 than the review holds/);
});
