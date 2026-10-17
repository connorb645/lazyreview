/**
 * The markup of a review's bar and of a stack's pull request strip.
 *
 *   headingContext(pieces)   the line under the title, [[text, emphasis]]
 *                            pieces with the emphasis in bold
 *   totals(t)                a count of files with its lines added and
 *                            removed, {files, adds, dels}
 *   filteredTotals(pulls, pull, t)
 *                            the same for one pull request, after its pill
 *   allPulls(pulls, base)    the strip's words with no filter: every pull
 *                            request, from `base` to the top's head
 *   onePull(member, t)       the strip's words for one pull request
 *   pullButtons(pulls)       the strip's buttons, All and one per pull request
 *   warnLine(words)          a warning about the stack
 *   backLabel(from)          the back button's words, to the view it returns to
 */
import { esc, fmt } from '../lib/text.mjs';
import { pullPill } from './pulls.mjs';
import { english } from '/i18n/english.mjs';

const words = english.web.bar;

export function headingContext(pieces) {
  return pieces.map(part => (part[1] ? '<b>' + esc(part[0]) + '</b>' : esc(part[0]))).join('');
}

export function totals(t) {
  return english.web.counts.files(t.files) + ' &nbsp;<span class="plus">+' + fmt(t.adds) + '</span> <span class="minus">&minus;' + fmt(t.dels) + '</span>';
}

export function filteredTotals(pulls, pull, t) {
  return pullPill(pulls, pull) + ' ' + totals(t);
}

export function allPulls(pulls, base) {
  return words.allPulls(pulls.members.length, esc(base), esc(pulls.members[pulls.members.length - 1].pull));
}

export function onePull(member, t) {
  return (member.title ? '<b>' + esc(member.title) + '</b> &middot; ' : '') + words.ofCombined(totals(t));
}

export function pullButtons(pulls) {
  return '<button type="button" class="pf" data-pull="" aria-pressed="true">' + words.all + '</button>' + pulls.members.map(m =>
    '<button type="button" class="pf" data-pull="' + m.pull + '" aria-pressed="false" style="--pc:' + pulls.pullColor(m.pull) + '"' +
    (m.title ? ' title="' + esc(m.title) + '"' : '') + '><i></i>#' + m.pull + '</button>').join('');
}

export function warnLine(warning) {
  return '<p class="warn-line">' + esc(warning) + '</p>';
}

export function backLabel(from) {
  return from === 'review' ? words.backToReview : words.backToDiagram;
}
