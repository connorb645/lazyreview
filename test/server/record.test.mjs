import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync, writeFileSync, mkdirSync, rmSync, readdirSync, existsSync, realpathSync } from 'node:fs';
import { join, dirname, relative } from 'node:path';
import { fileURLToPath } from 'node:url';
import { request } from 'node:http';
import { workbench } from '../support/workbench.mjs';
import { fakeClaude } from '../support/fake-claude.mjs';
import { AppServer } from '../../src/server/app-server.mjs';
import { ThreadService } from '../../src/ask/thread-service.mjs';
import { Claude } from '../../src/commands/claude/claude.mjs';
import { BranchLoader } from '../../src/loader/branch-loader.mjs';
import { PullRef } from '../../src/models/pull-ref.mjs';
import { OpenPulls } from '../../src/home/open-pulls.mjs';
import { HiddenRepositories } from '../../src/home/hidden-repositories.mjs';
import { SETTINGS, homeRoutes, accountsConfig } from '../support/home-routes.mjs';

const RECORD = join(dirname(fileURLToPath(import.meta.url)), '..', '..', 'terminal', 'testdata', 'record.json');
// The data home the answers were recorded from, before any change, and the fake GitHub's routes while they were: the Go
// code that answers in the server's place reads them, and gives the same answers.
const HOME = join(dirname(RECORD), 'home');
const ROUTES = join(dirname(RECORD), 'github.json');
const ASKED = join(dirname(RECORD), 'asked.json');
const HOMES = join(dirname(RECORD), 'homes.json');
const STAND_IN_HOME = '/home/reviewer/.lazyreview';
const STAND_IN_ORIGIN = '/stand-in/origin.git';
const STACK = 'github.com/octo/widgets/21';
const SINGLE = 'github.com/octo/widgets/30';

const thread = (id, path, line, login, body, extra = {}) => ({
  id, isResolved: false, isOutdated: false, path, line, startLine: null, originalLine: line, originalStartLine: null, diffSide: 'RIGHT', startDiffSide: null,
  subjectType: 'LINE', resolvedBy: null, ...extra,
  comments: { totalCount: 1, nodes: [{ databaseId: Number(id.slice(1)), author: { login, __typename: 'User' }, body, createdAt: '2026-10-15T10:00:00Z', url: `https://github.com/octo/widgets/pull/21#discussion_r${id.slice(1)}`, diffHunk: '' }] },
});

const TOKEN = 'record-token';
const UNREACHABLE = { $fail: 'error connecting to api.github.com\ncheck your internet connection or https://githubstatus.com' };
// The config file the answers name, in place of the throwaway one, so the same review gives the same record.
const STAND_IN_CONFIG = '/home/reviewer/Library/Application Support/lazyreview/config.yml';

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

// A stack with comments queued, a conversation, threads on GitHub and a written review, and a single pull request with Markdown;
// and the home screen's open pull requests as GitHub gives them, as it fails to, and with no accounts set, with the stack
// loaded and the single pull request not yet, as a load from the home screen finds it.
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
    let tick = 0;
    const now = () => new Date(Date.UTC(2026, 9, 15, 13, 0, tick++)).toISOString();
    let sessions = 0;
    const session = () => `00000000-0000-4000-8000-${String(++sessions).padStart(12, '0')}`;
    const threads = new ThreadService({ store: w.store, claude: new Claude({ command: fake.command }), session, now });
    const server = new AppServer({
      store: w.store, loader: w.loader, branches: new BranchLoader({ store: w.store }), queue: w.queue, threads, pulls: new OpenPulls({ store: w.store, config: w.config }),
      hidden: new HiddenRepositories({ config: w.config }), token: TOKEN, port: 0,
    });
    const address = await server.listen();
    try {
      const answers = {};
      // What each home screen list was read from, and each change to the hidden repositories made to, for the Go replay.
      const homes = { lists: [], changes: [] };
      const routesNow = () => JSON.parse(readFileSync(w.gh.extra.FAKE_GH_ROUTES, 'utf8').replaceAll(w.repo.origin, STAND_IN_ORIGIN));
      const configNow = () => (existsSync(w.configFile) ? readFileSync(w.configFile, 'utf8') : null);
      const reviewsNow = () => Object.keys(filesUnder(w.home.root, 'reviews')).filter(p => p.endsWith('/review.json')).map(p => p.slice('reviews/'.length, -'/review.json'.length)).sort();
      const home = async (variant, settings, extra) => {
        if (settings) writeFileSync(w.configFile, accountsConfig(settings.accounts, settings.hidden));
        else rmSync(w.configFile, { force: true });
        w.routes({ reviews, extra });
        const answer = await send(address, 'GET', '/api/pulls');
        assert.equal(answer.status, 200, variant);
        answers[`GET /api/pulls${variant}`] = { ...answer, body: { ...answer.body, config: answer.body.config.replace(w.configFile, STAND_IN_CONFIG) } };
        homes.lists.push({ key: `GET /api/pulls${variant}`, config: configNow(), routes: routesNow(), reviews: reviewsNow() });
      };
      await home('', SETTINGS, homeRoutes());
      await home('#one repository', { accounts: [SETTINGS.accounts[1]] }, homeRoutes());
      await home('#none', { accounts: [{ login: 'reviewer' }] }, { 'token-reviewer *': { total_count: 0, items: [] } });
      await home('#one unreachable', SETTINGS, { ...homeRoutes(), 'token-reviewer-home *': UNREACHABLE });
      await home('#unreachable', SETTINGS, { 'token-reviewer *': UNREACHABLE, 'token-reviewer-home *': UNREACHABLE });
      await home('#no settings', null, {});
      await home('#hidden', { ...SETTINGS, hidden: ['octo/gadgets', 'reviewer-home/*'] }, homeRoutes());
      await home('#all hidden', { ...SETTINGS, hidden: ['octo/*', 'reviewer-home/notes'] }, homeRoutes());
      // The changes to the hidden repositories the home screen makes, each from the config the one before left, and one the
      // file's own style refuses.
      const names = ['octo/gadgets', 'octo/widgets', 'reviewer-home/notes'];
      const hiding = async (config, changes) => {
        writeFileSync(w.configFile, config);
        for (const body of changes) {
          const before = configNow();
          const answer = await send(address, 'POST', '/api/hidden', body);
          answers[key('POST', '/api/hidden', body)] = JSON.parse(JSON.stringify(answer).replaceAll(w.configFile, STAND_IN_CONFIG));
          homes.changes.push({ key: key('POST', '/api/hidden', body), before, after: configNow() });
        }
      };
      await hiding(accountsConfig(SETTINGS.accounts), [
        { hide: 'octo/widgets', repos: names }, { show: ['octo/widgets'], repos: names }, { hide: 'octo/*', repos: names }, { show: ['octo/*'], repos: names },
      ]);
      await hiding(accountsConfig(SETTINGS.accounts, ['octo/gadgets', 'reviewer-home/*']), [{ show: ['reviewer-home/*'], repos: names }]);
      await hiding(`github: {accounts: ${JSON.stringify(SETTINGS.accounts)}}\n`, [{ hide: 'reviewer-home/notes', repos: names }]);
      w.routes({ reviews });
      await w.loader.load({ links: [w.link(30)] });
      const stack = new PullRef('github.com', 'octo/widgets', 21);
      const build = w.store.pageBuild(stack);
      await w.queue.add(stack, { build, path: 'Sources/Catalog/MemoryCatalog.swift', start_line: 8, line: 10, side: 'RIGHT', body: 'Keep the order items came in.' });
      await w.queue.add(stack, { build, path: 'Sources/Basket/Basket.swift', start_line: 9, start_side: 'LEFT', line: 11, side: 'RIGHT', body: 'Why one?' });
      fake.answer({ text: 'It could: the basket only reads from it.', steps: [
        { tool: 'Grep', input: { pattern: 'Search(' } },
        { tool: 'Bash', input: { command: 'gh pr comment 22 --body "Pass the search in."' }, refused: 'lazyreview refused this: gh may only read, one command at a time.' },
      ] });
      await threads.start(stack, { path: 'Sources/Basket/Basket.swift', side: 'RIGHT', line: 7, text: 'Could this be passed in?', build });
      fake.answer('It returns early when the list is empty.');
      const single = new PullRef('github.com', 'octo/widgets', 30);
      await w.queue.add(single, { build: w.store.pageBuild(single), path: 'docs/guide.md', line: 3, side: 'RIGHT', body: 'Every item, or every item shown?' });
      await w.store.change([stack], async () => ({
        files: [[w.store.home.review(stack).written, `---\nstack: 21, 22\nhead: ${w.repo.sha('feature/basket')}\ntitle: Search, then a basket\n---\n## What this changes\n\n#21 adds a search over the catalog in \`Search.swift:4\`. #22 gives the basket a search of its own, in \`Basket.swift:7\`. [Likely]\n\n## Risks\n\n\`MemoryCatalog.swift:9\` appends without checking for duplicates. [Guessing]\n`]],
        result: null,
      }));

      const dataHome = filesUnder(w.home.root, 'reviews');
      const paths = [];
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
        ['POST', `${base}/comments`, { path: 'Sources/Basket/Basket.swift', side: 'RIGHT', line: 7, body: 'It could: the basket only reads from it.', build }],
        ['PATCH', `${base}/comments/1`, { body: 'Keep the order the items came in.', version: 'stale1', build }],
        ['PATCH', `${base}/comments/1`, { body: 'Keep the order the items came in.', version: version(1), build }],
        ['DELETE', `${base}/comments/2?${new URLSearchParams({ build, version: version(2) })}`, null],
        ['POST', `${base}/threads`, { path: 'Sources/Basket/Basket.swift', side: 'RIGHT', line: 3, text: 'Why keep a search here?', build }],
        ['POST', `${base}/threads/1/messages`, { text: 'Could it be passed in?', build }],
        // A conversation left waiting, as after the server stopped mid-answer, then stopped.
        () => w.store.change([stack], async () => ({
          files: [[w.store.home.review(stack).thread(9), JSON.stringify({
            id: 9, path: 'Sources/Basket/Basket.swift', side: 'RIGHT', line: 3, created_at: now(), session: session(),
            messages: [{ from: 'you', text: 'Does anything test this?', at: now() }], working: { steps: [{ words: 'Read Sources/Basket/Basket.swift' }, { words: 'Ran swift test' }] },
          }, null, 2) + '\n']],
          result: null,
        })),
        ['POST', `${base}/threads/9/stop`, {}],
      ];
      const one = `/api/reviews/${SINGLE}`;
      const oneState = (await send(address, 'GET', one)).body;
      const oneBuild = w.store.pageBuild(single);
      w.routes({ reviews, extra: {
        'POST repos/octo/widgets/pulls/30/reviews': { id: 8, state: 'APPROVED', html_url: 'https://github.com/octo/widgets/pull/30#pullrequestreview-8' },
        'GET repos/octo/widgets/pulls/30/comments?per_page=100&page=1': [{ pull_request_review_id: 8 }],
      } });
      const routes = JSON.parse(readFileSync(w.gh.extra.FAKE_GH_ROUTES, 'utf8').replaceAll(w.repo.origin, STAND_IN_ORIGIN));
      changes.push(
        ['POST', `${one}/post`, { pull: 30, action: 'comment', set: 'stale0000000', build: oneBuild }],
        ['POST', `${one}/post`, { pull: 30, action: 'approve-with-comments', set: oneState.queue.members[0].sets['approve-with-comments'].set, build: oneBuild }],
        ['POST', `${one}/comments`, { path: 'docs/guide.md', side: 'RIGHT', start_line: 3, start_side: 'LEFT', line: 3, body: 'Why every item?', build: oneBuild }],
      );
      for (const given of [{ path: 'docs/guide.md', side: 'RIGHT', line: 3 }, { path: 'docs/guide.md', side: 'RIGHT', start_line: 3, start_side: 'LEFT', line: 3 }]) {
        const path = where(one, { ...given, build: oneBuild });
        answers[key('GET', path)] = await send(address, 'GET', path);
      }
      changes.push(
        ['POST', `${one}/refresh`, {}],
        ['POST', '/api/load', { links: [w.link(30)], account: 'reviewer' }],
        ['POST', '/api/load', { links: ['https://github.com/octo/gadgets/pull/7'], account: 'reviewer' }],
      );
      for (const change of changes) {
        if (typeof change === 'function') {
          await change();
          continue;
        }
        const [method, path, body] = change;
        answers[key(method, path, body)] = await send(address, method, path, body);
      }
      // Each question the stand-in Claude was asked, in turn, with the data home it ran under named as the stand-in's.
      const roots = [realpathSync(w.home.root), w.home.root].sort((a, b) => b.length - a.length);
      const asked = fake.asked().map(c => ({
        system: c.args[c.args.indexOf('--append-system-prompt') + 1],
        prompt: roots.reduce((text, root) => text.replaceAll(root, STAND_IN_HOME), c.prompt),
      }));
      return { answers, home: dataHome, routes, asked, homes };
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

// Every file under `folder` of `root`, by its path from root, as text.
function filesUnder(root, folder) {
  const out = {};
  const walk = path => {
    for (const entry of readdirSync(join(root, path), { withFileTypes: true })) {
      const inner = join(path, entry.name);
      if (entry.name.startsWith('.')) continue;
      if (entry.isDirectory()) walk(inner);
      else out[inner] = readFileSync(join(root, inner), 'utf8');
    }
  };
  walk(folder);
  return out;
}

test('the server answers the terminal app is tested against are the ones it gives now', async () => {
  const { answers: now, home, routes, asked, homes } = await recordAnswers();
  if (process.env.LAZYREVIEW_RECORD) {
    mkdirSync(dirname(RECORD), { recursive: true });
    writeFileSync(RECORD, JSON.stringify(now, null, 1) + '\n');
    for (const [path, text] of Object.entries(home)) {
      mkdirSync(dirname(join(HOME, path)), { recursive: true });
      writeFileSync(join(HOME, path), text);
    }
    writeFileSync(ROUTES, JSON.stringify(routes, null, 1) + '\n');
    writeFileSync(ASKED, JSON.stringify(asked, null, 1) + '\n');
    writeFileSync(HOMES, JSON.stringify(homes, null, 1) + '\n');
    return;
  }
  for (const [path, text] of Object.entries(home)) {
    assert.ok(existsSync(join(HOME, path)) && readFileSync(join(HOME, path), 'utf8') === text,
      `${relative(dirname(RECORD), join(HOME, path))} is not the data home the answers come from; write it again with LAZYREVIEW_RECORD=1 npm test`);
  }
  assert.deepEqual(JSON.parse(readFileSync(ROUTES, 'utf8')), routes, `${ROUTES} is not the routes the answers come from; write it again with LAZYREVIEW_RECORD=1 npm test`);
  assert.deepEqual(JSON.parse(readFileSync(ASKED, 'utf8')), asked, `${ASKED} is not the questions Claude was asked; write it again with LAZYREVIEW_RECORD=1 npm test`);
  assert.deepEqual(JSON.parse(readFileSync(HOMES, 'utf8')), homes, `${HOMES} is not what the home screen was read from; write it again with LAZYREVIEW_RECORD=1 npm test`);
  let recorded;
  try {
    recorded = JSON.parse(readFileSync(RECORD, 'utf8'));
  } catch (error) {
    assert.fail(`could not read the record ${RECORD}: ${error.message}; write it with LAZYREVIEW_RECORD=1 npm test`);
  }
  const differs = firstDifference(recorded, now);
  assert.equal(differs, null, `the server now answers ${differs} differently from the record the terminal app's tests read; write it again with LAZYREVIEW_RECORD=1 npm test, then run the Go tests`);
});
