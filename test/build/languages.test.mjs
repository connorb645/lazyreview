import { test } from 'node:test';
import assert from 'node:assert/strict';
import { maskSource, SWIFT_SYNTAX, TYPESCRIPT_SYNTAX, PYTHON_SYNTAX, KOTLIN_SYNTAX } from '../../src/build/languages/source-mask.mjs';
import { readSwift } from '../../src/build/languages/swift.mjs';
import { readKotlin } from '../../src/build/languages/kotlin.mjs';
import { readJava } from '../../src/build/languages/java.mjs';
import { readTypeScript } from '../../src/build/languages/typescript.mjs';
import { readPython } from '../../src/build/languages/python.mjs';
import { languageFor } from '../../src/build/languages/languages.mjs';
import { bareTypeName, splitTopLevel } from '../../src/build/languages/header-text.mjs';

const tops = result => result.declarations.filter(d => d.top);
const byName = (result, name) => result.declarations.find(d => d.name === name);

test('masking keeps offsets and newlines and blanks comments and strings', () => {
  const text = 'let a = "x { y" // brace {\n/* { */ let b = 1';
  const masked = maskSource(text, SWIFT_SYNTAX);
  assert.equal(masked.length, text.length);
  assert.equal(masked.split('\n').length, 2);
  assert.ok(!masked.includes('{'));
  assert.ok(masked.includes('let b = 1'));
  assert.ok(masked.includes('"     "'));
});

test('masking skips Swift interpolation with nested quotes and raw strings', () => {
  const text = 'let s = "\\(d["k"]) {"; let r = #"a "{" b"#; struct After {}';
  const masked = maskSource(text, SWIFT_SYNTAX);
  assert.ok(masked.includes('struct After {}'));
  assert.equal((masked.match(/\{/g) || []).length, 1);
});

test('masking handles nested block comments in Swift and Kotlin', () => {
  for (const syntax of [SWIFT_SYNTAX, KOTLIN_SYNTAX]) {
    const masked = maskSource('/* a /* b */ { */ class C {}', syntax);
    assert.ok(masked.trim().startsWith('class C {}'));
  }
});

test('masking handles template literals and regular expressions in TypeScript', () => {
  const masked = maskSource('const t = `a ${ {x: "}"}.x } {`; const r = /[{"]/; class K {}', TYPESCRIPT_SYNTAX);
  assert.ok(masked.includes('class K {}'));
  assert.equal((masked.match(/\{/g) || []).length, 1);
});

test('masking handles Python prefixed and triple-quoted strings', () => {
  const masked = maskSource('x = rb"class Fake:"\n"""\nclass Doc:\n"""\nclass Real:\n    pass', PYTHON_SYNTAX);
  assert.ok(!masked.includes('Fake'));
  assert.ok(!masked.includes('Doc'));
  assert.ok(masked.includes('class Real:'));
});

test('Swift: kinds, inheritance, nesting and stored properties', () => {
  const result = readSwift(`
import Foundation

/// A session. struct NotThis {}
@MainActor public final class CheckoutSession: NSObject, CheckoutRunning, @unchecked Sendable {
    private let clock: ShopClock
    weak var delegate: CheckoutDelegate?
    static let shared = CheckoutSession()
    private lazy var reader = ReceiptReader(url: url)
    var computed: Int { 1 }

    struct Config: Codable {
        let speed: Double
    }

    func run() {
        let local: LocalOnly = LocalOnly()
        class func notAType() {}
    }
}

protocol CheckoutRunning: AnyObject {
    var progress: CheckoutProgress { get }
}

enum PaymentKind: String, CaseIterable { case card }

extension CheckoutSession: CustomStringConvertible where Self: AnyObject {
    var description: String { "" }
}

struct Box<T: Equatable>: Container where T: Hashable {}
actor Store {}
`);
  assert.deepEqual(tops(result).map(d => `${d.kind} ${d.name}`),
    ['class CheckoutSession', 'protocol CheckoutRunning', 'enum PaymentKind', 'extension CheckoutSession', 'struct Box', 'actor Store']);
  const session = byName(result, 'CheckoutSession');
  assert.deepEqual(session.inherits, ['NSObject', 'CheckoutRunning', 'Sendable']);
  assert.deepEqual(session.members, ['ShopClock', 'CheckoutDelegate?', 'Int']);
  assert.deepEqual(session.made, ['ReceiptReader']);
  assert.equal(byName(result, 'Config').parent, 'CheckoutSession');
  assert.equal(byName(result, 'Config').top, false);
  assert.ok(!result.declarations.some(d => d.name === 'NotThis' || d.name === 'notAType'));
  assert.equal(session.start, 5);
  assert.equal(session.end, 20);
  assert.deepEqual(byName(result, 'Box').inherits, ['Container']);
  assert.deepEqual(result.declarations.find(d => d.kind === 'extension').inherits, ['CustomStringConvertible']);
});

test('Swift: views and tests are flagged', () => {
  const result = readSwift('struct MapView: View { var body: some View { Text("") } }\nfinal class MapTests: XCTestCase {}\n@Suite struct NewTests {}');
  assert.equal(byName(result, 'MapView').view, true);
  assert.equal(byName(result, 'MapTests').test, true);
  assert.equal(byName(result, 'NewTests').test, true);
});

test('Kotlin: classes without bodies, constructors, supertypes and objects', () => {
  const result = readKotlin(`
package com.example

data class Price(val amount: Double, private val source: PriceSource, count: Int)

sealed interface Outcome

class Uploader(
    private val client: HttpClient,
) : BaseUploader(client), Outcome, Closeable by client {
    private val queue: UploadQueue = UploadQueue()
    val lazyThing by lazy { Parser() }
    companion object {
        const val LIMIT = 3
    }
    inner class Part
}

object Registry : Lookup

enum class Mode { FAST, SLOW }

class UploaderTest {
    @Test fun works() {}
}
`);
  assert.deepEqual(tops(result).map(d => `${d.kind} ${d.name}`),
    ['class Price', 'interface Outcome', 'class Uploader', 'object Registry', 'enum Mode', 'class UploaderTest']);
  const price = byName(result, 'Price');
  assert.equal(price.label, 'data class');
  assert.deepEqual(price.members, ['Double', 'PriceSource']);
  const uploader = byName(result, 'Uploader');
  assert.deepEqual(uploader.inherits, ['BaseUploader', 'Outcome', 'Closeable']);
  assert.deepEqual(uploader.members, ['HttpClient', 'UploadQueue']);
  assert.deepEqual(uploader.made, ['UploadQueue', 'Parser']);
  assert.equal(byName(result, 'Part').parent, 'Uploader');
  assert.equal(byName(result, 'Outcome').label, 'sealed interface');
  assert.equal(byName(result, 'UploaderTest').test, true);
});

test('Java: extends, implements, records and fields', () => {
  const result = readJava(`
package x;
public abstract class Store<T> extends BaseStore<T> implements Closeable, Flushable {
    private final Cache cache = new Cache();
    private static Logger LOG;
    protected List<Entry> entries;
    public Store() { Local l = null; }
    static class Inner {}
}
interface Flushable extends Thing {}
record Point(int x, Label label) implements Shape {}
`);
  assert.deepEqual(tops(result).map(d => `${d.kind} ${d.name}`), ['class Store', 'interface Flushable', 'record Point']);
  const store = byName(result, 'Store');
  assert.equal(store.label, 'abstract class');
  assert.deepEqual(store.inherits, ['BaseStore', 'Closeable', 'Flushable']);
  assert.deepEqual(store.members, ['Cache', 'List<Entry>']);
  assert.deepEqual(store.made, ['Cache']);
  assert.deepEqual(byName(result, 'Point').members, ['int', 'Label']);
  assert.equal(byName(result, 'Inner').top, false);
});

test('TypeScript: classes, interfaces, aliases, namespaces and imports', () => {
  const result = readTypeScript(`
import { Fetcher, type Options as Opts } from './fetcher';
import * as path from "node:path";
import Default, { Named } from '../shared/named.js';
export * from './everything';
const lazy = await import('./lazy');
const old = require('./old');
// import { Ghost } from './ghost';

@Injectable()
export class Service extends Base<Item> implements Runner, Stopper {
  private readonly cache: Map<string, Item> = new Map();
  #secret = new Secret();
  constructor(private readonly fetcher: Fetcher, plain: number) { super(); }
  run(): void {}
}

export interface Runner extends Thing { run(): void; status: Status }
export type Result = { ok: boolean; item: Item } | Failure;
export const enum Flag { On, Off }
export namespace Shapes { export class Circle {} }
function inner() { class Local {} }
`);
  assert.deepEqual(tops(result).map(d => `${d.kind} ${d.name}`),
    ['class Service', 'interface Runner', 'type Result', 'enum Flag', 'class Circle']);
  const service = byName(result, 'Service');
  assert.deepEqual(service.inherits, ['Base', 'Runner', 'Stopper']);
  assert.deepEqual(service.members, ['Map<string, Item>', 'Fetcher']);
  assert.deepEqual(service.made, ['Map', 'Secret']);
  assert.deepEqual(byName(result, 'Runner').members, ['Status']);
  assert.ok(byName(result, 'Result').end >= byName(result, 'Result').start);
  assert.equal(byName(result, 'Local').top, false);
  assert.deepEqual(result.imports, [
    { spec: './fetcher', names: ['Fetcher', 'Opts'] },
    { spec: 'node:path', names: null },
    { spec: '../shared/named.js', names: ['Named', 'Default'] },
    { spec: './everything', names: null },
    { spec: './lazy', names: null },
    { spec: './old', names: null },
  ]);
});

test('Python: classes, bases, labels, attributes and imports', () => {
  const result = readPython(`
from __future__ import annotations
from .store import (Store,
    Entry as E)
import os, pkg.sub.mod as m

"""
class NotThis:
"""

@dataclass(frozen=True)
class Price(Base, metaclass=ABCMeta):
    amount: float
    source: PriceSource = None

    def __init__(self, uploader: Uploader):
        self.uploader: Uploader = uploader
        self.queue = UploadQueue()
        local: Ignored = 1

    class Inner:
        pass

class Reader(Protocol):
    def read(self) -> bytes: ...

class Mode(Enum):
    A = 1
`);
  assert.deepEqual(tops(result).map(d => `${d.kind} ${d.name}`), ['class Price', 'protocol Reader', 'enum Mode']);
  const price = byName(result, 'Price');
  assert.equal(price.label, 'dataclass');
  assert.deepEqual(price.inherits, ['Base']);
  assert.deepEqual(price.members, ['float', 'PriceSource', 'Uploader']);
  assert.deepEqual(price.made, ['UploadQueue']);
  assert.equal(price.start, 12);
  assert.equal(price.end, 22);
  assert.equal(byName(result, 'Inner').parent, 'Price');
  assert.deepEqual(result.imports.map(i => [i.spec, i.names]), [
    ['__future__', ['annotations']], ['.store', ['Store', 'E']], ['os', null], ['pkg.sub.mod', null]]);
});

test('header helpers', () => {
  assert.deepEqual(splitTopLevel('A<B, C>, D(e, f), G'), ['A<B, C>', 'D(e, f)', 'G']);
  assert.equal(bareTypeName('@unchecked Sendable'), 'Sendable');
  assert.equal(bareTypeName('Foundation.NSObject'), 'NSObject');
  assert.equal(bareTypeName('Generic[T]'), 'Generic');
  assert.equal(bareTypeName('~Copyable'), 'Copyable');
});

test('languages by extension', () => {
  assert.equal(languageFor('a/B.swift').name, 'Swift');
  assert.equal(languageFor('a/b.kt').family, 'jvm');
  assert.equal(languageFor('a/b.java').family, 'jvm');
  assert.equal(languageFor('a/b.tsx').name, 'TypeScript');
  assert.equal(languageFor('a/b.py').name, 'Python');
  assert.equal(languageFor('a/README.md'), null);
});
