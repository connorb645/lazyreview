/**
 * The types each changed file declares, and which of them the change touches.
 *
 * readChangedTypes(files) takes the diff's files, each
 * {path, status, hunks, text, baseText, binary}, and returns:
 *   nodes      one per type the change adds or edits, one per file of
 *              extensions to types it does not draw, and one file box per
 *              file it draws nothing else for
 *   untouched  the other top-level types in those files, which may still be
 *              drawn as unchanged context
 *   imports    a Map from path to the file's imports
 *
 * Every node carries `code`: its masked source, which the relationships are
 * read from, and `family`, so a name only ever resolves within one language
 * family.
 */
import { languageFor } from '../languages/languages.mjs';
import { touches } from '../diff/diff-parser.mjs';

export function readChangedTypes(files) {
  const nodes = [];
  const untouched = [];
  const pendingExtensions = [];
  const conformances = [];
  const imports = new Map();

  for (const file of files) {
    const language = languageFor(file.path);
    const text = file.status === 'D' ? file.baseText : file.text;
    if (!language || file.binary || text == null) {
      nodes.push(fileBox(file, null, null));
      continue;
    }
    const read = language.read(text);
    const tops = read.declarations.filter(d => d.top);
    const nested = nestedNames(read.declarations);
    const baseNames = file.status === 'M' || file.status === 'R'
      ? new Set((file.baseText ? language.read(file.baseText).declarations : []).filter(d => d.top).map(d => d.name))
      : new Set();
    const before = nodes.length;

    for (const d of tops) {
      const touched = file.status === 'A' || file.status === 'D' || touches(file.hunks, d.start, d.end);
      const code = read.masked.slice(d.span[0], d.span[1]);
      if (d.kind === 'extension') {
        const extended = d.name.split('.')[0];
        if (touched) pendingExtensions.push({ file, language, extended, declaration: d, code });
        else conformances.push({ extended, family: language.family, inherits: d.inherits });
        continue;
      }
      const node = typeNode(file, language, d, code, nested.get(d.name) || []);
      if (file.status === 'D') node.state = 'removed';
      else if (touched) node.state = file.status === 'A' || !baseNames.has(d.name) ? 'new' : 'changed';
      else { node.state = 'same'; untouched.push(node); continue; }
      nodes.push(node);
    }
    // A file box stands for code with no types; a file whose types the
    // change leaves alone keeps no code here, so it pulls in no context.
    if (nodes.length === before && !pendingExtensions.some(p => p.file === file)) {
      nodes.push(fileBox(file, language, tops.length ? '' : read.masked));
    }
    if (read.imports.length) imports.set(file.path, read.imports);
  }

  // A changed extension of a type the map draws as changed is part of that
  // type. Any other changed extension stands for its file.
  const byName = new Map();
  for (const n of nodes) if (n.kind !== 'artifact') byName.set(`${n.family}:${n.name}`, n);
  const extensionFiles = new Map();
  for (const p of pendingExtensions) {
    // A type whose own file extends it is changed by that extension.
    const local = untouched.findIndex(u => u.file === p.file.path && u.name === p.extended);
    if (local >= 0) {
      const [promoted] = untouched.splice(local, 1);
      promoted.state = 'changed';
      nodes.push(promoted);
      byName.set(`${promoted.family}:${promoted.name}`, promoted);
    }
    const owner = byName.get(`${p.language.family}:${p.extended}`);
    if (owner && owner.state !== 'removed') {
      owner.code += `\n${p.code}`;
      owner.inherits = union(owner.inherits, p.declaration.inherits);
      if (p.file.path !== owner.file) owner.extFiles = union(owner.extFiles, [p.file.path]);
      continue;
    }
    let node = extensionFiles.get(p.file.path);
    if (!node) {
      node = extensionNode(p.file, p.language);
      extensionFiles.set(p.file.path, node);
      nodes.push(node);
    }
    node.extends = union(node.extends, [p.extended]);
    node.code += `\n${p.code}`;
    node.line = Math.min(node.line, p.declaration.start);
  }
  for (const node of extensionFiles.values()) node.name = extensionName(node.file, node.extends);
  for (const c of conformances) {
    const owner = byName.get(`${c.family}:${c.extended}`) || untouched.find(u => u.family === c.family && u.name === c.extended);
    if (owner) owner.inherits = union(owner.inherits, c.inherits);
  }
  return { nodes, untouched, imports };
}

/** A node for a type read from a file, before it is given a state. */
export function typeNode(file, language, d, code, nested) {
  return {
    id: null,
    name: d.name,
    kind: d.kind,
    label: d.label || null,
    file: file.path,
    line: d.start,
    end: d.end,
    family: language.family,
    inherits: d.inherits,
    properties: d.properties || [],
    nested,
    view: !!d.view,
    test: !!d.test || isTestPath(file.path),
    code,
    extFiles: [],
    extends: [],
  };
}

function fileBox(file, language, masked) {
  return {
    id: null,
    name: baseName(file.path),
    kind: 'artifact',
    label: null,
    file: file.path,
    line: 1,
    family: language ? language.family : null,
    inherits: [],
    properties: [],
    nested: [],
    view: false,
    test: isTestPath(file.path),
    code: masked || '',
    extFiles: [],
    extends: [],
    state: file.status === 'A' ? 'new' : file.status === 'D' ? 'removed' : 'changed',
  };
}

function extensionNode(file, language) {
  return { ...fileBox(file, language, ''), kind: 'extension', line: Infinity, state: file.status === 'A' ? 'new' : 'changed' };
}

/** The name a box of extensions shows: its file's name, or what it extends. */
export function extensionName(path, extended) {
  const stem = baseName(path).replace(/\.[^.]+$/, '');
  return stem.includes('+') ? stem : stem === extended[0] ? `${stem} extension` : `${extended[0]} · ${stem}`;
}

/** Whether a path holds test code, by the usual folder and file names. */
export function isTestPath(path) {
  return /(?:^|\/)(?:Tests?|tests?|__tests__|spec|androidTest|testFixtures)\//.test(path) ||
    /(?:Tests?|Spec)\.(?:swift|kt|java)$/.test(path) ||
    /(?:^|\/)test_[^/]*\.py$|_test\.py$|\.(?:test|spec)\.[cm]?[jt]sx?$/.test(path);
}

function nestedNames(declarations) {
  const out = new Map();
  for (const d of declarations) {
    if (d.top || d.kind === 'extension' || d.name === 'CodingKeys' || !d.owner || d.owner === d.name) continue;
    if (!out.has(d.owner)) out.set(d.owner, []);
    out.get(d.owner).push(d.name);
  }
  return out;
}

function union(a, b) {
  return [...new Set([...(a || []), ...(b || [])])];
}

function baseName(path) {
  return path.split('/').pop();
}
