/**
 * Positions in masked source text: line numbers, brace depth and matching
 * braces, all worked out in one pass.
 *
 * indexSource(masked) returns:
 *   line(offset)      the 1-based line an offset sits on
 *   depth(offset)     how many braces are open just before the offset
 *   close(offset)     the offset of the brace that closes the one at offset,
 *                     or the end of the text when it never closes
 *   lineStart(line)   the offset where a 1-based line starts
 */
export function indexSource(masked) {
  const n = masked.length;
  const starts = [0];
  const depths = new Int32Array(n + 1);
  const closes = new Int32Array(n).fill(-1);
  const stack = [];
  let depth = 0;
  for (let k = 0; k < n; k++) {
    depths[k] = depth;
    const c = masked[k];
    if (c === '\n') starts.push(k + 1);
    else if (c === '{') { stack.push(k); depth++; }
    else if (c === '}') {
      if (stack.length) closes[stack.pop()] = k;
      if (depth > 0) depth--;
    }
  }
  depths[n] = depth;

  function line(offset) {
    let lo = 0, hi = starts.length - 1;
    while (lo < hi) {
      const mid = (lo + hi + 1) >> 1;
      if (starts[mid] <= offset) lo = mid; else hi = mid - 1;
    }
    return lo + 1;
  }

  return {
    line,
    depth: offset => depths[Math.max(0, Math.min(n, offset))],
    close: offset => (closes[offset] >= 0 ? closes[offset] : n),
    lineStart: number => starts[Math.max(0, Math.min(starts.length - 1, number - 1))],
    lineCount: starts.length,
  };
}
