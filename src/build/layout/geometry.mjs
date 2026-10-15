/**
 * Small helpers for the points and boxes ELK returns.
 */

/** The points of an edge's first section, moved by (ox, oy). */
export function sectionPoints(edge, ox = 0, oy = 0) {
  const s = edge && edge.sections && edge.sections[0];
  if (!s) return null;
  return [s.startPoint, ...(s.bendPoints || []), s.endPoint].map(p => [p.x + ox, p.y + oy]);
}

/** The centre of an edge's first label, moved by (ox, oy), or null. */
export function labelCentre(edge, ox = 0, oy = 0) {
  const l = edge && edge.labels && edge.labels[0];
  return l && l.x !== undefined ? [ox + l.x + l.width / 2, oy + l.y + l.height / 2] : null;
}

export function offsetRoute(r, dx, dy) {
  return { ...r, points: r.points.map(([x, y]) => [x + dx, y + dy]), labelAt: r.labelAt ? [r.labelAt[0] + dx, r.labelAt[1] + dy] : null };
}

export function roundRoute(r) {
  const r2 = v => Math.round(v * 10) / 10;
  return { ...r, points: r.points.map(([x, y]) => [r2(x), r2(y)]), labelAt: r.labelAt ? [r2(r.labelAt[0]), r2(r.labelAt[1])] : null };
}

/** Widens a laid-out graph to `width`, keeping its contents centred. */
export function shiftToWidth(out, width) {
  if (out.width >= width) return;
  const dx = (width - out.width) / 2;
  for (const c of out.children) c.x += dx;
  for (const e of out.edges || []) {
    for (const s of e.sections || []) for (const p of [s.startPoint, s.endPoint, ...(s.bendPoints || [])]) p.x += dx;
    for (const l of e.labels || []) l.x += dx;
  }
  out.width = width;
}
