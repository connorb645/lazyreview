/**
 * The lines a note covers in a file's diff table: one line, or a range that
 * runs between two lines in the order the table shows them.
 *
 *   pointOf(cell)              the line a line number cell names, {side, line}
 *   rangeOf(file, from, to, wide)
 *                              the range from one point to another, as
 *                              {side, line, start_line, start_side}, the last
 *                              line first and null start_line for one line;
 *                              in the unified view it may start on a removed
 *                              line and end on an added one, and in the split
 *                              view it stays on the side it started on
 *   rowFor(file, side, line)   the table row showing a line, or null
 *   rowsOf(file, lines)        every row from a range's first line to its
 *                              last, in order
 *   linesWords(lines)          a range in words, such as "lines 7-10"
 */
export function pointOf(cell) {
  const at = cell.getAttribute('data-at');
  return { side: at.charAt(0) === 'L' ? 'LEFT' : 'RIGHT', line: +at.slice(2) };
}

export function rowFor(file, side, line) {
  return file.querySelector('tr[data-l~="' + (side === 'LEFT' ? 'L:' : 'R:') + line + '"]');
}

export function rangeOf(file, from, to, wide) {
  const a = { ...from, row: rowFor(file, from.side, from.line) };
  let b = { ...to, row: rowFor(file, to.side, to.line) };
  if (!a.row || !b.row) return null;
  if (wide && b.side !== a.side) {
    const line = numbers(b.row)[a.side];
    if (line === null) return null;
    b = { side: a.side, line, row: b.row };
  }
  const [first, last] = before(a, b) ? [a, b] : [b, a];
  const startSide = !wide && isUnchanged(first.row) ? last.side : first.side;
  const start = startSide === first.side ? first.line : numbers(first.row)[startSide];
  if (startSide === last.side && start === last.line) return { side: last.side, line: last.line, start_line: null, start_side: null };
  return { side: last.side, line: last.line, start_line: start, start_side: startSide };
}

export function rowsOf(file, n) {
  const end = rowFor(file, n.side, n.line);
  if (!end) return [];
  const start = n.start_line ? rowFor(file, n.start_side || n.side, n.start_line) : end;
  if (!start) return [end];
  const rows = [];
  for (let row = start; row; row = row.nextElementSibling) {
    if (row.hasAttribute('data-l')) rows.push(row);
    if (row === end) return rows;
  }
  return [end];
}

export function linesWords(n) {
  const side = s => (s === 'LEFT' ? 'removed line' : 'line');
  const startSide = n.start_side || n.side;
  if (!n.start_line) return side(n.side) + ' ' + n.line;
  if (startSide !== n.side) return side(startSide) + ' ' + n.start_line + ' to ' + side(n.side) + ' ' + n.line;
  return side(n.side) + 's ' + n.start_line + '-' + n.line;
}

function numbers(row) {
  const out = { LEFT: null, RIGHT: null };
  (row.getAttribute('data-l') || '').split(' ').forEach(key => {
    if (key.charAt(1) === ':') out[key.charAt(0) === 'L' ? 'LEFT' : 'RIGHT'] = +key.slice(2);
  });
  return out;
}

function isUnchanged(row) {
  const n = numbers(row);
  return n.LEFT !== null && n.RIGHT !== null && !row.querySelector('td.cdel, td.cadd');
}

function before(a, b) {
  if (a.row === b.row) return a.side === 'LEFT' || b.side === 'RIGHT';
  return !!(a.row.compareDocumentPosition(b.row) & Node.DOCUMENT_POSITION_FOLLOWING);
}
