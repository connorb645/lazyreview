/**
 * Reads, through a Git service, the diffs that tie each pull request of a
 * stack to the stack's combined diff.
 *
 * readStackSteps(git, {combinedBase, baseCommit, top, members}) returns
 * {heads, steps, warnings, counts}, with one step per member, bottom first:
 *
 *   pull       the member's number
 *   own        the diff that pull request shows: from where it starts (the
 *              merge base of the member below's head, or of the base, with
 *              its own head) to its head
 *   fromTop    from its head to the stack's top, which says where a line of
 *              the top sits at its head, if it is there
 *   fromBase   from the combined base to where it starts, which says where a
 *              line of the base sits there, if it is there
 *
 * Each diff maps paths to {path, oldPath, status, hunks}, by new path in
 * `byNew` and by old path in `byOld`; a diff between a commit and itself is
 * empty. `warnings` say where the stack is out of step: a pull request not
 * built on the head of the one below. `counts` give each member's own files,
 * added and removed lines, and commits.
 */
import { splitDiff } from '../diff/diff-parser.mjs';
import { readHunkLines } from './hunk-lines.mjs';

export function readStackSteps(git, { combinedBase, baseCommit, top, members }) {
  const heads = members.map(m => {
    try {
      return git.resolve(m.head);
    } catch {
      throw new Error(`#${m.pull}'s head ${m.head.slice(0, 12)} is not a commit in this repository; fetch it with git fetch origin pull/${m.pull}/head`);
    }
  });
  const warnings = [];
  const steps = members.map((m, i) => {
    const start = git.mergeBase(i ? heads[i - 1] : baseCommit, heads[i]);
    if (i && start !== heads[i - 1]) {
      const under = `#${members[i - 1].pull}`;
      warnings.push(`#${m.pull} is not built on ${under}'s head: ${under} has commits #${m.pull} does not hold. ` +
        `Until #${m.pull} is rebased, its own diff also shows the older ${under} it holds; a comment on a line ${under}'s head still holds goes to ${under}.`);
    }
    return { pull: m.pull, start, own: indexed(git, start, heads[i]), fromTop: indexed(git, heads[i], top), fromBase: indexed(git, combinedBase, start) };
  });
  const counts = steps.map((s, i) => {
    const lines = [...git.counts(s.start, heads[i]).values()];
    return {
      pull: s.pull, files: lines.length, commits: git.commitCount(s.start, heads[i]),
      adds: lines.reduce((t, c) => t + c.adds, 0), dels: lines.reduce((t, c) => t + c.dels, 0),
    };
  });
  return { heads, steps, warnings, counts };
}

function indexed(git, from, to) {
  const byNew = new Map();
  const byOld = new Map();
  if (from === to) return { byNew, byOld };
  for (const chunk of splitDiff(git.diff(from, to))) {
    const file = { path: chunk.status === 'D' ? null : chunk.path, oldPath: chunk.oldPath, status: chunk.status, hunks: readHunkLines(chunk.text) };
    if (file.path) byNew.set(file.path, file);
    if (file.oldPath) byOld.set(file.oldPath, file);
  }
  return { byNew, byOld };
}
