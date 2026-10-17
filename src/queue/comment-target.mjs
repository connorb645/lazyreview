/**
 * Where a comment on lines the app shows goes, or why none can: the one
 * answer that both the apps' preview and the queue take, so they never
 * disagree.
 *
 * commentTarget(review, page, given) is pure. `given` is the lines as the
 * app shows them, {path, side, line, start_line, start_side, pull}, where a
 * range may start on the other side from its last line, and `pull` names one
 * pull request where an unchanged line of a stack shows in several. `page` is
 * the review's page data. It returns one of
 *
 *   {goes: {pull, path, side, line, start_line, start_side}, words}
 *       the pull request that takes the comment, at its own path and lines,
 *       the ones GitHub takes for its diff
 *   {choose: [{pull, path, line}, ...], words}
 *       an unchanged line of a stack that several pull requests show, each
 *       of which takes the whole range; `pull` names one
 *   {refused: words}
 *
 * where `words` say it as a sentence the apps show as it is. The lines must
 * be lines of the change's diff, and a range must stay inside one hunk of
 * it. On a stack, the line is traced through the stack's line map to the
 * pull request that last changed it. GitHub's own check, against the pull
 * request's diff as GitHub has it, stays with the queue.
 */
import { diffPlaces } from './queue-rules.mjs';
import { lineEntry } from '../build/stack/line-map.mjs';
import { english } from '../i18n/english.mjs';

export function commentTarget(review, page, given) {
  if (review.kind === 'branch') return refused(english.queue.branchTakesNone);
  if (review.moved) return refused(english.queue.moved);
  if (!page.model.files.some(f => f.path === given.path)) return refused(english.queue.notInChange);
  const startSide = given.start_line ? given.start_side || given.side : null;
  const places = diffPlaces(page.diff[given.path]);
  const end = places[given.side].get(given.line);
  if (!end) return refused(english.queue.notADiffLine);
  if (given.start_line) {
    const start = places[startSide].get(given.start_line);
    if (!start || start.hunk !== end.hunk) return refused(english.queue.leavesHunk);
    if (start.row >= end.row) return refused(english.queue.backwards);
  }
  if (!page.stack) {
    const [member] = review.members;
    if (given.pull && given.pull !== member.pull) return refused(english.queue.notThisReview(member.pull, given.pull));
    return goes(review, given.path, { pull: member.pull, path: given.path, side: given.side, line: given.line, start_line: given.start_line || null, start_side: startSide });
  }
  return onStack(review, page.stack, given, startSide);
}

function onStack(review, stack, given, startSide) {
  const heads = new Map(stack.members.map(m => [m.pull, m.head]));
  const out = review.members.filter(m => heads.get(m.pull) !== m.head).map(m => m.pull);
  if (out.length) return refused(english.queue.otherHeads(out));
  const end = lineEntry(stack.lines, given.path, given.line, given.side);
  if (end.none) return refused(english.stack.noPull(end.none));
  const ends = end.pull ? [end] : end.candidates;
  if (!ends.length) return refused(english.stack.noneShows);
  const start = given.start_line ? lineEntry(stack.lines, given.path, given.start_line, startSide) : null;
  const taking = ends.map(e => ({ end: e, start: start && startOn(start, e) })).filter(t => !start || t.start);
  if (given.pull) {
    const chosen = taking.find(t => t.end.pull === given.pull);
    if (chosen) return goes(review, given.path, target(chosen, given.side, startSide));
    if (end.pull && end.pull !== given.pull) return refused(english.queue.lastChangedBy(end.pull, given.pull));
    if (!ends.some(e => e.pull === given.pull)) return refused(english.queue.notShownBy(given.pull));
    return refused(english.queue.notAllTo(given.pull));
  }
  if (!taking.length) return refused(english.queue.notAllToOne);
  if (taking.length === 1) return goes(review, given.path, target(taking[0], given.side, startSide));
  return {
    choose: taking.map(t => ({ pull: t.end.pull, path: t.end.path, line: t.end.line })),
    words: english.queue.choose(taking.map(t => t.end.pull)),
  };
}

// The start of a range on the same pull request and file as its end, or null.
function startOn(start, end) {
  const found = start.pull ? start : (start.candidates || []).find(c => c.pull === end.pull);
  return found && found.pull === end.pull && found.path === end.path ? found : null;
}

function target({ end, start }, side, startSide) {
  return { pull: end.pull, path: end.path, side, line: end.line, start_line: start ? start.line : null, start_side: start ? startSide : null };
}

function goes(review, shownPath, place) {
  const member = review.members.find(m => m.pull === place.pull);
  if (!member) return refused(english.queue.leftStack(place.pull));
  if (member.state && member.state !== 'open') return refused(english.queue.ended(place.pull, member.state));
  return { goes: place, words: english.queue.goes(place.pull, english.review.lines(place), place.path !== shownPath ? place.path : null) };
}

function refused(words) {
  return { refused: words };
}
