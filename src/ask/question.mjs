/**
 * Puts together the one message that asks Claude about a line, and says
 * which commit's code it may read to answer.
 *
 * headAsked({review, page, thread}) is pure. It takes the review, the page
 * data and the conversation, and returns {commit, pull, path, lines}: the
 * commit whose code Claude reads, on a stack the pull request it heads, else
 * null, and where the lines asked about are in that code, as the file's path
 * and its [first, last] lines, or null lines for removed ones. The commit is
 * the head of the pull request that changed the lines, or of the stack's top
 * when none did, or a branch's commit as it was loaded.
 *
 * copyBranches({review, page, thread}) is pure. It returns {branches,
 * branch, base}: the branches the copy of the code holds, {name: commit},
 * the one checked out, at headAsked's commit, and the base's: the base
 * branch, and each pull request's head as pull/<number>, or the branch.
 *
 * questionFor({review, page, thread, folder}) is pure. It takes the same,
 * whose last message is the question, and `folder`, the copy, and returns
 * {system, prompt}: the instructions, and the message itself. The message
 * holds the pull request's title and description, or a branch's name and
 * its base, where the copy is and its branches, the files the change
 * touches, the file asked about and where the lines are, the file's part of
 * the diff around them, the file's text around them (all of it when the file
 * is short), and the conversation so far. followUpFor({thread}) is the
 * message that asks the conversation's last question in a Claude Code
 * session that already holds the rest.
 */
import { join } from 'node:path';
import { lineEntry } from '../build/stack/line-map.mjs';
import { english } from '../i18n/english.mjs';

const WHOLE_FILE = 400;
const AROUND = 80;
const LISTED = 200;

export function headAsked({ review, page, thread }) {
  if (review.kind === 'branch') return { commit: review.head.sha, pull: null, ...placeShown(thread) };
  if (review.members.length === 1) return { commit: review.members[0].head, pull: null, ...placeShown(thread) };
  const owner = ownerOf(review, page, thread);
  if (owner) return { commit: owner.head, pull: owner.pull, ...placeTraced(page.stack.lines, thread) };
  const top = review.members[review.members.length - 1];
  return { commit: top.head, pull: top.pull, ...placeShown(thread) };
}

export function copyBranches({ review, page, thread }) {
  if (review.kind === 'branch') {
    return { branches: { [review.base.ref]: review.base.sha, [review.head.ref]: review.head.sha }, branch: review.head.ref, base: review.base.ref };
  }
  const head = headAsked({ review, page, thread });
  const branches = { [review.base.ref]: review.base.sha };
  for (const m of review.members) branches[`pull/${m.pull}`] = m.head;
  return { branches, branch: `pull/${head.pull ?? review.members[0].pull}`, base: review.base.ref };
}

export function followUpFor({ thread }) {
  return { system: english.ask.system, prompt: english.ask.followUp(thread.messages[thread.messages.length - 1].text) };
}

export function questionFor({ review, page, thread, folder }) {
  const file = page.model.files.find(f => f.path === thread.path);
  const contextLines = page.context[thread.path];
  const owner = ownerOf(review, page, thread);
  const head = headAsked({ review, page, thread });
  const top = review.kind !== 'branch' && review.members.length > 1 ? review.members[review.members.length - 1].pull : null;
  const parts = [about(review, owner)];
  const copy = copyBranches({ review, page, thread });
  parts.push(english.ask.codebase(folder, head.pull, copy.branch, copy.base));
  const files = page.model.files.map(f => ({ path: f.path, state: fileState(f) }));
  parts.push(english.ask.changes(files.slice(0, LISTED), Math.max(0, files.length - LISTED)));
  parts.push(english.ask.file(thread.path, fileState(file), file?.status === 'D' ? null : join(folder, head.path)));
  parts.push(english.ask.about(english.review.lines(thread), owner && review.members.length > 1 ? owner.pull : null));
  if (head.lines && (head.path !== thread.path || head.lines[0] !== firstLine(thread) || head.lines[1] !== thread.line)) {
    parts.push(english.ask.there(head.pull, english.review.lines({ side: 'RIGHT', line: head.lines[1], start_line: head.lines[0] !== head.lines[1] ? head.lines[0] : null })));
  }
  const hunk = file ? hunkAround(page.diff[thread.path] || '', thread) : null;
  if (hunk) parts.push(english.ask.diff(hunk));
  const text = thread.side === 'LEFT' ? null : page.sources[thread.path] || contextLines || addedText(page.diff[thread.path], file);
  if (text && text.length) parts.push(english.ask.head(window(text.length, thread), numbered(text, thread), top !== head.pull ? top : null));
  const earlier = thread.messages.slice(0, -1).filter(m => m.text);
  if (earlier.length) parts.push(english.ask.conversation(earlier));
  parts.push(english.ask.question(thread.messages[thread.messages.length - 1].text));
  return { system: english.ask.system, prompt: parts.join('\n\n') };
}

function about(review, owner) {
  if (review.kind === 'branch') return english.ask.branch(review.head.ref, review.title, review.base.ref);
  if (review.members.length === 1) return describe(review.members[0]);
  return english.ask.stack(review.members, owner ? describe(owner) : null);
}

function describe(member) {
  return english.ask.pull(member.pull, member.title, member.description);
}

function ownerOf(review, page, thread) {
  if (review.kind === 'branch') return null;
  if (review.members.length === 1) return review.members[0];
  const end = lineEntry(page.stack?.lines || {}, thread.path, thread.line, thread.side);
  return end.pull ? review.members.find(m => m.pull === end.pull) || null : null;
}

function placeShown(thread) {
  return { path: thread.path, lines: thread.side === 'RIGHT' ? [firstLine(thread), thread.line] : null };
}

function placeTraced(lines, thread) {
  const end = lineEntry(lines, thread.path, thread.line, thread.side);
  if (thread.side !== 'RIGHT') return { path: end.path, lines: null };
  const start = firstLine(thread) === thread.line ? end : lineEntry(lines, thread.path, thread.start_line, 'RIGHT');
  return { path: end.path, lines: [start.pull === end.pull && start.path === end.path ? start.line : end.line, end.line] };
}

function fileState(file) {
  if (!file) return english.ask.unchanged;
  if (file.status === 'A') return english.ask.added;
  if (file.status === 'D') return english.ask.deleted;
  if (file.status === 'R') return english.ask.renamed(file.oldPath);
  return english.ask.changed;
}

function hunkAround(diff, thread) {
  const hunks = [];
  let current = null;
  for (const row of diff.split('\n')) {
    const m = /^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@/.exec(row);
    if (m) {
      current = { rows: [row], old: [+m[1], +m[1] + (m[2] === undefined ? 1 : +m[2]) - 1], now: [+m[3], +m[3] + (m[4] === undefined ? 1 : +m[4]) - 1] };
      hunks.push(current);
    } else if (current) current.rows.push(row);
  }
  if (!hunks.length) return null;
  const span = thread.side === 'LEFT' ? 'old' : 'now';
  const first = firstLine(thread);
  const distance = h => (thread.line < h[span][0] ? h[span][0] - thread.line : first > h[span][1] ? first - h[span][1] : 0);
  return hunks.reduce((best, h) => (distance(h) < distance(best) ? h : best)).rows.join('\n').replace(/\n$/, '');
}

function addedText(diff, file) {
  if (!file || file.status !== 'A' || !diff) return null;
  return diff.split('\n').filter(r => r.startsWith('+') && !r.startsWith('+++')).map(r => r.slice(1));
}

function firstLine(thread) {
  return thread.start_line && (thread.start_side || thread.side) === thread.side ? thread.start_line : thread.line;
}

function range(count, thread) {
  if (count <= WHOLE_FILE) return [1, count];
  const first = firstLine(thread);
  return [Math.max(1, first - AROUND), Math.min(count, thread.line + AROUND)];
}

function window(count, thread) {
  const [from, to] = range(count, thread);
  return from === 1 && to === count ? null : { from, to, count };
}

function numbered(text, thread) {
  const [from, to] = range(text.length, thread);
  const width = String(to).length;
  return text.slice(from - 1, to).map((row, i) => `${String(from + i).padStart(width)} | ${row}`).join('\n');
}
