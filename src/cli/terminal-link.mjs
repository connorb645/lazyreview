/**
 * Reads the link `lazyreview terminal` is given.
 *
 * readTerminalLink(text) is pure. It returns {review}, the review that a
 * review's address in the app names, as load and list print it, on any
 * port, or that a review id names; or {pull}, the pull request a pull
 * request's address names. It throws one line saying so when the text is
 * none of these.
 */
import { PullRef } from '../models/pull-ref.mjs';
import { reviewRef } from '../models/review-ref.mjs';
import { english } from '../i18n/english.mjs';

const REVIEW_ADDRESS = /^https?:\/\/(?:127\.0\.0\.1|localhost)(?::\d+)?\/review\/([^?#]+?)\/*(?:[?#].*)?$/;

export function readTerminalLink(text) {
  try {
    const address = REVIEW_ADDRESS.exec(text);
    if (address) return { review: reviewRef(address[1]) };
    if (/^https?:\/\//.test(text)) return { pull: PullRef.parse(text) };
    return { review: reviewRef(text) };
  } catch {
    throw new Error(english.cli.notALink(text));
  }
}
