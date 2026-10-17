/**
 * Lays the map out with ELK in three passes: the types inside each package,
 * the packages inside each component, and the components, with any areas
 * around them, on the canvas.
 *
 * layoutModel(model, elk) returns everything the page draws, in canvas
 * coordinates: {w, h, areas, components, boxes, routes, pkgArrows, summaries,
 * farTitle}.
 * `boxes` holds every type, group and package frame by id; `routes` are the
 * links drawn at rest, with the points ELK routed them along and where their
 * label sits. An arrow between packages, in `pkgArrows`, and each way of an
 * arrow between components, in `summaries`, keep the links between types
 * they count, as [source, target].
 */
import { measure, nodeSize, farTitleRoom, PACKAGE_HEAD, PACKAGE_HEAD_WITH_NOTE, COMPONENT_HEAD, FAR_TITLE } from './sizes.mjs';
import { sectionPoints, labelCentre, offsetRoute, shiftToWidth } from './geometry.mjs';
import { layoutCanvas } from './canvas-layout.mjs';
import { english } from '../../i18n/english.mjs';

// Drawn at rest inside a package: its structure, and the links the config names.
const AT_REST = e => ['realize', 'inherit', 'has', 'extend'].includes(e.kind) || (e.kind === 'use' && e.key);
// Supertypes sit above their subtypes, so those links point up for the layout.
const UPWARD = e => ['realize', 'inherit', 'extend'].includes(e.kind);
const LABELLED = e => e.label && (e.key || e.kind !== 'has');

export async function layoutModel(model, elk) {
  const nodes = new Map(model.nodes.map(n => [n.id, n]));
  const groups = new Map(model.groups.map(g => [`group:${g.id}`, g]));
  const laidAs = id => (nodes.get(id)?.group ? `group:${nodes.get(id).group}` : id);
  const homeOf = id => {
    const n = groups.has(id) ? nodes.get(groups.get(id).members[0]) : nodes.get(id);
    return n ? { component: n.component, package: n.package } : null;
  };

  const results = {};
  for (const c of model.components) {
    results[c.id] = await layoutComponent(c, model, elk, nodes, groups, laidAs, homeOf);
  }
  return { ...(await layoutCanvas(model, elk, results)), farTitle: FAR_TITLE };
}

async function layoutPackage(component, pkg, model, elk, nodes, groups, laidAs) {
  const members = model.nodes.filter(n => n.component === component.id && n.package === pkg.id);
  if (!members.length) return null;
  const loose = members.filter(n => !n.group);
  const groupIds = [...new Set(members.filter(n => n.group).map(n => n.group))];
  const ids = new Set([...loose.map(n => n.id), ...groupIds.map(g => `group:${g}`)]);
  const taken = new Set();
  const edges = [];
  for (const e of model.edges) {
    if (!AT_REST(e)) continue;
    const s = laidAs(e.source), t = laidAs(e.target);
    if (s === t || !ids.has(s) || !ids.has(t)) continue;
    const key = `${s}>${t}`;
    if (taken.has(key)) continue;
    taken.add(key);
    const [a, b] = UPWARD(e) ? [t, s] : [s, t];
    const label = LABELLED(e) ? e.label : null;
    edges.push({ id: `e${edges.length}`, sources: [a], targets: [b], model: { ...e, source: s, target: t, label }, flipped: UPWARD(e),
      labels: label ? [{ text: label, ...measure(label, 'edge') }] : [] });
  }
  const head = pkg.bare ? 0 : pkg.description ? PACKAGE_HEAD_WITH_NOTE : PACKAGE_HEAD;
  const graph = {
    id: `pkg:${component.id}:${pkg.id}`,
    layoutOptions: edges.length ? {
      'elk.algorithm': 'layered',
      'elk.direction': 'DOWN',
      'elk.edgeRouting': 'ORTHOGONAL',
      'elk.layered.spacing.nodeNodeBetweenLayers': '40',
      'elk.spacing.nodeNode': '20',
      'elk.spacing.edgeNode': '12',
      'elk.spacing.edgeEdge': '7',
      'elk.spacing.edgeLabel': '4',
      'elk.edgeLabels.placement': 'CENTER',
      'elk.layered.nodePlacement.strategy': 'NETWORK_SIMPLEX',
      'elk.layered.considerModelOrder.strategy': 'NODES_AND_EDGES',
      'elk.separateConnectedComponents': 'true',
      'elk.spacing.componentComponent': '26',
      'elk.aspectRatio': '1.7',
      'elk.padding': `[top=${head + 10},left=18,bottom=18,right=18]`,
    } : packed(1.9, 20, `[top=${head + 10},left=18,bottom=18,right=18]`),
    children: [
      ...loose.map(n => ({ id: n.id, ...nodeSize(n) })),
      ...groupIds.map(g => {
        const group = groups.get(`group:${g}`);
        const caption = Math.max(measure(group.title, 'groupTitle').width, group.description ? measure(group.description, 'groupNote').width : 0) + 28;
        return {
          id: `group:${g}`,
          layoutOptions: {
            'elk.algorithm': 'rectpacking',
            'elk.padding': `[top=${group.description ? 48 : 34},left=14,bottom=14,right=14]`,
            'elk.spacing.nodeNode': '12',
            'elk.aspectRatio': '1.9',
            'elk.nodeSize.constraints': 'MINIMUM_SIZE',
            'elk.nodeSize.minimum': `(${Math.ceil(caption)}, 60)`,
          },
          children: group.members.map(id => ({ id, ...nodeSize(nodes.get(id)) })),
        };
      }),
    ],
    edges: edges.map(({ id, sources, targets, labels }) => ({ id, sources, targets, labels })),
  };
  const out = await elk.layout(graph);

  // The frame is at least as wide as its header and its groups' captions.
  let needed = pkg.bare ? 0 : measure(pkg.title, 'packageTitle').width + measure(pkg.path, 'packagePath').width + 64;
  if (pkg.description) needed = Math.max(needed, measure(pkg.description, 'packageNote').width + 40);
  shiftToWidth(out, needed);
  const boxes = {};
  for (const c of out.children) {
    boxes[c.id] = { x: c.x, y: c.y, w: c.width, h: c.height };
    for (const m of c.children || []) boxes[m.id] = { x: c.x + m.x, y: c.y + m.y, w: m.width, h: m.height };
  }
  const byId = new Map((out.edges || []).map(e => [e.id, e]));
  const routes = edges.map(({ id, model: e, flipped }) => {
    const laid = byId.get(id);
    let points = sectionPoints(laid);
    if (!points) return null;
    if (flipped) points = points.reverse();
    return { source: e.source, target: e.target, kind: e.kind, label: e.label, key: !!e.key, points, labelAt: labelCentre(laid) };
  }).filter(Boolean);
  return { w: out.width, h: out.height, boxes, routes };
}

async function layoutComponent(component, model, elk, nodes, groups, laidAs, homeOf) {
  const pkgs = {};
  for (const p of component.packages) {
    const laid = await layoutPackage(component, p, model, elk, nodes, groups, laidAs);
    if (laid) pkgs[p.id] = laid;
  }
  const testPackages = new Set(component.packages.filter(p =>
    model.nodes.some(n => n.package === p.id && n.component === component.id) &&
    model.nodes.filter(n => n.package === p.id && n.component === component.id).every(n => n.test)).map(p => p.id));

  // Links between packages, one arrow per pair
  const pairs = new Map();
  for (const e of model.edges) {
    const a = homeOf(e.source), b = homeOf(e.target);
    if (!a || !b || a.component !== component.id || b.component !== component.id || a.package === b.package) continue;
    if (testPackages.has(a.package) || testPackages.has(b.package)) continue;
    const key = `${a.package}>${b.package}`;
    if (!pairs.has(key)) pairs.set(key, { from: a.package, to: b.package, count: 0, tests: 0, labels: [], links: [] });
    const p = pairs.get(key);
    p.count++;
    p.links.push([e.source, e.target]);
    if (e.kind === 'test') p.tests++;
    if (e.key && e.label) p.labels.push(e.label);
  }
  const pairList = [...pairs.values()].filter(p => pkgs[p.from] && pkgs[p.to]);
  for (const p of pairList) {
    const words = [...new Set(p.labels)].slice(0, 2).join(', ') || (p.tests === p.count ? english.diagram.tests : '');
    p.words = words;
    p.countText = english.diagram.links(p.count);
    p.text = words ? `${words} · ${p.countText}` : p.countText;
  }
  const graph = {
    id: `comp:${component.id}`,
    layoutOptions: pairList.length ? {
      'elk.algorithm': 'layered',
      'elk.direction': 'DOWN',
      'elk.edgeRouting': 'ORTHOGONAL',
      'elk.layered.spacing.nodeNodeBetweenLayers': '90',
      'elk.spacing.nodeNode': '70',
      'elk.spacing.edgeNode': '30',
      'elk.spacing.edgeEdge': '18',
      'elk.spacing.edgeLabel': '6',
      'elk.edgeLabels.placement': 'CENTER',
      'elk.layered.nodePlacement.strategy': 'NETWORK_SIMPLEX',
      'elk.layered.considerModelOrder.strategy': 'NODES_AND_EDGES',
      'elk.separateConnectedComponents': 'true',
      'elk.spacing.componentComponent': '70',
      'elk.aspectRatio': '2.4',
      'elk.padding': `[top=${COMPONENT_HEAD + 16},left=36,bottom=36,right=36]`,
    } : packed(1.6, 70, `[top=${COMPONENT_HEAD + 16},left=36,bottom=36,right=36]`),
    children: Object.entries(pkgs).map(([id, p]) => ({ id, width: p.w, height: p.h })),
    edges: pairList.map((p, i) => ({ id: `p${i}`, sources: [p.from], targets: [p.to], labels: [{ text: p.text, ...measure(p.text, 'packageArrow') }] })),
  };
  const out = await elk.layout(graph);
  const counts = componentCounts(model, component);
  // Wide enough for its header and, beside other components, for its title
  // to stay as readable as theirs at far zoom.
  const needed = Math.max(460,
    measure(component.title, 'componentTitle').width + measure(english.diagram.openDiff, 'componentOpen').width + 80,
    measure(`${component.tag} · ${counts}`, 'componentTag').width + 56,
    model.components.length > 1 ? farTitleRoom(component.title) : 0);
  shiftToWidth(out, needed);
  const pos = new Map(out.children.map(c => [c.id, { x: c.x, y: c.y }]));
  const byId = new Map((out.edges || []).map(e => [e.id, e]));
  const boxes = {}, routes = [];
  for (const [id, p] of Object.entries(pkgs)) {
    const o = pos.get(id);
    boxes[`pkg:${component.id}:${id}`] = { x: o.x, y: o.y, w: p.w, h: p.h };
    for (const [bid, b] of Object.entries(p.boxes)) boxes[bid] = { x: b.x + o.x, y: b.y + o.y, w: b.w, h: b.h };
    for (const r of p.routes) routes.push(offsetRoute(r, o.x, o.y));
  }
  const pkgArrows = pairList.map((p, i) => {
    const laid = byId.get(`p${i}`);
    const points = sectionPoints(laid);
    if (!points) return null;
    return { from: p.from, to: p.to, count: p.count, words: p.words, countText: p.countText, links: p.links, points, labelAt: labelCentre(laid) };
  }).filter(Boolean);
  return { w: out.width, h: out.height, boxes, routes, pkgArrows };
}

// With nothing linking them, boxes sit in tidy rows rather than layers.
function packed(aspectRatio, spacing, padding) {
  return {
    'elk.algorithm': 'rectpacking',
    'elk.aspectRatio': String(aspectRatio),
    'elk.spacing.nodeNode': String(spacing),
    'elk.padding': padding,
  };
}

function componentCounts(model, component) {
  const files = (component.context ? model.contextFiles : model.files).filter(f => f.component === component.id);
  if (component.context) return english.diagram.unchangedFiles(files.length);
  const adds = files.reduce((s, f) => s + f.adds, 0), dels = files.reduce((s, f) => s + f.dels, 0);
  return english.diagram.changedFiles(files.length, adds, dels);
}
