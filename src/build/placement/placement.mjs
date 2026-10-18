/**
 * Puts each file in a component and a package.
 *
 * placeFiles(paths, config, {context}) matches every path against the
 * config's patterns, the most specific pattern winning and the first in the
 * config breaking a tie. Changed files match the ordinary components; with
 * `context`, the paths are unchanged files shown as context and match only
 * components marked `context`, and whatever matches none goes to one
 * "Unchanged code" component. Files no pattern names are grouped by folder,
 * each group named by its folder as it is on disk; a group of the whole
 * repository takes `rootTitle`, such as the repository's name.
 *
 * Returns {components, placement}: the components that hold at least one
 * path, each with its packages, and a Map from path to {component, package}.
 */
import { pathPattern } from './path-pattern.mjs';
import { groupByFolder } from './folder-grouping.mjs';
import { english } from '../../i18n/english.mjs';

export const CONTEXT_COMPONENT = 'context';

export function placeFiles(paths, config, { context = false, rootTitle = english.diagram.repositoryRoot } = {}) {
  const declared = config.components.filter(c => !!c.context === context);
  const rules = [];
  declared.forEach(c => {
    (c.packages || []).forEach(p => (p.paths || []).forEach(text => rules.push({ pattern: pathPattern(text), component: c.id, package: p.id })));
    (c.paths || []).forEach(text => rules.push({ pattern: pathPattern(text), component: c.id, package: null }));
  });
  rules.forEach((r, i) => { r.order = i; });

  const assigned = new Map();
  const unmatched = [];
  for (const path of paths) {
    const hits = rules.filter(r => r.pattern.test(path));
    if (!hits.length) { unmatched.push(path); continue; }
    hits.sort((a, b) => b.pattern.specificity - a.pattern.specificity || a.order - b.order);
    assigned.set(path, { component: hits[0].component, package: hits[0].package });
  }

  const components = [];
  for (const c of declared) {
    const own = paths.filter(p => assigned.get(p)?.component === c.id);
    if (!own.length) continue;
    const packages = (c.packages || []).map(p => ({ ...p, paths: undefined }))
      .filter(p => own.some(path => assigned.get(path).package === p.id));
    const loose = own.filter(path => assigned.get(path).package === null);
    packages.push(...folderPackages(loose, c.id, packages.map(p => p.id)).map(p => {
      p.paths.forEach(path => assigned.set(path, { component: c.id, package: p.id }));
      return { id: p.id, title: p.title, description: '' };
    }));
    components.push(finish({ ...c, packages, paths: undefined, declared: true }, own, assigned));
  }

  if (unmatched.length) {
    const extra = context
      ? [{ folder: null, paths: unmatched }]
      : mergeSmall(groupByFolder(unmatched, { maxGroups: declared.length ? 4 : 8, splitAbove: Math.max(8, unmatched.length / 4), balanced: true }));
    const taken = new Set([...config.components.map(c => c.id), CONTEXT_COMPONENT]);
    for (const group of extra) {
      const id = context ? CONTEXT_COMPONENT : uniqueId(group.folder === null ? 'other' : slug(group.folder || 'root'), taken);
      const title = context ? english.diagram.unchangedCode : group.folder === null ? english.diagram.otherFiles : group.folder === '' ? rootTitle : titleFor(group.folder, extra);
      const packages = folderPackages(group.paths, id, [], context ? 6 : 3);
      packages.forEach(p => p.paths.forEach(path => assigned.set(path, { component: id, package: p.id })));
      components.push(finish({
        id, title,
        description: context ? english.diagram.contextDescription : '',
        context,
        declared: false,
        packages: packages.map(p => ({ id: p.id, title: p.title, description: '' })),
      }, group.paths, assigned));
    }
    components.sort((a, b) => {
      if (a.declared !== b.declared) return a.declared ? -1 : 1;
      if (a.declared) return 0;
      return (a.id === 'other') - (b.id === 'other') || b.size - a.size;
    });
  }
  return { components, placement: assigned };
}

function finish(component, paths, assigned) {
  const packages = component.packages.map(p => {
    const own = paths.filter(path => assigned.get(path).package === p.id);
    return { ...p, path: whereFiles(own) };
  });
  return {
    ...component,
    tag: component.tag || whereFiles(paths) || english.diagram.repositoryRoot,
    size: paths.length,
    packages,
  };
}

// Packages for files no config package names: one per folder group.
function folderPackages(paths, componentId, taken, splitAbove = 3) {
  if (!paths.length) return [];
  const used = new Set(taken);
  const base = commonFolder(paths).replace(/\/$/, '');
  const groups = groupByFolder(paths, { maxGroups: 12, splitAbove, splitTop: true });
  const baseName = base ? base.split('/').pop() : english.diagram.repositoryRoot;
  return groups.map(g => {
    const relative = g.folder === base ? '' : base ? g.folder.slice(base.length + 1) : g.folder;
    const title = relative || (groups.length > 1 && base ? english.diagram.topLevel(baseName) : baseName);
    return { id: uniqueId(slug(relative || 'top'), used), title, paths: g.paths };
  });
}

// Groups of three files or fewer are not worth a component each; they join any group of other files.
function mergeSmall(groups) {
  if (groups.length <= 3) return groups;
  const small = groups.filter(g => g.paths.length <= 3 || g.folder === null);
  if (small.length < 2) return groups;
  return [...groups.filter(g => g.paths.length > 3), { folder: null, paths: small.flatMap(g => g.paths) }];
}

function titleFor(folder, groups) {
  if (folder === '') return english.diagram.repositoryRoot;
  const segments = folder.split('/');
  const last = segments[segments.length - 1];
  const clash = groups.some(g => g.folder && g.folder !== folder && g.folder.split('/').pop() === last);
  return clash && segments.length > 1 ? segments.slice(-2).join('/') : last;
}

/**
 * Where a set of files lives, for a heading: their common folder, or when
 * they share none, the folders and root files that hold most of them, such
 * as "Tests/AppTests/ · Package.swift · scripts/".
 */
export function whereFiles(paths) {
  const common = commonFolder(paths);
  if (common || !paths.length) return common;
  const counts = new Map();
  for (const p of paths) {
    const parts = p.split('/');
    const key = parts.length > 1 ? `${parts[0]}/` : p;
    counts.set(key, (counts.get(key) || 0) + 1);
  }
  const places = [...counts.entries()].sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0])).map(([k]) => k);
  return places.slice(0, 3).join(' · ') + (places.length > 3 ? ' · …' : '');
}

/** The deepest folder holding every path, with a trailing slash, or '' at the root. */
export function commonFolder(paths) {
  if (!paths.length) return '';
  const folders = paths.map(p => p.split('/').slice(0, -1));
  const first = folders[0];
  let depth = first.length;
  for (const f of folders) {
    let k = 0;
    while (k < depth && k < f.length && f[k] === first[k]) k++;
    depth = k;
  }
  return depth ? `${first.slice(0, depth).join('/')}/` : '';
}

function slug(text) {
  return text.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '') || 'part';
}

function uniqueId(base, taken) {
  let id = base;
  for (let k = 2; taken.has(id); k++) id = `${base}-${k}`;
  taken.add(id);
  return id;
}
