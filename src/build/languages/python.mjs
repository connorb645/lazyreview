/**
 * Reads the classes in a Python file, with their base classes and their
 * attributes, and the file's imports. A class runs until the first line
 * indented no deeper than its own `class` line. An attribute is one the
 * class body annotates, or one `__init__` gives `self`, typed or made by
 * calling a class; each is {name, type, line, inferred}, where `type` is the
 * written type, or the class called, with `inferred` true.
 */
import { maskSource, PYTHON_SYNTAX } from './source-mask.mjs';
import { skipBalanced, splitTopLevel, bareTypeName } from './header-text.mjs';
import { nestDeclarations } from './declaration-tree.mjs';

const CLASS = /^([ \t]*)class[ \t]+([A-Za-z_]\w*)[ \t]*(?=[(:\[])/;
const ENUMS = new Set(['Enum', 'IntEnum', 'StrEnum', 'Flag', 'IntFlag']);
const LABELS = { TypedDict: 'typed dict', NamedTuple: 'named tuple', ABC: 'abstract class' };

export function readPython(text) {
  const masked = maskSource(text, PYTHON_SYNTAX);
  const lines = masked.split('\n');
  const starts = [];
  let offset = 0;
  for (const line of lines) { starts.push(offset); offset += line.length + 1; }
  const indentOf = line => line.length - line.trimStart().length;
  const blank = line => !line.trim();

  const found = [];
  lines.forEach((line, i) => {
    const m = CLASS.exec(line);
    if (!m) return;
    const indent = m[1].replace(/\t/g, '    ').length;
    let k = starts[i] + m[0].length;
    let bases = [];
    if (masked[k] === '[') k = skipBalanced(masked, k);
    if (masked[k] === '(') {
      const close = skipBalanced(masked, k);
      bases = splitTopLevel(masked.slice(k + 1, close - 1));
      k = close;
    }
    const headerEnd = lineIndexOf(starts, masked.indexOf(':', k));
    let last = headerEnd;
    for (let j = headerEnd + 1; j < lines.length; j++) {
      if (blank(lines[j])) continue;
      if (indentOf(lines[j].replace(/\t/g, '    ')) <= indent) break;
      last = j;
    }
    const decorators = [];
    for (let j = i - 1; j >= 0 && /^\s*@/.test(lines[j]); j--) decorators.push(lines[j].trim().slice(1).split('(')[0]);
    const named = bases.filter(b => !/=/.test(b));
    const meta = bases.find(b => /^metaclass\s*=/.test(b)) || '';
    const inherits = named.map(b => bareTypeName(b)).filter(Boolean);
    found.push({
      name: m[2],
      kind: inherits.includes('Protocol') ? 'protocol' : inherits.some(b => ENUMS.has(b)) ? 'enum' : 'class',
      label: decorators.some(d => /(?:^|\.)dataclass$/.test(d)) ? 'dataclass'
        : /ABCMeta/.test(meta) ? 'abstract class'
          : (inherits.map(b => LABELS[b]).find(Boolean) || null),
      inherits,
      at: starts[i] + m[1].length,
      span: [starts[i], starts[last] + lines[last].length],
      indent,
      start: i + 1,
      end: last + 1,
      bodyLines: [headerEnd + 1, last],
      view: false,
      test: inherits.some(b => /TestCase$/.test(b)),
    });
  });

  const declarations = nestDeclarations(found, d => d.indent === 0).map(d => ({ ...d, properties: properties(lines, d) }));
  return { masked, declarations, imports: readImports(masked) };
}

function lineIndexOf(starts, offset) {
  let lo = 0, hi = starts.length - 1;
  while (lo < hi) {
    const mid = (lo + hi + 1) >> 1;
    if (starts[mid] <= offset) lo = mid; else hi = mid - 1;
  }
  return lo;
}

function properties(lines, declaration) {
  const [from, to] = declaration.bodyLines;
  const out = [];
  const add = (name, type, line, inferred) => { if (!out.some(p => p.name === name)) out.push({ name, type, line, inferred }); };
  let bodyIndent = null;
  let inInit = false;
  let initIndent = 0;
  for (let j = from; j <= to; j++) {
    const line = lines[j].replace(/\t/g, '    ');
    if (!line.trim()) continue;
    const indent = line.length - line.trimStart().length;
    if (bodyIndent === null) bodyIndent = indent;
    if (indent === bodyIndent) {
      inInit = /^\s*(?:async\s+)?def\s+__init__\b/.test(line);
      initIndent = indent;
      const annotated = /^\s*([A-Za-z_]\w*)\s*:\s*([^=\n]+)/.exec(line);
      if (annotated && !/^(?:def|class|if|for|while|with|try|except|else|elif|return|async|match|case)$/.test(annotated[1])) {
        add(annotated[1], annotated[2].trim(), j + 1, false);
      }
      continue;
    }
    if (!inInit || indent <= initIndent) continue;
    const typed = /\bself\.(\w+)\s*:\s*([^=\n]+)/.exec(line);
    if (typed) add(typed[1], typed[2].trim(), j + 1, false);
    const assigned = /\bself\.(\w+)\s*=\s*([A-Z]\w*)\s*\(/.exec(line);
    if (assigned) add(assigned[1], assigned[2], j + 1, true);
  }
  return out;
}

function readImports(masked) {
  const imports = [];
  const text = masked.replace(/\\\n/g, ' ');
  for (const m of text.matchAll(/^[ \t]*from[ \t]+(\.*[\w.]*)[ \t]+import[ \t]+(\([^)]*\)|[^\n#;]+)/gm)) {
    const names = m[2].replace(/[()]/g, ' ').split(',').map(s => s.trim().split(/\s+as\s+/).pop()).filter(s => /^\w+$/.test(s));
    imports.push({ spec: m[1], names: m[2].trim() === '*' ? null : names, python: true });
  }
  for (const m of text.matchAll(/^[ \t]*import[ \t]+([^\n#;]+)/gm)) {
    for (const part of m[1].split(',')) {
      const module = part.trim().split(/\s+as\s+/)[0];
      if (/^[\w.]+$/.test(module)) imports.push({ spec: module, names: null, python: true });
    }
  }
  return imports;
}
