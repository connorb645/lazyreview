/**
 * Everything a test of loading and reviewing needs, wired together: a
 * throwaway repository holding a stack of two pull requests and a single one,
 * a fake gh answering for them, and a data home in a temporary folder.
 *
 * workbench() returns {repo, gh, home, store, loader, queue, routes, pulls,
 * env, done}. `routes(changes)` answers GitHub as the repository now stands,
 * with `changes` to the pull requests or extra routes; `env` is the
 * environment a child process needs to use the same fake gh and data home.
 * This process's own PATH points at the fake gh until done() is called.
 *
 * The repository's pull requests:
 *   #21  feature/search on main         adds Search, edits MemoryCatalog
 *   #22  feature/basket on #21's branch  edits Basket and cart.ts
 *   #30  fix/wording on main             edits docs/guide.md
 */
import { mkdtempSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { throwawayRepo } from './throwaway-repo.mjs';
import { fakeGh } from './fake-gh.mjs';
import { githubRoutes } from './github-routes.mjs';
import { DataHome } from '../../src/store/data-home.mjs';
import { ReviewStore } from '../../src/store/review-store.mjs';
import { Clones } from '../../src/clone/clones.mjs';
import { Loader } from '../../src/loader/loader.mjs';
import { QueueService } from '../../src/queue/queue-service.mjs';

export const PULLS = [
  { number: 21, base: 'main', head: 'feature/search', title: 'Add search to the catalog', author: 'sam', body: 'Adds a search over the catalog by name.' },
  { number: 22, base: 'feature/search', head: 'feature/basket', title: 'Add a basket that uses search', author: 'sam' },
  { number: 30, base: 'main', head: 'fix/wording', title: 'Fix the guide wording', author: 'alex' },
];

export function workbench() {
  const repo = throwawayRepo();
  repo.commit('main', {
    'Sources/Catalog/Catalog.swift': 'protocol Catalog {\n    func items() -> [Item]\n}\n\nstruct Item {\n    let name: String\n}\n',
    'Sources/Catalog/MemoryCatalog.swift': 'final class MemoryCatalog: Catalog {\n    private var stored: [Item] = []\n\n    func items() -> [Item] {\n        stored\n    }\n}\n',
    'Sources/Basket/Basket.swift': 'final class Basket {\n    let catalog: Catalog\n\n    init(catalog: Catalog) {\n        self.catalog = catalog\n    }\n\n    func total() -> Int {\n        0\n    }\n}\n',
    'web/src/cart.ts': 'export function cartTotal(prices: number[]): number {\n  return prices.reduce((a, b) => a + b, 0);\n}\n',
    'docs/guide.md': '# Guide\n\nThe catalog lists items.\n\n- Browse\n- Search\n',
  }, 'base');
  repo.commit('feature/search', {
    'Sources/Catalog/Search.swift': 'struct Search {\n    let catalog: Catalog\n\n    func find(_ text: String) -> [Item] {\n        catalog.items().filter { $0.name.contains(text) }\n    }\n}\n',
    'Sources/Catalog/MemoryCatalog.swift': 'final class MemoryCatalog: Catalog {\n    private var stored: [Item] = []\n\n    func items() -> [Item] {\n        stored\n    }\n\n    func add(_ item: Item) {\n        stored.append(item)\n    }\n}\n',
  }, 'Add search');
  repo.commit('feature/basket', {
    'Sources/Basket/Basket.swift': 'final class Basket {\n    let catalog: Catalog\n    let search: Search\n\n    init(catalog: Catalog) {\n        self.catalog = catalog\n        self.search = Search(catalog: catalog)\n    }\n\n    func total() -> Int {\n        1\n    }\n}\n',
    'web/src/cart.ts': 'export function cartTotal(prices: number[]): number {\n  if (!prices.length) return 0;\n  return prices.reduce((a, b) => a + b, 0);\n}\n',
  }, 'Add basket');
  repo.checkout('main');
  repo.commit('fix/wording', { 'docs/guide.md': '# Guide\n\nThe catalog lists every item.\n\n- Browse\n- Search\n' }, 'Fix wording');
  repo.checkout('main');
  const published = { 21: 'feature/search', 22: 'feature/basket', 30: 'fix/wording' };
  repo.publish(published);

  const gh = fakeGh();
  const before = { PATH: process.env.PATH, FAKE_GH_ROUTES: process.env.FAKE_GH_ROUTES, FAKE_GH_LOG: process.env.FAKE_GH_LOG };
  Object.assign(process.env, { PATH: `${gh.bin}:${process.env.PATH}`, ...gh.extra });
  const routes = (changes = {}) => {
    repo.publish(published);
    gh.routes(githubRoutes({ repo, pulls: PULLS.map(p => ({ ...p, ...(changes[p.number] || {}) })), reviews: changes.reviews, extra: changes.extra }));
  };
  routes();

  const root = mkdtempSync(join(tmpdir(), 'lazyreview-home-'));
  const home = new DataHome(root);
  const store = new ReviewStore(home);
  let tick = 0;
  const now = () => new Date(Date.UTC(2026, 9, 15, 12, 0, tick++)).toISOString();
  return {
    repo, gh, home, store, routes,
    loader: new Loader({ store, clones: new Clones(home), now }),
    queue: new QueueService({ store, now }),
    env: { ...gh.env, LAZYREVIEW_HOME: root },
    link: n => `https://github.com/octo/widgets/pull/${n}`,
    done() {
      for (const [key, value] of Object.entries(before)) {
        if (value === undefined) delete process.env[key]; else process.env[key] = value;
      }
      gh.done();
      repo.done();
      rmSync(root, { recursive: true, force: true });
    },
  };
}
