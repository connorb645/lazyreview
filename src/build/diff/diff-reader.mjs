/**
 * Reads a diff and the text of every file it touches, through a Git service.
 *
 * readDiff(git, {base, head, exact}) diffs from the merge base of `base` and
 * `head` to `head`, or straight from `base` with `exact`. It returns
 * {baseCommit, headCommit, mergeBase, chunks, files}: `chunks` are the diff
 * split per file, and each of `files` is
 * {path, oldPath, status, binary, hunks, adds, dels, text, baseText, collapsed},
 * with the base text read only where the map needs it, for deleted files and
 * for changed code.
 */
import { splitDiff } from './diff-parser.mjs';
import { languageFor } from '../languages/languages.mjs';
import { english } from '../../i18n/english.mjs';

// Files that start closed in the page: generated files, data files of any
// size, and any file with a very large change.
const GENERATED = /(?:^|\/)(?:package-lock\.json|yarn\.lock|pnpm-lock\.yaml|Podfile\.lock|Package\.resolved|Gemfile\.lock|poetry\.lock|Cargo\.lock|composer\.lock)$|\.(?:pbxproj|snap|min\.js|min\.css|svg)$/;
const DATA = /\.(?:json|jsonl|ndjson|csv|tsv|xml|plist|geojson|gpx)$/i;

export function readDiff(git, { base, head, exact }) {
  const headCommit = git.resolve(head);
  const baseCommit = git.resolve(base);
  const mergeBase = exact ? baseCommit : git.mergeBase(baseCommit, headCommit);
  const chunks = splitDiff(git.diff(mergeBase, headCommit));
  if (!chunks.length) throw new Error(english.loader.noDifference(base, head));
  const counts = git.counts(mergeBase, headCommit);
  const headTexts = git.read(headCommit, chunks.filter(c => c.status !== 'D').map(c => c.path));
  const baseTexts = git.read(mergeBase, chunks
    .filter(c => c.status === 'D' || ((c.status === 'M' || c.status === 'R') && languageFor(c.path)))
    .map(c => c.oldPath));
  const files = chunks.map(c => {
    const count = counts.get(c.path) || { adds: 0, dels: 0, binary: c.binary };
    const changedLines = count.adds + count.dels;
    return {
      path: c.path, oldPath: c.oldPath, status: c.status, binary: c.binary || count.binary, hunks: c.hunks,
      adds: count.adds, dels: count.dels,
      text: c.status === 'D' ? null : headTexts.get(c.path) ?? null,
      baseText: c.oldPath ? baseTexts.get(c.oldPath) ?? null : null,
      collapsed: GENERATED.test(c.path) || (DATA.test(c.path) && changedLines > 60) || changedLines > 1500,
    };
  });
  return { baseCommit, headCommit, mergeBase, chunks, files };
}
