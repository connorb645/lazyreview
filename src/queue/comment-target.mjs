/**
 * Where a comment on a line the app shows goes.
 *
 * commentTarget(review, stack, given) is pure. `given` is the line as the
 * app shows it, {path, line, side, start_line, start_side, pull}, where a
 * range may start on the other side from its last line, and `stack` the
 * page's stack data, or null for a single pull request. For a single pull
 * request the comment goes where it was made. For a stack, the line is
 * traced through the stack's line map to the pull request that last changed
 * it, and the comment goes at that pull request's own path and line, the one
 * GitHub takes for its diff; `pull` picks one where an unchanged line shows
 * in several.
 *
 * It returns {pull, target: {path, line, side, start_line, start_side},
 * shown}, where `shown` is the line as the app showed it, for a stack, or an
 * error saying why no single pull request can take the comment.
 */
export function commentTarget(review, stack, given) {
  const startSide = given.start_line ? given.start_side || given.side : null;
  if (!stack) {
    const [member] = review.members;
    if (given.pull && given.pull !== member.pull) throw new Error(`this review is #${member.pull}'s, not #${given.pull}'s`);
    return { pull: member.pull, target: { path: given.path, line: given.line, side: given.side, start_line: given.start_line || null, start_side: startSide }, shown: null };
  }
  const heads = new Map(stack.members.map(m => [m.pull, m.head]));
  const out = review.members.filter(m => heads.get(m.pull) !== m.head).map(m => `#${m.pull}`);
  if (out.length) throw new Error(`not queued: the page shows other heads of ${out.join(', ')} than the review holds; load the stack again, then queue`);
  const end = traced(stack.lines, given.path, given.line, given.side, given.pull);
  const target = { path: end.path, line: end.line, side: given.side, start_line: null, start_side: null };
  if (given.start_line) {
    let start;
    try {
      start = traced(stack.lines, given.path, given.start_line, startSide, end.pull);
    } catch {
      throw new Error(`not queued: lines ${given.start_line}-${given.line} do not all go to #${end.pull}; comment on one line, or one comment per pull request`);
    }
    if (start.path !== end.path) {
      throw new Error(`not queued: lines ${given.start_line}-${given.line} do not run on in #${end.pull}'s diff; comment on one line, or one comment per pull request`);
    }
    target.start_line = start.line;
    target.start_side = startSide;
  }
  const shown = { path: given.path, line: given.line, side: given.side, ...(given.start_line ? { start_line: given.start_line, start_side: startSide } : {}) };
  return { pull: end.pull, target, shown };
}

function traced(lines, path, line, side, pull) {
  const entry = lines[path]?.[side]?.[String(line)];
  const where = `${path}:${line} (${side === 'LEFT' ? 'removed side' : 'new side'})`;
  if (entry === undefined) throw new Error(`not queued: ${where} is not a line of the stack's combined diff`);
  if (Array.isArray(entry)) {
    const found = { pull: entry[0], line: entry[1], path: entry[2] || path };
    if (pull && pull !== found.pull) throw new Error(`not queued: #${found.pull} last changed ${where}, not #${pull}`);
    return found;
  }
  if (entry.x) throw new Error(`not queued: no single pull request can take ${where}: ${entry.x}`);
  const shown = entry.c.map(c => ({ pull: c[0], line: c[1], path: c[2] || path }));
  if (pull) {
    const chosen = shown.find(c => c.pull === pull);
    if (!chosen) throw new Error(`not queued: #${pull}'s diff does not show ${where}`);
    return chosen;
  }
  if (shown.length === 1) return shown[0];
  if (!shown.length) throw new Error(`not queued: ${where} is unchanged, and no pull request's own diff shows it`);
  throw new Error(`not queued: ${where} is unchanged, and ${shown.map(c => `#${c.pull}`).join(' and ')} show it; say which`);
}
