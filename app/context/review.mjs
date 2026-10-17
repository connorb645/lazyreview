/**
 * One review in the app, as every view of it shares it.
 *
 * createReview(state, page) returns ctx: `state`, the review as the server
 * gave it; `data`, its page data indexed for drawing; `filter`, the pull
 * request the page is filtered to; and `pulls`, the stack's pull requests.
 * ctx.setState(next, message) puts in a new state, then calls each function
 * ctx.onChange was given, with `message` saying what the last action did.
 */
import { indexPage } from './review-data.mjs';
import { createPulls } from './pulls.mjs';

export function createReview(state, page) {
  const changes = [];
  const ctx = {
    state,
    data: indexPage(page),
    filter: { pull: null },
    setState(next, message) {
      ctx.state = next;
      changes.forEach(change => change(message));
    },
    onChange(change) {
      changes.push(change);
    },
  };
  ctx.pulls = createPulls(ctx);
  return ctx;
}
