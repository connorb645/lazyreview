/**
 * The markup of the button a selected passage of a Rendered view shows.
 *
 * passageButton() is the button, hidden until a passage is selected, and
 * passageLabel(queue, lines) its words for `lines`: a comment on them when
 * the review has a queue, otherwise a question to Claude.
 */
import { esc } from '../lib/text.mjs';
import { english } from '/i18n/english.mjs';

export function passageButton() {
  return '<button type="button" class="passage-pick" hidden=""></button>';
}

export function passageLabel(queue, lines) {
  return (queue ? english.web.notes.commentOn : english.web.notes.askClaudeAbout) + esc(english.review.lines(lines));
}
