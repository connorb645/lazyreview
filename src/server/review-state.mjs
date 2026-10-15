/**
 * What the app shows of one review, apart from its page data.
 *
 * reviewState(store, ref) returns the review's pull requests, its queue as
 * the buttons need it, its conversations and its written review, or null
 * when there is no review `ref`. A written review is marked `stale` when it
 * read another head than the review now shows.
 */
import { queueView } from '../queue/queue-rules.mjs';
import { readWrittenReview } from '../review/written-review.mjs';

export function reviewState(store, ref) {
  const review = store.read(ref);
  if (!review) return null;
  const text = store.readWritten(ref);
  let written = null;
  if (text) {
    const read = readWrittenReview(text);
    const head = review.members[review.members.length - 1].head;
    written = { ...read, stale: !head.startsWith(read.head) };
  }
  return {
    id: review.id, host: review.host, repo: review.repo, title: review.title, me: review.me, account: review.account,
    base: review.base, context: review.context, loaded_at: review.loaded_at, notes: review.notes, moved: review.moved,
    members: review.members.map(summary),
    earlier: review.earlier.map(summary),
    queue: queueView(review),
    threads: store.threads(ref),
    written,
  };
}

/** One line per review, for the list of every review kept. */
export function reviewSummary({ ref, review }) {
  return {
    id: ref.id, repo: review.repo, title: review.title, loaded_at: review.loaded_at, moved: review.moved,
    members: review.members.map(summary),
    queued: review.members.reduce((n, m) => n + m.comments.length, 0),
  };
}

function summary(m) {
  return { pull: m.pull, head: m.head, title: m.title, url: m.url, author: m.author, base: m.base, state: m.state, ended_at: m.ended_at, color: m.color };
}
