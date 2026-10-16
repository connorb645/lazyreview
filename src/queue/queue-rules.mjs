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
 * head with one set of comments, and nothing else. Each queued comment has a
 * version, worked out from the comment as it stands, so an edit or a drop
 * names the comment exactly as its window showed it.
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
const QUOTE = 120;

/** The fields GitHub takes for one comment of a review. */
export function postedFields(comment) {
  const fields = { path: comment.path, line: comment.line, side: comment.side, body: comment.body };
  if (comment.start_line) {
    fields.start_line = comment.start_line;
    fields.start_side = comment.start_side || comment.side;
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

/**
 * Where each line of a file's patch sits, on each side: {RIGHT, LEFT}, each a
 * Map from a line number to {hunk, row}, the hunk it is in and its row in
 * the patch. The unchanged line of a row has the same place on both sides,
 * so two places compare in the order the diff shows them.
 */
export function diffPlaces(patch) {
  const places = { RIGHT: new Map(), LEFT: new Map() };
  let hunk = -1;
  let old = 0;
  let now = 0;
  const rows = patch ? patch.split('\n') : [];
  if (rows[rows.length - 1] === '') rows.pop();
  rows.forEach((row, index) => {
    const start = HUNK.exec(row);
    const at = { hunk, row: index };
    if (start) { hunk++; old = +start[1]; now = +start[2]; return; }
    if (row.startsWith('+')) places.RIGHT.set(now++, at);
    else if (row.startsWith('-')) places.LEFT.set(old++, at);
    else if (!row.startsWith('\\')) { places.RIGHT.set(now++, at); places.LEFT.set(old++, at); }
  });
  return places;
}

/**
 * What stops GitHub taking a comment on a file whose diff has `places`, as
 * words that follow the comment's name, or null when nothing does. Its line
 * must be a line of the diff, and a range's first line must be an earlier
 * line of the same hunk: GitHub refuses a range that leaves its hunk.
 */
export function placeProblem(places, comment) {
  const end = places[comment.side].get(comment.line);
  if (!end) return 'is not on a line of the diff';
  if (!comment.start_line) return null;
  const side = comment.start_side || comment.side;
  const start = places[side].get(comment.start_line);
  const first = `${side === 'LEFT' ? 'removed ' : ''}line ${comment.start_line}`;
  if (!start) return `starts on ${first}, which is not on a line of the diff`;
  if (start.hunk !== end.hunk) return `runs from ${first} across more than one hunk of the diff, and GitHub takes a range only inside one hunk`;
  if (start.row >= end.row) return `starts on ${first}, which does not come before its last line in the diff`;
  return null;
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

/** The version of a queued comment: a short hash of the comment as it stands. */
export function commentVersion(comment) {
  return createHash('sha256').update(stableJSON(comment)).digest('hex').slice(0, 12);
}

/**
 * Throws, with `refused` saying what did not happen, unless `version` is the
 * version of `comment` as it stands: it changed since its window showed it.
 */
export function checkVersion(comment, version, refused) {
  if (!version) throw new Error(`${refused}: the change does not name the version of comment ${comment.id} it was made on`);
  if (version !== commentVersion(comment)) {
    throw new Error(`${refused}: comment ${comment.id} was changed after this window showed it, and now reads "${clipped(comment.body)}"`);
  }
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
      comments: member.comments.map(c => ({ ...c, version: commentVersion(c) })),
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

function clipped(text) {
  return text.length > QUOTE ? `${text.slice(0, QUOTE)}…` : text;
}

function stableJSON(value) {
  if (Array.isArray(value)) return `[${value.map(stableJSON).join(',')}]`;
  if (value && typeof value === 'object') {
    return `{${Object.keys(value).sort().map(k => `${JSON.stringify(k)}:${stableJSON(value[k])}`).join(',')}}`;
  }
  return JSON.stringify(value);
}
