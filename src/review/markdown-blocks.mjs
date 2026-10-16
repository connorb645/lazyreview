/**
 * The Rendered view of a Markdown file in a diff, as blocks the apps draw one
 * at a time, each with the source lines it was read from.
 *
 * markdownBlocks(file, view) is pure. `file` is the file's entry in the
 * page's model and `view` its hunks and whole text, as fileView gives them.
 * It returns {blocks, partial}: `partial` is true when the page holds only
 * the diff of the file, so the blocks show only its changed parts. Each
 * block is
 *
 *   {state, markdown, side, start_line, line, text_line, left, list, part}
 *
 *   state       add, del or same: a block of an added file is added, and of
 *               a deleted file removed
 *   markdown    its source, ready to draw alone: an item of an ordered list
 *               carries its own number
 *   side        RIGHT for the file after the change, LEFT before it
 *   start_line, line
 *               its first and last line on that side
 *   text_line   the line its text starts on, where a line break in its text
 *               is a line break in the source: a paragraph, a heading, a
 *               code block or a list item; null for any other block
 *   left        an unchanged block's {start_line, line} before the change
 *   list        a number shared by the items of one list, or null
 *   part        the hunk it comes from, when `partial`
 *
 * A list is read item by item, so a list that changes is marked item by
 * item. When a run of removed and added blocks holds exactly one list of a
 * kind, ordered or not, on each side, it is one list edited, and its items
 * are compared.
 */
import { createRequire } from 'node:module';

const { Marked } = createRequire(import.meta.url)('../../vendor/marked-12.0.2.min.js');
const lexer = new Marked({ gfm: true });
const LINED = new Set(['paragraph', 'heading', 'code', 'list_item']);

export function markdownBlocks(file, view) {
  const lists = { next: 0 };
  const side = (hunk, keep) => hunk.rows.filter(r => r.kind === 'same' || r.kind === keep).map(r => r.text);
  if (file.status === 'A') return { blocks: drawn('add', tokens(side(view.hunks[0], 'add').join('\n'), 1, 'RIGHT'), lists), partial: false };
  if (file.status === 'D') return { blocks: drawn('del', tokens(side(view.hunks[0], 'del').join('\n'), 1, 'LEFT'), lists), partial: false };
  if (view.whole) {
    let before = [];
    let next = 1;
    for (const hunk of view.hunks) {
      const first = hunk.new_count === 0 ? hunk.new_start + 1 : hunk.new_start;
      before = before.concat(view.whole.slice(next - 1, first - 1), side(hunk, 'del'));
      next = first + hunk.new_count;
    }
    const old = before.concat(view.whole.slice(next - 1)).join('\n');
    return { blocks: changes(tokens(old, 1, 'LEFT'), tokens(view.whole.join('\n'), 1, 'RIGHT'), lists), partial: false };
  }
  const blocks = view.hunks.flatMap((hunk, part) =>
    changes(tokens(side(hunk, 'del').join('\n'), hunk.old_start || 1, 'LEFT'), tokens(side(hunk, 'add').join('\n'), hunk.new_start || 1, 'RIGHT'), lists).map(b => ({ ...b, part })));
  return { blocks, partial: true };
}

// The blocks of a text, each with the lines it spans, starting at line `first`.
function tokens(text, first, side) {
  const source = text || '';
  const out = [];
  let pos = 0;
  let line = first;
  for (const token of lexer.lexer(source)) {
    const at = source.indexOf(token.raw, pos);
    if (at < 0) {
      if (token.type !== 'space') out.push({ token, lines: null, items: [] });
      continue;
    }
    line += newlines(source, pos, at);
    const read = { token, lines: span(token, line, side), items: [] };
    if (token.type === 'list') {
      let cursor = at;
      let cursorLine = line;
      for (const item of token.items) {
        const i = source.indexOf(item.raw, cursor);
        if (i < 0) break;
        const itemLine = cursorLine + newlines(source, cursor, i);
        read.items.push(span(item, itemLine, side));
        cursor = i + item.raw.length;
        cursorLine = itemLine + newlines(source, i, cursor);
      }
    }
    line += newlines(source, at, at + token.raw.length);
    pos = at + token.raw.length;
    if (token.type !== 'space') out.push(read);
  }
  return out;
}

function newlines(text, from, to) {
  let n = 0;
  for (let i = text.indexOf('\n', from); i >= 0 && i < to; i = text.indexOf('\n', i + 1)) n++;
  return n;
}

function span(token, from, side) {
  const body = token.raw.replace(/\s+$/, '');
  const fenced = token.type === 'code' && /^\s*(```|~~~)/.test(body);
  return { side, start_line: from, line: from + newlines(body, 0, body.length), text_line: LINED.has(token.type) ? from + (fenced ? 1 : 0) : null };
}

function key(token) {
  return token.raw.replace(/\s+$/, '');
}

// Longest common subsequence over two sequences of keys: the steps that turn
// `before` into `after`, same, del or add, in reading order.
function steps(before, after) {
  const n = before.length, m = after.length, width = m + 1;
  const table = new Uint32Array((n + 1) * width);
  for (let i = n - 1; i >= 0; i--) {
    for (let j = m - 1; j >= 0; j--) {
      table[i * width + j] = before[i] === after[j]
        ? table[(i + 1) * width + j + 1] + 1
        : Math.max(table[(i + 1) * width + j], table[i * width + j + 1]);
    }
  }
  const out = [];
  let i = 0, j = 0;
  while (i < n && j < m) {
    if (before[i] === after[j]) out.push({ kind: 'same', a: i++, b: j++ });
    else if (table[(i + 1) * width + j] >= table[i * width + j + 1]) out.push({ kind: 'del', a: i++ });
    else out.push({ kind: 'add', b: j++ });
  }
  while (i < n) out.push({ kind: 'del', a: i++ });
  while (j < m) out.push({ kind: 'add', b: j++ });
  return out;
}

function block(state, markdown, lines, left, list) {
  return {
    state, markdown, side: lines ? lines.side : null, start_line: lines ? lines.start_line : null, line: lines ? lines.line : null,
    text_line: lines ? lines.text_line : null, left: left ? { start_line: left.start_line, line: left.line } : null, list,
  };
}

function itemMarkdown(list, item, number) {
  const raw = item.raw.replace(/\s+$/, '');
  return list.ordered ? raw.replace(/^(\s*)\d+(?=[.)])/, `$1${number}`) : raw;
}

function start(list) {
  return list.ordered && list.start !== '' ? Number(list.start) : 1;
}

function drawn(state, reads, lists, olds = []) {
  return reads.flatMap((read, i) => {
    const old = olds[i];
    if (read.token.type !== 'list') return [block(state, key(read.token), read.lines, old && old.lines, null)];
    const list = lists.next++;
    return read.token.items.map((item, k) => block(state, itemMarkdown(read.token, item, start(read.token) + k), read.items[k], old && old.items[k], list));
  });
}

// Two versions of one list: each item marked, rather than the whole list.
function listChange(oldRead, newRead, lists) {
  const oldList = oldRead.token, newList = newRead.token;
  const list = lists.next++;
  return steps(oldList.items.map(key), newList.items.map(key)).map(step => {
    if (step.kind === 'add') return block('add', itemMarkdown(newList, newList.items[step.b], start(newList) + step.b), newRead.items[step.b], null, list);
    if (step.kind === 'del') return block('del', itemMarkdown(oldList, oldList.items[step.a], start(oldList) + step.a), oldRead.items[step.a], null, list);
    return block('same', itemMarkdown(newList, newList.items[step.b], start(newList) + step.b), newRead.items[step.b], oldRead.items[step.a], list);
  });
}

function run(dels, adds, lists) {
  const edited = new Map();
  for (const ordered of [true, false]) {
    const kind = r => r.token.type === 'list' && r.token.ordered === ordered;
    const olds = dels.filter(kind), news = adds.filter(kind);
    if (olds.length === 1 && news.length === 1) edited.set(news[0], olds[0]);
  }
  const paired = new Set(edited.values());
  return [
    ...drawn('del', dels.filter(r => !paired.has(r)), lists),
    ...adds.flatMap(r => (edited.has(r) ? listChange(edited.get(r), r, lists) : drawn('add', [r], lists))),
  ];
}

function changes(olds, news, lists) {
  const path = steps(olds.map(r => key(r.token)), news.map(r => key(r.token)));
  const out = [];
  let k = 0;
  while (k < path.length) {
    const step = path[k];
    if (step.kind === 'same') { out.push(...drawn('same', [news[step.b]], lists, [olds[step.a]])); k++; continue; }
    const dels = [], adds = [];
    while (k < path.length && path[k].kind !== 'same') {
      if (path[k].kind === 'del') dels.push(olds[path[k].a]); else adds.push(news[path[k].b]);
      k++;
    }
    out.push(...run(dels, adds, lists));
  }
  return out;
}
