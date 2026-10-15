/**
 * Reads a written review, the Markdown an agent or a person attaches to a
 * review for the app to open with.
 *
 * It starts with a header between `---` lines naming what it read, then its
 * sections as `##` headings, each with something in it:
 *
 *   ---
 *   pull: 12
 *   head: 3f2a9c1d0e
 *   ---
 *   ## What this changes
 *   ...
 *
 * A stack's review names its pull requests, bottom first, in place of one,
 * as `stack: 12, 13, 14`, and its head is the top's. A `title` may name it.
 *
 * readWrittenReview(text) returns {pulls, head, title, sections: [{id,
 * title, markdown}]}, or throws an error naming every problem at once.
 *
 * checkWrittenReview(written, review) throws unless it names the review's
 * pull requests, in order, and the head of its top one.
 */
const FIELDS = ['pull', 'stack', 'head', 'title'];

export function readWrittenReview(text) {
  const problems = [];
  const m = /^---\n([\s\S]*?)\n---\n?([\s\S]*)$/.exec(String(text).replace(/\r\n/g, '\n').replace(/^﻿/, ''));
  if (!m) throw new Error('the written review must start with a header between --- lines');
  const header = {};
  for (const line of m[1].split('\n')) {
    if (!line.trim()) continue;
    const field = /^([a-z]+):\s*(.*)$/.exec(line);
    if (!field) { problems.push(`the header line "${line}" is not "name: value"`); continue; }
    if (!FIELDS.includes(field[1])) problems.push(`the header has an unknown field "${field[1]}"`);
    header[field[1]] = field[2].trim();
  }
  if (!header.head || !/^[0-9a-f]{7,40}$/i.test(header.head)) problems.push('the header needs "head", the head commit the review read');
  if (!!header.pull === !!header.stack) problems.push('the header needs "pull", or "stack" for a stack, and not both');
  const pulls = String(header.pull || header.stack || '').split(/[\s,]+/).filter(Boolean).map(p => p.replace(/^#/, ''));
  if (header.pull && (pulls.length !== 1 || !/^\d+$/.test(pulls[0]))) problems.push('the header\'s "pull" must be the pull request number');
  if (header.stack && (pulls.length < 2 || pulls.some(p => !/^\d+$/.test(p)))) problems.push('the header\'s "stack" must list two or more pull request numbers, bottom first');

  const sections = [];
  let current = null;
  let fence = null;
  for (const line of m[2].split('\n')) {
    const marker = /^(```+|~~~+)/.exec(line);
    if (marker) fence = fence === null ? marker[1][0] : (line.startsWith(fence) ? null : fence);
    const heading = fence === null && !line.startsWith('###') && /^##\s+(.+?)\s*#*\s*$/.exec(line);
    if (heading) {
      current = { title: heading[1], lines: [] };
      sections.push(current);
    } else if (current) current.lines.push(line);
    else if (line.trim()) problems.push('text before the first ## section heading');
  }
  if (!sections.length) problems.push('the written review needs at least one ## section');
  for (const s of sections) if (!s.lines.join('\n').trim()) problems.push(`"## ${s.title}" is empty; a section with nothing to say says so in one line`);
  if (problems.length) {
    throw new Error(`the written review has ${problems.length} problem${problems.length === 1 ? '' : 's'}:\n${problems.map(p => `  - ${p}`).join('\n')}`);
  }
  const ids = new Set();
  return {
    pulls: pulls.map(Number),
    head: header.head.toLowerCase(),
    title: header.title || '',
    sections: sections.map(s => ({ id: uniqueId(s.title, ids), title: s.title, markdown: s.lines.join('\n').trim() })),
  };
}

export function checkWrittenReview(written, review) {
  const pulls = review.members.map(m => m.pull);
  if (written.pulls.join(',') !== pulls.join(',')) {
    throw new Error(`the written review names ${written.pulls.map(p => `#${p}`).join(', ')}, but the review shows ${pulls.map(p => `#${p}`).join(', ')}`);
  }
  const head = review.members[review.members.length - 1].head;
  if (!head.startsWith(written.head)) {
    throw new Error(`the written review read ${written.head}, but the review shows head ${head.slice(0, 12)}; review the head the app shows`);
  }
}

function uniqueId(title, ids) {
  const base = title.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '') || 'section';
  let id = base;
  for (let k = 2; ids.has(id); k++) id = `${base}-${k}`;
  ids.add(id);
  return id;
}
