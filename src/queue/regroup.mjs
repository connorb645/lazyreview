/**
 * Takes a review's queue to the pull requests a load names, bottom first.
 *
 * regroup({previous, fresh, leaving, absorbed, anchor, now}) is pure:
 *
 *   previous  the review kept at this review's folder, or null for a new one
 *   fresh     each pull request as GitHub has it now, bottom first, as
 *             {pull, head, start, title, url, author, base, state, ended_at}
 *   leaving   {pull: state} for each live member of `previous` that the load
 *             no longer names, as GitHub has it now
 *   absorbed  [{ref, review}]: other reviews whose every live pull request
 *             the load names, whose queues move into this one
 *   anchor    this review's id, for the moved reviews to point at
 *
 * It returns {members, earlier, moved, notes}. A member keeps its queued
 * comments, the reviews sent from it and its colour. A comment whose pull
 * request's head moved is marked stale, as is one on a removed line, or
 * whose range starts on one, whose pull request's diff now starts from
 * another commit, as when the one below
 * is rebased or merged: its line may mean something else now, so no button
 * sends it until it is queued again or dropped. A member the load no longer
 * names leaves the stack, kept with what it had, unless it is still open with
 * comments queued, which is refused. Each absorbed review's comments get new
 * numbers here, and the review itself is returned in `moved`, emptied and
 * pointing at this one. `notes` say what moved and what went stale.
 */
import { nextId } from './queue-rules.mjs';
import { english } from '../i18n/english.mjs';

const COLORS = 8;

export function regroup({ previous, fresh, leaving = {}, absorbed = [], anchor, now }) {
  const notes = [];
  const old = new Map();
  if (previous && !previous.moved) for (const m of [...previous.members, ...previous.earlier]) old.set(m.pull, m);
  const live = new Set(previous && !previous.moved ? previous.members.map(m => m.pull) : []);

  const members = fresh.map(member => {
    const before = old.get(member.pull);
    old.delete(member.pull);
    if (!before) return { ...member, color: null, comments: [], posted: [] };
    return { ...member, color: before.color, comments: staled(before.comments, before, member, notes), posted: before.posted };
  });
  const taken = new Set(members.map(m => m.color).filter(Boolean));
  for (const m of members) {
    if (m.color) continue;
    m.color = Array.from({ length: COLORS }, (_, i) => i + 1).find(c => !taken.has(c)) || (taken.size % COLORS) + 1;
    taken.add(m.color);
  }

  const earlier = [];
  for (const [pull, member] of old) {
    if (!live.has(pull)) { earlier.push(member); continue; }
    const state = leaving[pull] || { state: member.state || 'open' };
    if (state.state === 'open' && member.comments.length) {
      throw new Error(english.queue.stillOpen(pull, member.comments.length));
    }
    earlier.push({ ...member, state: state.state, ended_at: state.ended_at || null });
    notes.push(english.queue.leftTheStack(pull, state.state));
  }

  const review = { members, earlier };
  let next = nextId(review);
  const moved = absorbed.map(({ ref, review: other }) => {
    for (const from of other.members) {
      const target = members.find(m => m.pull === from.pull);
      for (const comment of staled(from.comments, from, target, notes)) {
        target.comments.push({ ...comment, id: next });
        notes.push(english.queue.renumbered(from.pull, comment.id, next));
        next++;
      }
      for (const p of from.posted) if (!target.posted.some(q => q.review === p.review)) target.posted.push(p);
    }
    return {
      ref,
      review: {
        ...other,
        members: other.members.map(m => ({ ...m, comments: [] })),
        moved: { to: anchor, stack: fresh.map(m => m.pull), at: now },
      },
    };
  });
  return { ...review, moved, notes };
}

function staled(comments, before, now, notes) {
  let why;
  let sides;
  if (before.head !== now.head) {
    why = english.queue.staleHead(before.head.slice(0, 8), now.pull, now.head.slice(0, 8));
    sides = ['RIGHT', 'LEFT'];
  } else if (before.start && now.start && before.start !== now.start) {
    why = english.queue.staleStart(now.pull, now.start.slice(0, 8), before.start.slice(0, 8));
    sides = ['LEFT'];
  } else {
    return comments;
  }
  const touched = c => sides.includes(c.side) || (c.start_line && sides.includes(c.start_side || c.side));
  const count = comments.filter(touched).length;
  if (count) notes.push(english.queue.markedStale(now.pull, count, why));
  return comments.map(c => (touched(c) ? { ...c, stale: c.stale || why } : c));
}
