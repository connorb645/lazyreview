/**
 * Reads the type declarations in a Java file: classes, interfaces, enums,
 * records and annotation types, with what each extends or implements and its
 * fields and record components, each {name, type, line, inferred}, where
 * `inferred` is always false: Java writes every field's type.
 */
import { maskSource, JAVA_SYNTAX } from './source-mask.mjs';
import { indexSource } from './source-index.mjs';
import { skipBalanced, splitTopLevel, bareTypeName } from './header-text.mjs';
import { nestDeclarations } from './declaration-tree.mjs';

const DECLARATION = /(?<=^|[\s;{}()])((?:@[\w.]+(?:\([^()\n]*\))?\s+)*)((?:(?:public|private|protected|static|final|abstract|sealed|non-sealed|strictfp)\s+)*)(class|interface|enum|record|@interface)\s+([A-Za-z_]\w*)/gd;
const FIELD = /(?:^|[;{}])[ \t]*((?:@[\w.]+(?:\([^()\n]*\))?\s+)*)((?:(?:private|protected|public|static|final|transient|volatile)[ \t]+)*)([A-Z][\w.]*(?:<[^;=(){}]*>)?(?:\[\])*)[ \t]+(\w+)[ \t]*(?:=[ \t]*(?:new[ \t]+([A-Z]\w*))?|;)/gmd;

export function readJava(text) {
  const masked = maskSource(text, JAVA_SYNTAX);
  const index = indexSource(masked);
  const found = [];
  for (const m of masked.matchAll(DECLARATION)) {
    const header = readHeader(masked, m.indices[4][1], m[3]);
    if (!header) continue;
    const keyword = m.indices[3][0];
    found.push({
      name: m[4],
      kind: m[3] === '@interface' ? 'annotation' : m[3] === 'interface' ? 'interface' : m[3],
      label: /\babstract\b/.test(m[2]) && m[3] === 'class' ? 'abstract class' : null,
      inherits: [...header.extends, ...header.implements].map(bareTypeName).filter(Boolean),
      at: keyword,
      span: [m.index, index.close(header.open) + 1],
      body: [header.open, index.close(header.open)],
      depth: index.depth(keyword),
      start: index.line(m.index),
      end: index.line(index.close(header.open)),
      recordComponents: header.components,
      componentsAt: header.componentsAt,
      view: false,
      test: false,
    });
  }
  const declarations = nestDeclarations(found, d => d.depth === 0).map(d => {
    const test = /@(?:Test|ParameterizedTest)\b/.test(text.slice(d.body[0], d.body[1]));
    return { ...d, properties: [...components(masked, index, d), ...fields(masked, index, d)], test };
  });
  return { masked, declarations, imports: [] };
}

function readHeader(masked, k, word) {
  let j = k;
  while (/\s/.test(masked[j] || '')) j++;
  if (masked[j] === '<') j = skipBalanced(masked, j);
  let components = [];
  let componentsAt = -1;
  if (word === 'record') {
    while (/\s/.test(masked[j] || '')) j++;
    if (masked[j] === '(') {
      const close = skipBalanced(masked, j);
      componentsAt = j + 1;
      components = splitTopLevel(masked.slice(j + 1, close - 1));
      j = close;
    }
  }
  const rest = masked.slice(j, j + 4000);
  const brace = rest.indexOf('{');
  if (brace < 0) return null;
  const clause = rest.slice(0, brace);
  if (/[;}=]/.test(clause)) return null;
  const part = keyword => {
    const m = new RegExp(`\\b${keyword}\\b([\\s\\S]*?)(?=\\b(?:extends|implements|permits)\\b|$)`).exec(clause);
    return m ? splitTopLevel(m[1]) : [];
  };
  return { open: j + brace, extends: part('extends'), implements: part('implements'), components, componentsAt };
}

function components(masked, index, declaration) {
  const out = [];
  let cursor = declaration.componentsAt;
  for (const component of declaration.recordComponents) {
    const text = component.trim();
    const at = masked.indexOf(text, cursor);
    cursor = at + text.length;
    const m = /^([\s\S]*\S)\s+(\w+)$/.exec(text.replace(/^(?:@[\w.]+(?:\([^()]*\))?\s+)*/, ''));
    if (m) out.push({ name: m[2], type: m[1], line: index.line(at), inferred: false });
  }
  return out;
}

function fields(masked, index, declaration) {
  const [open, close] = declaration.body;
  const inner = index.depth(open) + 1;
  const out = [];
  for (const m of masked.slice(open, close).matchAll(FIELD)) {
    const at = open + m.indices[3][0];
    if (index.depth(at) !== inner) continue;
    if (/\bstatic\b/.test(m[2]) || /^(?:return|throw|new|else|case)$/.test(m[3])) continue;
    out.push({ name: m[4], type: m[3], line: index.line(at), inferred: false });
  }
  return out;
}
