/**
 * The whole text of a file of a review's page, as lines.
 *
 * wholeText(page, path) is pure. It gives a changed file's text at the head
 * when the page kept it, an added file's from its diff, an unchanged file of
 * the context its own, and null for any other: a deleted file, or one whose
 * text was too large to keep.
 */
import { readHunks } from './diff-hunks.mjs';

export function wholeText(page, path) {
  if (page.sources[path]) return page.sources[path];
  if (page.context[path]) return page.context[path];
  const file = page.model.files.find(f => f.path === path);
  if (!file || file.status !== 'A') return null;
  return readHunks(page.diff[path]).flatMap(hunk => hunk.rows.filter(row => row.kind === 'add').map(row => row.text));
}
