/**
 * Which pull requests of a stack changed each type the map draws.
 *
 * nodePulls(nodes, chunks, mapped, pulls) takes the nodes to draw, the
 * combined diff split per file, mapStackLines' result and the members'
 * numbers, bottom first. It returns one list per node, in stack order: the
 * pull requests that changed a line inside the type, or removed one from
 * between two of its lines, as the combined diff shows it. A file box, an
 * extension or a removed type takes its file's pull requests, and a type
 * also takes those of every other file whose extensions are folded into it.
 * An unchanged type has none.
 */
import { readHunkLines } from './hunk-lines.mjs';

export function nodePulls(nodes, chunks, mapped, pulls) {
  const hunksByPath = new Map(chunks.map(c => [c.path, readHunkLines(c.text)]));
  const fileOf = path => new Set(mapped.files[path]?.pulls || []);
  return nodes.map(n => {
    if (n.state === 'same' || n.state === 'context') return [];
    const found = n.kind === 'artifact' || n.kind === 'extension' || n.state === 'removed' || !n.end
      ? fileOf(n.file)
      : inSpan(hunksByPath.get(n.file) || [], mapped.lines[n.file], n.line, n.end);
    for (const path of n.extFiles || []) for (const p of fileOf(path)) found.add(p);
    return pulls.filter(p => found.has(p));
  });
}

function inSpan(hunks, lines, start, end) {
  const found = new Set();
  if (!lines) return found;
  for (const h of hunks) {
    for (const n of h.added) if (n >= start && n <= end && Array.isArray(lines.RIGHT[n])) found.add(lines.RIGHT[n][0]);
    for (const [o, at] of h.removedAt) if (at > start && at <= end && Array.isArray(lines.LEFT[o])) found.add(lines.LEFT[o][0]);
  }
  return found;
}
