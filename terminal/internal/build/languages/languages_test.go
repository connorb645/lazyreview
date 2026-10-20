package languages

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func tops(r Reading) []string {
	var out []string
	for _, d := range r.Declarations {
		if d.Top {
			out = append(out, d.Kind+" "+d.Name)
		}
	}
	return out
}

func byName(t *testing.T, r Reading, name string) Declaration {
	t.Helper()
	for _, d := range r.Declarations {
		if d.Name == name {
			return d
		}
	}
	t.Fatalf("no declaration named %s", name)
	return Declaration{}
}

func named(d Declaration) []string {
	var out []string
	for _, p := range d.Properties {
		written := "<nil>"
		if p.Type != nil {
			written = *p.Type
		}
		mark := ""
		if p.Inferred {
			mark = " inferred"
		}
		out = append(out, fmt.Sprintf("%s: %s @%02d%s", p.Name, written, p.Line, mark))
	}
	return out
}

func lines(given ...string) string {
	return strings.Join(given, "\n") + "\n"
}

func want(t *testing.T, what string, got, wanted any) {
	t.Helper()
	if !reflect.DeepEqual(got, wanted) {
		t.Errorf("%s: got %q, want %q", what, got, wanted)
	}
}

// Masking keeps every offset and newline, and blanks comments and the
// insides of strings.
func TestMaskingKeepsOffsetsAndBlanksCommentsAndStrings(t *testing.T) {
	text := "let a = \"x { y\" // brace {\n/* { */ let b = 1"
	masked := maskSource(text, swiftSyntax)
	want(t, "length", len(masked), len(text))
	want(t, "lines", strings.Count(masked, "\n"), 1)
	want(t, "braces", strings.Contains(masked, "{"), false)
	want(t, "code kept", strings.Contains(masked, "let b = 1"), true)
	want(t, "string blanked", strings.Contains(masked, `"     "`), true)
}

// Swift interpolation with quotes inside it and raw strings are skipped
// whole; nested block comments end where they close, in Swift and Kotlin.
func TestMaskingSkipsInterpolationRawStringsAndNestedComments(t *testing.T) {
	masked := maskSource(`let s = "\(d["k"]) {"; let r = #"a "{" b"#; struct After {}`, swiftSyntax)
	want(t, "struct kept", strings.Contains(masked, "struct After {}"), true)
	want(t, "braces", strings.Count(masked, "{"), 1)
	for _, s := range []syntax{swiftSyntax, kotlinSyntax} {
		masked := maskSource("/* a /* b */ { */ class C {}", s)
		want(t, "nested comment", strings.HasPrefix(strings.TrimSpace(masked), "class C {}"), true)
	}
}

// TypeScript's template literals and regular expressions, and Python's
// prefixed and triple-quoted strings, are masked.
func TestMaskingTemplatesRegularExpressionsAndPythonStrings(t *testing.T) {
	masked := maskSource("const t = `a ${ {x: \"}\"}.x } {`; const r = /[{\"]/; class K {}", typescriptSyntax)
	want(t, "class kept", strings.Contains(masked, "class K {}"), true)
	want(t, "braces", strings.Count(masked, "{"), 1)
	masked = maskSource("x = rb\"class Fake:\"\n\"\"\"\nclass Doc:\n\"\"\"\nclass Real:\n    pass", pythonSyntax)
	want(t, "prefixed string", strings.Contains(masked, "Fake"), false)
	want(t, "triple-quoted string", strings.Contains(masked, "Doc"), false)
	want(t, "class kept", strings.Contains(masked, "class Real:"), true)
}

// A non-ASCII letter in a string blanks as many bytes as it takes, so the
// offsets stay Go's, and a type that holds the string shows one blank per
// letter, as Node showed it.
func TestNonASCIILettersBlankByByteAndShowByLetter(t *testing.T) {
	r := readTypeScript("class K {\n  kind: 'é' | '😀';\n}\n")
	want(t, "masked", strings.Contains(r.Masked, "'  ' | '    '"), true)
	want(t, "properties", named(byName(t, r, "K")), []string{"kind: ' ' | '  ' @02"})
}

// Swift: kinds, inheritance, nesting and stored properties; a computed
// property stores nothing, so it is left out.
func TestSwiftKindsInheritanceNestingAndStoredProperties(t *testing.T) {
	r := readSwift(lines("", "import Foundation", "", "/// A session. struct NotThis {}",
		"@MainActor public final class CheckoutSession: NSObject, CheckoutRunning, @unchecked Sendable {",
		"    private let clock: ShopClock", "    weak var delegate: CheckoutDelegate?", "    static let shared = CheckoutSession()",
		"    private lazy var reader = ReceiptReader(url: url)", "    var computed: Int { 1 }", "",
		"    struct Config: Codable {", "        let speed: Double", "    }", "",
		"    func run() {", "        let local: LocalOnly = LocalOnly()", "        class func notAType() {}", "    }", "}", "",
		"protocol CheckoutRunning: AnyObject {", "    var progress: CheckoutProgress { get }", "}", "",
		"enum PaymentKind: String, CaseIterable { case card }", "",
		"extension CheckoutSession: CustomStringConvertible where Self: AnyObject {", "    var description: String { \"\" }", "}", "",
		"struct Box<T: Equatable>: Container where T: Hashable {}", "actor Store {}"))
	want(t, "tops", tops(r), []string{"class CheckoutSession", "protocol CheckoutRunning", "enum PaymentKind", "extension CheckoutSession", "struct Box", "actor Store"})
	session := byName(t, r, "CheckoutSession")
	want(t, "inherits", session.Inherits, []string{"NSObject", "CheckoutRunning", "Sendable"})
	want(t, "properties", named(session), []string{"clock: ShopClock @06", "delegate: CheckoutDelegate? @07", "reader: ReceiptReader @09 inferred"})
	want(t, "parent", *byName(t, r, "Config").Parent, "CheckoutSession")
	want(t, "nested top", byName(t, r, "Config").Top, false)
	want(t, "lines", [2]int{session.Start, session.End}, [2]int{5, 20})
	want(t, "box", byName(t, r, "Box").Inherits, []string{"Container"})
	want(t, "extension", r.Declarations[4].Inherits, []string{"CustomStringConvertible"})
	for _, d := range r.Declarations {
		if d.Name == "NotThis" || d.Name == "notAType" {
			t.Errorf("%s is no declaration", d.Name)
		}
	}
}

// Swift views and tests are flagged.
func TestSwiftViewsAndTestsAreFlagged(t *testing.T) {
	r := readSwift("struct MapView: View { var body: some View { Text(\"\") } }\nfinal class MapTests: XCTestCase {}\n@Suite struct NewTests {}")
	want(t, "view", byName(t, r, "MapView").View, true)
	want(t, "test case", byName(t, r, "MapTests").Test, true)
	want(t, "suite", byName(t, r, "NewTests").Test, true)
}

// Kotlin: classes without bodies, constructors, supertypes and objects.
func TestKotlinClassesConstructorsSupertypesAndObjects(t *testing.T) {
	r := readKotlin(lines("", "package com.example", "", "data class Price(val amount: Double, private val source: PriceSource, count: Int)", "",
		"sealed interface Outcome", "", "class Uploader(", "    private val client: HttpClient,", ") : BaseUploader(client), Outcome, Closeable by client {",
		"    private val queue: UploadQueue = UploadQueue()", "    val lazyThing by lazy { Parser() }", "    companion object {",
		"        const val LIMIT = 3", "    }", "    inner class Part", "}", "", "object Registry : Lookup", "", "enum class Mode { FAST, SLOW }", "",
		"class UploaderTest {", "    @Test fun works() {}", "}"))
	want(t, "tops", tops(r), []string{"class Price", "interface Outcome", "class Uploader", "object Registry", "enum Mode", "class UploaderTest"})
	price := byName(t, r, "Price")
	want(t, "label", *price.Label, "data class")
	want(t, "price properties", named(price), []string{"amount: Double @04", "source: PriceSource @04"})
	uploader := byName(t, r, "Uploader")
	want(t, "inherits", uploader.Inherits, []string{"BaseUploader", "Outcome", "Closeable"})
	want(t, "uploader properties", named(uploader), []string{"client: HttpClient @09", "queue: UploadQueue @11", "lazyThing: Parser @12 inferred"})
	want(t, "parent", *byName(t, r, "Part").Parent, "Uploader")
	want(t, "sealed", *byName(t, r, "Outcome").Label, "sealed interface")
	want(t, "test", byName(t, r, "UploaderTest").Test, true)
}

// Java: extends, implements, records and fields.
func TestJavaExtendsImplementsRecordsAndFields(t *testing.T) {
	r := readJava(lines("", "package x;", "public abstract class Store<T> extends BaseStore<T> implements Closeable, Flushable {",
		"    private final Cache cache = new Cache();", "    private static Logger LOG;", "    protected List<Entry> entries;",
		"    public Store() { Local l = null; }", "    static class Inner {}", "}", "interface Flushable extends Thing {}",
		"record Point(int x, Label label) implements Shape {}"))
	want(t, "tops", tops(r), []string{"class Store", "interface Flushable", "record Point"})
	store := byName(t, r, "Store")
	want(t, "label", *store.Label, "abstract class")
	want(t, "inherits", store.Inherits, []string{"BaseStore", "Closeable", "Flushable"})
	want(t, "fields", named(store), []string{"cache: Cache @04", "entries: List<Entry> @06"})
	want(t, "record components", named(byName(t, r, "Point")), []string{"x: int @11", "label: Label @11"})
	want(t, "nested top", byName(t, r, "Inner").Top, false)
}

// TypeScript: classes, interfaces, aliases, namespaces and imports.
func TestTypeScriptClassesInterfacesAliasesNamespacesAndImports(t *testing.T) {
	r := readTypeScript(lines("", "import { Fetcher, type Options as Opts } from './fetcher';", `import * as path from "node:path";`,
		"import Default, { Named } from '../shared/named.js';", "export * from './everything';", "const lazy = await import('./lazy');",
		"const old = require('./old');", "// import { Ghost } from './ghost';", "", "@Injectable()",
		"export class Service extends Base<Item> implements Runner, Stopper {", "  private readonly cache: Map<string, Item> = new Map();",
		"  #secret = new Secret();", "  constructor(private readonly fetcher: Fetcher, plain: number) { super(); }", "  run(): void {}", "}", "",
		"export interface Runner extends Thing { run(): void; status: Status }", "export type Result = { ok: boolean; item: Item } | Failure;",
		"export const enum Flag { On, Off }", "export namespace Shapes { export class Circle {} }", "function inner() { class Local {} }"))
	want(t, "tops", tops(r), []string{"class Service", "interface Runner", "type Result", "enum Flag", "class Circle"})
	service := byName(t, r, "Service")
	want(t, "inherits", service.Inherits, []string{"Base", "Runner", "Stopper"})
	want(t, "fields", named(service), []string{"cache: Map<string, Item> @12", "#secret: Secret @13 inferred", "fetcher: Fetcher @14"})
	want(t, "interface fields", named(byName(t, r, "Runner")), []string{"status: Status @18"})
	want(t, "alias lines", byName(t, r, "Result").End >= byName(t, r, "Result").Start, true)
	want(t, "local top", byName(t, r, "Local").Top, false)
	want(t, "imports", r.Imports, []Import{
		{Spec: "./fetcher", Names: []string{"Fetcher", "Opts"}}, {Spec: "node:path"}, {Spec: "../shared/named.js", Names: []string{"Named", "Default"}},
		{Spec: "./everything"}, {Spec: "./lazy"}, {Spec: "./old"},
	})
}

// Python: classes, bases, labels, attributes and imports.
func TestPythonClassesBasesLabelsAttributesAndImports(t *testing.T) {
	r := readPython(lines("", "from __future__ import annotations", "from .store import (Store,", "    Entry as E)", "import os, pkg.sub.mod as m", "",
		`"""`, "class NotThis:", `"""`, "", "@dataclass(frozen=True)", "class Price(Base, metaclass=ABCMeta):", "    amount: float",
		"    source: PriceSource = None", "", "    def __init__(self, uploader: Uploader):", "        self.uploader: Uploader = uploader",
		"        self.queue = UploadQueue()", "        local: Ignored = 1", "", "    class Inner:", "        pass", "",
		"class Reader(Protocol):", "    def read(self) -> bytes: ...", "", "class Mode(Enum):", "    A = 1"))
	want(t, "tops", tops(r), []string{"class Price", "protocol Reader", "enum Mode"})
	price := byName(t, r, "Price")
	want(t, "label", *price.Label, "dataclass")
	want(t, "inherits", price.Inherits, []string{"Base"})
	want(t, "attributes", named(price), []string{"amount: float @13", "source: PriceSource @14", "uploader: Uploader @17", "queue: UploadQueue @18 inferred"})
	want(t, "lines", [2]int{price.Start, price.End}, [2]int{12, 22})
	want(t, "parent", *byName(t, r, "Inner").Parent, "Price")
	want(t, "imports", r.Imports, []Import{
		{Spec: "__future__", Names: []string{"annotations"}, Python: true}, {Spec: ".store", Names: []string{"Store", "E"}, Python: true},
		{Spec: "os", Python: true}, {Spec: "pkg.sub.mod", Python: true},
	})
}

// A span is byte offsets into the masked text, so slicing the text by it
// gives the declaration's code, and a body that never closes ends the span
// at the end of the text.
func TestASpanSlicesTheMaskedTextAtItsCode(t *testing.T) {
	r := readKotlin("// é\nclass Café {}\nclass Open {")
	d := byName(t, r, "Open")
	want(t, "unclosed", r.Masked[d.Span[0]:d.Span[1]], "class Open {")
	r = readSwift("let é = 1\nstruct Box { let x: Int }\n")
	d = byName(t, r, "Box")
	want(t, "closed", r.Masked[d.Span[0]:d.Span[1]], "struct Box { let x: Int }")
}

// The header helpers split a header at its top-level commas and read the
// bare type name an entry refers to.
func TestHeaderHelpers(t *testing.T) {
	want(t, "split", splitTopLevel("A<B, C>, D(e, f), G"), []string{"A<B, C>", "D(e, f)", "G"})
	for entry, name := range map[string]string{"@unchecked Sendable": "Sendable", "Foundation.NSObject": "NSObject", "Generic[T]": "Generic", "~Copyable": "Copyable"} {
		want(t, entry, bareTypeName(entry), name)
	}
	want(t, "identifiers", Identifiers("a.b(a, $c) _d"), []string{"a", "b", "$c", "_d"})
}

// A path's extension chooses its language, and so its family.
func TestLanguagesByExtension(t *testing.T) {
	want(t, "swift", LanguageFor("a/B.swift").Name, "Swift")
	want(t, "kotlin", LanguageFor("a/b.kt").Family, "jvm")
	want(t, "java", LanguageFor("a/b.java").Family, "jvm")
	want(t, "tsx", LanguageFor("a/b.tsx").Name, "TypeScript")
	want(t, "python", LanguageFor("a/b.py").Name, "Python")
	want(t, "markdown", LanguageFor("a/README.md") == nil, true)
	want(t, "jvm globs", Families["jvm"].Globs, []string{"*.kt", "*.kts", "*.java"})
}
