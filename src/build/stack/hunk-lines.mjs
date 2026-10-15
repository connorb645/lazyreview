/**
 * The line numbers inside one file's diff, on both sides, and how a line
 * number on one side maps to the other.
 *
 * readHunkLines(text) reads a file's part of a diff, as splitDiff gives it,
 * and returns its hunks, each {oldStart, oldLength, newStart, newLength,
 * added, removed, removedAt, newToOld, oldToNew}: the new-side numbers it
 * adds, the old-side numbers it removes with the new-side position each was
 * removed at, and its unchanged lines both ways.
 *
 * towardOld(hunks, n) says what happens to new-side line n going back to the
 * old side: {kind: 'added'}, or {kind: 'context' | 'outside', line} where
 * 'context' means a hunk shows it unchanged and 'outside' means no hunk shows
 * it. towardNew(hunks, o) does the same for old-side line o going forward,
 * with {kind: 'removed'} for a line the diff removes.
 */
export function readHunkLines(text) {
  const hunks = [];
  let hunk = null;
  let oldLine = 0;
  let newLine = 0;
  for (const line of text.split('\n')) {
    const m = /^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@/.exec(line);
    if (m) {
      hunk = {
        oldStart: +m[1], oldLength: m[2] === undefined ? 1 : +m[2],
        newStart: +m[3], newLength: m[4] === undefined ? 1 : +m[4],
        added: new Set(), removed: new Set(), removedAt: new Map(), newToOld: new Map(), oldToNew: new Map(),
      };
      hunks.push(hunk);
      oldLine = hunk.oldStart;
      newLine = hunk.newStart;
      continue;
    }
    if (!hunk) continue;
    const done = oldLine >= hunk.oldStart + hunk.oldLength && newLine >= hunk.newStart + hunk.newLength;
    const c = line[0];
    if (c === '+' && !done) hunk.added.add(newLine++);
    else if (c === '-' && !done) {
      hunk.removed.add(oldLine);
      hunk.removedAt.set(oldLine++, newLine);
    }
    else if ((c === ' ' || line === '') && !done) {
      hunk.newToOld.set(newLine, oldLine);
      hunk.oldToNew.set(oldLine, newLine);
      oldLine++;
      newLine++;
    }
  }
  return hunks;
}

export function towardOld(hunks, n) {
  let offset = 0;
  for (const h of hunks) {
    const start = rangeStart(h.newStart, h.newLength);
    if (n < start) break;
    if (n < start + h.newLength) {
      if (h.added.has(n)) return { kind: 'added' };
      return { kind: 'context', line: h.newToOld.get(n) };
    }
    offset = (rangeStart(h.oldStart, h.oldLength) + h.oldLength) - (start + h.newLength);
  }
  return { kind: 'outside', line: n + offset };
}

export function towardNew(hunks, o) {
  let offset = 0;
  for (const h of hunks) {
    const start = rangeStart(h.oldStart, h.oldLength);
    if (o < start) break;
    if (o < start + h.oldLength) {
      if (h.removed.has(o)) return { kind: 'removed' };
      return { kind: 'context', line: h.oldToNew.get(o) };
    }
    offset = (rangeStart(h.newStart, h.newLength) + h.newLength) - (start + h.oldLength);
  }
  return { kind: 'outside', line: o + offset };
}

function rangeStart(start, length) {
  return length === 0 ? start + 1 : start;
}
