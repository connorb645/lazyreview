/**
 * Reads the type declarations in a TypeScript or JavaScript file: classes,
 * interfaces, enums and type aliases, with what each extends or implements,
 * its fields, and the file's imports. Each field, and each constructor
 * parameter that declares one, is {name, type, line, inferred}: `type` is
 * the written type, or the class its initializer constructs, with
 * `inferred` true, or null. A getter is no field.
 */
import { maskSource, TYPESCRIPT_SYNTAX } from './source-mask.mjs';
import { indexSource } from './source-index.mjs';
import { skipBalanced, splitTopLevel, bareTypeName } from './header-text.mjs';
import { nestDeclarations } from './declaration-tree.mjs';

const DECLARATION = /(?<=^|[;{}\n])[ \t]*((?:@[\w.]+(?:\([^()]*\))?\s+)*)((?:(?:export|default|declare|abstract|const)\s+)*)(class|interface|enum|type|namespace|module)\s+([A-Za-z_$][\w$]*)/gd;
const FIELD = /(?:^|[;{])[ \t]*((?:@[\w.]+(?:\([^()\n]*\))?\s+)*)((?:(?:public|private|protected|readonly|declare|override|abstract|static|accessor)[ \t]+)*)(#?[A-Za-z_$][\w$]*)[?!]?[ \t]*(?::[ \t]*([^=;\n]+?))?[ \t]*(?:=[ \t]*(?:new[ \t]+([A-Z][\w$]*))?[^;\n]*)?(?=;|$)/gmd;
const CONSTRUCTOR_PARAMETER = /^(?:(?:public|private|protected|readonly|override)\s+)+([A-Za-z_$][\w$]*)\??\s*:\s*([\s\S]+?)\s*(?:=[\s\S]*)?$/;
const IMPORT = /(?:^|[;\n])[ \t]*(?:import|export)\s+(?:type\s+)?([\s\S]*?)\s*from\s*(['"])([^'"\n]*)\2|(?:^|[;\n])[ \t]*import\s*(['"])([^'"\n]*)\4|\brequire\s*\(\s*(['"])([^'"\n]*)\6\s*\)|\bimport\s*\(\s*(['"])([^'"\n]*)\8\s*\)/gm;

export function readTypeScript(text) {
  const masked = maskSource(text, TYPESCRIPT_SYNTAX);
  const index = indexSource(masked);
  const found = [];
  for (const m of masked.matchAll(DECLARATION)) {
    const word = m[3];
    const keyword = m.indices[3][0];
    const header = word === 'type' ? readAlias(masked, index, m.indices[4][1]) : readHeader(masked, m.indices[4][1]);
    if (!header) continue;
    const end = header.open >= 0 ? index.close(header.open) + 1 : header.end;
    found.push({
      name: m[4],
      kind: word,
      label: /\babstract\b/.test(m[2]) && word === 'class' ? 'abstract class' : null,
      inherits: [...header.extends, ...header.implements].map(bareTypeName).filter(Boolean),
      at: keyword,
      span: [m.index + m[0].length - m[0].trimStart().length, end],
      body: header.open >= 0 ? [header.open, index.close(header.open)] : null,
      depth: index.depth(keyword),
      start: index.line(m.index + m[0].length - m[0].trimStart().length),
      end: index.line(Math.max(keyword, end - 1)),
      view: false,
      test: false,
    });
  }
  const containers = found.filter(d => d.kind === 'namespace' || d.kind === 'module');
  const insideContainerOnly = d => d.depth === containers.filter(c => c.span[0] < d.at && d.at < c.span[1]).length;
  const declarations = nestDeclarations(found.filter(d => d.kind !== 'namespace' && d.kind !== 'module'), insideContainerOnly)
    .map(d => ({ ...d, properties: properties(masked, index, d) }));
  return { masked, declarations, imports: readImports(text, masked) };
}

function readHeader(masked, k) {
  let j = k;
  while (/\s/.test(masked[j] || '')) j++;
  if (masked[j] === '<') j = skipBalanced(masked, j);
  const rest = masked.slice(j, j + 4000);
  const brace = rest.search(/\{/);
  if (brace < 0) return null;
  const clause = rest.slice(0, brace);
  if (/[;}=]/.test(clause.replace(/=>/g, ''))) return null;
  const part = keyword => {
    const m = new RegExp(`\\b${keyword}\\b([\\s\\S]*?)(?=\\b(?:extends|implements)\\b|$)`).exec(clause);
    return m ? splitTopLevel(m[1]) : [];
  };
  return { open: j + brace, extends: part('extends'), implements: part('implements') };
}

// A type alias runs from its name to the semicolon or line end that closes
// it, wherever no bracket is open.
function readAlias(masked, index, k) {
  let j = k;
  while (/\s/.test(masked[j] || '')) j++;
  if (masked[j] === '<') j = skipBalanced(masked, j);
  while (/\s/.test(masked[j] || '')) j++;
  if (masked[j] !== '=') return null;
  const from = j + 1;
  let depth = 0;
  for (j = from; j < masked.length; j++) {
    const c = masked[j];
    if ('([{<'.includes(c)) depth++;
    else if (')]}'.includes(c) || (c === '>' && masked[j - 1] !== '=')) depth = Math.max(0, depth - 1);
    else if (depth === 0 && c === ';') break;
    else if (depth === 0 && c === '\n') {
      const next = masked.slice(j + 1, j + 80).trimStart();
      if (!/^[|&?:]/.test(next) && !/[|&=:,(<{]\s*$/.test(masked.slice(from, j))) break;
    }
  }
  return { open: -1, end: j, extends: [], implements: [] };
}

function properties(masked, index, declaration) {
  if (declaration.kind === 'type' || !declaration.body || declaration.kind === 'enum') return [];
  const [open, close] = declaration.body;
  const inner = index.depth(open) + 1;
  const out = [];
  for (const m of masked.slice(open, close).matchAll(FIELD)) {
    const at = open + m.indices[3][0];
    if (index.depth(at) !== inner) continue;
    if (/\bstatic\b/.test(m[2])) continue;
    const written = m[4] ? m[4].trim() : null;
    out.push({ name: m[3], type: written || m[5] || null, line: index.line(at), inferred: !written && !!m[5] });
  }
  const constructor = /\bconstructor\s*\(/.exec(masked.slice(open, close));
  if (constructor) {
    const start = open + constructor.index + constructor[0].length - 1;
    let cursor = start + 1;
    for (const parameter of splitTopLevel(masked.slice(start + 1, skipBalanced(masked, start) - 1))) {
      const at = masked.indexOf(parameter.trim(), cursor);
      cursor = at + parameter.trim().length;
      const m = CONSTRUCTOR_PARAMETER.exec(parameter.trim());
      if (m) out.push({ name: m[1], type: m[2], line: index.line(at), inferred: false });
    }
  }
  return out;
}

// Imports are found in the masked text, so commented-out ones never count,
// and their module names are read back from the original at the same place.
function readImports(text, masked) {
  const imports = [];
  for (const m of masked.matchAll(IMPORT)) {
    const groups = [[3, 2], [5, 4], [7, 6], [9, 8]];
    const hit = groups.find(([g]) => m[g] !== undefined);
    if (!hit) continue;
    const start = m.index + m[0].lastIndexOf(m[hit[1]] + m[hit[0]]) + 1;
    const spec = text.slice(start, start + m[hit[0]].length);
    imports.push({ spec, names: hit[0] === 3 ? importedNames(m[1]) : null });
  }
  return imports;
}

function importedNames(clause) {
  if (/^\s*\*/.test(clause)) return null;
  const names = [];
  const braces = /\{([\s\S]*)\}/.exec(clause);
  if (braces) {
    for (const entry of splitTopLevel(braces[1])) {
      const m = /^(?:type\s+)?([\w$]+)(?:\s+as\s+([\w$]+))?$/.exec(entry.trim());
      if (m) names.push(m[2] || m[1]);
    }
  }
  const head = clause.replace(/\{[\s\S]*\}/, '').split(',')[0].trim();
  if (/^[\w$]+$/.test(head)) names.push(head);
  return names;
}
