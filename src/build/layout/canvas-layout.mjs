/**
 * Places the components on the canvas and routes the arrows between them.
 *
 * layoutCanvas(model, elk, results) takes each component's laid-out size in
 * `results` and returns the whole canvas layout. Components are placed in
 * blocks: one per area, framed and titled, and one for the components in no
 * area. Inside a block the components sit in rows, the largest on top, and
 * the arrows between them run straight where two frames face each other.
 * ELK places the blocks and routes the arrows between blocks. Such an arrow
 * meets a block at a fixed point facing the component it is for, and runs on
 * from there to that component's frame.
 */
import { measure, AREA_HEAD } from './sizes.mjs';
import { sectionPoints, labelCentre, offsetRoute, roundRoute } from './geometry.mjs';
import { shelfPack, route, START, END } from './shelf.mjs';
import { english } from '../../i18n/english.mjs';

const CANVAS_ASPECT = 1.45;
const AREA_PAD = 90;
const MARGIN = { top: 170, side: 80, bottom: 80 };
const PORT_SPACING = 170;
const OPTIONS = {
  'elk.algorithm': 'layered',
  'elk.edgeRouting': 'ORTHOGONAL',
  'elk.layered.spacing.nodeNodeBetweenLayers': '420',
  'elk.spacing.nodeNode': '300',
  'elk.spacing.edgeNode': '60',
  'elk.spacing.edgeEdge': '60',
  'elk.spacing.edgeLabel': '10',
  'elk.edgeLabels.placement': 'CENTER',
  'elk.layered.nodePlacement.strategy': 'NETWORK_SIMPLEX',
  'elk.layered.considerModelOrder.strategy': 'NODES_AND_EDGES',
  'elk.separateConnectedComponents': 'true',
  'elk.spacing.componentComponent': '300',
};
const SIDES = { RIGHT: { out: 'EAST', in: 'WEST' }, DOWN: { out: 'SOUTH', in: 'NORTH' } };
// When the run from a side would cross another component, the sides to try next.
const TURNS = { EAST: ['SOUTH', 'NORTH', 'WEST'], WEST: ['NORTH', 'SOUTH', 'EAST'], SOUTH: ['EAST', 'WEST', 'NORTH'], NORTH: ['WEST', 'EAST', 'SOUTH'] };

export async function layoutCanvas(model, elk, results) {
  const pairs = summaryPairs(model.summary);
  const areaOf = new Map(model.areas.flatMap(a => a.components.map(id => [id, a.id])));
  const blockOf = id => (areaOf.has(id) ? `area:${areaOf.get(id)}` : 'loose');
  const loose = model.components.filter(c => !areaOf.has(c.id)).map(c => c.id);

  const parts = [
    ...model.areas.map(a => ({ id: `area:${a.id}`, area: a, components: a.components })),
    ...(loose.length ? [{ id: 'loose', area: null, components: loose }] : []),
  ];
  const blocks = parts.map(b => layoutBlock(b, pairs.filter(p => blockOf(p.fwd.source) === b.id && blockOf(p.fwd.target) === b.id), results, parts.length === 1));
  const cross = pairs.filter(p => blockOf(p.fwd.source) !== blockOf(p.fwd.target));

  if (blocks.length === 1) {
    const only = blocks[0];
    const placed = new Map([[only.id, { x: MARGIN.side, y: MARGIN.top }]]);
    return assemble({ w: only.w + MARGIN.side * 2, h: only.h + MARGIN.top + MARGIN.bottom }, placed, blocks, model, results, [], new Map(), new Map());
  }
  let best = null;
  for (const direction of ['RIGHT', 'DOWN']) {
    const graph = rootGraph(direction, blocks, cross, blockOf);
    const out = await elk.layout(graph.elk);
    const fit = Math.abs(Math.log(out.width / out.height / CANVAS_ASPECT));
    if (!best || fit < best.fit) best = { out, fit, ports: graph.ports };
  }
  const placed = new Map(best.out.children.map(c => [c.id, { x: c.x, y: c.y }]));
  const edges = new Map((best.out.edges || []).map(e => [e.id, e]));
  return assemble({ w: best.out.width, h: best.out.height }, placed, blocks, model, results, cross, edges, best.ports);
}

function layoutBlock(block, inner, results, alone) {
  const framed = !!block.area;
  const title = framed ? measure(block.area.title, 'areaTitle').width : 0;
  const packed = shelfPack(block.components.map(id => ({ id, w: results[id].w, h: results[id].h })),
    framed ? { top: AREA_HEAD, side: AREA_PAD, bottom: AREA_PAD } : { top: 0, side: 0, bottom: 0 },
    alone ? CANVAS_ASPECT : CANVAS_ASPECT / 2);
  const w = framed ? Math.max(packed.w, title + 140, block.area.description ? measure(block.area.description, 'areaTag').width + 140 : 0) : packed.w;
  const dx = (w - packed.w) / 2;
  const at = Object.fromEntries(Object.entries(packed.at).map(([id, b]) => [id, { ...b, x: b.x + dx }]));
  return {
    ...block,
    w,
    h: packed.h,
    title,
    at,
    routes: inner.map(p => ({ pair: p, ...route(at[p.fwd.source], at[p.fwd.target], labelSize(p.lines)) })),
  };
}

function rootGraph(direction, blocks, cross, blockOf) {
  const byId = new Map(blocks.map(b => [b.id, b]));
  const ports = new Map();
  const used = new Map();
  const portFor = (componentId, role) => {
    const block = byId.get(blockOf(componentId));
    const preferred = SIDES[direction][role];
    const tries = [preferred, ...TURNS[preferred]].map(side =>
      ({ side, ...portPlace(block, componentId, side, used.get(`${block.id}:${componentId}:${side}`) || 0, role) }));
    const place = tries.find(t => t.clear) || tries[0];
    const key = `${block.id}:${componentId}:${place.side}`;
    used.set(key, (used.get(key) || 0) + 1);
    const id = `port${ports.size}`;
    ports.set(id, { id, block: block.id, ...place });
    return id;
  };
  const edges = cross.map((p, i) => labelledEdge(`s${i}`, portFor(p.fwd.source, 'out'), portFor(p.fwd.target, 'in'), p.lines));
  return {
    ports,
    elk: {
      id: 'canvas',
      layoutOptions: {
        ...OPTIONS, 'elk.direction': direction, 'elk.aspectRatio': String(CANVAS_ASPECT),
        'elk.padding': `[top=${MARGIN.top},left=${MARGIN.side},bottom=${MARGIN.bottom},right=${MARGIN.side}]`,
      },
      children: blocks.map(b => ({
        id: b.id,
        width: b.w,
        height: b.h,
        layoutOptions: { 'elk.portConstraints': 'FIXED_POS' },
        ports: [...ports.values()].filter(p => p.block === b.id).map(p => ({
          id: p.id, width: 2, height: 2, x: p.port[0], y: p.port[1], layoutOptions: { 'elk.port.side': p.side },
        })),
      })),
      edges,
    },
  };
}

// Where an arrow meets a block: on the given side, in line with the
// component, at a point from which a straight run reaches the component
// without crossing another one. Returns the port, relative to the block, the
// point just off the component's frame where the run ends, and whether the
// run is clear.
function portPlace(block, componentId, side, count, role) {
  const c = block.at[componentId];
  const blockers = Object.entries(block.at).filter(([id]) => id !== componentId).map(([, b]) => b);
  if (block.area) blockers.push({ x: 0, y: 0, w: block.title + 100, h: AREA_HEAD - 30 });
  const horizontal = side === 'WEST' || side === 'EAST';
  const from = horizontal ? c.y + 40 : c.x + 40;
  const to = horizontal ? c.y + c.h - 40 : c.x + c.w - 40;
  const centre = (from + to) / 2 + (count ? Math.ceil(count / 2) * PORT_SPACING * (count % 2 ? 1 : -1) : 0);
  const candidates = [];
  for (let v = from; v <= to; v += 30) candidates.push(v);
  candidates.sort((a, b) => Math.abs(a - centre) - Math.abs(b - centre));
  const clear = v => blockers.every(b => {
    if (horizontal) {
      if (v < b.y || v > b.y + b.h) return true;
      return side === 'WEST' ? b.x >= c.x : b.x + b.w <= c.x + c.w;
    }
    if (v < b.x || v > b.x + b.w) return true;
    return side === 'NORTH' ? b.y >= c.y : b.y + b.h <= c.y + c.h;
  });
  const free = candidates.find(clear);
  const v = free ?? Math.min(to, Math.max(from, centre));
  const gap = role === 'in' ? END : START;
  const result = { clear: free !== undefined };
  if (side === 'WEST') return { ...result, port: [-2, v], frame: [c.x - gap, v] };
  if (side === 'EAST') return { ...result, port: [block.w, v], frame: [c.x + c.w + gap, v] };
  if (side === 'NORTH') return { ...result, port: [v, -2], frame: [v, c.y - gap] };
  return { ...result, port: [v, block.h], frame: [v, c.y + c.h + gap] };
}

function assemble(size, placed, blocks, model, results, cross, edges, ports = new Map()) {
  const r2 = v => Math.round(v * 10) / 10;
  const layout = { w: Math.round(size.w), h: Math.round(size.h), areas: {}, components: {}, boxes: {}, routes: [], pkgArrows: [], summaries: [] };
  for (const block of blocks) {
    const origin = placed.get(block.id);
    if (block.area) layout.areas[block.area.id] = { x: r2(origin.x), y: r2(origin.y), w: r2(block.w), h: r2(block.h) };
    for (const id of block.components) {
      const at = { x: origin.x + block.at[id].x, y: origin.y + block.at[id].y };
      const r = results[id];
      layout.components[id] = { x: r2(at.x), y: r2(at.y), w: r2(r.w), h: r2(r.h) };
      for (const [bid, b] of Object.entries(r.boxes)) layout.boxes[bid] = { x: r2(b.x + at.x), y: r2(b.y + at.y), w: r2(b.w), h: r2(b.h) };
      for (const e of r.routes) layout.routes.push(roundRoute(offsetRoute(e, at.x, at.y)));
      for (const a of r.pkgArrows) layout.pkgArrows.push(roundRoute({ ...offsetRoute(a, at.x, at.y), comp: id }));
    }
    for (const inner of block.routes) {
      layout.summaries.push(roundRoute(summary(inner.pair, offsetRoute({ points: inner.points, labelAt: inner.labelAt }, origin.x, origin.y))));
    }
  }
  cross.forEach((pair, i) => {
    const edge = edges.get(`s${i}`);
    let points = sectionPoints(edge);
    if (!points) return;
    const run = portId => {
      const p = ports.get(portId);
      const origin = placed.get(p.block);
      return [origin.x + p.frame[0], origin.y + p.frame[1]];
    };
    const start = run(edge.sources[0]);
    const end = run(edge.targets[0]);
    points = [start, ...snap(points, 0, start, ports.get(edge.sources[0]).side)];
    points = [...snap(points, points.length - 1, end, ports.get(edge.targets[0]).side), end];
    layout.summaries.push(roundRoute(summary(pair, { points: straighten(points), labelAt: labelCentre(edge) })));
  });
  return layout;
}

// Lines the arrow's end at a port up with the run on to the frame, so the
// join stays square.
function snap(points, index, frame, side) {
  const out = points.map(p => [...p]);
  const axis = side === 'WEST' || side === 'EAST' ? 1 : 0;
  const neighbour = index === 0 ? 1 : out.length - 2;
  if (out[neighbour] && Math.abs(out[neighbour][axis] - out[index][axis]) < 3) out[neighbour][axis] = frame[axis];
  out[index][axis] = frame[axis];
  return out;
}

function summary(pair, r) {
  return { source: pair.fwd.source, target: pair.fwd.target, both: !!pair.rev, lines: pair.lines, ways: pair.ways, points: r.points, labelAt: r.labelAt };
}

// Drops points that repeat the one before, so a run on from a port joins the
// arrow without a zero-length bend.
function straighten(points) {
  return points.filter((p, i) => i === 0 || Math.abs(p[0] - points[i - 1][0]) + Math.abs(p[1] - points[i - 1][1]) > 0.5);
}

function labelSize(lines) {
  const sizes = lines.map(l => measure(l, 'summary'));
  return { width: Math.max(...sizes.map(s => s.width)), height: sizes.reduce((h, s) => h + s.height, 0) };
}

function labelledEdge(id, source, target, lines) {
  return { id, sources: [source], targets: [target], labels: [{ text: lines.join('\n'), ...labelSize(lines) }] };
}

// One arrow per pair of components; a pair that links both ways shares one
// arrow with a head at each end and a line of words for each way. Each way
// keeps its arrow, its label and the links between types it counts, so an
// app can count them again under a pull request filter.
function summaryPairs(summary) {
  const byKey = new Map(summary.map(s => [`${s.source}>${s.target}`, s]));
  const pairs = [];
  for (const s of summary) {
    const rev = byKey.get(`${s.target}>${s.source}`);
    if (rev && (rev.count > s.count || (rev.count === s.count && rev.source < s.source))) continue;
    const words = (x, arrow) => english.diagram.summaryLine(arrow, x.label, x.count);
    const way = (x, arrow) => ({ arrow, label: x.label, links: x.links });
    pairs.push({
      fwd: s, rev: rev || null,
      lines: rev ? [words(s, '→'), words(rev, '←')] : [words(s, '→')],
      ways: rev ? [way(s, '→'), way(rev, '←')] : [way(s, '→')],
    });
  }
  return pairs;
}
