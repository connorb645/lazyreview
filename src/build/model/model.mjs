/**
 * Puts the map's model together: components and their packages, the types
 * placed in them, the relationships between the types, groups of look-alike
 * types and the arrows between components.
 *
 * buildModel({files, nodes, imports, config, rootTitle}) takes the diff's files with
 * their counts, every node to draw (changed, unchanged in a changed file, or
 * unchanged context), the files' imports and the read config. It returns the
 * model the page draws, with nothing in it the page does not use. On a
 * stack's page, files and nodes carry `pulls`, the pull requests that changed
 * them, and files `byPull`, each one's added and removed lines.
 */
import { placeFiles, CONTEXT_COMPONENT } from '../placement/placement.mjs';
import { findRelations } from './relations.mjs';
import { COLORS } from '../config/config.mjs';

const GROUP_AT = 6;

export function buildModel({ files, nodes: given, imports, config, rootTitle }) {
  const nodes = given.map(n => ({ ...n }));
  assignIds(nodes);

  const changedPlacement = placeFiles(files.map(f => f.path), config, { rootTitle });
  const contextPaths = [...new Set(nodes.filter(n => n.state === 'context').map(n => n.file))];
  const contextPlacement = placeFiles(contextPaths, config, { context: true });
  const place = path => changedPlacement.placement.get(path) || contextPlacement.placement.get(path);
  for (const n of nodes) Object.assign(n, place(n.file));

  let edges = findRelations(nodes, imports);
  const byId = new Map(nodes.map(n => [n.id, n]));
  const groups = makeGroups(nodes, edges, config);
  edges = regroupEdges(edges, groups, byId);
  edges = applyLinks(edges, config.links, nodes, groups);

  const components = colorComponents([...changedPlacement.components, ...contextPlacement.components]);
  const componentIds = new Set(components.map(c => c.id));
  const areas = config.areas
    .map(a => ({ ...a, components: a.components.filter(id => componentIds.has(id)) }))
    .filter(a => a.components.length);
  const areaOf = new Map(areas.flatMap(a => a.components.map(id => [id, a])));
  for (const c of components) {
    c.area = areaOf.get(c.id)?.id || null;
    c.group = c.group || areaOf.get(c.id)?.title || '';
  }

  const names = new Set(nodes.map(n => n.name));
  for (const n of nodes) n.stereotype = stereotypeOf(n, names);

  const componentOf = id => (id.startsWith('group:') ? groups.find(g => `group:${g.id}` === id).component : byId.get(id).component);
  const summary = summarize(edges, componentOf, config.summaries, components.length);

  return {
    components: components.map(c => ({
      id: c.id, title: c.title, description: c.description, tag: c.tag, color: c.color, group: c.group,
      area: c.area, context: !!c.context,
      // A lone package that only repeats its component's name or folder is drawn without a header.
      packages: c.packages.map(p => ({
        id: p.id, title: p.title, description: p.description || '', path: p.path,
        bare: c.packages.length === 1 && !p.description && (p.title === c.title || p.path === c.tag),
      })),
    })),
    areas: areas.map(a => ({ id: a.id, title: a.title, description: a.description, components: a.components })),
    files: files.map((f, index) => ({
      path: f.path, oldPath: f.oldPath, status: f.status, adds: f.adds, dels: f.dels, binary: !!f.binary, collapsed: !!f.collapsed,
      pulls: f.pulls, byPull: f.byPull, index, ...changedPlacement.placement.get(f.path),
    })),
    contextFiles: contextPaths.map(path => ({ path, ...contextPlacement.placement.get(path) })),
    nodes: nodes.map(n => ({
      id: n.id, name: n.name, kind: n.kind, stereotype: n.stereotype, file: n.file, line: n.line === Infinity ? 1 : n.line,
      component: n.component, package: n.package, state: n.state,
      inherits: n.inherits, nested: n.nested, extends: n.extends, extFiles: n.extFiles,
      view: n.view, test: n.test, group: n.group || null, pulls: n.pulls,
    })),
    edges,
    groups: groups.map(g => ({ id: g.id, title: g.title, description: g.description, members: g.members, target: g.target })),
    summary,
  };
}

function assignIds(nodes) {
  const count = new Map();
  for (const n of nodes) if (n.kind !== 'artifact' && n.kind !== 'extension') count.set(n.name, (count.get(n.name) || 0) + 1);
  const taken = new Set();
  for (const n of nodes) {
    let id = n.kind === 'artifact' ? `file:${n.file}` : n.kind === 'extension' ? `ext:${n.file}` : count.get(n.name) > 1 ? `${n.name}@${n.file}` : n.name;
    for (let k = 2; taken.has(id); k++) id = `${id}#${k}`;
    taken.add(id);
    n.id = id;
  }
}

// Many types in one package that all conform to, or inherit from, the same
// type are drawn as one box holding them all, with one arrow for the lot.
function makeGroups(nodes, edges, config) {
  const groups = [];
  const grouped = new Set();
  const byId = new Map(nodes.map(n => [n.id, n]));
  const structural = edges.filter(e => e.kind === 'realize' || e.kind === 'inherit');

  function add(id, title, description, members, target, kind) {
    if (members.length < 2) return;
    const first = byId.get(members[0]);
    members.forEach(m => { byId.get(m).group = id; grouped.add(m); });
    groups.push({ id, title: `${members.length} ${title}`, description, members, target, kind, component: first.component, package: first.package });
  }

  for (const g of config.groups) {
    const target = nodes.find(n => n.name === g.of && n.kind !== 'artifact');
    const links = structural.filter(e => target && e.target === target.id && !grouped.has(e.source));
    const packages = new Map();
    for (const e of links) {
      const n = byId.get(e.source);
      const key = `${n.component}/${n.package}`;
      if (!packages.has(key)) packages.set(key, []);
      packages.get(key).push(e);
    }
    const best = [...packages.values()].sort((a, b) => b.length - a.length)[0] || [];
    add(g.id, g.title, g.description, best.map(e => e.source), target && target.id, best[0]?.kind || 'realize');
  }

  const buckets = new Map();
  for (const e of structural) {
    const n = byId.get(e.source);
    if (grouped.has(n.id) || n.test) continue;
    const key = `${n.component}/${n.package}>${e.target}>${e.kind}`;
    if (!buckets.has(key)) buckets.set(key, []);
    buckets.get(key).push(n.id);
  }
  for (const [key, members] of buckets) {
    const fresh = members.filter(m => !grouped.has(m));
    if (fresh.length < GROUP_AT) continue;
    const [, target, kind] = key.split('>');
    const name = byId.get(target).name;
    const id = `of-${name.toLowerCase().replace(/[^a-z0-9]+/g, '-')}-${groups.length + 1}`;
    add(id, kind === 'realize' ? `types conforming to ${name}` : `types inheriting from ${name}`, '', fresh, target, kind);
  }
  return groups;
}

function regroupEdges(edges, groups, byId) {
  if (!groups.length) return edges;
  const out = edges.filter(e => !groups.some(g => g.members.includes(e.source) && e.target === g.target && (e.kind === g.kind)));
  for (const g of groups) {
    if (!g.target || !byId.has(g.target)) continue;
    out.push({ source: `group:${g.id}`, target: g.target, kind: g.kind, label: g.kind === 'realize' ? 'each conforms to' : 'each inherits from' });
  }
  return out;
}

// The config's named links: a label on the link that is already there, or a
// new one when the code reading missed it.
function applyLinks(edges, links, nodes, groups) {
  const find = name => {
    if (name.startsWith('group:')) return groups.some(g => `group:${g.id}` === name) ? name : null;
    const exact = nodes.find(n => n.id === name);
    if (exact) return exact.id;
    const named = nodes.filter(n => n.name === name);
    return named.length === 1 ? named[0].id : null;
  };
  const out = edges.map(e => ({ ...e, key: false }));
  for (const l of links) {
    const source = find(l.from), target = find(l.to);
    if (!source || !target) continue;
    const had = out.find(e => e.source === source && e.target === target);
    if (had) Object.assign(had, { label: l.label, key: true, kind: l.kind === 'has' ? 'has' : had.kind === 'test' ? 'use' : had.kind });
    else out.push({ source, target, kind: l.kind, label: l.label, key: true });
  }
  return out;
}

function summarize(edges, componentOf, labels, componentCount) {
  const pairs = new Map();
  for (const e of edges) {
    const a = componentOf(e.source), b = componentOf(e.target);
    if (a === b) continue;
    const key = `${a}>${b}`;
    pairs.set(key, (pairs.get(key) || 0) + 1);
  }
  const named = new Map(labels.map(l => [`${l.from}>${l.to}`, l.label]));
  return [...pairs.entries()]
    .filter(([key, count]) => count >= 2 || named.has(key) || componentCount <= 4)
    .map(([key, count]) => {
      const [source, target] = key.split('>');
      return { source, target, count, label: named.get(key) || null };
    })
    .sort((a, b) => a.source.localeCompare(b.source) || a.target.localeCompare(b.target));
}

// «protocol», «data class» or «extension» of X, then the supertypes the map
// does not draw, since a drawn one has its own arrow.
function stereotypeOf(n, names) {
  if (n.kind === 'artifact') return '«file»';
  if (n.kind === 'extension') return `«extension»${n.extends.length ? ` of ${n.extends.join(', ')}` : ''}`;
  const supers = n.inherits.filter(s => !names.has(s));
  const head = n.label ? `«${n.label}»` : n.kind === 'class' ? '' : `«${n.kind}»`;
  const tail = supers.length ? `: ${supers.slice(0, 2).join(', ')}${supers.length > 2 ? ', …' : ''}` : '';
  return [head, tail].filter(Boolean).join(' ');
}

function colorComponents(components) {
  const used = new Set(components.map(c => c.color).filter(Boolean));
  const free = COLORS.filter(c => c !== 'grey' && !used.has(c));
  let k = 0;
  return components.map(c => {
    if (c.color) return c;
    if (c.id === CONTEXT_COMPONENT || c.context) return { ...c, color: 'grey' };
    const color = free.length ? free[k++ % free.length] : COLORS[k++ % (COLORS.length - 1)];
    return { ...c, color };
  });
}
