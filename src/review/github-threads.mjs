/**
 * The review threads already on GitHub, other people's and bots' and one's
 * own, as the app shows them on the lines they belong to.
 *
 * readThreads(pull, nodes) is pure. It takes one pull request's threads as
 * GitHub's GraphQL API gives them and returns one record each:
 *
 *   {id, pull, path, side, line, start_line, start_side, original_line,
 *    resolved, resolved_by, outdated, file, comments, more, hunk}
 *
 * where `line` is null for an outdated thread, `file` marks a thread on the
 * whole file, `comments` are {id, author, bot, body, at, url}, oldest first,
 * `more` counts the comments GitHub holds beyond them, and `hunk` is the end
 * of the diff the first comment was made on.
 *
 * placeThreads(threads, stack) is pure. It gives each thread `at`, the line
 * the app shows it on, {path, side, line, start_line, start_side}, or null
 * when no line shown carries it: an outdated thread, one on the whole file,
 * or on a stack one whose line a later pull request changed again. A single
 * pull request's lines are the app's own; a stack's are found in its line
 * map, from each pull request's own line back to the combined diff's.
 */
const HUNK_TAIL = 4;

export function readThreads(pull, nodes) {
  return nodes.map(node => {
    const comments = node.comments.nodes;
    return {
      id: node.id,
      pull,
      path: node.path,
      side: node.diffSide,
      line: node.line ?? null,
      start_line: node.startLine ?? null,
      start_side: node.startLine ? node.startDiffSide || node.diffSide : null,
      original_line: node.originalLine ?? null,
      resolved: !!node.isResolved,
      resolved_by: node.resolvedBy?.login || null,
      outdated: !!node.isOutdated,
      file: node.subjectType === 'FILE',
      comments: comments.map(c => ({
        id: c.databaseId, author: c.author?.login || 'ghost', bot: c.author?.__typename === 'Bot', body: c.body, at: c.createdAt, url: c.url,
      })),
      more: Math.max(0, node.comments.totalCount - comments.length),
      hunk: comments.length && comments[0].diffHunk ? comments[0].diffHunk.split('\n').slice(-HUNK_TAIL).join('\n') : '',
    };
  });
}

export function placeThreads(threads, stack) {
  const back = stack ? ownToShown(stack.lines) : null;
  return threads.map(t => ({ ...t, at: placed(t, back) }));
}

function placed(t, back) {
  if (t.outdated || t.file || !t.line) return null;
  const startSide = t.start_side || t.side;
  if (!back) {
    return { path: t.path, side: t.side, line: t.line, ...(t.start_line ? { start_line: t.start_line, start_side: startSide } : {}) };
  }
  const end = back.get(key(t.pull, t.side, t.path, t.line));
  if (!end) return null;
  const start = t.start_line ? back.get(key(t.pull, startSide, t.path, t.start_line)) : null;
  const ranged = start && start.path === end.path;
  return { path: end.path, side: t.side, line: end.line, ...(ranged ? { start_line: start.line, start_side: startSide } : {}) };
}

function ownToShown(lines) {
  const back = new Map();
  const add = (entry, path, side, line) => {
    const k = key(entry[0], side, entry[2] || path, entry[1]);
    if (!back.has(k)) back.set(k, { path, line: Number(line) });
  };
  for (const [path, sides] of Object.entries(lines)) {
    for (const side of ['RIGHT', 'LEFT']) {
      for (const [line, entry] of Object.entries(sides[side] || {})) {
        if (Array.isArray(entry)) add(entry, path, side, line);
        else if (entry.c) entry.c.forEach(c => add(c, path, side, line));
      }
    }
  }
  return back;
}

function key(pull, side, path, line) {
  return `${pull} ${side} ${line} ${path}`;
}
