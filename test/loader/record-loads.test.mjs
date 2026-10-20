import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync, writeFileSync, mkdtempSync, rmSync, readdirSync, statSync, existsSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, dirname, relative } from 'node:path';
import { fileURLToPath } from 'node:url';
import { throwawayRepo } from '../support/throwaway-repo.mjs';
import { fakeGh } from '../support/fake-gh.mjs';
import { githubRoutes } from '../support/github-routes.mjs';
import { DataHome } from '../../src/store/data-home.mjs';
import { ReviewStore } from '../../src/store/review-store.mjs';
import { Loader } from '../../src/loader/loader.mjs';
import { PullRef } from '../../src/models/pull-ref.mjs';

// Made-up loads of pull requests, one after another into one data home, as
// the commits that make the repository, the fake GitHub's routes at each
// load, the changes made between loads, and what each load answered and
// left in the data home, layouts aside, which the Go loader is tested
// against. The origin's path is written as STAND_IN_ORIGIN.
const RECORD = join(dirname(fileURLToPath(import.meta.url)), '..', '..', 'terminal', 'testdata', 'loads.json');
const STAND_IN_ORIGIN = '/stand-in/origin.git';
const LINK = n => `https://github.com/octo/widgets/pull/${n}`;

const text = rows => rows.join('\n') + '\n';
const CATALOG = text(['protocol Catalog {', '    func items() -> [Item]', '}', '', 'struct Item {', '    let name: String', '}']);
const MEMORY = text(['final class MemoryCatalog: Catalog {', '    private var stored: [Item] = []', '', '    func items() -> [Item] {', '        stored', '    }', '}']);
const MEMORY_ADD = text(['final class MemoryCatalog: Catalog {', '    private var stored: [Item] = []', '', '    func items() -> [Item] {', '        stored', '    }', '', '    func add(_ item: Item) {', '        stored.append(item)', '    }', '}']);
const BASKET = text(['final class Basket {', '    let catalog: Catalog', '', '    init(catalog: Catalog) {', '        self.catalog = catalog', '    }', '}']);
const BASKET_SEARCH = text(['final class Basket {', '    let catalog: Catalog', '    let search: Search', '', '    init(catalog: Catalog) {', '        self.catalog = catalog', '        self.search = Search(catalog: catalog)', '    }', '}']);
const SEARCH = text(['struct Search {', '    let catalog: Catalog', '}']);

const PULLS = [
  { number: 21, base: 'main', head: 'feature/search', title: 'Add search', author: 'sam', body: 'Adds a search.\n\n'.repeat(300) },
  { number: 22, base: 'feature/search', head: 'feature/basket', title: 'Use search in the basket', author: 'sam' },
  { number: 30, base: 'main', head: 'fix/wording', title: 'Fix the wording', author: 'alex' },
];

const thread = (id, path, line, extra = {}) => ({
  id, isResolved: false, isOutdated: false, path, line, startLine: null, originalLine: line, originalStartLine: null, diffSide: 'RIGHT', startDiffSide: null,
  subjectType: 'LINE', resolvedBy: null, ...extra,
  comments: { totalCount: 2, nodes: [{ databaseId: Number(id.slice(1)), author: { login: 'sam', __typename: 'User' }, body: 'Why?', createdAt: '2026-10-15T10:00:00Z', url: `${LINK(21)}#discussion_r${id.slice(1)}`, diffHunk: '@@ -1,2 +1,3 @@\n a\n b\n+c\n d\n e' }] },
});
const REVIEWS = {
  21: { threads: [thread('T1', 'Sources/Catalog/MemoryCatalog.swift', 9), thread('T3', 'Sources/Catalog/Catalog.swift', null, { isOutdated: true })] },
  22: { threads: [thread('T2', 'Sources/Basket/Basket.swift', 3, { isResolved: true, resolvedBy: { login: 'sam' }, startLine: 2, startDiffSide: 'RIGHT' })] },
};

const comment = (id, path, line) => ({ id, path, line, side: 'RIGHT', body: `Comment ${id}`, code: '', shown: { path, line, side: 'RIGHT' } });

async function recorded() {
  const repo = throwawayRepo();
  const gh = fakeGh();
  const root = mkdtempSync(join(tmpdir(), 'lazyreview-home-'));
  const before = { PATH: process.env.PATH, FAKE_GH_ROUTES: process.env.FAKE_GH_ROUTES, FAKE_GH_LOG: process.env.FAKE_GH_LOG };
  Object.assign(process.env, { PATH: `${gh.bin}:${process.env.PATH}`, ...gh.extra });
  const script = [];
  const play = op => {
    script.push(op);
    if (op.commit) repo.commit(...op.commit);
    else if (op.checkout) repo.checkout(op.checkout);
  };
  const store = new ReviewStore(new DataHome(root));
  let tick = 0;
  const loader = new Loader({ store, now: () => new Date(Date.UTC(2026, 9, 15, 12, 0, tick++)).toISOString() });
  const steps = [];
  const files = () => {
    const out = {};
    const walk = dir => {
      for (const name of readdirSync(dir)) {
        const path = join(dir, name);
        if (statSync(path).isDirectory()) { if (name !== 'clones') walk(path); continue; }
        const rel = relative(root, path);
        if (rel.endsWith('.lock')) continue;
        let content = readFileSync(path, 'utf8');
        if (rel.endsWith('page.json')) { const { layout, ...rest } = JSON.parse(content); content = JSON.stringify(rest); }
        out[rel] = content;
      }
    };
    if (existsSync(join(root, 'reviews'))) walk(join(root, 'reviews'));
    return out;
  };
  const routesNow = (changes = {}) => {
    repo.publish({ 21: 'feature/search', 22: 'feature/basket', 30: 'fix/wording' });
    const routes = githubRoutes({ repo, pulls: PULLS.map(p => ({ ...p, ...(changes[p.number] || {}) })), reviews: REVIEWS, extra: changes.extra });
    gh.routes(routes);
    return JSON.parse(JSON.stringify(routes).replaceAll(repo.origin, STAND_IN_ORIGIN));
  };
  const load = async (name, given, changes) => {
    const routes = routesNow(changes);
    const request = { links: given.links, ...(given.context !== undefined ? { context: given.context } : {}), ...(given.config !== undefined ? { config: given.config } : {}), ...(given.account !== undefined ? { account: given.account } : {}) };
    let answer;
    try {
      const loaded = given.refresh ? await loader.refresh(PullRef.parse(given.refresh)) : await loader.load(request);
      answer = { ref: loaded.ref.id, counts: loaded.counts, notes: loaded.notes };
    } catch (error) {
      answer = { error: error.message };
    }
    steps.push({ name, script: script.splice(0), routes, request: given.refresh ? { refresh: given.refresh } : request, answer, files: files() });
  };
  const queue = (id, pull, comments) => {
    const path = join(root, 'reviews', 'github.com', 'octo', 'widgets', String(id), 'review.json');
    const review = JSON.parse(readFileSync(path, 'utf8'));
    review.members.find(m => m.pull === pull).comments.push(...comments);
    const written = JSON.stringify(review, null, 2) + '\n';
    writeFileSync(path, written);
    script.push({ write: [relative(root, path), written] });
  };
  try {
    play({ commit: ['main', { 'Sources/Catalog/Catalog.swift': CATALOG, 'Sources/Catalog/MemoryCatalog.swift': MEMORY, 'Sources/Basket/Basket.swift': BASKET, 'docs/guide.md': '# Guide\n' }, 'base'] });
    play({ commit: ['feature/search', { 'Sources/Catalog/MemoryCatalog.swift': MEMORY_ADD, 'Sources/Catalog/Search.swift': SEARCH }, 'search'] });
    play({ commit: ['feature/basket', { 'Sources/Basket/Basket.swift': BASKET_SEARCH }, 'basket'] });
    play({ checkout: 'main' });
    play({ commit: ['fix/wording', { 'docs/guide.md': '# The guide\n' }, 'wording'] });
    play({ checkout: 'main' });

    await load('no links', { links: [] });
    await load('two repositories', { links: [LINK(21), 'https://github.com/octo/gadgets/pull/2'] });
    await load('a pull request named twice', { links: [LINK(21), LINK(21)] });
    await load('not a stack', { links: [LINK(21), LINK(30)] });
    await load('the lower pull request alone', { links: [LINK(21)], account: 'sam' });
    queue(21, 21, [comment(1, 'Sources/Catalog/MemoryCatalog.swift', 9)]);
    await load('the upper pull request alone, with a config', { links: [LINK(22)], config: { components: [{ id: 'basket', title: 'Basket', paths: ['Sources/Basket/'] }] } });
    queue(22, 22, [comment(1, 'Sources/Basket/Basket.swift', 3)]);
    await load('both as a stack, the upper review moving into it', { links: [LINK(21), LINK(22)], context: true });
    play({ checkout: 'feature/basket' });
    play({ commit: ['feature/basket', { 'Sources/Basket/Basket.swift': BASKET_SEARCH.replace('let search: Search', 'let search: Search\n    let size = 1') }, 'basket again'] });
    play({ checkout: 'main' });
    await load('the stack again, its upper head moved', { links: [LINK(21), LINK(22)] });
    await load('the stack refreshed with its lower pull request merged', { refresh: 'github.com/octo/widgets/21' }, { 21: { state: 'closed', merged: true } });
    await load('a single pull request, its threads refused', { links: [LINK(30)] }, { extra: { 'POST graphql octo/widgets#30': { $fail: 'gh: Something went wrong' } } });
    await load('a moved review refreshed', { refresh: 'github.com/octo/widgets/22' });
    return { steps };
  } finally {
    for (const [key, value] of Object.entries(before)) {
      if (value === undefined) delete process.env[key]; else process.env[key] = value;
    }
    gh.done();
    repo.done();
    rmSync(root, { recursive: true, force: true });
  }
}

test('the loads the Go loader is tested against load as Node loads them now', async () => {
  const now = JSON.parse(JSON.stringify(await recorded()));
  if (process.env.LAZYREVIEW_RECORD) {
    writeFileSync(RECORD, JSON.stringify(now, null, 1) + '\n');
    return;
  }
  assert.deepEqual(JSON.parse(readFileSync(RECORD, 'utf8')), now, `${RECORD} is not what Node loads now; write it again with LAZYREVIEW_RECORD=1 npm test, then run the Go tests`);
});
