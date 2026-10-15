/**
 * Traces a line of a stack's combined diff to the pull request that last
 * changed it, and to the line that pull request's own diff gives it.
 *
 * traceNew(steps, path, line) starts from a line of the top head (the new
 * side). The member whose head holds the line as the top has it, and whose
 * own diff adds it, owns it: {pull, path, line, side: 'RIGHT'}, at that
 * member's head, which is the line GitHub takes for its diff. In a stack in
 * step only one member can; in one out of step, where a member below was
 * rebased and the one above still holds its older commits, both can, and the
 * lower one owns it, since the change is its own and stays there once the
 * upper one is rebased. A line no member adds is unchanged by the stack:
 * {context: [...]}, listing each member whose head holds it and whose own
 * diff shows it as an unchanged line, at that member's numbers, bottom first.
 *
 * traceOld(steps, path, line) does the same for a line of the combined base
 * (the old side): the lowest member that starts with the line and removes it
 * owns it, with side 'LEFT', the line's number where that member starts, and
 * the path the member's diff gives the file, which is its new name when the
 * member renames it, as GitHub takes it.
 *
 * `steps` are readStackSteps' steps, bottom first.
 */
import { towardOld, towardNew } from './hunk-lines.mjs';

export function traceNew(steps, path, line) {
  const shown = [];
  for (const s of steps) {
    const at = back(s.fromTop, { path, line });
    const file = at && s.own.byNew.get(at.path);
    if (!file) continue;
    const step = towardOld(file.hunks, at.line);
    if (step.kind === 'added') return { pull: s.pull, ...at, side: 'RIGHT' };
    if (step.kind === 'context') shown.push({ pull: s.pull, ...at, side: 'RIGHT' });
  }
  return { context: shown };
}

export function traceOld(steps, path, line) {
  const shown = [];
  for (const s of steps) {
    const at = forward(s.fromBase, { path, line });
    const file = at && s.own.byOld.get(at.path);
    if (!file) continue;
    const step = towardNew(file.hunks, at.line);
    const target = { pull: s.pull, path: file.path || at.path, line: at.line, side: 'LEFT' };
    if (step.kind === 'removed') return target;
    if (step.kind === 'context') shown.push(target);
  }
  return { context: shown };
}

function back(diff, at) {
  const file = diff.byNew.get(at.path);
  if (!file) return at;
  const step = towardOld(file.hunks, at.line);
  return step.kind === 'added' ? null : { path: file.oldPath || at.path, line: step.line };
}

function forward(diff, at) {
  const file = diff.byOld.get(at.path);
  if (!file) return at;
  const step = towardNew(file.hunks, at.line);
  return step.kind === 'removed' ? null : { path: file.path || at.path, line: step.line };
}
