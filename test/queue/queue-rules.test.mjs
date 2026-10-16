import { test } from 'node:test';
import assert from 'node:assert/strict';
import { setId, offered, diffPlaces, placeProblem, postedFields, queueView, liveMember, pressWords } from '../../src/queue/queue-rules.mjs';
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

test('the lines of a patch are placed on both sides, each in its hunk and its row', () => {
  const places = diffPlaces('@@ -3,3 +3,4 @@\n one\n-two\n+TWO\n+three\n four\n\\ No newline at end of file\n@@ -20 +21 @@\n-x\n+y');
  assert.deepEqual([...places.RIGHT.keys()], [3, 4, 5, 6, 21]);
  assert.deepEqual([...places.LEFT.keys()], [3, 4, 5, 20]);
  assert.deepEqual([places.LEFT.get(3), places.RIGHT.get(3)], [{ hunk: 0, row: 1 }, { hunk: 0, row: 1 }], 'an unchanged line has one place');
  assert.deepEqual(places.RIGHT.get(21), { hunk: 1, row: 9 });
  assert.deepEqual(diffPlaces(undefined), { RIGHT: new Map(), LEFT: new Map() }, 'a file GitHub shows no patch for, such as a binary one, has no lines');
  assert.deepEqual([...diffPlaces('@@ -1 +1 @@\n-a\n+b\n').RIGHT.keys()], [1], 'a trailing newline adds no line');
});

test('GitHub takes a comment on a line of the diff, and a range only inside one hunk, first line first', () => {
  const places = diffPlaces('@@ -3,3 +3,4 @@\n one\n-two\n+TWO\n+three\n four\n@@ -20,2 +21,2 @@\n-x\n+y\n z');
  const at = (line, side = 'RIGHT', start_line, start_side) => placeProblem(places, { line, side, start_line, start_side });
  assert.equal(at(4), null);
  assert.equal(at(4, 'LEFT'), null);
  assert.equal(at(9), 'is not on a line of the diff');
  assert.equal(at(6, 'RIGHT', 3), null);
  assert.equal(at(5, 'RIGHT', 4, 'LEFT'), null, 'a range may start on a removed line and end on an added one');
  assert.equal(at(22, 'RIGHT', 4), 'runs from line 4 across more than one hunk of the diff, and GitHub takes a range only inside one hunk');
  assert.equal(at(5, 'RIGHT', 8), 'starts on line 8, which is not on a line of the diff');
  assert.equal(at(4, 'RIGHT', 5), 'starts on line 5, which does not come before its last line in the diff');
  assert.equal(at(4, 'LEFT', 4, 'RIGHT'), 'starts on line 4, which does not come before its last line in the diff', 'the removed line comes before the line added in its place');
  assert.deepEqual(postedFields({ path: 'a.ts', line: 5, side: 'RIGHT', start_line: 4, start_side: 'LEFT', body: 'b' }),
    { path: 'a.ts', line: 5, side: 'RIGHT', start_line: 4, start_side: 'LEFT', body: 'b' });
  assert.equal(postedFields({ path: 'a.ts', line: 6, side: 'RIGHT', start_line: 3, body: 'b' }).start_side, 'RIGHT', 'a range queued before ranges had a side starts on its own side');
});

test('the queue view offers no button for an ended pull request, one that left, or a moved review', () => {
  const review = { repo: 'o/r', me: 'me', moved: null, members: [member(1, { comments: [comment(1)] }), member(2, { state: 'merged' })], earlier: [member(3)] };
  const view = queueView(review);
  assert.deepEqual(view.members.map(m => Object.keys(m.sets)), [['comment', 'approve-with-comments', 'approve'], []]);
  assert.deepEqual(Object.keys(view.earlier[0].sets), []);
  assert.deepEqual(queueView({ ...review, moved: { to: 'x' } }).members.map(m => Object.keys(m.sets)), [[], []]);
  assert.deepEqual(Object.values(view.members[0].sets).map(s => s.words), [
    'This comment goes on o/r#1 under your name, exactly as shown.',
    'You approve o/r#1, and this comment goes with the approval under your name, exactly as shown.',
    'You approve o/r#1 with no comments. The queued 1 comment stays queued, unposted.',
  ], 'each press comes with the sentence both apps ask before it posts');
  assert.match(pressWords(review, member(1, { comments: [comment(1), comment(2)] }), 'comment'), /^These 2 comments go on o\/r#1/);
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

test('a comment on a single pull request goes where it was made, on a line of the diff and a range inside one hunk', () => {
  const review = { members: [member(7)] };
  const page = { model: { files: [{ path: 'a.ts' }] }, diff: { 'a.ts': 'diff --git a/a.ts b/a.ts\n--- a/a.ts\n+++ b/a.ts\n@@ -1,3 +1,4 @@\n one\n+two\n three\n four\n@@ -20,2 +21,2 @@\n-x\n+y\n z' }, stack: null };
  const at = given => commentTarget(review, page, { path: 'a.ts', side: 'RIGHT', ...given });
  assert.deepEqual(at({ line: 3, start_line: 2 }), { goes: { pull: 7, path: 'a.ts', side: 'RIGHT', line: 3, start_line: 2, start_side: 'RIGHT' }, words: 'Goes on #7, lines 2-3.' });
  assert.deepEqual(at({ line: 21, start_line: 20, start_side: 'LEFT' }).words, 'Goes on #7, removed line 20 to line 21.');
  assert.deepEqual(at({ line: 9 }), { refused: 'GitHub takes comments only on lines of the diff, and this one is not. Ask Claude about it instead.' });
  assert.deepEqual(at({ line: 1, start_line: null }).goes.line, 1, 'the header of the file\'s diff is no line');
  assert.match(at({ line: 21, start_line: 2 }).refused, /^GitHub takes a range only inside one hunk/);
  assert.match(at({ line: 2, start_line: 3 }).refused, /^A range runs from its first line to its last/);
  assert.deepEqual(at({ line: 2, pull: 8 }), { refused: 'This review is #7\'s, not #8\'s.' });
  assert.deepEqual(commentTarget(review, page, { path: 'b.ts', side: 'RIGHT', line: 1 }).refused, 'This file is not in the change, so a comment cannot go on it. Ask Claude about it instead.');
  assert.deepEqual(commentTarget({ members: [member(7, { state: 'closed' })] }, page, { path: 'a.ts', side: 'RIGHT', line: 2 }), { refused: '#7 is closed, so nothing can be queued for it.' });
  assert.match(commentTarget({ ...review, moved: { to: 'x' } }, page, { path: 'a.ts', side: 'RIGHT', line: 2 }).refused, /^This review moved/);
  assert.match(commentTarget({ kind: 'branch' }, page, { path: 'a.ts', side: 'RIGHT', line: 2 }).refused, /^A branch with no pull request takes no comments/);
});

test('on a stack, a comment goes to the pull request that last changed its line, or names one of those that show it', () => {
  const review = { members: [member(7), member(8)] };
  const page = {
    model: { files: [{ path: 'a.ts' }] },
    diff: { 'a.ts': '@@ -1,3 +1,4 @@\n one\n+two\n three\n four\n@@ -20,2 +21,2 @@\n-x\n+y\n z' },
    stack: {
      members: review.members.map(m => ({ pull: m.pull, head: m.head })),
      lines: { 'a.ts': { RIGHT: { 1: { c: [[7, 1], [8, 1]] }, 2: [8, 2], 3: { c: [[8, 3]] }, 4: { c: [] }, 21: { x: 'no pull request\'s own diff adds it' }, 22: [7, 30, 'old/a.ts'] }, LEFT: {} } },
    },
  };
  const at = given => commentTarget(review, page, { path: 'a.ts', side: 'RIGHT', ...given });
  assert.deepEqual(at({ line: 2 }), { goes: { pull: 8, path: 'a.ts', side: 'RIGHT', line: 2, start_line: null, start_side: null }, words: 'Goes on #8, line 2.' });
  assert.deepEqual(at({ line: 1 }), { choose: [{ pull: 7, path: 'a.ts', line: 1 }, { pull: 8, path: 'a.ts', line: 1 }], words: 'Unchanged, and shown by #7 and #8. Choose the pull request it goes on.' });
  assert.equal(at({ line: 1, pull: 8 }).words, 'Goes on #8, line 1.');
  assert.deepEqual(at({ line: 2, pull: 7 }), { refused: '#8 last changed this line, not #7.' });
  assert.equal(at({ line: 2, start_line: 1 }).words, 'Goes on #8, lines 1-2.', 'a range may start on a line several pull requests show');
  assert.deepEqual(at({ line: 4 }), { refused: 'Unchanged, and no pull request in the stack shows this line, so it takes no comment.' });
  assert.deepEqual(at({ line: 21 }), { refused: 'No pull request in the stack can take a comment here: no pull request\'s own diff adds it.' });
  assert.equal(at({ line: 22 }).words, 'Goes on #7, line 30 of old/a.ts.', 'a comment goes at the pull request\'s own path and line');
  assert.equal(at({ line: 3, start_line: 2 }).words, 'Goes on #8, lines 2-3.');
  assert.match(at({ line: 3, start_line: 1, pull: 7 }).refused, /^#7's diff does not show this line\./);
  assert.deepEqual(commentTarget({ members: [member(7), member(8, { state: 'merged' })] }, page, { path: 'a.ts', side: 'RIGHT', line: 2 }), { refused: '#8 is merged, so nothing can be queued for it.' });
  const moved = { ...page, stack: { ...page.stack, members: [{ pull: 7, head: 'other' }, { pull: 8, head: review.members[1].head }] } };
  assert.match(commentTarget(review, moved, { path: 'a.ts', side: 'RIGHT', line: 2 }).refused, /^This page shows other heads of #7 than the review holds/);
});
