/**
 * Reads the type declarations in a Swift file: classes, structs, enums,
 * protocols, actors and extensions, with what each inherits or conforms to
 * and its stored properties, each {name, type, line, inferred}: `type` is
 * the written type, or the type its initializer constructs, with `inferred`
 * true, or null. A computed property, whose name or type is followed by a
 * body that is not only willSet or didSet, stores nothing and is left out.
 */
import { maskSource, SWIFT_SYNTAX } from './source-mask.mjs';
import { indexSource } from './source-index.mjs';
import { skipBalanced, splitTopLevel, bareTypeName } from './header-text.mjs';
import { nestDeclarations } from './declaration-tree.mjs';

const ATTRIBUTE = String.raw`@[\w.]+(?:\([^()\n]*\))?\s+`;
const MODIFIER = String.raw`(?:public|internal|private|fileprivate|open|final|indirect|nonisolated|package|dynamic|(?:public|internal|private|fileprivate|package)\(set\))\s+`;
const DECLARATION = new RegExp(String.raw`(?<=^|[\s;{}()])((?:${ATTRIBUTE})*)((?:${MODIFIER})*)(class|struct|enum|protocol|actor|extension)\s+([A-Za-z_]\w*(?:\.[A-Za-z_]\w*)*)`, 'gd');
const PROPERTY = /(?:^|[;{])[ \t]*((?:@[\w.]+(?:\([^()\n]*\))?\s+)*)((?:(?:public|internal|private|fileprivate|open|final|weak|unowned(?:\(\w+\))?|lazy|nonisolated|static|class|override|dynamic|package|(?:public|internal|private|fileprivate|package)\(set\))[ \t]+)*)(let|var)[ \t]+(\w+)[ \t]*(?::[ \t]*([^={\n]+))?(?:=[ \t]*([A-Z]\w*)(?:\.\w+)*[ \t]*[({])?/gmd;

const NOT_NAMES = new Set(['func', 'var', 'let', 'subscript', 'init', 'deinit', 'case', 'typealias', 'static', 'class',
  'override', 'convenience', 'required', 'mutating', 'some', 'any', 'where', 'protocol']);
const VIEW_SUPERTYPES = new Set(['View', 'App', 'Scene', 'ViewModifier', 'Shape', 'UIViewRepresentable',
  'UIViewControllerRepresentable', 'NSViewRepresentable', 'NSViewControllerRepresentable']);
const TEST_SUPERTYPES = new Set(['XCTestCase', 'QuickSpec', 'AsyncSpec']);

export function readSwift(text) {
  const masked = maskSource(text, SWIFT_SYNTAX);
  const index = indexSource(masked);
  const found = [];
  for (const m of masked.matchAll(DECLARATION)) {
    const [attributes, , kind, name] = [m[1], m[2], m[3], m[4]];
    if (NOT_NAMES.has(name)) continue;
    const header = readHeader(masked, m.indices[4][1]);
    if (!header) continue;
    const keyword = m.indices[3][0];
    const inherits = header.inherits.map(bareTypeName).filter(Boolean);
    found.push({
      name,
      kind,
      label: null,
      inherits,
      at: keyword,
      span: [m.index, index.close(header.open) + 1],
      body: [header.open, index.close(header.open)],
      depth: index.depth(keyword),
      start: index.line(m.index),
      end: index.line(index.close(header.open)),
      view: inherits.some(s => VIEW_SUPERTYPES.has(s)),
      test: inherits.some(s => TEST_SUPERTYPES.has(s)) || /@Suite\b/.test(attributes),
    });
  }
  const declarations = nestDeclarations(found, d => d.depth === 0).map(d => ({
    ...d,
    properties: properties(masked, index, d),
  }));
  return { masked, declarations, imports: [] };
}

// Everything between the name and the opening brace: generics, the
// inheritance clause and a where clause. Null when this is not a declaration.
function readHeader(masked, k) {
  let j = k;
  while (masked[j] === ' ' || masked[j] === '\t') j++;
  if (masked[j] === '<') j = skipBalanced(masked, j);
  const rest = masked.slice(j, j + 4000);
  const brace = rest.indexOf('{');
  if (brace < 0) return null;
  const clause = rest.slice(0, brace);
  if (/[;}=]/.test(clause) || /^\s*\(/.test(clause)) return null;
  const colon = clause.match(/^\s*:/);
  const inheritText = colon ? clause.slice(colon[0].length).split(/\bwhere\b/)[0] : '';
  if (!colon && clause.trim() && !/^\s*where\b/.test(clause)) return null;
  return { open: j + brace, inherits: splitTopLevel(inheritText) };
}

function properties(masked, index, declaration) {
  if (declaration.kind === 'extension' || declaration.kind === 'protocol') return [];
  const [open, close] = declaration.body;
  const inner = index.depth(open) + 1;
  const out = [];
  for (const m of masked.slice(open, close).matchAll(PROPERTY)) {
    const at = open + m.indices[3][0];
    if (index.depth(at) !== inner) continue;
    if (/\b(?:static|class)\b/.test(m[2])) continue;
    if (!m[6] && computed(masked, index, open + (m.indices[5] || m.indices[4])[1])) continue;
    const written = m[5] ? m[5].trim() : null;
    out.push({ name: m[4], type: written || m[6] || null, line: index.line(at), inferred: !written && !!m[6] });
  }
  return out;
}

function computed(masked, index, end) {
  const brace = /^[ \t]*\{/.exec(masked.slice(end, end + 200));
  if (!brace) return false;
  const open = end + brace[0].length - 1;
  return !/^\s*(?:willSet|didSet)\b/.test(masked.slice(open + 1, index.close(open)));
}
