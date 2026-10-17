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
import { english } from '../i18n/english.mjs';

const FIELDS = ['pull', 'stack', 'head', 'title'];

export function readWrittenReview(text) {
  const problems = [];
  const m = /^---\n([\s\S]*?)\n---\n?([\s\S]*)$/.exec(String(text).replace(/\r\n/g, '\n').replace(/^﻿/, ''));
  if (!m) throw new Error(english.written.noHeader);
  const header = {};
  for (const line of m[1].split('\n')) {
    if (!line.trim()) continue;
    const field = /^([a-z]+):\s*(.*)$/.exec(line);
    if (!field) { problems.push(english.written.headerLine(line)); continue; }
    if (!FIELDS.includes(field[1])) problems.push(english.written.unknownField(field[1]));
    header[field[1]] = field[2].trim();
  }
  if (!header.head || !/^[0-9a-f]{7,40}$/i.test(header.head)) problems.push(english.written.needsHead);
  if (!!header.pull === !!header.stack) problems.push(english.written.pullOrStack);
  const pulls = String(header.pull || header.stack || '').split(/[\s,]+/).filter(Boolean).map(p => p.replace(/^#/, ''));
  if (header.pull && (pulls.length !== 1 || !/^\d+$/.test(pulls[0]))) problems.push(english.written.pullNumber);
  if (header.stack && (pulls.length < 2 || pulls.some(p => !/^\d+$/.test(p)))) problems.push(english.written.stackNumbers);

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
    else if (line.trim()) problems.push(english.written.textBefore);
  }
  if (!sections.length) problems.push(english.written.noSection);
  for (const s of sections) if (!s.lines.join('\n').trim()) problems.push(english.written.emptySection(s.title));
  if (problems.length) throw new Error(english.written.problems(problems));
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
  if (written.pulls.join(',') !== pulls.join(',')) throw new Error(english.written.otherPulls(written.pulls, pulls));
  const head = review.members[review.members.length - 1].head;
  if (!head.startsWith(written.head)) throw new Error(english.written.otherHead(written.head, head.slice(0, 12)));
}

function uniqueId(title, ids) {
  const base = title.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '') || 'section';
  let id = base;
  for (let k = 2; ids.has(id); k++) id = `${base}-${k}`;
  ids.add(id);
  return id;
}
