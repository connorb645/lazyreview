import { test } from 'node:test';
import assert from 'node:assert/strict';
import { existsSync, readFileSync } from 'node:fs';
import { workbench } from '../support/workbench.mjs';
import { PullRef } from '../../src/github/pull-ref.mjs';

const ref = n => new PullRef('github.com', 'octo/widgets', n);

test('a pull request loads into a review: its record, its page and its clone', async () => {
  const w = workbench();
  try {
    const loaded = await w.loader.load({ links: [w.link(30)] });
    assert.equal(loaded.ref.id, 'github.com/octo/widgets/30');
    assert.deepEqual(loaded.counts, { files: 1, adds: 1, dels: 1, components: 1, types: 0 });
    const review = w.store.read(ref(30));
    assert.equal(review.me, 'reviewer');
    assert.equal(review.title, 'Fix the guide wording');
    assert.deepEqual(review.base, { ref: 'main', sha: w.repo.sha('main') });
    assert.equal(review.context, false, 'unchanged context is drawn only when asked for');
    assert.deepEqual(review.members.map(m => [m.pull, m.head, m.author, m.base, m.state, m.color, m.comments.length]),
      [[30, w.repo.sha('fix/wording'), 'alex', 'main', 'open', 1, 0]]);
    assert.equal(review.members[0].start, w.repo.sha('main'));
    const page = JSON.parse(w.store.readPage(ref(30)));
    assert.equal(page.stack, null);
    assert.deepEqual(Object.keys(page.diff), ['docs/guide.md']);
    assert.match(page.model.heading.context.map(p => p[0]).join(''), /^Pull request #30 on main: head [0-9a-f]{8}, merge base [0-9a-f]{8}, 1 commit$/);
    assert.ok(existsSync(w.home.clone('github.com', 'octo/widgets')), 'the clone is kept under the data home');
  } finally {
    w.done();
  }
});

test('a stack loads bottom first into one review, each pull request in its own colour', async () => {
  const w = workbench();
  try {
    const loaded = await w.loader.load({ links: [w.link(21), w.link(22)] });
    assert.equal(loaded.ref.id, 'github.com/octo/widgets/21');
    const review = w.store.read(ref(21));
    assert.equal(review.title, 'Stack #21 to #22');
    assert.deepEqual(review.members.map(m => [m.pull, m.color, m.start]), [[21, 1, w.repo.sha('main')], [22, 2, w.repo.sha('feature/search')]]);
    const page = JSON.parse(w.store.readPage(ref(21)));
    assert.deepEqual(page.stack.members.map(m => m.pull), [21, 22]);
    assert.deepEqual(page.model.files.find(f => f.path === 'Sources/Basket/Basket.swift').pulls, [22]);
    assert.deepEqual(page.stack.lines['Sources/Catalog/MemoryCatalog.swift'].RIGHT['9'], [21, 9]);
  } finally {
    w.done();
  }
});

test('a load keeps the review threads on GitHub with the page, each on the line it shows them on', async () => {
  const w = workbench();
  const thread = (id, path, line, login, extra = {}) => ({
    id, isResolved: false, isOutdated: false, path, line, startLine: null, originalLine: line, originalStartLine: null, diffSide: 'RIGHT', startDiffSide: null,
    subjectType: 'LINE', resolvedBy: null, ...extra,
    comments: { totalCount: 1, nodes: [{ databaseId: 1, author: { login, __typename: 'User' }, body: 'Why?', createdAt: '2026-10-15T10:00:00Z', url: 'https://github.com/x', diffHunk: '' }] },
  });
  try {
    w.routes({ reviews: {
      21: { threads: [thread('T1', 'Sources/Catalog/MemoryCatalog.swift', 9, 'alex'), thread('T2', 'Sources/Catalog/Search.swift', null, 'alex', { isOutdated: true, isResolved: true })] },
      22: { threads: [thread('T3', 'Sources/Basket/Basket.swift', 7, 'sam')] },
    } });
    await w.loader.load({ links: [w.link(21), w.link(22)] });
    const page = JSON.parse(w.store.readPage(ref(21)));
    assert.deepEqual(page.threads.map(t => [t.id, t.pull, t.resolved, t.outdated, t.at && t.at.line]), [['T1', 21, false, false, 9], ['T2', 21, true, true, null], ['T3', 22, false, false, 7]]);

    w.routes({ extra: { 'POST graphql octo/widgets#30': { errors: [{ message: 'Resource not accessible' }] } } });
    const loaded = await w.loader.load({ links: [w.link(30)] });
    assert.deepEqual(JSON.parse(w.store.readPage(ref(30))).threads, []);
    assert.deepEqual(loaded.notes, ["#30's review threads on GitHub are not shown, as they could not be read: GitHub refused the review threads of octo/widgets#30: Resource not accessible"]);
    assert.deepEqual(w.store.read(ref(30)).notes, loaded.notes, 'the Review view says so too');
  } finally {
    w.done();
  }
});

test('a load refuses pull requests that are not a stack bottom first, or not of one repository', async () => {
  const w = workbench();
  try {
    await assert.rejects(w.loader.load({ links: [w.link(21), w.link(30)] }), /these are not a stack, bottom first: #30 is based on main, not on #21's branch feature\/search/);
    await assert.rejects(w.loader.load({ links: [w.link(22), w.link(21)] }), /not a stack, bottom first/);
    await assert.rejects(w.loader.load({ links: [w.link(21), 'https://github.com/octo/other/pull/2'] }), /a stack is in one repository/);
    await assert.rejects(w.loader.load({ links: [w.link(21), w.link(21)] }), /named twice/);
    await assert.rejects(w.loader.load({ links: ['not a link'] }), /is not a pull request's address/);
    assert.deepEqual(w.store.list(), [], 'nothing is written when a load is refused');
  } finally {
    w.done();
  }
});

test('loading again after new commits marks the queued comments stale, and keeps the rest', async () => {
  const w = workbench();
  try {
    await w.loader.load({ links: [w.link(30)] });
    await w.queue.add(ref(30), { path: 'docs/guide.md', line: 3, side: 'RIGHT', body: 'Every item, or every item shown?' });
    w.repo.commit('fix/wording', { 'docs/guide.md': '# Guide\n\nThe catalog lists every single item.\n\n- Browse\n- Search\n' }, 'Again');
    w.repo.checkout('main');
    w.routes();
    const loaded = await w.loader.load({ links: [w.link(30)] });
    assert.match(loaded.notes.join('\n'), /#30: 1 queued comment marked stale \(queued at [0-9a-f]{8}, and #30 is now at [0-9a-f]{8}\)/);
    const [member] = w.store.read(ref(30)).members;
    assert.equal(member.head, w.repo.sha('fix/wording'));
    assert.equal(member.comments.length, 1);
    assert.match(member.comments[0].stale, /^queued at /);
  } finally {
    w.done();
  }
});

test('a stack takes in the queues of its pull requests\' own reviews, which then point at it', async () => {
  const w = workbench();
  try {
    await w.loader.load({ links: [w.link(22)] });
    await w.queue.add(ref(22), { path: 'Sources/Basket/Basket.swift', line: 7, side: 'RIGHT', body: 'Could this be passed in?' });
    await w.loader.load({ links: [w.link(21)] });
    await w.queue.add(ref(21), { path: 'Sources/Catalog/Search.swift', line: 5, side: 'RIGHT', body: 'Is this case sensitive on purpose?' });
    const loaded = await w.loader.load({ links: [w.link(21), w.link(22)] });
    assert.equal(loaded.ref.id, 'github.com/octo/widgets/21', 'the stack lives with its lowest pull request that had a review');
    assert.match(loaded.notes.join('\n'), /#22's comment 1 is now 2/);
    const stack = w.store.read(ref(21));
    assert.deepEqual(stack.members.map(m => m.comments.map(c => [c.id, c.body])), [[[1, 'Is this case sensitive on purpose?']], [[2, 'Could this be passed in?']]]);
    const moved = w.store.read(ref(22));
    assert.deepEqual(moved.moved.stack, [21, 22]);
    assert.equal(moved.moved.to, 'github.com/octo/widgets/21');
    assert.deepEqual(moved.members[0].comments, []);
    assert.equal(w.store.find(ref(22)).ref.id, 'github.com/octo/widgets/21', 'a pull request is found in the review that holds it');
  } finally {
    w.done();
  }
});

test('a pull request leaves a stack only with nothing queued while it is open', async () => {
  const w = workbench();
  try {
    await w.loader.load({ links: [w.link(21), w.link(22)] });
    await w.queue.add(ref(21), { path: 'Sources/Basket/Basket.swift', line: 7, side: 'RIGHT', body: 'Could this be passed in?' });
    await assert.rejects(w.loader.load({ links: [w.link(21)] }), /#22 is still open with 1 queued comment, so it cannot leave the stack/);
    w.routes({ 21: { merged: true, state: 'closed' } });
    const refreshed = await w.loader.refresh(ref(21));
    assert.match(refreshed.notes.join('\n'), /#21 left the stack \(merged\)/);
    const review = w.store.read(ref(21));
    assert.deepEqual(review.members.map(m => m.pull), [22]);
    assert.deepEqual(review.earlier.map(m => [m.pull, m.state]), [[21, 'merged']]);
    assert.equal(review.members[0].comments.length, 1, 'the comment stays with the pull request it goes on');
  } finally {
    w.done();
  }
});

test('the account a review was loaded as acts for every later call, through its own token', async () => {
  const w = workbench();
  try {
    await w.loader.load({ links: [w.link(30)], account: 'second-account' });
    assert.equal(w.store.read(ref(30)).account, 'second-account');
    assert.ok(w.gh.calls().every(c => c.token === 'token-second-account'));
    await w.loader.load({ links: [w.link(30)] });
    assert.equal(w.store.read(ref(30)).account, 'second-account', 'loading again keeps the account');
  } finally {
    w.done();
  }
});

test('a components config is kept with the review and used again on the next load', async () => {
  const w = workbench();
  try {
    const config = { components: [{ id: 'catalog', title: 'Catalog', paths: ['Sources/Catalog/'] }] };
    await w.loader.load({ links: [w.link(21)], config, context: true });
    await w.loader.load({ links: [w.link(21)] });
    const page = JSON.parse(w.store.readPage(ref(21)));
    assert.deepEqual(page.model.components.map(c => c.title), ['Catalog', 'Unchanged code']);
    assert.equal(w.store.read(ref(21)).context, true, 'the choice of context is kept too');
    assert.deepEqual(JSON.parse(readFileSync(w.home.review(ref(21)).config, 'utf8')), config);
    await assert.rejects(w.loader.load({ links: [w.link(21)], config: { components: [{ id: 'Bad' }] } }), /the config has \d problems/);
  } finally {
    w.done();
  }
});
