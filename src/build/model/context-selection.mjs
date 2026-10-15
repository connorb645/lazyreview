/**
 * Chooses the unchanged types the map draws around the change.
 *
 * chooseContext({drawn, untouched, candidates, options}) scores each unchanged
 * type by how the changed types link to it: conforming to it or inheriting
 * from it counts most, holding it next, and naming it least. A link the other
 * way, unchanged code that uses a changed type, counts too, less so when the
 * changed type is used everywhere, and a mention counts less from code that
 * mentions a great many types.
 *
 *   drawn       the nodes for what the change adds or edits
 *   untouched   unchanged types inside the changed files
 *   candidates  unchanged types from unchanged files
 *   options     {enabled, max, include, exclude}
 *
 * Without `enabled`, only untouched types a changed type conforms to,
 * inherits from, holds or extends are kept, and no candidate. With it, up to
 * `max` of the best-scoring types are kept, `include` names always and
 * `exclude` names never. Returns {same, context}.
 */
import { findRelations } from './relations.mjs';
import { identifiers } from '../languages/header-text.mjs';

const OUTWARD = { inherit: 6, realize: 6, has: 4, extend: 4, use: 1.5, test: 0.5 };
const INWARD = { inherit: 3, realize: 3, has: 3, extend: 2, use: 1.5, test: 0 };
const STRUCTURAL = new Set(['inherit', 'realize', 'has', 'extend']);

export function chooseContext({ drawn, untouched, candidates, options }) {
  const originals = [...drawn, ...untouched, ...(options.enabled ? candidates : [])];
  const pool = originals.map((n, i) => ({ ...n, id: `n${i}` }));
  const original = new Map(pool.map((n, i) => [n.id, originals[i]]));
  const drawnIds = new Set(pool.slice(0, drawn.length).map(n => n.id));

  const spread = new Map();
  for (const c of pool.slice(drawn.length + untouched.length)) {
    const words = identifiers(c.code);
    for (const d of pool.slice(0, drawn.length)) if (words.has(d.name)) spread.set(d.id, (spread.get(d.id) || 0) + 1);
  }

  const relations = findRelations(pool);
  const reach = new Map();
  for (const e of relations) {
    if (drawnIds.has(e.source) && !drawnIds.has(e.target)) reach.set(e.source, (reach.get(e.source) || 0) + 1);
  }
  const score = new Map();
  const structural = new Set();
  for (const e of relations) {
    if (drawnIds.has(e.source) && !drawnIds.has(e.target)) {
      const damping = e.kind === 'use' ? 1 / Math.log2(1 + reach.get(e.source)) : 1;
      score.set(e.target, (score.get(e.target) || 0) + OUTWARD[e.kind] * Math.min(1, damping));
      if (STRUCTURAL.has(e.kind)) structural.add(e.target);
    } else if (!drawnIds.has(e.source) && drawnIds.has(e.target)) {
      const damping = 1 / Math.log2(2 + (spread.get(e.target) || 0));
      score.set(e.source, (score.get(e.source) || 0) + INWARD[e.kind] * damping);
    }
  }

  const unchanged = pool.slice(drawn.length);
  if (!options.enabled) {
    const kept = new Set(unchanged.filter(n => structural.has(n.id)).map(n => original.get(n.id)));
    return { same: untouched.filter(n => kept.has(n)), context: [] };
  }
  const include = new Set(options.include);
  const exclude = new Set(options.exclude);
  const ranked = unchanged
    .filter(n => !exclude.has(n.name) && !n.test)
    .filter(n => include.has(n.name) || (score.get(n.id) || 0) >= 0.5)
    .sort((a, b) => Number(include.has(b.name)) - Number(include.has(a.name)) ||
      (score.get(b.id) || 0) - (score.get(a.id) || 0) || a.name.localeCompare(b.name));
  const kept = new Set();
  const names = new Set();
  for (const n of ranked) {
    if (kept.size >= options.max && !include.has(n.name)) continue;
    const key = `${n.family}:${n.name}`;
    if (names.has(key)) continue;
    names.add(key);
    kept.add(original.get(n.id));
  }
  return { same: untouched.filter(n => kept.has(n)), context: candidates.filter(n => kept.has(n)) };
}
