/**
 * Splits `git diff` output into one chunk per file and reads each chunk's
 * paths, status and hunks.
 *
 * splitDiff(text) returns [{path, oldPath, status, binary, text, hunks}],
 * where `text` is the file's own part of the diff, `status` is A, M, D or R,
 * and each hunk is {oldStart, oldLength, newStart, newLength, added, removedAt}:
 * the new-side line numbers it adds, and the new-side positions where it
 * removes lines.
 */
export function splitDiff(text) {
  const chunks = [];
  const lines = text.split('\n');
  let start = -1;
  for (let i = 0; i <= lines.length; i++) {
    if (i === lines.length || lines[i].startsWith('diff --git ')) {
      if (start >= 0) chunks.push(lines.slice(start, trimEnd(lines, start, i)));
      start = i;
    }
  }
  return chunks.map(readChunk);
}

function trimEnd(lines, start, end) {
  let e = end;
  while (e > start + 1 && lines[e - 1] === '') e--;
  return e;
}

function unquote(path) {
  if (!path.startsWith('"')) return path;
  const inner = path.slice(1, -1).replace(/\\([0-7]{3})/g, (_, o) => String.fromCharCode(parseInt(o, 8)))
    .replace(/\\(.)/g, (_, c) => ({ n: '\n', t: '\t' }[c] || c));
  return Buffer.from(inner, 'latin1').toString('utf8');
}

function stripPrefix(path, prefix) {
  const p = unquote(path.replace(/\t.*$/, ''));
  return p.startsWith(prefix) ? p.slice(prefix.length) : p;
}

// `diff --git a/x b/x`: when both sides name the same path, it is the
// header's two halves, which is the only reliable reading when the path holds
// spaces and the chunk has no ---/+++ lines.
function headerPath(header) {
  const rest = header.slice('diff --git '.length);
  if (rest.startsWith('"')) {
    const m = /^("(?:[^"\\]|\\.)*") ("(?:[^"\\]|\\.)*"|.*)$/.exec(rest);
    if (m) return { a: stripPrefix(m[1], 'a/'), b: stripPrefix(m[2], 'b/') };
  }
  const half = (rest.length - 1) / 2;
  if (Number.isInteger(half) && rest.slice(0, half).slice(2) === rest.slice(half + 1).slice(2)) {
    return { a: rest.slice(2, half), b: rest.slice(half + 3) };
  }
  const m = /^a\/(.*) b\/(.*)$/.exec(rest);
  return m ? { a: m[1], b: m[2] } : { a: rest, b: rest };
}

function readChunk(lines) {
  const fromHeader = headerPath(lines[0]);
  let oldPath = null;
  let newPath = null;
  let status = 'M';
  let binary = false;
  const hunks = [];
  let hunk = null;
  let oldLine = 0;
  let newLine = 0;
  for (let i = 1; i < lines.length; i++) {
    const line = lines[i];
    if (!hunk) {
      if (line.startsWith('new file mode')) status = 'A';
      else if (line.startsWith('deleted file mode')) status = 'D';
      else if (line.startsWith('rename from ')) { status = 'R'; oldPath = unquote(line.slice(12)); }
      else if (line.startsWith('rename to ')) newPath = unquote(line.slice(10));
      else if (line.startsWith('copy from ')) { status = 'A'; oldPath = unquote(line.slice(10)); }
      else if (line.startsWith('copy to ')) newPath = unquote(line.slice(8));
      else if (line.startsWith('Binary files ') || line === 'GIT binary patch') binary = true;
      else if (line.startsWith('--- ')) { const p = line.slice(4); if (p !== '/dev/null') oldPath = stripPrefix(p, 'a/'); }
      else if (line.startsWith('+++ ')) { const p = line.slice(4); if (p !== '/dev/null') newPath = stripPrefix(p, 'b/'); }
    }
    const m = /^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@/.exec(line);
    if (m) {
      hunk = {
        oldStart: +m[1], oldLength: m[2] === undefined ? 1 : +m[2],
        newStart: +m[3], newLength: m[4] === undefined ? 1 : +m[4],
        added: [], removedAt: [],
      };
      hunks.push(hunk);
      oldLine = hunk.oldStart;
      newLine = hunk.newStart;
      continue;
    }
    if (!hunk) continue;
    const c = line[0];
    if (c === '+') { hunk.added.push(newLine++); }
    else if (c === '-') { hunk.removedAt.push(newLine); oldLine++; }
    else if (c === ' ' || line === '') { oldLine++; newLine++; }
  }
  if (status === 'D') newPath = null;
  if (status === 'A' && !oldPath) oldPath = null;
  const path = newPath || oldPath || fromHeader.b;
  return {
    path,
    oldPath: status === 'R' || status === 'D' ? (oldPath || fromHeader.a) : (status === 'A' ? null : path),
    status,
    binary,
    text: lines.join('\n'),
    hunks,
  };
}

/**
 * Whether the file's hunks add a line from `start` to `end` (1-based,
 * inclusive, new side), or remove lines from between two of them.
 */
export function touches(hunks, start, end) {
  return hunks.some(h =>
    h.added.some(n => n >= start && n <= end) ||
    h.removedAt.some(n => n > start && n <= end));
}
