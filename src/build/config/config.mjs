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
export const COLORS = ['blue', 'teal', 'purple', 'amber', 'olive', 'rose', 'indigo', 'slate', 'grey'];

function problemsError(problems) {
  return new Error(`the config has ${problems.length} problem${problems.length === 1 ? '' : 's'}:\n${problems.map(p => `  - ${p}`).join('\n')}`);
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
  const where = (kind, i, item) => `${kind} ${item && typeof item.id === 'string' ? `"${item.id}"` : i + 1}`;

  function keys(item, allowed, label) {
    if (!isObject(item)) { problems.push(`${label} is not an object`); return false; }
    for (const k of Object.keys(item)) if (!allowed.includes(k)) problems.push(`${label} has an unknown key "${k}"`);
    return true;
  }
  function text(item, key, label, required) {
    const v = item[key];
    if (v === undefined) { if (required) problems.push(`${label} needs "${key}"`); return; }
    if (typeof v !== 'string' || !v.trim()) problems.push(`${label}: "${key}" must be a non-empty string`);
  }
  function id(item, label, seen) {
    text(item, 'id', label, true);
    if (typeof item.id !== 'string') return;
    if (!/^[a-z][a-z0-9-]*$/.test(item.id)) problems.push(`${label}: id must be lower case letters, digits and dashes`);
    if (seen.has(item.id)) problems.push(`${label}: id "${item.id}" is used twice`);
    seen.add(item.id);
  }
  function list(item, key, label) {
    const v = item[key];
    if (v === undefined) return [];
    if (!Array.isArray(v)) { problems.push(`${label}: "${key}" must be a list`); return []; }
    return v;
  }
  function strings(item, key, label) {
    const v = list(item, key, label);
    if (v.some(s => typeof s !== 'string' || !s)) problems.push(`${label}: "${key}" must hold non-empty strings`);
    return v;
  }

  if (!keys(config, KEYS.root, 'the config')) throw problemsError(problems);
  text(config, 'title', 'the config');
  text(config, 'summary', 'the config');

  const componentIds = new Set();
  const components = list(config, 'components', 'the config').map((c, i) => {
    const label = where('component', i, c);
    if (!keys(c, KEYS.component, label)) return null;
    id(c, label, componentIds);
    text(c, 'title', label, true);
    text(c, 'description', label);
    text(c, 'tag', label);
    text(c, 'group', label);
    if (c.color !== undefined && !COLORS.includes(c.color)) problems.push(`${label}: color must be one of ${COLORS.join(', ')}`);
    if (c.context !== undefined && typeof c.context !== 'boolean') problems.push(`${label}: "context" must be true or false`);
    const paths = strings(c, 'paths', label);
    const packageIds = new Set();
    const packages = list(c, 'packages', label).map((p, j) => {
      const plabel = `${label}, ${where('package', j, p)}`;
      if (!keys(p, KEYS.package, plabel)) return null;
      id(p, plabel, packageIds);
      text(p, 'title', plabel, true);
      text(p, 'description', plabel);
      const ppaths = strings(p, 'paths', plabel);
      if (!ppaths.length) problems.push(`${plabel} needs at least one path`);
      return { id: p.id, title: p.title, description: p.description || '', paths: ppaths };
    }).filter(Boolean);
    if (!paths.length && !packages.length) problems.push(`${label} needs paths or packages`);
    return {
      id: c.id, title: c.title, description: c.description || '', tag: c.tag || '', color: c.color || null,
      group: c.group || '', context: !!c.context, paths, packages,
    };
  }).filter(Boolean);

  const areaIds = new Set();
  const inArea = new Set();
  const areas = list(config, 'areas', 'the config').map((a, i) => {
    const label = where('area', i, a);
    if (!keys(a, KEYS.area, label)) return null;
    id(a, label, areaIds);
    text(a, 'title', label, true);
    text(a, 'description', label);
    const members = strings(a, 'components', label);
    if (!members.length) problems.push(`${label} needs at least one component`);
    for (const m of members) {
      if (inArea.has(m)) problems.push(`${label}: component "${m}" is already in another area`);
      inArea.add(m);
    }
    return { id: a.id, title: a.title, description: a.description || '', components: members };
  }).filter(Boolean);

  const links = list(config, 'links', 'the config').map((l, i) => {
    const label = `link ${i + 1}`;
    if (!keys(l, KEYS.link, label)) return null;
    text(l, 'from', label, true);
    text(l, 'to', label, true);
    text(l, 'label', label, true);
    if (l.kind !== undefined && !['use', 'has'].includes(l.kind)) problems.push(`${label}: kind must be use or has`);
    return { from: l.from, to: l.to, label: l.label, kind: l.kind || 'use' };
  }).filter(Boolean);

  const summaries = list(config, 'summaries', 'the config').map((s, i) => {
    const label = `summary ${i + 1}`;
    if (!keys(s, KEYS.summary, label)) return null;
    text(s, 'from', label, true);
    text(s, 'to', label, true);
    text(s, 'label', label, true);
    return { from: s.from, to: s.to, label: s.label };
  }).filter(Boolean);

  const groupIds = new Set();
  const groups = list(config, 'groups', 'the config').map((g, i) => {
    const label = where('group', i, g);
    if (!keys(g, KEYS.group, label)) return null;
    id(g, label, groupIds);
    text(g, 'of', label, true);
    text(g, 'title', label, true);
    text(g, 'description', label);
    return { id: g.id, of: g.of, title: g.title, description: g.description || '' };
  }).filter(Boolean);

  let contextTypes = { include: [], exclude: [], max: 40 };
  if (config.contextTypes !== undefined && keys(config.contextTypes, KEYS.contextTypes, 'contextTypes')) {
    const c = config.contextTypes;
    if (c.max !== undefined && !(Number.isInteger(c.max) && c.max >= 0)) problems.push('contextTypes: max must be a whole number, 0 or more');
    contextTypes = { include: strings(c, 'include', 'contextTypes'), exclude: strings(c, 'exclude', 'contextTypes'), max: c.max ?? 40 };
  }

  const known = new Set([...componentIds, 'context']);
  for (const a of areas) for (const m of a.components) if (!known.has(m)) problems.push(`area "${a.id}" names component "${m}", which the config does not declare`);
  for (const s of summaries) for (const end of [s.from, s.to]) if (!known.has(end)) problems.push(`a summary names component "${end}", which the config does not declare`);

  if (problems.length) throw problemsError(problems);
  return { title: config.title || '', summary: config.summary || '', areas, components, links, summaries, groups, contextTypes };
}
