/**
 * Lines of a file in words, as the apps and Claude's questions name them:
 * "line 3", "lines 2-3", "removed line 4" or "removed line 4 to line 5".
 *
 * linesWords({side, line, start_line, start_side}) is pure. A range may
 * start on the other side from its last line.
 */
export function linesWords(n) {
  const side = s => (s === 'LEFT' ? 'removed line' : 'line');
  const startSide = n.start_side || n.side;
  if (!n.start_line) return `${side(n.side)} ${n.line}`;
  if (startSide !== n.side) return `${side(startSide)} ${n.start_line} to ${side(n.side)} ${n.line}`;
  return `${side(n.side)}s ${n.start_line}-${n.line}`;
}
