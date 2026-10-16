/**
 * The relationships between the types on the map, read from their code.
 *
 * findRelations(nodes, imports) returns [{source, target, kind, label}]:
 *   realize   a type conforms to a protocol or implements an interface
 *   inherit   a type inherits from a class, or a protocol from a protocol
 *   has       a stored property holds the other type; the link names those
 *             properties in `properties`
 *   extend    a file of extensions extends the other type
 *   use       the code names the other type, or imports the other file
 *   test      any link from a test
 * A name resolves within its language family only, to the nearest type of
 * that name by folder when there are several. `imports` maps a path to the
 * file's imports, which link file boxes for code with no types of its own.
 */
import { identifiers } from '../languages/header-text.mjs';

const STRENGTH = { inherit: 5, realize: 5, has: 4, extend: 3, use: 2 };
const INTERFACES = new Set(['protocol', 'interface']);
const SCRIPT_EXTENSIONS = ['.ts', '.tsx', '.mts', '.cts', '.js', '.jsx', '.mjs', '.cjs'];

export function findRelations(nodes, imports = new Map()) {
  const byName = new Map();
  for (const n of nodes) {
    if (n.kind === 'artifact' || n.kind === 'extension' || !n.family) continue;
    const key = `${n.family}:${n.name}`;
    if (!byName.has(key)) byName.set(key, []);
    byName.get(key).push(n);
  }
  const resolve = (name, from) => nearest((byName.get(`${from.family}:${name}`) || []).filter(c => c !== from), from);

  const edges = new Map();
  const add = (source, target, kind, label = null, properties = null) => {
    if (!target || source === target) return;
    const key = `${source.id}>${target.id}`;
    const had = edges.get(key);
    if (had && STRENGTH[had.kind] >= STRENGTH[kind]) return;
    edges.set(key, { source: source.id, target: target.id, kind, label, ...(properties ? { properties } : {}) });
  };

  for (const n of nodes) {
    if (!n.family) continue;
    for (const s of n.inherits) {
      const t = resolve(s, n);
      if (t) add(n, t, INTERFACES.has(t.kind) && !INTERFACES.has(n.kind) ? 'realize' : 'inherit');
    }
    for (const e of n.extends) add(n, resolve(e, n), 'extend');
    const holders = new Map();
    for (const p of n.properties) {
      for (const word of identifiers(p.type || '')) {
        const t = resolve(word, n);
        if (!t) continue;
        if (!holders.has(t)) holders.set(t, []);
        if (!holders.get(t).includes(p.name)) holders.get(t).push(p.name);
      }
    }
    for (const [t, names] of holders) add(n, t, 'has', n.view && t.view ? 'shows' : null, names);
    for (const word of identifiers(n.code)) {
      if (word === n.name) continue;
      const t = resolve(word, n);
      if (t) add(n, t, n.view && t.view ? 'has' : 'use', n.view && t.view ? 'shows' : null);
    }
  }

  const inFile = new Map();
  for (const n of nodes) {
    if (!inFile.has(n.file)) inFile.set(n.file, []);
    inFile.get(n.file).push(n);
  }
  const paths = [...inFile.keys()];
  for (const [path, list] of imports) {
    for (const i of list) {
      const target = i.python ? pythonTarget(path, i.spec, i.names, paths) : scriptTarget(path, i.spec, paths);
      if (!target || target === path) continue;
      const from = inFile.get(path) || [];
      const to = inFile.get(target) || [];
      for (const s of from) {
        const words = s.kind === 'artifact' ? null : identifiers(s.code);
        for (const t of to) {
          const named = i.names === null || i.names.includes(t.name);
          if (!words && (t.kind === 'artifact' || named)) add(s, t, 'use');
          else if (words && t.kind === 'artifact' && i.names && i.names.some(name => words.has(name))) add(s, t, 'use');
        }
      }
    }
  }

  const list = [...edges.values()];
  const tests = new Set(nodes.filter(n => n.test).map(n => n.id));
  for (const e of list) if (tests.has(e.source)) e.kind = 'test';
  return list;
}

function nearest(candidates, from) {
  if (candidates.length <= 1) return candidates[0] || null;
  const shared = c => {
    if (c.file === from.file) return Infinity;
    const a = c.file.split('/'), b = from.file.split('/');
    let k = 0;
    while (k < a.length && k < b.length && a[k] === b[k]) k++;
    return k;
  };
  return [...candidates].sort((x, y) => shared(y) - shared(x))[0];
}

function normalize(parts) {
  const out = [];
  for (const p of parts) {
    if (p === '' || p === '.') continue;
    if (p === '..') out.pop(); else out.push(p);
  }
  return out.join('/');
}

function scriptTarget(from, spec, paths) {
  if (!spec.startsWith('.')) return null;
  const folder = from.split('/').slice(0, -1);
  const base = normalize([...folder, ...spec.split('/')]);
  const stem = base.replace(/\.(?:[cm]?js|jsx)$/, '');
  const tries = [base, ...SCRIPT_EXTENSIONS.map(e => stem + e), ...SCRIPT_EXTENSIONS.map(e => `${base}/index${e}`)];
  return tries.find(t => paths.includes(t)) || null;
}

function pythonTarget(from, spec, names, paths) {
  const dots = /^\.*/.exec(spec)[0].length;
  const module = spec.slice(dots).split('.').filter(Boolean);
  const modules = [module, ...(names || []).map(n => [...module, n])];
  const files = parts => [`${parts.join('/')}.py`, `${parts.join('/')}/__init__.py`];
  if (dots) {
    const folder = from.split('/').slice(0, -1);
    const up = folder.slice(0, folder.length - (dots - 1));
    for (const m of modules.slice().reverse()) {
      const hit = files([...up, ...m]).map(f => normalize(f.split('/'))).find(f => paths.includes(f));
      if (hit) return hit;
    }
    return null;
  }
  for (const m of modules.slice().reverse()) {
    if (!m.length) continue;
    const hit = paths.find(p => files(m).some(f => p === f || p.endsWith(`/${f}`)));
    if (hit) return hit;
  }
  return null;
}

