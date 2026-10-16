/**
 * Finds the unchanged types the change touches, as candidates for context.
 *
 * findContextCandidates({git, head, drawn, changedPaths, include}) looks both
 * ways: types the changed code names, found by searching the repository at
 * `head` for their declarations, and unchanged code that names a changed
 * type. Types named in `include` are found whatever links them. It
 * reads only those files, at most `maxFiles` of them, and returns
 * {candidates, texts}: a node for each unchanged top-level type that links
 * to the change, and the text of every file read, by path.
 *
 * findContextExtensions({git, head, types, skip}) finds, for the unchanged
 * types the map draws as context, the files at `head` that extend them, and
 * returns {nodes, texts}: one context node per such file, holding every
 * extension it declares of those types, and each file's text. A file in
 * `skip`, a changed file or one a drawn type comes from, is on the map
 * already and is left out, as is test code, which is never drawn as context.
 */
import { languageFor, FAMILIES } from '../languages/languages.mjs';
import { identifiers } from '../languages/header-text.mjs';
import { typeNode, extensionName, isTestPath } from '../model/changed-types.mjs';

export function findContextCandidates({ git, head, drawn, changedPaths, include = [], maxFiles = 600 }) {
  const changed = new Set(changedPaths);
  const families = [...new Set(drawn.map(n => n.family).filter(Boolean))];
  const outward = new Map();
  const inward = new Map();
  const wanted = new Map();

  for (const family of families) {
    const { keywords, globs } = FAMILIES[family];
    const declared = new Map();
    const pattern = `(${keywords.join('|')})[[:space:]]+[A-Za-z_]`;
    const scan = new RegExp(`\\b(?:${keywords.join('|')})\\s+([A-Za-z_]\\w*)`, 'g');
    for (const hit of git.grep(head, pattern, globs)) {
      if (changed.has(hit.path)) continue;
      for (const m of hit.text.matchAll(scan)) {
        if (!declared.has(m[1])) declared.set(m[1], new Set());
        declared.get(m[1]).add(hit.path);
      }
    }
    const own = drawn.filter(n => n.family === family);
    const drawnNames = new Set(own.map(n => n.name));
    const named = new Set();
    for (const n of own) {
      for (const word of identifiers([n.code, ...n.properties.map(p => p.type || ''), ...n.inherits, ...n.extends].join(' '))) {
        if (declared.has(word) && !drawnNames.has(word)) named.add(word);
      }
    }
    for (const name of include) if (declared.has(name)) named.add(name);
    for (const name of named) {
      for (const path of declared.get(name)) {
        if (!outward.has(path)) outward.set(path, new Set());
        outward.get(path).add(name);
      }
    }
    const existing = own.filter(n => n.state === 'changed' && n.kind !== 'artifact').map(n => n.name)
      .concat(own.flatMap(n => n.extends));
    if (existing.length) {
      for (const [path, count] of git.countMentions(head, [...new Set(existing)], globs)) {
        if (!changed.has(path)) inward.set(path, count);
      }
    }
    wanted.set(family, new Set(existing));
  }

  const paths = [...outward.keys()].sort();
  const rest = [...inward.entries()].filter(([p]) => !outward.has(p)).sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0]));
  for (const [p] of rest) {
    if (paths.length >= maxFiles) break;
    paths.push(p);
  }
  const texts = git.read(head, paths.slice(0, maxFiles));

  const candidates = [];
  for (const [path, text] of texts) {
    const language = languageFor(path);
    if (!language || text == null) continue;
    const read = language.read(text);
    const names = outward.get(path);
    const changedNames = wanted.get(language.family) || new Set();
    for (const d of read.declarations) {
      if (!d.top || d.kind === 'extension') continue;
      const code = read.masked.slice(d.span[0], d.span[1]);
      const named = names && names.has(d.name);
      const naming = !named && [...identifiers(code)].some(w => changedNames.has(w));
      if (!named && !naming) continue;
      const node = typeNode({ path }, language, d, code, []);
      node.state = 'context';
      candidates.push(node);
    }
  }
  return { candidates, texts };
}

export function findContextExtensions({ git, head, types, skip }) {
  const nodes = [];
  const texts = new Map();
  for (const family of [...new Set(types.map(t => t.family).filter(Boolean))]) {
    const names = new Set(types.filter(t => t.family === family).map(t => t.name));
    const words = [...names].filter(n => /^[A-Za-z_]\w*$/.test(n));
    if (!words.length) continue;
    const pattern = `extension[[:space:]]+(${words.join('|')})([^A-Za-z0-9_]|$)`;
    const paths = [...new Set(git.grep(head, pattern, FAMILIES[family].globs).map(hit => hit.path))].filter(p => !skip.has(p)).sort();
    for (const [path, text] of git.read(head, paths)) {
      const language = languageFor(path);
      if (!language || text == null) continue;
      const read = language.read(text);
      const found = read.declarations.filter(d => d.top && d.kind === 'extension' && names.has(d.name.split('.')[0]));
      if (!found.length || isTestPath(path)) continue;
      const node = typeNode({ path }, language, found[0], found.map(d => read.masked.slice(d.span[0], d.span[1])).join('\n'), []);
      node.kind = 'extension';
      node.extends = [...new Set(found.map(d => d.name.split('.')[0]))];
      node.inherits = [...new Set(found.flatMap(d => d.inherits))];
      node.name = extensionName(path, node.extends);
      node.state = 'context';
      nodes.push(node);
      texts.set(path, text);
    }
  }
  return { nodes, texts };
}
