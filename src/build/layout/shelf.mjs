/**
 * The arrangement inside a block of components: in rows, largest first, the
 * largest alone on top when it dominates the rest, each row no wider than the
 * block, and the arrows between them drawn straight where two frames face
 * each other.
 */
const GAP_X = 460;
const GAP_Y = 300;
// An arrow stops short of the frames it joins, so both its heads show.
export const START = 8;
export const END = 14;

/**
 * shelfPack(boxes, padding, aspect) takes [{id, w, h}] and returns {w, h, at},
 * where `at` gives each box's place, inside `padding` ({top, side, bottom}).
 * Of every row width the boxes allow, it keeps the one that makes the block
 * closest to `aspect` times as wide as it is tall.
 */
export function shelfPack(boxes, padding, aspect = 1.2) {
  const sorted = [...boxes].sort((a, b) => b.w * b.h - a.w * a.h || a.id.localeCompare(b.id));
  const total = sorted.reduce((s, b) => s + b.w * b.h, 0);
  // The largest has the top row to itself only when it dominates the rest.
  const alone = sorted.length > 2 && sorted[0].w * sorted[0].h > total * 0.4;
  const widest = Math.max(...sorted.map(b => b.w));
  const limits = new Set([widest]);
  for (let i = alone ? 1 : 0; i < sorted.length; i++) {
    let width = 0;
    for (let j = i; j < sorted.length; j++) {
      width += (j > i ? GAP_X : 0) + sorted[j].w;
      if (width >= widest) limits.add(width);
    }
  }
  let best = null;
  for (const limit of limits) {
    const packed = pack(sorted, padding, limit, alone);
    const fit = Math.abs(Math.log(packed.w / packed.h / aspect));
    if (!best || fit < best.fit - 1e-9) best = { packed, fit };
  }
  return best.packed;
}

function pack(sorted, padding, limit, alone) {
  const rows = [{ items: [sorted[0]], width: sorted[0].w }];
  for (const box of sorted.slice(1)) {
    const row = rows[rows.length - 1];
    if ((rows.length > 1 || !alone) && row.width + GAP_X + box.w <= limit) {
      row.items.push(box);
      row.width += GAP_X + box.w;
    } else rows.push({ items: [box], width: box.w });
  }
  const width = Math.max(...rows.map(r => r.width));
  const at = {};
  let y = padding.top;
  for (const row of rows) {
    let x = padding.side + (width - row.width) / 2;
    for (const box of row.items) {
      at[box.id] = { x, y, w: box.w, h: box.h };
      x += box.w + GAP_X;
    }
    y += Math.max(...row.items.map(b => b.h)) + GAP_Y;
  }
  return { w: width + padding.side * 2, h: y - GAP_Y + padding.bottom, at };
}

/**
 * route(a, b, label) draws an arrow from box a to box b: straight across
 * where they face each other side by side, straight down or up where one sits
 * above the other, and otherwise down into the gap between them, across and
 * on. `label` is {width, height}; returns {points, labelAt}.
 */
export function route(a, b, label) {
  const xOverlap = Math.min(a.x + a.w, b.x + b.w) - Math.max(a.x, b.x);
  const yOverlap = Math.min(a.y + a.h, b.y + b.h) - Math.max(a.y, b.y);
  if (yOverlap > 120 && xOverlap <= 0) {
    const y = Math.max(a.y, b.y) + Math.min(yOverlap, 500) / 2;
    const right = a.x < b.x;
    const x1 = right ? a.x + a.w + START : a.x - START, x2 = right ? b.x - END : b.x + b.w + END;
    return { points: [[x1, y], [x2, y]], labelAt: [(x1 + x2) / 2, y - label.height / 2 - 12] };
  }
  if (xOverlap > 120) {
    const x = Math.max(a.x, b.x) + Math.min(xOverlap, 700) / 2;
    const down = a.y < b.y;
    const y1 = down ? a.y + a.h + START : a.y - START, y2 = down ? b.y - END : b.y + b.h + END;
    return { points: [[x, y1], [x, y2]], labelAt: [x + 24 + label.width / 2, (y1 + y2) / 2] };
  }
  const down = a.y < b.y;
  const y1 = down ? a.y + a.h + START : a.y - START;
  const y2 = down ? b.y - END : b.y + b.h + END;
  const mid = (y1 + y2) / 2;
  const x1 = a.x + a.w / 2, x2 = b.x + b.w / 2;
  return { points: [[x1, y1], [x1, mid], [x2, mid], [x2, y2]], labelAt: [(x1 + x2) / 2, mid - label.height / 2 - 12] };
}
