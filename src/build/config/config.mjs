/**
 * Reads and checks a map config: the components, packages, areas and links a change is drawn with.
 *
 * readConfig(json) returns the config with every optional part filled in, or
 * throws an error listing every problem at once. With no config at all,
 * readConfig(undefined) returns the empty config, which groups files by folder.
 *
 * The shape (every key optional):
 *   title        the page's name
 *   summary      one line saying what the diff is, in place of the branch line
 *   areas        [{id, title, description, components: [component ids]}]
 *   components   [{id, title, description, tag, color, group, context,
 *                  paths: [patterns], packages: [{id, title, description, paths}]}]
 *   links        [{from, to, label, kind}]  named links between types; kind is
 *                use (the default) or has
 *   summaries    [{from, to, label}]  words on the arrow between two components
 *   groups       [{id, of, title, description}]  types conforming to `of`,
 *                drawn as one box
 *   contextTypes {include: [names], exclude: [names], max}
 */
import { english } from '../../i18n/english.mjs';

export const COLORS = ['blue', 'teal', 'purple', 'amber', 'olive', 'rose', 'indigo', 'slate', 'grey'];

function problemsError(problems) {
  return new Error(english.components.problems(problems));
}

const KEYS = {
  root: ['title', 'summary', 'areas', 'components', 'links', 'summaries', 'groups', 'contextTypes'],
  area: ['id', 'title', 'description', 'components'],
  component: ['id', 'title', 'description', 'tag', 'color', 'group', 'context', 'paths', 'packages'],
  package: ['id', 'title', 'description', 'paths'],
  link: ['from', 'to', 'label', 'kind'],
  summary: ['from', 'to', 'label'],
  group: ['id', 'of', 'title', 'description'],
  contextTypes: ['include', 'exclude', 'max'],
};

export function readConfig(json) {
  const problems = [];
  const config = json === undefined ? {} : json;
  const isObject = v => v && typeof v === 'object' && !Array.isArray(v);
  const idOf = item => (item && typeof item.id === 'string' ? item.id : null);

  function keys(item, allowed, label) {
    if (!isObject(item)) { problems.push(english.components.notAnObject(label)); return false; }
    for (const k of Object.keys(item)) if (!allowed.includes(k)) problems.push(english.components.unknownKey(label, k));
    return true;
  }
  function text(item, key, label, required) {
    const v = item[key];
    if (v === undefined) { if (required) problems.push(english.components.needs(label, key)); return; }
    if (typeof v !== 'string' || !v.trim()) problems.push(english.components.notText(label, key));
  }
  function id(item, label, seen) {
    text(item, 'id', label, true);
    if (typeof item.id !== 'string') return;
    if (!/^[a-z][a-z0-9-]*$/.test(item.id)) problems.push(english.components.badId(label));
    if (seen.has(item.id)) problems.push(english.components.idTwice(label, item.id));
    seen.add(item.id);
  }
  function list(item, key, label) {
    const v = item[key];
    if (v === undefined) return [];
    if (!Array.isArray(v)) { problems.push(english.components.notAList(label, key)); return []; }
    return v;
  }
  function strings(item, key, label) {
    const v = list(item, key, label);
    if (v.some(s => typeof s !== 'string' || !s)) problems.push(english.components.notStrings(label, key));
    return v;
  }

  const root = english.components.root;
  if (!keys(config, KEYS.root, root)) throw problemsError(problems);
  text(config, 'title', root);
  text(config, 'summary', root);

  const componentIds = new Set();
  const components = list(config, 'components', root).map((c, i) => {
    const label = english.components.component(idOf(c), i + 1);
    if (!keys(c, KEYS.component, label)) return null;
    id(c, label, componentIds);
    text(c, 'title', label, true);
    text(c, 'description', label);
    text(c, 'tag', label);
    text(c, 'group', label);
    if (c.color !== undefined && !COLORS.includes(c.color)) problems.push(english.components.badColor(label, COLORS));
    if (c.context !== undefined && typeof c.context !== 'boolean') problems.push(english.components.badContext(label));
    const paths = strings(c, 'paths', label);
    const packageIds = new Set();
    const packages = list(c, 'packages', label).map((p, j) => {
      const plabel = english.components.package(label, idOf(p), j + 1);
      if (!keys(p, KEYS.package, plabel)) return null;
      id(p, plabel, packageIds);
      text(p, 'title', plabel, true);
      text(p, 'description', plabel);
      const ppaths = strings(p, 'paths', plabel);
      if (!ppaths.length) problems.push(english.components.needsPath(plabel));
      return { id: p.id, title: p.title, description: p.description || '', paths: ppaths };
    }).filter(Boolean);
    if (!paths.length && !packages.length) problems.push(english.components.needsPaths(label));
    return {
      id: c.id, title: c.title, description: c.description || '', tag: c.tag || '', color: c.color || null,
      group: c.group || '', context: !!c.context, paths, packages,
    };
  }).filter(Boolean);

  const areaIds = new Set();
  const inArea = new Set();
  const areas = list(config, 'areas', root).map((a, i) => {
    const label = english.components.area(idOf(a), i + 1);
    if (!keys(a, KEYS.area, label)) return null;
    id(a, label, areaIds);
    text(a, 'title', label, true);
    text(a, 'description', label);
    const members = strings(a, 'components', label);
    if (!members.length) problems.push(english.components.needsComponent(label));
    for (const m of members) {
      if (inArea.has(m)) problems.push(english.components.inTwoAreas(label, m));
      inArea.add(m);
    }
    return { id: a.id, title: a.title, description: a.description || '', components: members };
  }).filter(Boolean);

  const links = list(config, 'links', root).map((l, i) => {
    const label = english.components.link(i + 1);
    if (!keys(l, KEYS.link, label)) return null;
    text(l, 'from', label, true);
    text(l, 'to', label, true);
    text(l, 'label', label, true);
    if (l.kind !== undefined && !['use', 'has'].includes(l.kind)) problems.push(english.components.badKind(label));
    return { from: l.from, to: l.to, label: l.label, kind: l.kind || 'use' };
  }).filter(Boolean);

  const summaries = list(config, 'summaries', root).map((s, i) => {
    const label = english.components.summary(i + 1);
    if (!keys(s, KEYS.summary, label)) return null;
    text(s, 'from', label, true);
    text(s, 'to', label, true);
    text(s, 'label', label, true);
    return { from: s.from, to: s.to, label: s.label };
  }).filter(Boolean);

  const groupIds = new Set();
  const groups = list(config, 'groups', root).map((g, i) => {
    const label = english.components.group(idOf(g), i + 1);
    if (!keys(g, KEYS.group, label)) return null;
    id(g, label, groupIds);
    text(g, 'of', label, true);
    text(g, 'title', label, true);
    text(g, 'description', label);
    return { id: g.id, of: g.of, title: g.title, description: g.description || '' };
  }).filter(Boolean);

  let contextTypes = { include: [], exclude: [], max: 40 };
  const contextLabel = english.components.contextTypes;
  if (config.contextTypes !== undefined && keys(config.contextTypes, KEYS.contextTypes, contextLabel)) {
    const c = config.contextTypes;
    if (c.max !== undefined && !(Number.isInteger(c.max) && c.max >= 0)) problems.push(english.components.badMax);
    contextTypes = { include: strings(c, 'include', contextLabel), exclude: strings(c, 'exclude', contextLabel), max: c.max ?? 40 };
  }

  const known = new Set([...componentIds, 'context']);
  for (const a of areas) for (const m of a.components) if (!known.has(m)) problems.push(english.components.areaUnknown(a.id, m));
  for (const s of summaries) for (const end of [s.from, s.to]) if (!known.has(end)) problems.push(english.components.summaryUnknown(end));

  if (problems.length) throw problemsError(problems);
  return { title: config.title || '', summary: config.summary || '', areas, components, links, summaries, groups, contextTypes };
}
