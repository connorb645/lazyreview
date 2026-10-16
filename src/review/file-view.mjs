/**
 * One file of a review's page, ready to draw.
 *
 * fileView(page, path) is pure. It returns null for a path the page does not
 * draw, and otherwise {path, hunks, gaps, whole, rendered}:
 *
 *   hunks  the file's diff, hunk by hunk, each {header, old_start, old_count,
 *          new_start, new_count, rows}, each row {kind, old, new, text,
 *          owner, words}: `kind` is add, del, same or note; on a stack,
 *          `owner` is the pull request that changed an added or removed
 *          row, and `words` say who changed the row and where a comment on
 *          it goes; on a single pull request or a branch both are null
 *   gaps   the runs of unchanged lines between the hunks, before the first
 *          and after the last, each {before, from, to, delta}: it comes
 *          before hunk `before`, runs from new line `from` to `to`, and
 *          each of its lines is old line `delta` further on; only a changed
 *          file whose whole text the page kept has them
 *   whole  the file's whole text at the head, as lines, or null
 *   rendered
 *          for a Markdown file with a diff, its Rendered view as
 *          markdownBlocks gives it, {blocks, partial}; otherwise null
 *
 * An unchanged file of the context has no hunks and no gaps, only its text.
 */
import { readHunks } from './diff-hunks.mjs';
import { wholeText } from './page-text.mjs';
import { markdownBlocks } from './markdown-blocks.mjs';
import { lineEntry } from '../build/stack/line-map.mjs';

const MARKDOWN = /\.(md|markdown)$/i;

export function fileView(page, path) {
  const file = page.model.files.find(f => f.path === path);
  if (!file) {
    return page.model.contextFiles.some(f => f.path === path) ? { path, hunks: [], gaps: [], whole: wholeText(page, path), rendered: null } : null;
  }
  const hunks = readHunks(page.diff[path]).map(hunk => ({ ...hunk, rows: hunk.rows.map(row => ({ ...row, ...mark(page.stack, path, row) })) }));
  const whole = wholeText(page, path);
  const rendered = MARKDOWN.test(path) && hunks.length ? markdownBlocks(file, { hunks, whole }) : null;
  return { path, hunks, gaps: file.status === 'M' || file.status === 'R' ? gaps(hunks, whole) : [], whole, rendered };
}

function mark(stack, path, row) {
  if (!stack || row.kind === 'note') return { owner: null, words: null };
  const entry = row.kind === 'del' ? lineEntry(stack.lines, path, row.old, 'LEFT') : lineEntry(stack.lines, path, row.new, 'RIGHT');
  const changed = row.kind !== 'same';
  return { owner: changed && entry.pull ? entry.pull : null, words: words(entry, changed, path) };
}

function words(entry, changed, path) {
  const at = x => `#${x.pull}, line ${x.line}${x.path !== path ? ` of ${x.path}` : ''}`;
  if (entry.pull) return `${changed ? `Changed in #${entry.pull}.` : `Unchanged here; #${entry.pull} last changed it.`} A comment here goes on ${at(entry)}.`;
  if (entry.none) return `No pull request in the stack can take a comment here: ${entry.none}.`;
  if (!entry.candidates.length) return 'Unchanged, and no pull request in the stack shows this line, so it takes no comment.';
  if (entry.candidates.length === 1) return `Unchanged. A comment here goes on ${at(entry.candidates[0])}.`;
  return `Unchanged, and shown by ${entry.candidates.map(at).join(' and ')}. A comment here names which one it is for.`;
}

function gaps(hunks, whole) {
  if (!whole || !hunks.length) return [];
  const out = [];
  let next = 1;
  hunks.forEach((hunk, i) => {
    if (hunk.new_start > next) out.push({ before: i, from: next, to: hunk.new_start - 1, delta: hunk.old_start - hunk.new_start });
    next = hunk.new_start + hunk.new_count;
  });
  const last = hunks[hunks.length - 1];
  if (next <= whole.length) out.push({ before: hunks.length, from: next, to: whole.length, delta: (last.old_start + last.old_count) - (last.new_start + last.new_count) });
  return out;
}
