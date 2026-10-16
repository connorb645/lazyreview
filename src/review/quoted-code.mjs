/**
 * The code a comment or a question quotes, read from the review's page.
 *
 * quotedCode(page, lines) is pure. `lines` are {path, side, line,
 * start_line, start_side} as the app shows them. Inside a hunk of the page's
 * diff it gives the rows from the first line to the last: those of the side
 * the lines are on, or every row between, removed and added, for a range from
 * a removed line to an added one. Elsewhere, on the new side, it gives the
 * lines of the file's whole text, when the page holds it, and otherwise
 * nothing.
 */
import { readHunks } from './diff-hunks.mjs';

export function quotedCode(page, n) {
  const startSide = n.start_line ? n.start_side || n.side : n.side;
  const first = n.start_line || n.line;
  const keep = row => row.kind !== 'note' && (startSide !== n.side || row.kind !== (n.side === 'LEFT' ? 'add' : 'del'));
  for (const hunk of readHunks(page.diff[n.path])) {
    const from = hunk.rows.findIndex(row => on(row, startSide, first));
    const to = hunk.rows.findIndex(row => on(row, n.side, n.line));
    if (from >= 0 && to >= from) return hunk.rows.slice(from, to + 1).filter(keep).map(row => row.text).join('\n');
  }
  const whole = startSide === 'RIGHT' && n.side === 'RIGHT' ? wholeText(page, n.path) : null;
  return whole ? whole.slice(first - 1, n.line).join('\n') : '';
}

function on(row, side, line) {
  return side === 'LEFT' ? row.kind !== 'add' && row.old === line : row.kind !== 'del' && row.new === line;
}

function wholeText(page, path) {
  if (page.sources[path]) return page.sources[path];
  if (page.context[path]) return page.context[path];
  const file = page.model.files.find(f => f.path === path);
  if (!file || file.status !== 'A') return null;
  return readHunks(page.diff[path]).flatMap(hunk => hunk.rows.filter(row => row.kind === 'add').map(row => row.text));
}
