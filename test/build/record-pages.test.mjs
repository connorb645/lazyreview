import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync, writeFileSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { throwawayRepo } from '../support/throwaway-repo.mjs';
import { Git } from '../../src/commands/git/git.mjs';
import { readConfig } from '../../src/build/config/config.mjs';
import { buildPage } from '../../src/build/page-builder.mjs';
import { pullsSubject, branchSubject } from '../../src/build/page/heading.mjs';
import { defaultElk } from '../../src/build/layout/elk.mjs';

// Made-up changes, each as the commits that make its repository, the page
// request and the page Node builds for it, its layout aside, which the Go
// page builder is tested against; and made-up components configs with what
// Node reads of each.
const RECORD = join(dirname(fileURLToPath(import.meta.url)), '..', '..', 'terminal', 'testdata', 'pages.json');

const text = rows => rows.join('\n') + '\n';
const lines = (prefix, n) => Array.from({ length: n }, (_, i) => `${prefix} ${i + 1}`);

const CATALOG = text(['protocol Catalog {', '    func items() -> [Item]', '}', '', 'struct Item: Codable {', '    let name: String', '    let price: Int', '}']);
const MEMORY = text(['final class MemoryCatalog: Catalog {', '    private var stored: [Item] = []', '', '    func items() -> [Item] {', '        stored', '    }', '}']);
const MEMORY_ADD = text(['final class MemoryCatalog: Catalog {', '    private var stored: [Item] = []', '', '    func items() -> [Item] {', '        stored', '    }', '', '    func add(_ item: Item) {', '        stored.append(item)', '    }', '}']);
const BASKET = text(['final class Basket {', '    let catalog: Catalog', '    var lines: [Item] = []', '', '    init(catalog: Catalog) {', '        self.catalog = catalog', '    }', '', '    func total() -> Int {', '        lines.reduce(0) { $0 + $1.price }', '    }', '}']);
const SEARCH = text(['struct Search {', '    let catalog: Catalog', '    let clock = Clock()', '', '    func find(_ text: String) -> [Item] {', '        catalog.items().filter { $0.name.contains(text) }', '    }', '}', '', 'final class Clock {', '    var now = 0', '}']);
const VIEWS = text(['import SwiftUI', '', 'struct CatalogView: View {', '    let list: ItemList', '    var body: some View { list }', '}', '', 'struct ItemList: View {', '    let items: [Item]', '    var body: some View { Text("") }', '}']);
const ITEMS = json => JSON.stringify(json, null, 2) + '\n';

const SHAPES = names => text(['protocol Shape {', '    var area: Double { get }', '}', '', ...names.flatMap(n => [`struct ${n}: Shape {`, '    var area: Double { 1 }', '}', ''])]);

const CASES = [
  {
    name: 'a Swift pull request with renames, deletions, extensions, views, tests, data and binary files',
    script: [
      { commit: ['main', {
        'Sources/Catalog/Catalog.swift': CATALOG,
        'Sources/Catalog/MemoryCatalog.swift': MEMORY,
        'Sources/Basket/Basket.swift': BASKET,
        'Sources/Old/Legacy.swift': text(['final class Legacy {', '    let catalog: Catalog', '}']),
        'Sources/Name.swift': text(['struct Name {', '    let first: String', '    let last: String', '}']),
        'Sources/Catalog/Item+Format.swift': text(['extension Item: CustomStringConvertible {', '    var description: String { name }', '}']),
        'Tests/CatalogTests/CatalogTests.swift': text(['import XCTest', '', 'final class CatalogTests: XCTestCase {', '    let catalog = MemoryCatalog()', '}']),
        'docs/guide.md': text(['# Guide', '', 'The catalog lists items.']),
        'Package.resolved': ITEMS({ pins: [] }),
        'data/items.json': ITEMS(lines('item', 10)),
        'assets/logo.png': '\u0000PNG\u0001\u0002',
      }, 'base'] },
      { commit: ['feature/search', {
        'Sources/Catalog/MemoryCatalog.swift': MEMORY_ADD,
        'Sources/Catalog/Search.swift': SEARCH,
        'Sources/Catalog/MemoryCatalog+Search.swift': text(['extension MemoryCatalog: Searchable {', '    func search(_ text: String) -> [Item] { [] }', '}']),
        'Sources/Util/String+Trim.swift': text(['extension String {', '    var trimmed: String { self }', '}']),
        'Sources/UI/CatalogView.swift': VIEWS,
        'Sources/Basket/Basket.swift': BASKET.replace('$0 + $1.price', '$0 + $1.price * 2'),
        'Sources/Old/Legacy.swift': null,
        'Sources/Name.swift': null,
        'Sources/People/Name.swift': text(['struct Name {', '    let first: String', '    let last: String', '    let middle: String', '}']),
        'Tests/CatalogTests/CatalogTests.swift': text(['import XCTest', '', 'final class CatalogTests: XCTestCase {', '    let catalog = MemoryCatalog()', '    let search = Search(catalog: MemoryCatalog())', '}']),
        'docs/guide.md': text(['# Guide', '', 'The catalog lists every item.']),
        'Package.resolved': ITEMS({ pins: ['a'] }),
        'data/items.json': ITEMS(lines('item', 80)),
        'assets/logo.png': '\u0000PNG\u0001\u0003',
      }, 'Add search'] },
    ],
    request: { base: 'main', members: [{ pull: 12, head: 'feature/search' }], baseBranch: 'main', title: 'Add search', subject: { pulls: [12] }, config: null, context: false, rootTitle: 'widgets' },
  },
  {
    name: 'the same pull request drawn with the unchanged code around it',
    same: 0,
    extra: [
      { commit: ['main', {
        'Sources/Shop/Checkout.swift': text(['final class Checkout {', '    let basket: Basket', '    let search: Searching', '}']),
        'Sources/Shop/Searching.swift': text(['protocol Searching {', '    func find(_ text: String) -> [Item]', '}']),
        'Sources/Shop/Searching+Basket.swift': text(['extension Searching {', '    func first() -> Item? { nil }', '}']),
        'Sources/Shop/Ledger.swift': text(['struct Ledger {', '    let entries: [String]', '}']),
        'Sources/Shop/Report.swift': text(['struct Report {', '    let catalog: Catalog', '    let memory: MemoryCatalog', '}']),
      }, 'shop'] },
      { git: ['rebase', '-q', 'main', 'feature/search'] },
    ],
    request: { base: 'main', members: [{ pull: 12, head: 'feature/search' }], baseBranch: 'main', title: 'Add search', subject: { pulls: [12] }, config: { contextTypes: { include: ['Ledger'], exclude: ['Report'], max: 5 } }, context: true, rootTitle: 'widgets' },
  },
  {
    name: 'the same pull request drawn with a components config',
    same: 0,
    request: {
      base: 'main', members: [{ pull: 12, head: 'feature/search' }], baseBranch: 'main', title: 'Add search', subject: { pulls: [12] }, context: false, rootTitle: 'widgets',
      config: {
        title: 'Search, as configured',
        summary: 'Adds search over the catalog.',
        areas: [{ id: 'shop', title: 'Shop', description: 'What people buy from', components: ['catalog', 'basket'] }],
        components: [
          { id: 'catalog', title: 'Catalog', tag: 'Sources/Catalog', color: 'teal', paths: ['Sources/Catalog/'],
            packages: [{ id: 'search', title: 'Search', description: 'Finding items', paths: ['Sources/Catalog/Search*.swift', 'Sources/Catalog/MemoryCatalog+Search.swift'] }] },
          { id: 'basket', title: 'Basket', paths: ['Sources/Basket/**'] },
          { id: 'tests', title: 'Tests', paths: ['Tests/'] },
          { id: 'shop-context', title: 'Shop code', context: true, paths: ['Sources/Shop/'] },
        ],
        links: [{ from: 'Search', to: 'Catalog', label: 'searches', kind: 'has' }, { from: 'Basket', to: 'Search', label: 'finds with' }],
        summaries: [{ from: 'basket', to: 'catalog', label: 'reads items' }],
        groups: [{ id: 'catalogs', of: 'Catalog', title: 'catalogs', description: 'Every catalog' }],
      },
    },
  },
  {
    name: 'many look-alike types grouped, and a type named twice',
    script: [
      { commit: ['main', { 'Sources/Shapes/Shape.swift': SHAPES([]), 'Sources/Other/Circle.swift': text(['struct Circle {', '    let radius: Double', '}']) }, 'base'] },
      { commit: ['shapes', {
        'Sources/Shapes/Shape.swift': SHAPES(['Square', 'Oval', 'Star', 'Heart', 'Kite', 'Ring', 'Wave']),
        'Sources/Shapes/Circle.swift': text(['struct Circle: Shape {', '    var area: Double { 3 }', '}']),
        'Sources/Other/Circle.swift': text(['struct Circle {', '    let radius: Double', '    let centre: Double', '}']),
      }, 'shapes'] },
    ],
    request: { base: 'main', members: [{ pull: 7, head: 'shapes' }], baseBranch: 'main', title: 'Shapes', subject: { pulls: [7] }, config: null, context: false, rootTitle: 'shapes' },
  },
  {
    name: 'Kotlin, Java, Python and TypeScript across many folders, with imports between files',
    script: [
      { commit: ['main', {
        'app/src/main/kotlin/shop/Cart.kt': text(['package shop', '', 'data class Cart(val lines: List<Line>)', '', 'data class Line(val sku: String, val count: Int)']),
        'server/java/shop/Pricing.java': text(['package shop;', '', 'public interface Pricing {', '    int price(String sku);', '}']),
        'tools/py/report.py': text(['from .rows import Row', '', 'def build():', '    return Row()']),
        'tools/py/rows.py': text(['class Row:', '    def __init__(self):', '        self.cells = []']),
        'web/src/cart.ts': text(['import { total } from "./total";', '', 'export function cart(prices: number[]): number {', '  return total(prices);', '}']),
        'web/src/total.ts': text(['export function total(prices: number[]): number {', '  return prices.reduce((a, b) => a + b, 0);', '}']),
        ...Object.fromEntries(lines('note', 6).map((n, i) => [`notes/n${i}.md`, n + '\n'])),
      }, 'base'] },
      { commit: ['mixed', {
        'app/src/main/kotlin/shop/Cart.kt': text(['package shop', '', 'data class Cart(val lines: List<Line>, val pricing: Pricing)', '', 'data class Line(val sku: String, val count: Int)', '', 'sealed interface Event', '', 'object Cleared : Event']),
        'app/src/main/kotlin/shop/CartView.kt': text(['package shop', '', 'class CartView(private val cart: Cart) {', '    companion object {', '        const val TITLE = "Cart"', '    }', '}']),
        'server/java/shop/Pricing.java': text(['package shop;', '', 'public interface Pricing {', '    int price(String sku);', '    int discount(String sku);', '}']),
        'server/java/shop/ListPricing.java': text(['package shop;', '', 'public abstract class ListPricing implements Pricing {', '    private final Cart cart;', '}', '', 'final class SalePricing extends ListPricing {', '}']),
        'tools/py/report.py': text(['from .rows import Row', 'import os', '', 'def build():', '    return [Row(), os.sep]']),
        'tools/py/rows.py': text(['class Row:', '    def __init__(self):', '        self.cells = []', '', 'class Header(Row):', '    pass']),
        'web/src/cart.ts': text(['import { total } from "./total";', 'import type { Price } from "./price";', '', 'export function cart(prices: Price[]): number {', '  return total(prices.map(p => p.amount));', '}']),
        'web/src/price.ts': text(['export interface Price {', '  amount: number;', '}', '', 'export class Sale implements Price {', '  amount = 0;', '}']),
        'web/src/total.test.ts': text(['import { total } from "./total";', '', 'test("adds", () => total([1]));']),
        ...Object.fromEntries(lines('note', 6).map((n, i) => [`notes/n${i}.md`, n + ' changed\n'])),
        'scripts/a.sh': 'echo a\n',
        'scripts/b.sh': 'echo b\n',
        'README.md': '# Shop\n',
      }, 'mixed'] },
    ],
    request: { base: 'main', members: [{ pull: 40, head: 'mixed' }], baseBranch: 'main', title: 'Mixed', subject: { pulls: [40] }, config: null, context: false, rootTitle: 'shop' },
  },
  {
    name: 'a stack of three pull requests',
    script: [
      { commit: ['main', { 'Sources/Catalog/Catalog.swift': CATALOG, 'Sources/Catalog/MemoryCatalog.swift': MEMORY, 'Sources/Basket/Basket.swift': BASKET }, 'base'] },
      { commit: ['one', { 'Sources/Catalog/MemoryCatalog.swift': MEMORY_ADD }, 'one'] },
      { commit: ['two', { 'Sources/Catalog/Search.swift': SEARCH, 'Sources/Catalog/MemoryCatalog+Search.swift': text(['extension MemoryCatalog {', '    func clear() {}', '}']) }, 'two'] },
      { commit: ['three', { 'Sources/Basket/Basket.swift': BASKET.replace('let catalog: Catalog', 'let catalog: Catalog\n    let search: Search'), 'Sources/Catalog/Search.swift': SEARCH.replace('var now = 0', 'var now = 1') }, 'three'] },
    ],
    request: { base: 'main', members: [{ pull: 21, head: 'one' }, { pull: 22, head: 'two' }, { pull: 23, head: 'three' }], baseBranch: 'main', title: 'Stack #21 to #23', subject: { pulls: [21, 22, 23] }, config: null, context: false, rootTitle: 'widgets' },
  },
  {
    name: 'a stack out of step',
    script: [
      { commit: ['main', { 'web/cart.ts': text(lines('// cart', 20)), 'web/total.ts': text(lines('// total', 10)) }, 'base'] },
      { commit: ['lower', { 'web/cart.ts': text(lines('// cart', 20).map(l => (l === '// cart 4' ? '// cart four' : l))) }, 'lower'] },
      { commit: ['upper', { 'web/total.ts': text(lines('// total', 10).map(l => (l === '// total 2' ? '// total two' : l))) }, 'upper'] },
      { checkout: 'lower' },
      { commit: ['lower', { 'web/cart.ts': text(lines('// cart', 20).map(l => (l === '// cart 4' ? '// cart four' : l === '// cart 15' ? '// cart fifteen' : l))) }, 'lower again'] },
    ],
    request: { base: 'main', members: [{ pull: 31, head: 'lower' }, { pull: 32, head: 'upper' }], baseBranch: 'main', title: 'Stack #31 to #32', subject: { pulls: [31, 32] }, config: null, context: false, rootTitle: 'widgets' },
  },
  {
    name: 'a branch drawn with context, its context types extended elsewhere',
    script: [
      { commit: ['main', {
        'Sources/Core/Store.swift': text(['protocol Store {', '    func load() -> Data', '}']),
        'Sources/Core/Disk.swift': text(['final class Disk: Store {', '    func load() -> Data { Data() }', '}']),
        'Sources/Core/Disk+Cache.swift': text(['extension Disk {', '    func cached() -> Data { load() }', '}']),
        'Sources/App/Reader.swift': text(['struct Reader {', '    let store: Store', '}']),
      }, 'base'] },
      { commit: ['feature/reader', { 'Sources/App/Reader.swift': text(['struct Reader {', '    let store: Store', '    let disk: Disk', '}']) }, 'reader'] },
    ],
    request: { base: 'main', members: [{ head: 'feature/reader' }], baseBranch: 'main', title: 'feature/reader', subject: { branch: 'feature/reader' }, config: null, context: true, rootTitle: 'core' },
  },
  {
    name: 'a change one folder dominates, with small groups beside it',
    script: [
      { commit: ['main', { 'README.md': '# Spread\n' }, 'base'] },
      { commit: ['spread', Object.fromEntries([
        ...['a', 'b', 'c'].flatMap(part => Array.from({ length: 27 }, (_, i) => [`big/${part}/Part${part.toUpperCase()}${i}.swift`, `struct Part${part.toUpperCase()}${i} {}\n`])),
        ...Array.from({ length: 3 }, (_, i) => [`big/small/Small${i}.swift`, `struct Small${i} {}\n`]),
        ...['x', 'y', 'z'].flatMap(side => Array.from({ length: 5 }, (_, i) => [`${side}/${side.toUpperCase()}${i}.swift`, `struct ${side.toUpperCase()}${i} {}\n`])),
        ...Array.from({ length: 5 }, (_, i) => [`Top${i}.swift`, `struct Top${i} {}\n`]),
      ]), 'spread'] },
    ],
    request: { base: 'main', members: [{ head: 'spread' }], baseBranch: 'main', title: 'spread', subject: { branch: 'spread' }, config: null, context: false, rootTitle: 'spread' },
  },
  {
    name: 'no difference',
    script: [{ commit: ['main', { 'a.txt': 'a\n' }, 'base'] }, { commit: ['same', {}, 'empty'] }],
    request: { base: 'main', members: [{ pull: 1, head: 'same' }], baseBranch: 'main', title: 'Same', subject: { pulls: [1] }, config: null, context: false, rootTitle: 'same' },
  },
];

const CONFIGS = [
  null,
  { title: 'T', contextTypes: { max: 0 } },
  CASES[2].request.config,
  [],
  { color: 'red', components: 'x', areas: [{}], links: [5], summaries: [{ from: 'a' }], groups: [{ id: 'Bad Id', of: '' }], contextTypes: { max: -1, include: [1] } },
  { components: [{ id: 'a', title: 'A' }, { id: 'a', title: '  ', color: 'pink', context: 'yes', packages: [{ id: 'p' }, 3] }], areas: [{ id: 'x', title: 'X', components: ['a', 'a', 'missing'] }], summaries: [{ from: 'a', to: 'nowhere', label: 'l' }] },
  { links: [{ from: 'A', to: 'B', label: 'l', kind: 'owns' }], contextTypes: { max: 2.5, exclude: [''] } },
];

function play(repo, script) {
  for (const op of script) {
    if (op.commit) repo.commit(...op.commit);
    else if (op.checkout) repo.checkout(op.checkout);
    else repo.git(...op.git);
  }
}

async function built(repo, request) {
  const git = new Git(repo.work);
  try {
    const { page, starts, counts, notes } = await buildPage({
      git,
      base: request.base,
      members: request.members,
      baseBranch: request.baseBranch,
      title: request.title,
      subject: request.subject.branch ? branchSubject(request.subject.branch) : pullsSubject(request.subject.pulls),
      config: readConfig(request.config ?? undefined),
      context: request.context,
      rootTitle: request.rootTitle,
      elk: defaultElk(),
    });
    const { layout, ...rest } = page;
    return { built: { page: rest, starts, counts, notes } };
  } catch (error) {
    return { error: error.message };
  }
}

async function recorded() {
  const pages = [];
  for (const c of CASES) {
    const script = c.same === undefined ? c.script : [...CASES[c.same].script, ...(c.extra || [])];
    const repo = throwawayRepo();
    try {
      play(repo, script);
      pages.push({ name: c.name, script, request: c.request, ...(await built(repo, c.request)) });
    } finally {
      repo.done();
    }
  }
  const configs = CONFIGS.map(given => {
    try {
      return { given, config: readConfig(given ?? undefined) };
    } catch (error) {
      return { given, error: error.message };
    }
  });
  return { pages, configs };
}

test('the pages and configs the Go page builder is tested against build as Node builds them now', async () => {
  const now = JSON.parse(JSON.stringify(await recorded()));
  if (process.env.LAZYREVIEW_RECORD) {
    writeFileSync(RECORD, JSON.stringify(now, null, 1) + '\n');
    return;
  }
  assert.deepEqual(JSON.parse(readFileSync(RECORD, 'utf8')), now, `${RECORD} is not what Node builds now; write it again with LAZYREVIEW_RECORD=1 npm test, then run the Go tests`);
});
