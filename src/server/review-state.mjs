/**
 * What the app shows of one review, apart from its page data.
 *
 * reviewState(store, ref) returns the review's pull requests, its queue as
 * the buttons need it, its conversations, the review threads on GitHub and
 * its written review, or null when there is no review `ref`. A written
 * review is marked `stale` when it read another head than the review now
 * shows, and each of its sections carries its `file:line` links, as
 * writtenLinks resolves them. A branch review gives its branch, its base and
 * its conversations, with `queue` null: it takes no comments and sends
 * nothing. Each state carries its `version`, a short hash of the rest of it,
 * which the change notices name.
 */
import { createHash } from 'node:crypto';
import { queueView } from '../queue/queue-rules.mjs';
import { readWrittenReview } from '../review/written-review.mjs';
import { writtenLinks } from '../review/written-links.mjs';

export function reviewState(store, ref) {
  const review = store.read(ref);
  if (!review) return null;
  if (review.kind === 'branch') {
    return versioned({
      id: review.id, kind: review.kind, repo: review.repo, path: review.path, title: review.title, base: review.base, head: review.head,
      context: review.context, loaded_at: review.loaded_at, notes: review.notes, moved: null, queue: null, threads: store.threads(ref), github: [], written: null,
    });
  }
  const text = store.readWritten(ref);
  let written = null;
  if (text) {
    const read = readWrittenReview(text);
    const head = review.members[review.members.length - 1].head;
    const drawn = store.readPage(ref);
    const paths = drawn ? [...drawn.page.model.files, ...drawn.page.model.contextFiles].map(f => f.path) : [];
    written = { ...read, sections: read.sections.map(s => ({ ...s, links: writtenLinks(s.markdown, paths) })), stale: !head.startsWith(read.head) };
  }
  return versioned({
    id: review.id, host: review.host, repo: review.repo, title: review.title, me: review.me, account: review.account,
    base: review.base, context: review.context, loaded_at: review.loaded_at, notes: review.notes, moved: review.moved,
    members: review.members.map(summary),
    earlier: review.earlier.map(summary),
    queue: queueView(review),
    threads: store.threads(ref),
    github: store.githubThreads(ref),
    written,
  });
}

function versioned(state) {
  return { ...state, version: createHash('sha256').update(JSON.stringify(state)).digest('hex').slice(0, 12) };
}

function summary(m) {
  return { pull: m.pull, head: m.head, title: m.title, url: m.url, author: m.author, base: m.base, state: m.state, ended_at: m.ended_at, color: m.color };
}
