import { test } from 'node:test';
import assert from 'node:assert/strict';
import { request } from 'node:http';
import { workbench } from '../support/workbench.mjs';
import { fakeClaude } from '../support/fake-claude.mjs';
import { AppServer } from '../../src/server/app-server.mjs';
import { ThreadService } from '../../src/ask/thread-service.mjs';
import { Claude } from '../../src/ask/claude.mjs';
import { BranchLoader } from '../../src/loader/branch-loader.mjs';

function call(address, method, path, { body, headers = {} } = {}) {
  const url = new URL(path, address);
  return new Promise((resolve, reject) => {
    const req = request({ host: url.hostname, port: url.port, path: url.pathname + url.search, method, headers: { Host: url.host, ...headers } }, res => {
      const chunks = [];
      res.on('data', c => chunks.push(c));
      res.on('end', () => {
        const text = Buffer.concat(chunks).toString('utf8');
        resolve({ status: res.statusCode, headers: res.headers, text, json: /json/.test(res.headers['content-type'] || '') ? JSON.parse(text) : null });
      });
    });
    req.on('error', reject);
    req.end(body === undefined ? undefined : JSON.stringify(body));
  });
}

async function started() {
  const w = workbench();
  const fake = fakeClaude();
  await w.loader.load({ links: [w.link(30)] });
  const branches = new BranchLoader({ store: w.store });
  const server = new AppServer({
    store: w.store, loader: w.loader, branches, queue: w.queue,
    threads: new ThreadService({ store: w.store, claude: new Claude({ command: fake.command }) }), port: 0,
  });
  const address = await server.listen();
  const own = { Origin: address, 'Content-Type': 'application/json' };
  return { w, fake, server, branches, address, own, done: async () => { await server.close(); fake.done(); w.done(); } };
}

test('the app\'s pages and files are served with a policy that loads nothing from elsewhere', async () => {
  const s = await started();
  try {
    for (const path of ['/', '/review/github.com/octo/widgets/30']) {
      const page = await call(s.address, 'GET', path);
      assert.equal(page.status, 200);
      assert.match(page.text, /<script type="module" src="\/app\/main\.mjs"><\/script>/);
      assert.match(page.headers['content-security-policy'], /default-src 'self'; script-src 'self'/);
    }
    assert.match((await call(s.address, 'GET', '/app/main.mjs')).headers['content-type'], /text\/javascript/);
    assert.equal((await call(s.address, 'GET', '/vendor/marked-12.0.2.min.js')).status, 200);
    for (const path of ['/vendor/plus-jakarta-sans-5.3.0-latin.woff2', '/vendor/plus-jakarta-sans-5.3.0-latin-ext.woff2']) {
      const font = await call(s.address, 'GET', path);
      assert.equal(font.status, 200, path);
      assert.equal(font.headers['content-type'], 'font/woff2', path);
    }
    for (const path of ['/app/../src/cli/main.mjs', '/app/%2e%2e/package.json', '/vendor/elk-0.9.3.bundled.js', '/app/nothing.mjs', '/app/%E0%A4%A']) {
      assert.equal((await call(s.address, 'GET', path)).status, 404, path);
    }
  } finally {
    await s.done();
  }
});

test('the server answers only to its own name, and takes changes only from its own pages', async () => {
  const s = await started();
  try {
    const port = new URL(s.address).port;
    assert.equal((await call(s.address, 'GET', '/api/reviews', { headers: { Host: `attacker.example:${port}` } })).status, 421);
    assert.equal((await call(s.address, 'GET', '/api/reviews', { headers: { Host: `localhost:${port}` } })).status, 200);
    const comment = { path: 'docs/guide.md', line: 3, side: 'RIGHT', body: 'Every item?' };
    assert.equal((await call(s.address, 'POST', '/api/reviews/github.com/octo/widgets/30/comments', { body: comment, headers: { 'Content-Type': 'application/json' } })).status, 403);
    assert.equal((await call(s.address, 'POST', '/api/reviews/github.com/octo/widgets/30/comments', { body: comment, headers: { Origin: 'https://attacker.example', 'Content-Type': 'application/json' } })).status, 403);
    assert.equal((await call(s.address, 'POST', '/api/reviews/github.com/octo/widgets/30/comments', { body: comment, headers: { Origin: s.address, 'Content-Type': 'text/plain' } })).status, 415);
    assert.equal(s.w.store.read(s.w.store.list()[0].ref).members[0].comments.length, 0, 'no refused request changed anything');
  } finally {
    await s.done();
  }
});

test('the API lists reviews, gives one\'s state and page, and queues, edits, drops and posts', async () => {
  const s = await started();
  const base = '/api/reviews/github.com/octo/widgets/30';
  try {
    const list = (await call(s.address, 'GET', '/api/reviews')).json;
    assert.deepEqual(list.map(r => [r.id, r.title, r.queued]), [['github.com/octo/widgets/30', 'Fix the guide wording', 0]]);
    const page = (await call(s.address, 'GET', `${base}/page`)).json;
    assert.deepEqual(page.model.files.map(f => f.path), ['docs/guide.md']);
    assert.deepEqual(Object.keys(page).sort(), ['build', 'layout', 'model', 'stack'], 'the page leaves out the diff and the whole texts');
    assert.equal(page.build, s.w.store.pageBuild(s.w.store.list()[0].ref), 'the page names its build, which each change names back');
    const build = page.build;
    const file = (await call(s.address, 'GET', `${base}/file?path=docs%2Fguide.md&build=${build}`)).json;
    assert.deepEqual(file.hunks[0].rows.map(r => [r.kind, r.old, r.new, r.text, r.owner]), [
      ['same', 1, 1, '# Guide', null], ['same', 2, 2, '', null], ['del', 3, null, 'The catalog lists items.', null], ['add', null, 3, 'The catalog lists every item.', null],
      ['same', 4, 4, '', null], ['same', 5, 5, '- Browse', null], ['same', 6, 6, '- Search', null],
    ]);
    assert.deepEqual([file.gaps, file.whole.length], [[], 6]);
    assert.match((await call(s.address, 'GET', `${base}/file?path=docs%2Fguide.md&build=older`)).json.error, /^no file: the review was loaded again/);
    assert.match((await call(s.address, 'GET', `${base}/file?path=nowhere.md&build=${build}`)).json.error, /nowhere\.md is not a file of this review/);
    assert.equal((await call(s.address, 'GET', '/api/reviews/github.com/octo/widgets/99')).status, 404);

    const unnamed = await call(s.address, 'POST', `${base}/comments`, { body: { path: 'docs/guide.md', line: 3, side: 'RIGHT', body: 'Every item?' }, headers: s.own });
    assert.equal(unnamed.status, 409);
    assert.match(unnamed.json.error, /does not name the page build/);
    const added = await call(s.address, 'POST', `${base}/comments`, { body: { path: 'docs/guide.md', line: 3, side: 'RIGHT', body: 'Every item?', build }, headers: s.own });
    assert.equal(added.status, 200);
    assert.equal(added.json.state.queue.members[0].comments[0].body, 'Every item?');
    const refused = await call(s.address, 'POST', `${base}/comments`, { body: { path: 'docs/guide.md', line: 99, side: 'RIGHT', body: 'x', build }, headers: s.own });
    assert.equal(refused.status, 409);
    assert.match(refused.json.error, /GitHub takes comments only on lines of the diff/);
    const where = query => call(s.address, 'GET', `${base}/where?${new URLSearchParams(query)}`).then(r => r.json);
    assert.deepEqual(await where({ path: 'docs/guide.md', side: 'RIGHT', line: 3, build }), { goes: { pull: 30, path: 'docs/guide.md', side: 'RIGHT', line: 3, start_line: null, start_side: null }, words: 'Goes on #30, line 3.' });
    assert.deepEqual(await where({ path: 'docs/guide.md', side: 'RIGHT', line: 99, build }), { refused: 'GitHub takes comments only on lines of the diff, and this one is not. Ask Claude about it instead.' });
    assert.match((await where({ path: 'docs/guide.md', side: 'RIGHT', line: 3, build: 'older' })).refused, /^The review was loaded again after this window opened/);
    assert.equal((await call(s.address, 'POST', `${base}/comments`, { body: { path: 'docs/guide.md', line: 'x', side: 'UP', body: 'x', build }, headers: s.own })).status, 409);

    const edited = await call(s.address, 'PATCH', `${base}/comments/1`, { body: { body: 'Every single item?', version: added.json.state.queue.members[0].comments[0].version, build }, headers: s.own });
    assert.equal(edited.json.state.queue.members[0].comments[0].body, 'Every single item?');

    s.w.routes({ extra: {
      'POST repos/octo/widgets/pulls/30/reviews': { id: 5, state: 'COMMENTED', html_url: 'https://github.com/octo/widgets/pull/30#r5' },
      'GET repos/octo/widgets/pulls/30/comments?per_page=100&page=1': [{ pull_request_review_id: 5 }],
    } });
    const state = (await call(s.address, 'GET', base)).json;
    const posted = await call(s.address, 'POST', `${base}/post`, { body: { pull: 30, action: 'comment', set: state.queue.members[0].sets.comment.set, build }, headers: s.own });
    assert.equal(posted.json.message, 'posted 1 comment as one review: https://github.com/octo/widgets/pull/30#r5');
    assert.equal(posted.json.state.queue.members[0].posted.length, 1);

    const more = await call(s.address, 'POST', `${base}/comments`, { body: { path: 'docs/guide.md', line: 3, side: 'RIGHT', body: 'One more.', build }, headers: s.own });
    const { version } = more.json.state.queue.members[0].comments[0];
    assert.equal((await call(s.address, 'DELETE', `${base}/comments/2?version=${version}`, { headers: { Origin: s.address } })).status, 409);
    const dropped = await call(s.address, 'DELETE', `${base}/comments/2?version=${version}&build=${build}`, { headers: { Origin: s.address } });
    assert.deepEqual(dropped.json.state.queue.members[0].comments, []);
  } finally {
    await s.done();
  }
});

test('the API asks Claude about a line and keeps the conversation', async () => {
  const s = await started();
  const base = '/api/reviews/github.com/octo/widgets/30';
  try {
    const { build } = (await call(s.address, 'GET', `${base}/page`)).json;
    const started = await call(s.address, 'POST', `${base}/threads`, { body: { path: 'docs/guide.md', side: 'RIGHT', line: 3, text: 'Every?', build }, headers: s.own });
    assert.equal(started.status, 200);
    assert.deepEqual(started.json.thread.messages.map(m => m.from), ['you', 'claude']);
    const replied = await call(s.address, 'POST', `${base}/threads/1/messages`, { body: { text: 'Sure?', build }, headers: s.own });
    assert.equal(replied.json.thread.messages.length, 4);
    assert.equal((await call(s.address, 'GET', base)).json.threads.length, 1);
  } finally {
    await s.done();
  }
});

test('a branch review is served read only: its state, its page and its conversations, but no queue', async () => {
  const s = await started();
  try {
    const loaded = await s.branches.load({ repo: s.w.repo.work, base: 'main', head: 'fix/wording', title: 'Wording' });
    const base = '/api/reviews/' + loaded.ref.id.split('/').map(encodeURIComponent).join('/');
    assert.equal((await call(s.address, 'GET', '/review/' + loaded.ref.id)).status, 200);
    const state = (await call(s.address, 'GET', base)).json;
    assert.deepEqual([state.kind, state.title, state.head.ref, state.base.ref, state.queue], ['branch', 'Wording', 'fix/wording', 'main', null]);
    const page = (await call(s.address, 'GET', `${base}/page`)).json;
    assert.deepEqual(page.model.files.map(f => f.path), ['docs/guide.md']);
    const listed = (await call(s.address, 'GET', '/api/reviews')).json.find(r => r.id === loaded.ref.id);
    assert.deepEqual([listed.kind, listed.branch, listed.base, listed.queued], ['branch', 'fix/wording', 'main', 0]);

    const comment = { path: 'docs/guide.md', line: 3, side: 'RIGHT', body: 'Every item?' };
    const where = await call(s.address, 'GET', `${base}/where?${new URLSearchParams({ path: 'docs/guide.md', side: 'RIGHT', line: 3, build: page.build })}`);
    assert.match(where.json.refused, /^A branch with no pull request takes no comments/);
    assert.equal((await call(s.address, 'POST', `${base}/comments`, { body: comment, headers: s.own })).status, 404);
    assert.equal((await call(s.address, 'POST', `${base}/post`, { body: { pull: 1, action: 'comment', set: 'x' }, headers: s.own })).status, 404);

    const asked = await call(s.address, 'POST', `${base}/threads`, { body: { path: 'docs/guide.md', side: 'RIGHT', line: 3, text: 'Every?', build: page.build }, headers: s.own });
    assert.deepEqual(asked.json.thread.messages.map(m => m.from), ['you', 'claude']);
    assert.match(s.fake.asked()[0].prompt, /^Branch fix\/wording: Wording\. It has no pull request yet, and its diff is against main\.\n\nFile: docs\/guide\.md \(changed\)/);

    s.w.repo.commit('fix/wording', { 'docs/guide.md': '# Guide\n\nThe catalog lists every single item.\n\n- Browse\n- Search\n' }, 'Again');
    s.w.repo.checkout('main');
    const refreshed = await call(s.address, 'POST', `${base}/refresh`, { body: {}, headers: s.own });
    assert.equal(refreshed.json.id, loaded.ref.id);
    const after = (await call(s.address, 'GET', base)).json;
    assert.equal(after.head.sha, s.w.repo.sha('fix/wording'));
    assert.equal(after.threads.length, 1, 'loading again keeps the conversations');
  } finally {
    await s.done();
  }
});
