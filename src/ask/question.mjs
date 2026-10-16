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
import { linesWords } from '../review/lines-words.mjs';

const WHOLE_FILE = 400;
const AROUND = 80;

export const SYSTEM = [
  'You help someone review a change to code.',
  'You see one file of it: the part of the diff around the lines they ask about, the code around those lines, and their question.',
  'Answer in plain words and short sentences, and only what they asked.',
  'Say so when the code you were shown is not enough to be sure.',
  'When they ask you to write a review comment, reply with the comment text alone, as they would write it: short, plain, one point.',
  'Never use the em dash; use a plain dash.',
].join(' ');

export function questionFor({ review, page, thread }) {
  const file = page.model.files.find(f => f.path === thread.path);
  const contextLines = page.context[thread.path];
  const owner = ownerOf(review, page, thread);
  const parts = [about(review, owner)];
  parts.push(`File: ${thread.path}${file ? fileState(file) : ' (unchanged by this change)'}`);
  parts.push(`They ask about ${linesWords(thread)}${owner && review.members.length > 1 ? `, which #${owner.pull} changed` : ''}.`);
  const hunk = file ? hunkAround(page.diff[thread.path] || '', thread) : null;
  if (hunk) parts.push(`The diff of this file around those lines:\n\`\`\`diff\n${hunk}\n\`\`\``);
  const text = thread.side === 'LEFT' ? null : page.sources[thread.path] || contextLines || addedText(page.diff[thread.path], file);
  if (text && text.length) parts.push(`The file at the head${window(text.length, thread)}:\n\`\`\`\n${numbered(text, thread)}\n\`\`\``);
  const earlier = thread.messages.slice(0, -1).filter(m => m.text);
  if (earlier.length) parts.push(`The conversation so far:\n${earlier.map(m => `${m.from === 'claude' ? 'You answered' : 'They asked'}: ${m.text}`).join('\n\n')}`);
  parts.push(`Their question:\n${thread.messages[thread.messages.length - 1].text}`);
  return { system: SYSTEM, prompt: parts.join('\n\n') };
}

function about(review, owner) {
  if (review.kind === 'branch') return branch(review);
  if (review.members.length === 1) return describe(review.members[0]);
  const list = review.members.map(m => `#${m.pull}: ${m.title}`).join('\n');
  return `A stack of pull requests, bottom first, reviewed together:\n${list}${owner ? `\n\n${describe(owner)}` : ''}`;
}

function describe(member) {
  return `Pull request #${member.pull}: ${member.title}${member.description ? `\n\nIts description:\n${member.description}` : ''}`;
}

function branch(review) {
  const named = review.title !== review.head.ref ? `: ${review.title}` : '';
  return `Branch ${review.head.ref}${named}. It has no pull request yet, and its diff is against ${review.base.ref}.`;
}

function ownerOf(review, page, thread) {
  if (review.kind === 'branch') return null;
  if (review.members.length === 1) return review.members[0];
  const entry = page.stack?.lines[thread.path]?.[thread.side]?.[String(thread.line)];
  return Array.isArray(entry) ? review.members.find(m => m.pull === entry[0]) || null : null;
}

function fileState(file) {
  if (file.status === 'A') return ' (new)';
  if (file.status === 'D') return ' (deleted)';
  if (file.status === 'R') return ` (renamed from ${file.oldPath})`;
  return ' (changed)';
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
  return from === 1 && to === count ? '' : `, lines ${from}-${to} of ${count}`;
}

function numbered(text, thread) {
  const [from, to] = range(text.length, thread);
  const width = String(to).length;
  return text.slice(from - 1, to).map((row, i) => `${String(from + i).padStart(width)} | ${row}`).join('\n');
}
