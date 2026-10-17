/**
 * Puts together the one message that asks Claude about a line, holding only
 * what the answer needs.
 *
 * questionFor({review, page, thread}) is pure. It takes the review, the page
 * data and the conversation, whose last message is the question, and returns
 * {system, prompt}: the instructions, and the message itself. The message
 * holds the pull request's title and description, or a branch's name and its
 * base, the file's part of the diff around the lines asked about, the file's
 * text around them (all of it when the file is short), and the conversation
 * so far.
 */
import { english } from '../i18n/english.mjs';

const WHOLE_FILE = 400;
const AROUND = 80;

export function questionFor({ review, page, thread }) {
  const file = page.model.files.find(f => f.path === thread.path);
  const contextLines = page.context[thread.path];
  const owner = ownerOf(review, page, thread);
  const parts = [about(review, owner)];
  parts.push(english.ask.file(thread.path, fileState(file)));
  parts.push(english.ask.about(english.review.lines(thread), owner && review.members.length > 1 ? owner.pull : null));
  const hunk = file ? hunkAround(page.diff[thread.path] || '', thread) : null;
  if (hunk) parts.push(english.ask.diff(hunk));
  const text = thread.side === 'LEFT' ? null : page.sources[thread.path] || contextLines || addedText(page.diff[thread.path], file);
  if (text && text.length) parts.push(english.ask.head(window(text.length, thread), numbered(text, thread)));
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
  const entry = page.stack?.lines[thread.path]?.[thread.side]?.[String(thread.line)];
  return Array.isArray(entry) ? review.members.find(m => m.pull === entry[0]) || null : null;
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
