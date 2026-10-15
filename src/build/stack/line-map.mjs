/**
 * Where a comment on each line of a stack's combined diff goes, and which
 * pull requests changed each file.
 *
 * mapStackLines(steps, chunks, pulls) takes readStackSteps' steps, the
 * combined diff split per file and the members' numbers, bottom first, and
 * traces every line the combined diff's hunks show. It returns {lines, files}:
 *
 *   lines  {path: {RIGHT: {line: entry}, LEFT: {line: entry}}}, on the
 *          combined diff's own numbers: RIGHT for an added or unchanged line
 *          at the top head, LEFT for a removed or unchanged line at the base.
 *          An entry is one of
 *            [pull, line]          the pull request that changed the line,
 *            [pull, line, path]    and its line there, with the path the
 *                                  pull request's own diff gives the file
 *                                  when it is not the key's
 *            {c: [[pull, line], ...]}  an unchanged line, with each pull
 *                                  request whose own diff shows it
 *            {x: reason}           no pull request can take a comment on it
 *   files  {path: {pulls, byPull}}: the pull requests that changed a line of
 *          the file as the combined diff shows it, in stack order, and each
 *          one's [added, removed] count there. A file with no lines to trace,
 *          such as a binary file or a pure rename, is owned by every member
 *          whose own diff touches it.
 *
 * lineEntry(lines, path, line, side) reads one entry back as
 * {pull, path, line, side}, {candidates: [...]} or {none: reason}.
 */
import { readHunkLines } from './hunk-lines.mjs';
import { traceNew, traceOld } from './line-trace.mjs';

export function mapStackLines(steps, chunks, pulls) {
  const lines = {};
  const files = {};
  for (const chunk of chunks) {
    const right = {};
    const left = {};
    const byPull = {};
    const count = (pull, k) => { (byPull[pull] = byPull[pull] || [0, 0])[k]++; };
    for (const h of readHunkLines(chunk.text)) {
      for (const n of h.added) {
        const found = traceNew(steps, chunk.path, n);
        right[n] = found.pull ? owned(found, chunk.path) : { x: 'no pull request\'s own diff adds this line as it reads here' };
        if (found.pull) count(found.pull, 0);
      }
      for (const o of h.removed) {
        const found = traceOld(steps, chunk.oldPath, o);
        left[o] = found.pull ? owned(found, chunk.path) : { x: 'no pull request\'s own diff removes this line as it reads here' };
        if (found.pull) count(found.pull, 1);
      }
      for (const [n, o] of h.newToOld) {
        right[n] = unchanged(traceNew(steps, chunk.path, n), chunk.path);
        left[o] = unchanged(traceOld(steps, chunk.oldPath, o), chunk.path);
      }
    }
    lines[chunk.path] = { RIGHT: right, LEFT: left };
    files[chunk.path] = { pulls: Object.keys(byPull).length ? pulls.filter(p => byPull[p]) : touching(steps, chunk), byPull };
  }
  return { lines, files };
}

export function lineEntry(lines, path, line, side) {
  const entry = lines[path]?.[side]?.[line];
  if (!entry) return { none: 'the line is not in the combined diff' };
  if (Array.isArray(entry)) return target(entry, path, side);
  if (entry.c) return { candidates: entry.c.map(e => target(e, path, side)) };
  return { none: entry.x };
}

function target([pull, line, at], path, side) {
  return { pull, path: at || path, line, side };
}

function owned(found, path) {
  return found.path === path ? [found.pull, found.line] : [found.pull, found.line, found.path];
}

function unchanged(found, path) {
  return found.pull ? owned(found, path) : { c: found.context.map(t => owned(t, path)) };
}

function touching(steps, chunk) {
  return steps.filter(s => {
    if (chunk.status === 'D') {
      const moved = s.fromBase.byOld.get(chunk.oldPath);
      const path = moved ? moved.path : chunk.oldPath;
      return !!path && s.own.byOld.has(path);
    }
    const moved = s.fromTop.byNew.get(chunk.path);
    const path = moved ? (moved.status === 'A' ? null : moved.oldPath) : chunk.path;
    return !!path && s.own.byNew.has(path);
  }).map(s => s.pull);
}
