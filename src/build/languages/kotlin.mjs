/**
 * Reads the type declarations in a Kotlin file: classes, interfaces, objects
 * and enums, with their supertypes and their properties, including those
 * declared in the primary constructor, each {name, type, line, inferred}:
 * `type` is the written type, or the type its initializer constructs, with
 * `inferred` true, or null. A property with a getter and no initializer
 * stores nothing and is left out.
 */
import { maskSource, KOTLIN_SYNTAX } from './source-mask.mjs';
import { indexSource } from './source-index.mjs';
import { skipBalanced, splitTopLevel, bareTypeName } from './header-text.mjs';
import { nestDeclarations } from './declaration-tree.mjs';

const MODIFIERS = 'public|private|internal|protected|open|abstract|sealed|data|enum|annotation|inner|value|inline|expect|actual|final|fun|override';
const DECLARATION = new RegExp(String.raw`(?<=^|[\s;{}()])((?:@[\w.:]+(?:\([^()\n]*\))?\s+)*)((?:(?:${MODIFIERS})\s+)*)(class|interface|object)\s+([A-Za-z_]\w*)`, 'gd');
const PROPERTY = /(?:^|[;{])[ \t]*((?:@[\w.:]+(?:\([^()\n]*\))?\s+)*)((?:(?:public|private|internal|protected|open|abstract|override|final|lateinit|const|inline|external|actual|expect)[ \t]+)*)(val|var)[ \t]+(\w+)[ \t]*(?::[ \t]*([^=\n{]+))?(?:(?:=|by)[ \t]*(?:lazy[ \t]*\{[ \t]*)?([A-Z]\w*)[ \t]*[({])?/gmd;
const PARAMETER = /^(?:@[\w.:]+(?:\([^()]*\))?\s+)*(?:(?:private|public|internal|protected|override|open|final)\s+)*(val|var)\s+(\w+)\s*:\s*([\s\S]+?)\s*(?:=[\s\S]*)?$/;

export function readKotlin(text) {
  const masked = maskSource(text, KOTLIN_SYNTAX);
  const index = indexSource(masked);
  const found = [];
  for (const m of masked.matchAll(DECLARATION)) {
    const modifiers = m[2];
    const word = m[3];
    if (word === 'object' && /\bcompanion\s*$/.test(masked.slice(Math.max(0, m.index - 12), m.index))) continue;
    const header = readHeader(masked, m.indices[4][1]);
    const keyword = m.indices[3][0];
    const end = header.open >= 0 ? index.close(header.open) + 1 : header.end;
    const supertypes = header.supertypes.map(s => s.replace(/\s+by\s+[\s\S]*$/, ''));
    found.push({
      name: m[4],
      kind: kindOf(word, modifiers),
      label: labelOf(word, modifiers),
      inherits: supertypes.map(bareTypeName).filter(Boolean),
      at: keyword,
      span: [m.index, end],
      body: header.open >= 0 ? [header.open, index.close(header.open)] : null,
      depth: index.depth(keyword),
      start: index.line(m.index),
      end: index.line(Math.max(m.index, end - 1)),
      parameters: header.parameters,
      parametersAt: header.parametersAt,
      view: false,
      test: false,
    });
  }
  const declarations = nestDeclarations(found, d => d.depth === 0).map(d => {
    const test = !!d.body && /@(?:Test|ParameterizedTest)\b/.test(text.slice(d.body[0], d.body[1]));
    return { ...d, properties: [...constructorProperties(masked, index, d), ...properties(masked, index, d)], test };
  });
  return { masked, declarations, imports: [] };
}

function kindOf(word, modifiers) {
  if (word === 'interface') return 'interface';
  if (word === 'object') return 'object';
  if (/\benum\b/.test(modifiers)) return 'enum';
  if (/\bannotation\b/.test(modifiers)) return 'annotation';
  return 'class';
}

function labelOf(word, modifiers) {
  if (word !== 'class') return /\bsealed\b/.test(modifiers) ? `sealed ${word}` : /\bfun\b/.test(modifiers) ? `fun ${word}` : null;
  for (const special of ['data', 'sealed', 'abstract', 'value', 'inline']) {
    if (new RegExp(`\\b${special}\\b`).test(modifiers)) return `${special === 'inline' ? 'value' : special} class`;
  }
  return null;
}

// Generics, the primary constructor, the supertypes and the opening brace.
// A Kotlin class may have no body, so the header can also end at a line end.
function readHeader(masked, k) {
  let j = k;
  const skipSpace = () => { while (masked[j] === ' ' || masked[j] === '\t') j++; };
  skipSpace();
  if (masked[j] === '<') { j = skipBalanced(masked, j); skipSpace(); }
  const constructor = /^(?:(?:@[\w.:]+(?:\([^()\n]*\))?\s+)*(?:(?:private|public|internal|protected)\s+)?constructor\s*)?(?=\()/.exec(masked.slice(j, j + 400));
  let parameters = [];
  let parametersAt = -1;
  if (constructor) {
    j += constructor[0].length;
    const close = skipBalanced(masked, j);
    parametersAt = j + 1;
    parameters = splitTopLevel(masked.slice(j + 1, close - 1));
    j = close;
  }
  let supertypeText = '';
  let open = -1;
  let depth = 0;
  let collecting = false;
  for (; j < masked.length; j++) {
    const c = masked[j];
    if (depth === 0) {
      if (c === '{') { open = j; break; }
      if (c === ';' || c === '}' || c === '=') break;
      if (c === '\n') {
        const before = (collecting ? supertypeText : masked.slice(k, j)).trimEnd();
        const after = masked.slice(j + 1, j + 200).trimStart();
        if (!/[,:]$/.test(before) && !/^(?:[:,{]|where\b)/.test(after)) break;
      }
      if (c === ':' && !collecting) { collecting = true; continue; }
    }
    if (c === '(' || c === '<' || c === '[') depth++;
    else if (c === ')' || (c === '>' && masked[j - 1] !== '-') || c === ']') depth = Math.max(0, depth - 1);
    if (collecting) supertypeText += c;
  }
  return {
    open,
    end: j,
    parameters,
    parametersAt,
    supertypes: splitTopLevel(supertypeText.split(/\bwhere\b/)[0]),
  };
}

function constructorProperties(masked, index, declaration) {
  const out = [];
  let cursor = declaration.parametersAt;
  for (const parameter of declaration.parameters) {
    const at = masked.indexOf(parameter.trim(), cursor);
    cursor = at + parameter.trim().length;
    const m = PARAMETER.exec(parameter.trim());
    if (m) out.push({ name: m[2], type: m[3].trim(), line: index.line(at), inferred: false });
  }
  return out;
}

function properties(masked, index, declaration) {
  if (!declaration.body) return [];
  const [open, close] = declaration.body;
  const inner = index.depth(open) + 1;
  const out = [];
  for (const m of masked.slice(open, close).matchAll(PROPERTY)) {
    const at = open + m.indices[3][0];
    if (index.depth(at) !== inner) continue;
    if (/\bconst\b/.test(m[2])) continue;
    const typed = m[5] || '';
    const accessor = /\s(?:get|set)\b/.exec(typed);
    const written = (accessor ? typed.slice(0, accessor.index) : typed).trim() || null;
    const after = open + (m.indices[5] ? m.indices[5][0] + written.length : m.indices[4][1]);
    const getterHere = !!accessor && /^\s+get\b/.test(typed.slice(accessor.index));
    const initialised = !getterHere && /^[ \t]*(?:=|by\b)/.test(masked.slice(after, close));
    const getterBelow = /^[^\n]*\n[ \t]*(?:(?:private|protected|internal|public)[ \t]+)?get\b/.test(masked.slice(after, close));
    if (getterHere || (getterBelow && !initialised)) continue;
    out.push({ name: m[4], type: written || m[6] || null, line: index.line(at), inferred: !written && !!m[6] });
  }
  return out;
}
