import { test } from 'node:test';
import assert from 'node:assert/strict';
import { request } from 'node:http';
import { workbench } from '../support/workbench.mjs';
import { fakeClaude } from '../support/fake-claude.mjs';
import { AppServer } from '../../src/server/app-server.mjs';
import { ThreadService } from '../../src/ask/thread-service.mjs';
import { Claude } from '../../src/ask/claude.mjs';

function call(address, method, path, { body, headers = {} } = {}) {
  const url = new URL(path, address);
  return new Promise((resolve, reject) => {
    const req = request({ host: url.hostname, port: url.port, path: url.pathname, method, headers: { Host: url.host, ...headers } }, res => {
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
  const server = new AppServer({
    store: w.store, loader: w.loader, queue: w.queue,
    threads: new ThreadService({ store: w.store, claude: new Claude({ command: fake.command }) }), port: 0,
  });
  const address = await server.listen();
  const own = { Origin: address, 'Content-Type': 'application/json' };
  return { w, fake, server, address, own, done: async () => { await server.close(); fake.done(); w.done(); } };
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
    assert.deepEqual(Object.keys(page.diff), ['docs/guide.md']);
    assert.equal((await call(s.address, 'GET', '/api/reviews/github.com/octo/widgets/99')).status, 404);

    const added = await call(s.address, 'POST', `${base}/comments`, { body: { path: 'docs/guide.md', line: 3, side: 'RIGHT', body: 'Every item?' }, headers: s.own });
    assert.equal(added.status, 200);
    assert.equal(added.json.state.queue.members[0].comments[0].body, 'Every item?');
    const refused = await call(s.address, 'POST', `${base}/comments`, { body: { path: 'docs/guide.md', line: 99, side: 'RIGHT', body: 'x' }, headers: s.own });
    assert.equal(refused.status, 409);
    assert.match(refused.json.error, /not on a line of the diff/);
    assert.equal((await call(s.address, 'POST', `${base}/comments`, { body: { path: 'docs/guide.md', line: 'x', side: 'UP', body: 'x' }, headers: s.own })).status, 409);

    const edited = await call(s.address, 'PATCH', `${base}/comments/1`, { body: { body: 'Every single item?' }, headers: s.own });
    assert.equal(edited.json.state.queue.members[0].comments[0].body, 'Every single item?');

    s.w.routes({ extra: {
      'POST repos/octo/widgets/pulls/30/reviews': { id: 5, state: 'COMMENTED', html_url: 'https://github.com/octo/widgets/pull/30#r5' },
      'GET repos/octo/widgets/pulls/30/comments?per_page=100&page=1': [{ pull_request_review_id: 5 }],
    } });
    const state = (await call(s.address, 'GET', base)).json;
    const posted = await call(s.address, 'POST', `${base}/post`, { body: { pull: 30, action: 'comment', set: state.queue.members[0].sets.comment }, headers: s.own });
    assert.equal(posted.json.message, 'posted 1 comment as one review: https://github.com/octo/widgets/pull/30#r5');
    assert.equal(posted.json.state.queue.members[0].posted.length, 1);

    await call(s.address, 'POST', `${base}/comments`, { body: { path: 'docs/guide.md', line: 3, side: 'RIGHT', body: 'One more.' }, headers: s.own });
    const dropped = await call(s.address, 'DELETE', `${base}/comments/2`, { headers: { Origin: s.address } });
    assert.deepEqual(dropped.json.state.queue.members[0].comments, []);
  } finally {
    await s.done();
  }
});

test('the API asks Claude about a line and keeps the conversation', async () => {
  const s = await started();
  const base = '/api/reviews/github.com/octo/widgets/30';
  try {
    const started = await call(s.address, 'POST', `${base}/threads`, { body: { path: 'docs/guide.md', side: 'RIGHT', line: 3, text: 'Every?' }, headers: s.own });
    assert.equal(started.status, 200);
    assert.deepEqual(started.json.thread.messages.map(m => m.from), ['you', 'claude']);
    const replied = await call(s.address, 'POST', `${base}/threads/1/messages`, { body: { text: 'Sure?' }, headers: s.own });
    assert.equal(replied.json.thread.messages.length, 4);
    assert.equal((await call(s.address, 'GET', base)).json.threads.length, 1);
  } finally {
    await s.done();
  }
});
