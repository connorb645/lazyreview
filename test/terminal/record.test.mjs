import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync, writeFileSync, mkdirSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { request } from 'node:http';
import { workbench } from '../support/workbench.mjs';
import { fakeClaude } from '../support/fake-claude.mjs';
import { AppServer } from '../../src/server/app-server.mjs';
import { ThreadService } from '../../src/ask/thread-service.mjs';
import { Claude } from '../../src/ask/claude.mjs';
import { BranchLoader } from '../../src/loader/branch-loader.mjs';
import { PullRef } from '../../src/github/pull-ref.mjs';

const RECORD = join(dirname(fileURLToPath(import.meta.url)), '..', '..', 'terminal', 'testdata', 'record.json');
const STACK = 'github.com/octo/widgets/21';
const SINGLE = 'github.com/octo/widgets/30';

const thread = (id, path, line, login, body, extra = {}) => ({
  id, isResolved: false, isOutdated: false, path, line, startLine: null, originalLine: line, originalStartLine: null, diffSide: 'RIGHT', startDiffSide: null,
  subjectType: 'LINE', resolvedBy: null, ...extra,
  comments: { totalCount: 1, nodes: [{ databaseId: Number(id.slice(1)), author: { login, __typename: 'User' }, body, createdAt: '2026-10-15T10:00:00Z', url: `https://github.com/octo/widgets/pull/21#discussion_r${id.slice(1)}`, diffHunk: '' }] },
});

const TOKEN = 'record-token';

// Sends a request as the terminal app does, a change carrying the token and its fields as JSON.
function send(address, method, path, body = null) {
  const url = new URL(path, address);
  const headers = { Host: url.host, ...(method === 'GET' ? {} : { Authorization: `Bearer ${TOKEN}` }), ...(body ? { 'Content-Type': 'application/json' } : {}) };
  return new Promise((resolve, reject) => {
    const req = request({ host: url.hostname, port: url.port, method, path: url.pathname + url.search, headers }, res => {
      const chunks = [];
      res.on('data', c => chunks.push(c));
      res.on('end', () => resolve({ status: res.statusCode, body: JSON.parse(Buffer.concat(chunks).toString('utf8')) }));
    });
    req.on('error', reject);
    req.end(body ? JSON.stringify(body) : undefined);
  });
}

// The key the terminal app's tests look an answer up by: the method, the path, the query sorted by
// name, and a change's fields as JSON with its keys sorted.
function key(method, path, body = null) {
  const url = new URL(path, 'http://x');
  const query = new URLSearchParams([...url.searchParams].sort(([a], [b]) => a.localeCompare(b)));
  const fields = body ? ` ${JSON.stringify(Object.fromEntries(Object.entries(body).sort(([a], [b]) => a.localeCompare(b))))}` : '';
  return `${method} ${url.pathname}${[...query].length ? `?${query}` : ''}${fields}`;
}

const where = (base, given) => `${base}/where?${new URLSearchParams(given)}`;

// A stack with comments queued, a conversation, threads on GitHub and a written review, and a single pull request with Markdown.
async function recordAnswers() {
  const w = workbench();
  const fake = fakeClaude();
  try {
    w.repo.checkout('fix/wording');
    w.repo.commit('fix/wording', {
      'docs/guide.md': '# Guide\n\nThe catalog lists every item.\n\n- Browse\n- Search\n- Keep a basket\n\n## Steps\n\n1. Open the catalog.\n1. Pick an item.\n1. Add it to the basket.\n\n```swift\nlet basket = Basket(catalog: catalog)\n```\n',
      'docs/basket.md': '# Basket\n\nA basket keeps its prices until checkout.\n\n1. Add an item\n2. Remove an item\n',
    }, 'More guide');
    w.repo.checkout('main');
    const reviews = {
      21: { threads: [thread('T1', 'Sources/Catalog/MemoryCatalog.swift', 9, 'sam', 'Should this keep the order items were added in?')] },
      22: { threads: [thread('T2', 'Sources/Basket/Basket.swift', 7, 'sam', 'Could the search be passed in?', { isResolved: true, resolvedBy: { login: 'sam' } })] },
    };
    w.routes({ reviews });
    await w.loader.load({ links: [w.link(21), w.link(22)], context: true });
    await w.loader.load({ links: [w.link(30)] });
    let tick = 0;
    const now = () => new Date(Date.UTC(2026, 9, 15, 13, 0, tick++)).toISOString();
    const threads = new ThreadService({ store: w.store, claude: new Claude({ command: fake.command }), now });
    const server = new AppServer({ store: w.store, loader: w.loader, branches: new BranchLoader({ store: w.store }), queue: w.queue, threads, token: TOKEN, port: 0 });
    const address = await server.listen();
    try {
      const stack = new PullRef('github.com', 'octo/widgets', 21);
      const build = w.store.pageBuild(stack);
      await w.queue.add(stack, { build, path: 'Sources/Catalog/MemoryCatalog.swift', start_line: 8, line: 10, side: 'RIGHT', body: 'Keep the order items came in.' });
      await w.queue.add(stack, { build, path: 'Sources/Basket/Basket.swift', start_line: 9, start_side: 'LEFT', line: 11, side: 'RIGHT', body: 'Why one?' });
      await threads.start(stack, { path: 'Sources/Basket/Basket.swift', side: 'RIGHT', line: 7, text: 'Could this be passed in?', build });
      const single = new PullRef('github.com', 'octo/widgets', 30);
      await w.queue.add(single, { build: w.store.pageBuild(single), path: 'docs/guide.md', line: 3, side: 'RIGHT', body: 'Every item, or every item shown?' });
      await w.store.change([stack], async () => ({
        files: [[w.store.home.review(stack).written, `---\nstack: 21, 22\nhead: ${w.repo.sha('feature/basket')}\ntitle: Search, then a basket\n---\n## What this changes\n\n#21 adds a search over the catalog in \`Search.swift:4\`. #22 gives the basket a search of its own, in \`Basket.swift:7\`. [Likely]\n\n## Risks\n\n\`MemoryCatalog.swift:9\` appends without checking for duplicates. [Guessing]\n`]],
        result: null,
      }));

      const paths = ['/api/reviews'];
      for (const id of [STACK, SINGLE]) {
        const base = `/api/reviews/${id}`;
        const page = (await send(address, 'GET', `${base}/page`)).body;
        paths.push(base, `${base}/page`);
        for (const f of [...page.model.files, ...page.model.contextFiles]) paths.push(`${base}/file?${new URLSearchParams({ path: f.path, build: page.build })}`);
      }
      const base = `/api/reviews/${STACK}`;
      paths.push(
        where(base, { path: 'Sources/Basket/Basket.swift', side: 'RIGHT', line: 3, build }),
        where(base, { path: 'Sources/Catalog/MemoryCatalog.swift', side: 'RIGHT', start_line: 7, start_side: 'RIGHT', line: 9, build }),
        where(base, { path: 'Sources/Catalog/Catalog.swift', side: 'RIGHT', line: 1, build }),
        where(base, { path: 'Sources/Basket/Basket.swift', side: 'RIGHT', line: 7, build }),
      );
      const answers = {};
      for (const path of paths) {
        const answer = await send(address, 'GET', path);
        assert.equal(answer.status, 200, path);
        answers[key('GET', path)] = answer;
      }
      // The changes the terminal app makes, in turn, each answered as the state then stands.
      const state = (await send(address, 'GET', base)).body;
      const version = n => state.queue.members.flatMap(m => m.comments).find(c => c.id === n).version;
      const changes = [
        ['POST', `${base}/comments`, { path: 'Sources/Basket/Basket.swift', side: 'RIGHT', line: 3, body: 'Name the search for what it finds.', build }],
        ['POST', `${base}/comments`, { path: 'Sources/Catalog/MemoryCatalog.swift', side: 'RIGHT', start_line: 7, start_side: 'RIGHT', line: 9, body: 'One add is enough.', build }],
        ['POST', `${base}/comments`, { path: 'Sources/Basket/Basket.swift', side: 'RIGHT', line: 7, body: 'It returns early when the list is empty.', build }],
        ['PATCH', `${base}/comments/1`, { body: 'Keep the order the items came in.', version: 'stale1', build }],
        ['PATCH', `${base}/comments/1`, { body: 'Keep the order the items came in.', version: version(1), build }],
        ['DELETE', `${base}/comments/2?${new URLSearchParams({ build, version: version(2) })}`, null],
        ['POST', `${base}/threads`, { path: 'Sources/Basket/Basket.swift', side: 'RIGHT', line: 3, text: 'Why keep a search here?', build }],
        ['POST', `${base}/threads/1/messages`, { text: 'Could it be passed in?', build }],
      ];
      const one = `/api/reviews/${SINGLE}`;
      const oneState = (await send(address, 'GET', one)).body;
      const oneBuild = w.store.pageBuild(single);
      w.routes({ reviews, extra: {
        'POST repos/octo/widgets/pulls/30/reviews': { id: 8, state: 'APPROVED', html_url: 'https://github.com/octo/widgets/pull/30#pullrequestreview-8' },
        'GET repos/octo/widgets/pulls/30/comments?per_page=100&page=1': [{ pull_request_review_id: 8 }],
      } });
      changes.push(
        ['POST', `${one}/post`, { pull: 30, action: 'comment', set: 'stale0000000', build: oneBuild }],
        ['POST', `${one}/post`, { pull: 30, action: 'approve-with-comments', set: oneState.queue.members[0].sets['approve-with-comments'].set, build: oneBuild }],
      );
      for (const [method, path, body] of changes) answers[key(method, path, body)] = await send(address, method, path, body);
      return answers;
    } finally {
      await server.close();
    }
  } finally {
    fake.done();
    w.done();
  }
}

// The first key whose answer differs, or null.
function firstDifference(recorded, now) {
  for (const k of new Set([...Object.keys(recorded), ...Object.keys(now)])) {
    if (JSON.stringify(recorded[k]) !== JSON.stringify(now[k])) return k;
  }
  return null;
}

test('the server answers the terminal app is tested against are the ones it gives now', async () => {
  const now = await recordAnswers();
  if (process.env.REVIEW_DESK_RECORD) {
    mkdirSync(dirname(RECORD), { recursive: true });
    writeFileSync(RECORD, JSON.stringify(now, null, 1) + '\n');
    return;
  }
  let recorded;
  try {
    recorded = JSON.parse(readFileSync(RECORD, 'utf8'));
  } catch (error) {
    assert.fail(`could not read the record ${RECORD}: ${error.message}; write it with REVIEW_DESK_RECORD=1 npm test`);
  }
  const differs = firstDifference(recorded, now);
  assert.equal(differs, null, `the server now answers ${differs} differently from the record the terminal app's tests read; write it again with REVIEW_DESK_RECORD=1 npm test, then run the Go tests`);
});
