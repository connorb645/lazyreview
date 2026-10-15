/**
 * The rules of a review's queue: the comments waiting to go on each pull
 * request, and the three actions that send a review.
 *
 *   comment                the queued comments, as a review that only comments
 *   approve-with-comments  an approval, with the queued comments
 *   approve                an approval alone; the queued comments stay queued
 *
 * No action requests changes. Each action has its own set id, worked out
 * from the repository, the pull request, its head, the action and the exact
 * comments it sends, so a set id names one action on one pull request at one
 * head with one set of comments, and nothing else.
 *
 * Every function here is pure. A review is the record a ReviewStore keeps:
 * {repo, me, members, earlier, moved}, where each member is a pull request
 * {pull, head, author, state, comments, posted, ...}.
 */
import { createHash } from 'node:crypto';

export const ACTIONS = {
  'comment': { event: 'COMMENT', state: 'COMMENTED', comments: true },
  'approve-with-comments': { event: 'APPROVE', state: 'APPROVED', comments: true },
  'approve': { event: 'APPROVE', state: 'APPROVED', comments: false },
};

const HUNK = /^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@/;

/** The fields GitHub takes for one comment of a review. */
export function postedFields(comment) {
  const fields = { path: comment.path, line: comment.line, side: comment.side, body: comment.body };
  if (comment.start_line) {
    fields.start_line = comment.start_line;
    fields.start_side = comment.side;
  }
  return fields;
}

/** The comments an action sends. */
export function sent(member, action) {
  return ACTIONS[action].comments ? member.comments : [];
}

export function setId(review, member, action) {
  const canonical = { repo: review.repo, pull: member.pull, head: member.head, action, comments: sent(member, action).map(postedFields) };
  return createHash('sha256').update(stableJSON(canonical)).digest('hex').slice(0, 12);
}

/** The actions offered for a member: none that sends a stale comment, and no approval of one's own pull request. */
export function offered(review, member) {
  const ready = member.comments.length > 0 && !member.comments.some(c => c.stale);
  const approvable = !(member.author && member.author === review.me);
  return Object.keys(ACTIONS)
    .filter(action => ready || !ACTIONS[action].comments)
    .filter(action => approvable || ACTIONS[action].event !== 'APPROVE');
}

/** What stops a comment's text being posted. */
export function bodyProblems(body) {
  return String(body || '').trim() ? [] : ['the text is empty'];
}

/** The lines a file's patch shows, on each side: {RIGHT: Set, LEFT: Set}. */
export function linesIn(patch) {
  const right = new Set();
  const left = new Set();
  let old = 0;
  let now = 0;
  const rows = patch ? patch.split('\n') : [];
  if (rows[rows.length - 1] === '') rows.pop();
  for (const row of rows) {
    const start = HUNK.exec(row);
    if (start) { old = +start[1]; now = +start[2]; continue; }
    if (row.startsWith('+')) right.add(now++);
    else if (row.startsWith('-')) left.add(old++);
    else if (!row.startsWith('\\')) { right.add(now++); left.add(old++); }
  }
  return { RIGHT: right, LEFT: left };
}

/** The live member for `pull`, or an error saying why there is none. */
export function liveMember(review, pull) {
  if (review.moved) throw new Error(`this review moved to ${review.moved.to}; nothing changed here`);
  const member = review.members.find(m => m.pull === pull);
  if (!member) throw new Error(`#${pull} is not ${review.earlier.some(m => m.pull === pull) ? 'in this stack any more' : 'part of this review'}`);
  return member;
}

/** The next free comment number across every member, live or not. */
export function nextId(review) {
  const ids = [...review.members, ...review.earlier].flatMap(m => [...m.comments.map(c => c.id), ...m.posted.flatMap(p => p.comments.map(c => c.id))]);
  return Math.max(0, ...ids) + 1;
}

/** The queued comment numbered `id`, with the member holding it, or an error. */
export function findComment(review, id) {
  for (const member of [...review.members, ...review.earlier]) {
    const comment = member.comments.find(c => c.id === id);
    if (comment) return { member, comment };
  }
  throw new Error(`no queued comment ${id}`);
}

/**
 * What the app shows for each pull request's queue: its comments, the reviews
 * sent from it, and a set id for each action on offer. A merged or closed
 * pull request, one that left the stack, and a review that moved get no set
 * ids, so no button.
 */
export function queueView(review) {
  const view = (member, live) => {
    const ended = member.state && member.state !== 'open';
    return {
      pull: member.pull, head: member.head, title: member.title, url: member.url, author: member.author, base: member.base,
      color: member.color, state: member.state || 'open', ended_at: member.ended_at || null,
      sets: live && !ended && !review.moved ? Object.fromEntries(offered(review, member).map(a => [a, setId(review, member, a)])) : {},
      comments: member.comments,
      posted: member.posted.map(p => ({ set: p.set, action: p.action, review: p.review, at: p.at, count: p.comments.length })),
    };
  };
  return {
    repo: review.repo,
    members: review.members.map(m => view(m, true)),
    earlier: review.earlier.map(m => view(m, false)),
    moved: review.moved || null,
  };
}

function stableJSON(value) {
  if (Array.isArray(value)) return `[${value.map(stableJSON).join(',')}]`;
  if (value && typeof value === 'object') {
    return `{${Object.keys(value).sort().map(k => `${JSON.stringify(k)}:${stableJSON(value[k])}`).join(',')}}`;
  }
  return JSON.stringify(value);
}
