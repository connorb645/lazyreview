import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync, writeFileSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { maskSource, SWIFT_SYNTAX, KOTLIN_SYNTAX, JAVA_SYNTAX, TYPESCRIPT_SYNTAX, PYTHON_SYNTAX } from '../../src/build/languages/source-mask.mjs';
import { languageFor } from '../../src/build/languages/languages.mjs';
import { bareTypeName, splitTopLevel, identifiers } from '../../src/build/languages/header-text.mjs';

// What each language reader makes of each text, which the Go readers are tested against. Offsets are written as
// UTF-8 byte offsets and the masked text with one blank per byte, as Go keeps them, and a span runs no further than
// the text, as Node slices it.
const RECORD = join(dirname(fileURLToPath(import.meta.url)), '..', '..', 'terminal', 'testdata', 'languages.json');

const lines = (...given) => `${given.join('\n')}\n`;
const crlf = text => text.replace(/\n/g, '\r\n');

const SWIFT = [
  lines('import Foundation', '', '/// A session. struct NotThis {}',
    '@MainActor public final class CheckoutSession: NSObject, CheckoutRunning, @unchecked Sendable {',
    '    private let clock: ShopClock', '    weak var delegate: CheckoutDelegate?', '    static let shared = CheckoutSession()',
    '    private lazy var reader = ReceiptReader(url: url)', '    var computed: Int { 1 }', '',
    '    struct Config: Codable {', '        let speed: Double', '    }', '',
    '    func run() {', '        let local: LocalOnly = LocalOnly()', '        class func notAType() {}', '    }', '}', '',
    'protocol CheckoutRunning: AnyObject {', '    var progress: CheckoutProgress { get }', '}', '',
    'enum PaymentKind: String, CaseIterable { case card }', '',
    'extension CheckoutSession: CustomStringConvertible where Self: AnyObject {', '    var description: String { "" }', '}', '',
    'struct Box<T: Equatable>: Container where T: Hashable {}', 'actor Store {}'),
  'struct MapView: View { var body: some View { Text("") } }\nfinal class MapTests: XCTestCase {}\n@Suite struct NewTests {}',
  lines('struct Settings {', '    var volume: Int = 3 {', '        didSet { save() }', '    }', '    var name: String {', '        get { "" }',
    '        set { }', '    }', '    private(set) var count = Counter.zero()', '    unowned(unsafe) var owner: Owner', '    let made = Factory.Part.make()',
    '    @Published var items: [Item] = []', '    let pair: (Int, String)', '    var closure: () -> Void = {}', '}'),
  lines('let s = "\\(d["k"]) {"; let r = #"a "{" b"#; struct After {}', 'let raw = #"x \\#(value) {"#', 'let multi = """',
    '  class Hidden {', '  """', '/* outer /* inner { */ still */ class Visible {}', 'extension Outer.Inner: Equatable {}',
    'indirect enum Tree<T> where T: Comparable { case leaf }', 'class func notThis() {}', 'class var notThisEither: Int { 0 }',
    'struct Generic<T: (Int) -> Void>: P {}', 'struct Open {', '    var x: Int'),
  crlf(lines('public struct Line: Hashable {', '    let a: Int', '    var b: String = ""', '    lazy var c = Maker()', '}')),
  lines('﻿struct Café { let name: String; let price: Price }', '// é 😀 { not a brace', 'let greeting = "héllo { 😀"',
    'struct Spaced { let x: Int }', 'class Owner { let tag: Tag<"é"> }'),
  lines('@available(iOS 15, *)', 'public class Old: NSObject {', '    @objc dynamic var flag: Bool = false', '    nonisolated let id: UUID', '}',
    'struct S { let a: Int; let b: Int }', 'enum E { case a; static let x = 1 }', 'protocol P where Self: AnyObject {}',
    'struct Q: some P {}', 'struct R : ~Copyable {}'),
];

const KOTLIN = [
  lines('package com.example', '', 'data class Price(val amount: Double, private val source: PriceSource, count: Int)', '', 'sealed interface Outcome', '',
    'class Uploader(', '    private val client: HttpClient,', ') : BaseUploader(client), Outcome, Closeable by client {',
    '    private val queue: UploadQueue = UploadQueue()', '    val lazyThing by lazy { Parser() }', '    companion object {',
    '        const val LIMIT = 3', '    }', '    inner class Part', '}', '', 'object Registry : Lookup', '', 'enum class Mode { FAST, SLOW }', '',
    'class UploaderTest {', '    @Test fun works() {}', '}'),
  lines('class Shop<T : Comparable<T>>(val item: T) : Base<T>(), Store', '    where T : Any {', '    val size: Int get() = 1',
    '    val other: Int', '        get() = 2', '    var counted: Int = 0', '        private set', '    val plain = 3', '    val made = Factory()',
    '    companion object Named {', '        val x = 1', '    }', '}', 'fun interface Callback { fun call() }', 'value class Id(val raw: String)',
    'inline class Old(val raw: Int)', 'annotation class Marker', 'abstract class Shape', 'sealed class Result {', '    object Done : Result()', '}',
    'enum class Level(val weight: Int) { LOW(1), HIGH(2) }', 'object : Listener {}'),
  lines('class Text(', '    @SerializedName("x") val x: String = "a, b",', '    val list: List<Map<String, Int>>,', ') {',
    '    val template = "${values.map { "}" }} {"', '    val raw = """', '        class Hidden {', '    """', "    val brace = '{'", '}',
    'class Multi :', '    Base(),', '    Other', '{', '    val y: Int = 0', '}', 'class Lambda(val f: (Int) -> Unit) : Handler<(String) -> Unit>',
    'class Parameterized {', '    @ParameterizedTest fun many() {}', '}', '/* outer /* inner */ class Hidden {} */', 'class Unclosed {'),
  crlf(lines('class Windows(val a: Int) {', '    val b: String = ""', '    val c by lazy { Thing() }', '}')),
  lines('﻿class Ünit(val café: String)', '// é 😀 {', 'class Label(val text: String = "é {") { val tag: Tag = Tag("😀") }',
    'private  companion object Inside', 'expect class Platform', 'actual class Platform'),
];

const JAVA = [
  lines('package x;', 'public abstract class Store<T> extends BaseStore<T> implements Closeable, Flushable {', '    private final Cache cache = new Cache();',
    '    private static Logger LOG;', '    protected List<Entry> entries;', '    public Store() { Local l = null; }', '    static class Inner {}', '}',
    'interface Flushable extends Thing {}', 'record Point(int x, Label label) implements Shape {}'),
  lines('@interface Marker {}', 'sealed interface Shape permits Circle, Square {}', 'non-sealed class Circle implements Shape {',
    '    String[] names;', '    final Map<String, List<Integer>> map = new HashMap<>();', '    transient volatile Counter counter;',
    '    void run() { return; }', '}', 'final class Square implements Shape, Comparable<Square> {', '    @Inject Service service;', '}',
    'record Pair<A, B>(@NotNull A first, List<Map<String, B>> second) {}', 'enum Mode { FAST, SLOW; private Speed speed; }',
    'class Outer<T extends Comparable<T>> extends Base<T> {', '    void make() { class Local {} }', '    @Test void works() {}', '}'),
  lines('class Text {', '    String block = """', '        class Hidden {', '        """;', "    char brace = '{';", '    String s = "}";', '}',
    '/* class Commented {} */', 'class Unclosed {'),
  crlf(lines('public class Windows {', '    private Name name;', '    Count count = new Count();', '}')),
  lines('﻿public class Café {', '    String é = "😀 {";', '    // é {', '    Label label;', '}'),
];

const TYPESCRIPT = [
  lines("import { Fetcher, type Options as Opts } from './fetcher';", 'import * as path from "node:path";',
    "import Default, { Named } from '../shared/named.js';", "export * from './everything';", "const lazy = await import('./lazy');",
    "const old = require('./old');", "// import { Ghost } from './ghost';", '', '@Injectable()',
    'export class Service extends Base<Item> implements Runner, Stopper {', '  private readonly cache: Map<string, Item> = new Map();',
    '  #secret = new Secret();', '  constructor(private readonly fetcher: Fetcher, plain: number) { super(); }', '  run(): void {}', '}', '',
    'export interface Runner extends Thing { run(): void; status: Status }', 'export type Result = { ok: boolean; item: Item } | Failure;',
    'export const enum Flag { On, Off }', 'export namespace Shapes { export class Circle {} }', 'function inner() { class Local {} }'),
  lines('export type Shape =', '  | Circle', '  | Square;', 'type Pair<A, B> = [A, B]', 'type Fn = (a: number) => void', 'type Obj = {',
    '  a: string', '}', 'declare class Declared { x: number }', 'export default class Main {', '  name?: string;', '  count!: number;',
    '  static shared = new Main();', '  readonly made = new Map<string, number>();', '  value = 3', '  get computed() { return 1; }', '}',
    'abstract class Base<T extends (x: number) => void> implements Runner {}', 'namespace A.B { class Deep {} }',
    'const ratio = a / b / c; class AfterDivision {}', 'const re = /[{"]/g; class AfterRegex {}', 'const t = `a ${ {x: "}"}.x } {`; class AfterTemplate {}'),
  lines("import type { A } from 'a';", "import x, * as y from 'z';", 'export { a as b } from "./c";', "import './polyfill';",
    'import {', '  One,', '  Two as Second,', "} from './multi';", "const s = \"import { Fake } from 'fake'\";", "import Default from './d'"),
  lines('export function View() {', '  return <div className="x">{value}</div>;', '}', 'class AfterJsx {}', 'interface Props { label: string }'),
  crlf(lines("import { A } from './a';", 'export class Windows {', '  name: string;', '  count = new Counter();', '  label: string', '}')),
  lines("﻿import { é } from './é';", 'export class Café {', "  kind: 'café' | '😀';", '  // é {', '  note: string = "é {";', '}',
    'interface Spaced { x: number }'),
];

const PYTHON = [
  lines('from __future__ import annotations', 'from .store import (Store,', '    Entry as E)', 'import os, pkg.sub.mod as m', '', '"""',
    'class NotThis:', '"""', '', '@dataclass(frozen=True)', 'class Price(Base, metaclass=ABCMeta):', '    amount: float',
    '    source: PriceSource = None', '', '    def __init__(self, uploader: Uploader):', '        self.uploader: Uploader = uploader',
    '        self.queue = UploadQueue()', '        local: Ignored = 1', '', '    class Inner:', '        pass', '',
    'class Reader(Protocol):', '    def read(self) -> bytes: ...', '', 'class Mode(Enum):', '    A = 1'),
  lines('class Point(NamedTuple):', '    x: int', '    y: int', 'class Movie(TypedDict):', '    title: str', 'class Shape(ABC):', '    pass',
    'class Abstract(metaclass=ABCMeta):', '    pass', '@dataclasses.dataclass', 'class Data:', '    a: int', 'class Box[T](Base):', '    item: T',
    'class Generic(Generic[T], Base):', '    pass', 'class Tests(unittest.TestCase):', '    def test_it(self):', '        pass',
    'class Mixed:', '    async def __init__(self):', '        self.a: int = 0', '        self.b = Maker()', '        self.c = maker()',
    '        self.a = Other()', '    value: int', '    def other(self):', '        self.d = Hidden()'),
  lines('x = rb"class Fake:"', 'y = f"{class_name}"', "z = '''", 'class Doc:', "'''", 'class Real:', '    pass', '# class Commented:',
    'from a import b, \\', '    c', 'from . import x', 'from .. import (a as b, c)', 'import a.b as c, d', 'from x import *',
    'import os # comment', 'class Open('),
  lines('class Tabbed:', '\tname: str', '\tdef __init__(self):', '\t\tself.size = Size()', 'class After:', '    pass'),
  crlf(lines('class Windows:', '    name: str', '    def __init__(self):', '        self.size = Size()', 'from a import b')),
  lines('﻿class Café:', '    note: str = "é"', '    """', 'é is here', '    """', '    label: Literal["é", "😀"]', 'class Next:', '    pass'),
];

const READINGS = [
  ...SWIFT.map(text => ['Swift', text]),
  ...KOTLIN.map(text => ['Kotlin', text]),
  ...JAVA.map(text => ['Java', text]),
  ...TYPESCRIPT.map(text => ['TypeScript', text]),
  ...PYTHON.map(text => ['Python', text]),
];

const SYNTAXES = { swift: SWIFT_SYNTAX, kotlin: KOTLIN_SYNTAX, java: JAVA_SYNTAX, typescript: TYPESCRIPT_SYNTAX, python: PYTHON_SYNTAX };
const MASKS = [
  'let a = "x { y" // brace {\n/* { */ let b = 1', '/* a /* b */ { */ class C {}', '"unterminated { string\nnext {', '/* never closed {',
  'x = rb"class Fake:"\n"""\nclass Doc:\n"""\nclass Real:\n    pass', "'a\\'b' \"c\\\"d\" `e ${f + `g ${h}`} i` { }",
  'a = b / c / d; e = /x{/; return /[/]/.test(s) {', 'x = "é😀" // é😀 {\n"""é""" { }', 'r#"a"# r##"b"## #"c"# { }', "'{' '\\'' \"{\" { }",
];

const ENTRIES = ['@unchecked Sendable', 'Foundation.NSObject', 'Generic[T]', '~Copyable', 'any P', 'some View', 'Base<T>(args)', 'out T', 'in T',
  '', '123', 'a.b.', '@Foo(bar) Baz', 'NSObject?', '[Int]', '(A, B)', 'Map<K, V>', 'public Foo', 'internal(set) X', '$Dollar', 'é', 'A B',
  'Interface', 'final  Base', 'abstract.Thing', 'Outer.Inner<T>.Deep'];
const SPLITS = ['A<B, C>, D(e, f), G', 'a -> b, c', '', ' , ,', 'F<(A) -> B>, C', 'x[1,2], {a, b}', '>, a', 'é, 😀 , b '];
const IDENTIFIERS = ['a b a', 'foo.bar(baz) $x _y 1z', 'émile x', '', 'class A extends A<B> { b: B }'];
const PATHS = ['a/B.swift', 'a/b.kt', 'a/b.kts', 'a/b.java', 'a/b.tsx', 'a/b.d.ts', 'a/b.mjs', 'a/b.cjs', 'a/b.jsx', 'a/b.mts', 'a/b.cts', 'a/b.py',
  'a/b.pyi', 'a/README.md', 'X.KT', 'A.Swift', '.swift', 'swift', 'a/b.ts.orig'];

function byteOffsets(text) {
  const offsets = [];
  let bytes = 0;
  for (let i = 0; i < text.length; i++) {
    offsets.push(bytes);
    const code = text.charCodeAt(i);
    bytes += code < 0x80 ? 1 : code < 0x800 ? 2 : code >= 0xd800 && code < 0xdc00 ? 4 : code >= 0xdc00 && code < 0xe000 ? 0 : 3;
  }
  offsets.push(bytes);
  return offset => offsets[Math.min(offset, text.length)];
}

function blankedByBytes(text, masked) {
  let out = '';
  for (let i = 0; i < text.length;) {
    const units = text.codePointAt(i) > 0xffff ? 2 : 1;
    const original = text.slice(i, i + units);
    const shown = masked.slice(i, i + units);
    if (shown !== original) assert.equal(shown, ' '.repeat(units), 'masking blanks a character whole');
    out += shown === original ? original : ' '.repeat(Buffer.byteLength(original));
    i += units;
  }
  return out;
}

function reading(name, text) {
  const read = languageFor(`file.${{ Swift: 'swift', Kotlin: 'kt', Java: 'java', TypeScript: 'ts', Python: 'py' }[name]}`).read(text);
  const bytes = byteOffsets(text);
  return {
    language: name,
    text,
    masked: blankedByBytes(text, read.masked),
    declarations: read.declarations.map(d => ({
      name: d.name, kind: d.kind, label: d.label ?? null, inherits: d.inherits, at: bytes(d.at),
      span: [bytes(d.span[0]), bytes(d.span[1])], body: d.body ? [bytes(d.body[0]), bytes(d.body[1])] : null, depth: d.depth ?? 0,
      start: d.start, end: d.end, view: d.view, test: d.test, parent: d.parent, owner: d.owner, top: d.top, properties: d.properties,
    })),
    imports: read.imports,
  };
}

function recorded() {
  return {
    readings: READINGS.map(([name, text]) => reading(name, text)),
    masks: Object.entries(SYNTAXES).flatMap(([syntax, spec]) => MASKS.map(text => ({ syntax, text, masked: blankedByBytes(text, maskSource(text, spec)) }))),
    bareNames: ENTRIES.map(entry => ({ entry, name: bareTypeName(entry) })),
    splits: SPLITS.map(text => ({ text, parts: splitTopLevel(text) })),
    identifiers: IDENTIFIERS.map(text => ({ text, words: [...identifiers(text)] })),
    languages: PATHS.map(path => {
      const language = languageFor(path);
      return { path, name: language?.name ?? null, family: language?.family ?? null };
    }),
  };
}

test('the language readings the Go readers are tested against are the ones Node gives now', () => {
  const now = recorded();
  if (process.env.LAZYREVIEW_RECORD) {
    writeFileSync(RECORD, JSON.stringify(now, null, 1) + '\n');
    return;
  }
  assert.deepEqual(JSON.parse(readFileSync(RECORD, 'utf8')), now, `${RECORD} is not what Node reads now; write it again with LAZYREVIEW_RECORD=1 npm test, then run the Go tests`);
});
