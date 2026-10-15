import { test } from 'node:test';
import assert from 'node:assert/strict';
import { workbench } from '../support/workbench.mjs';
import { PullRef } from '../../src/github/pull-ref.mjs';
import { queueView } from '../../src/queue/queue-rules.mjs';

const ref = n => new PullRef('github.com', 'octo/widgets', n);
const REVIEW_URL = 'https://github.com/octo/widgets/pull/30#pullrequestreview-7';

function sets(w, n, pull = n) {
  return queueView(w.store.read(ref(n))).members.find(m => m.pull === pull).sets;
}
function posts(w) {
  return w.gh.calls().filter(c => c.method === 'POST');
}

test('a comment is queued on a line of the diff, and refused anywhere else', async () => {
  const w = workbench();
  try {
    await w.loader.load({ links: [w.link(30)] });
    const done = await w.queue.add(ref(30), { path: 'docs/guide.md', line: 3, side: 'RIGHT', body: '  Every item, or every item shown?  ', code: 'The catalog lists every item.' });
    assert.deepEqual(done.comment, { id: 1, path: 'docs/guide.md', line: 3, side: 'RIGHT', body: 'Every item, or every item shown?', code: 'The catalog lists every item.' });
    await assert.rejects(w.queue.add(ref(30), { path: 'docs/guide.md', line: 40, side: 'RIGHT', body: 'Hm' }), /not queued: docs\/guide\.md:40 is not on a line of the diff/);
    await assert.rejects(w.queue.add(ref(30), { path: 'docs/guide.md', line: 3, side: 'RIGHT', body: '   ' }), /not queued: the text is empty/);
    await w.queue.add(ref(30), { path: 'docs/guide.md', line: 3, side: 'LEFT', body: 'Why drop this?' });
    assert.deepEqual(w.store.read(ref(30)).members[0].comments.map(c => [c.id, c.side]), [[1, 'RIGHT'], [2, 'LEFT']]);
  } finally {
    w.done();
  }
});

test('on a stack, a comment goes to the pull request that last changed its line, at that pull request\'s own line', async () => {
  const w = workbench();
  try {
    await w.loader.load({ links: [w.link(21), w.link(22)] });
    const first = await w.queue.add(ref(21), { path: 'Sources/Catalog/MemoryCatalog.swift', line: 9, side: 'RIGHT', body: 'Should add check for duplicates?' });
    assert.equal(first.pull, 21);
    assert.deepEqual(first.comment.shown, { path: 'Sources/Catalog/MemoryCatalog.swift', line: 9, side: 'RIGHT' });
    const second = await w.queue.add(ref(21), { path: 'Sources/Basket/Basket.swift', line: 7, side: 'RIGHT', body: 'Could this be passed in?' });
    assert.equal(second.pull, 22);
    await assert.rejects(w.queue.add(ref(21), { path: 'Sources/Basket/Basket.swift', line: 7, side: 'RIGHT', body: 'x', pull: 21 }), /#22 last changed Sources\/Basket\/Basket\.swift:7/);
    const range = await w.queue.add(ref(21), { path: 'Sources/Basket/Basket.swift', line: 7, start_line: 3, side: 'RIGHT', body: 'Both of these?' });
    assert.deepEqual([range.pull, range.comment.start_line, range.comment.line], [22, 3, 7]);
    await assert.rejects(w.queue.add(ref(21), { path: 'Sources/Catalog/Catalog.swift', line: 2, side: 'RIGHT', body: 'x' }), /is not a line of the stack's combined diff/);
    const review = w.store.read(ref(21));
    assert.deepEqual(review.members.map(m => m.comments.map(c => c.id)), [[1], [2, 3]]);
  } finally {
    w.done();
  }
});

test('a comment is refused once its pull request moved on GitHub', async () => {
  const w = workbench();
  try {
    await w.loader.load({ links: [w.link(30)] });
    w.repo.commit('fix/wording', { 'docs/guide.md': '# Guide\n\nAll items.\n' }, 'more');
    w.repo.checkout('main');
    w.routes();
    await assert.rejects(w.queue.add(ref(30), { path: 'docs/guide.md', line: 3, side: 'RIGHT', body: 'Hm' }), /not queued: #30 moved from [0-9a-f]{8} to [0-9a-f]{8}; load it again/);
  } finally {
    w.done();
  }
});

test('Add Comments posts exactly the queued comments as one review, and only with the set the page showed', async () => {
  const w = workbench();
  try {
    await w.loader.load({ links: [w.link(30)] });
    await w.queue.add(ref(30), { path: 'docs/guide.md', line: 3, side: 'RIGHT', body: 'Every item, or every item shown?' });
    const shown = sets(w, 30);
    assert.deepEqual(Object.keys(shown), ['comment', 'approve-with-comments', 'approve']);
    await assert.rejects(w.queue.post(ref(30), { pull: 30, action: 'comment', set: shown.approve }), /the page showed set [0-9a-f]{12}, but "comment" now gives set [0-9a-f]{12}; nothing was posted/);
    await w.queue.add(ref(30), { path: 'docs/guide.md', line: 3, side: 'LEFT', body: 'And this one?' });
    await assert.rejects(w.queue.post(ref(30), { pull: 30, action: 'comment', set: shown.comment }), /nothing was posted/, 'a set id shown before the queue changed no longer posts');
    assert.deepEqual(posts(w), []);

    w.routes({ extra: {
      'POST repos/octo/widgets/pulls/30/reviews': { id: 7, state: 'COMMENTED', html_url: REVIEW_URL },
      'GET repos/octo/widgets/pulls/30/comments?per_page=100&page=1': [{ pull_request_review_id: 7 }, { pull_request_review_id: 7 }],
    } });
    const done = await w.queue.post(ref(30), { pull: 30, action: 'comment', set: sets(w, 30).comment });
    assert.equal(done.message, `posted 2 comments as one review: ${REVIEW_URL}`);
    assert.deepEqual(posts(w).map(c => c.body), [{
      commit_id: w.repo.sha('fix/wording'), event: 'COMMENT',
      comments: [
        { path: 'docs/guide.md', line: 3, side: 'RIGHT', body: 'Every item, or every item shown?' },
        { path: 'docs/guide.md', line: 3, side: 'LEFT', body: 'And this one?' },
      ],
    }]);
    const member = w.store.read(ref(30)).members[0];
    assert.deepEqual(member.comments, []);
    assert.deepEqual(member.posted.map(p => [p.action, p.review, p.comments.length]), [['comment', REVIEW_URL, 2]]);
    assert.deepEqual(Object.keys(sets(w, 30)), ['approve'], 'with nothing queued only Approve is offered');
  } finally {
    w.done();
  }
});

test('Approve posts an approval alone and leaves the queued comments queued', async () => {
  const w = workbench();
  try {
    await w.loader.load({ links: [w.link(30)] });
    await w.queue.add(ref(30), { path: 'docs/guide.md', line: 3, side: 'RIGHT', body: 'Nit.' });
    w.routes({ extra: { 'POST repos/octo/widgets/pulls/30/reviews': { id: 8, state: 'APPROVED', html_url: REVIEW_URL } } });
    const done = await w.queue.post(ref(30), { pull: 30, action: 'approve', set: sets(w, 30).approve });
    assert.equal(done.message, `approved, with no comments: ${REVIEW_URL}`);
    assert.deepEqual(posts(w)[0].body, { commit_id: w.repo.sha('fix/wording'), event: 'APPROVE' });
    assert.equal(w.store.read(ref(30)).members[0].comments.length, 1);
  } finally {
    w.done();
  }
});

test('nothing is posted to a closed pull request, a moved head, a stale comment or one\'s own pull request', async () => {
  const w = workbench();
  try {
    await w.loader.load({ links: [w.link(30)] });
    await w.queue.add(ref(30), { path: 'docs/guide.md', line: 3, side: 'RIGHT', body: 'Nit.' });
    const shown = sets(w, 30);
    w.routes({ 30: { state: 'closed' } });
    await assert.rejects(w.queue.post(ref(30), { pull: 30, action: 'comment', set: shown.comment }), /nothing was posted: #30 is closed/);
    w.repo.commit('fix/wording', { 'docs/guide.md': '# Guide\n\nAll items.\n' }, 'more');
    w.repo.checkout('main');
    w.routes();
    await assert.rejects(w.queue.post(ref(30), { pull: 30, action: 'comment', set: shown.comment }), /the head moved from [0-9a-f]{9} to [0-9a-f]{9}/);
    await w.loader.load({ links: [w.link(30)] });
    assert.deepEqual(Object.keys(sets(w, 30)), ['approve'], 'a stale comment takes away the buttons that would send it');
    assert.deepEqual(posts(w), []);
  } finally {
    w.done();
  }
  const own = workbench();
  try {
    own.routes({ 30: { author: 'reviewer' } });
    await own.loader.load({ links: [own.link(30)] });
    await own.queue.add(ref(30), { path: 'docs/guide.md', line: 3, side: 'RIGHT', body: 'Note to self.' });
    assert.deepEqual(Object.keys(sets(own, 30)), ['comment'], 'no approval of one\'s own pull request');
    await assert.rejects(own.queue.post(ref(30), { pull: 30, action: 'approve', set: 'abc' }), /"approve" is not offered for #30; nothing was posted/);
  } finally {
    own.done();
  }
});

test('a press whose review GitHub already holds posts nothing new, and records it', async () => {
  const w = workbench();
  try {
    await w.loader.load({ links: [w.link(30)] });
    await w.queue.add(ref(30), { path: 'docs/guide.md', line: 3, side: 'RIGHT', body: 'Every item?' });
    const head = w.repo.sha('fix/wording');
    w.routes({ reviews: { 30: {
      reviews: [{ id: 9, user: { login: 'reviewer' }, commit_id: head, state: 'COMMENTED', html_url: REVIEW_URL }],
      comments: [{ pull_request_review_id: 9, path: 'docs/guide.md', body: 'Every item?\n' }],
    } } });
    const done = await w.queue.post(ref(30), { pull: 30, action: 'comment', set: sets(w, 30).comment });
    assert.equal(done.message, `already done as ${REVIEW_URL}; posted nothing new`);
    assert.deepEqual(posts(w), []);
    const member = w.store.read(ref(30)).members[0];
    assert.deepEqual(member.comments, []);
    assert.equal(member.posted[0].review, REVIEW_URL);
  } finally {
    w.done();
  }
});

test('nothing is posted when GitHub answers as another account than the review was loaded as', async () => {
  const w = workbench();
  try {
    await w.loader.load({ links: [w.link(30)] });
    w.routes({ extra: { 'GET user': { login: 'someone-else' } } });
    await assert.rejects(w.queue.post(ref(30), { pull: 30, action: 'approve', set: sets(w, 30).approve }), /GitHub answers as someone-else now, but this review was loaded as reviewer; nothing was posted/);
    assert.deepEqual(posts(w), []);
  } finally {
    w.done();
  }
});

test('on a stack, each pull request posts its own review with its own set', async () => {
  const w = workbench();
  try {
    await w.loader.load({ links: [w.link(21), w.link(22)] });
    await w.queue.add(ref(21), { path: 'Sources/Basket/Basket.swift', line: 7, side: 'RIGHT', body: 'Could this be passed in?' });
    const s21 = sets(w, 21, 21), s22 = sets(w, 21, 22);
    assert.deepEqual(Object.keys(s21), ['approve']);
    assert.deepEqual(Object.keys(s22), ['comment', 'approve-with-comments', 'approve']);
    await assert.rejects(w.queue.post(ref(21), { pull: 21, action: 'comment', set: s22.comment }), /"comment" is not offered for #21/);
    w.routes({ extra: {
      'POST repos/octo/widgets/pulls/22/reviews': { id: 11, state: 'APPROVED', html_url: 'https://github.com/octo/widgets/pull/22#r' },
      'GET repos/octo/widgets/pulls/22/comments?per_page=100&page=1': [{ pull_request_review_id: 11 }],
    } });
    const done = await w.queue.post(ref(21), { pull: 22, action: 'approve-with-comments', set: s22['approve-with-comments'] });
    assert.match(done.message, /^approved with 1 comment: /);
    assert.deepEqual(posts(w).map(c => [c.path, c.body.event, c.body.comments[0].line]), [['repos/octo/widgets/pulls/22/reviews', 'APPROVE', 7]]);
  } finally {
    w.done();
  }
});

test('a queued comment is edited or dropped by its number', async () => {
  const w = workbench();
  try {
    await w.loader.load({ links: [w.link(30)] });
    await w.queue.add(ref(30), { path: 'docs/guide.md', line: 3, side: 'RIGHT', body: 'First.' });
    await w.queue.add(ref(30), { path: 'docs/guide.md', line: 3, side: 'LEFT', body: 'Second.' });
    await w.queue.edit(ref(30), 1, ' First, reworded. ');
    await assert.rejects(w.queue.edit(ref(30), 1, ' '), /not changed: the text is empty/);
    await w.queue.drop(ref(30), 2);
    await assert.rejects(w.queue.drop(ref(30), 9), /no queued comment 9/);
    assert.deepEqual(w.store.read(ref(30)).members[0].comments.map(c => [c.id, c.body]), [[1, 'First, reworded.']]);
    await w.queue.add(ref(30), { path: 'docs/guide.md', line: 3, side: 'RIGHT', body: 'Third.' });
    assert.equal(w.store.read(ref(30)).members[0].comments[1].id, 2, 'a new comment takes the number after the highest still held');
  } finally {
    w.done();
  }
});
