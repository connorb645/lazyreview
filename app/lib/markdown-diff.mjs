/**
 * The Rendered view of a Markdown file in a diff.
 *
 * render(before, after) takes the whole file before and after the change
 * and returns HTML: the document as it reads, with each block marked added,
 * removed or unchanged. A list is drawn item by item, so a list that changes
 * is marked item by item. Pass null as before for a file the change adds,
 * and null as after for one it deletes.
 *
 * renderParts(parts) is for a file whose whole text the app does not hold:
 * parts is [{before, after, oldStart, newStart}], one per hunk, with the
 * line each side starts on, and the output says it shows only the changed
 * parts.
 *
 * Every block, and every list item, names the source lines it was read
 * from, so a passage selected in it can be traced back to them:
 *   data-side             RIGHT for the file after the change, LEFT before
 *   data-from, data-to    its first and last line on that side
 *   data-text             the line its text starts on, where a line break
 *                         in its text is a line break in the source: a
 *                         paragraph, a heading, a code block or a list item
 *   data-left-from, data-left-to
 *                         an unchanged block's lines before the change
 *
 * The output is safe to put in a page, as safeMarkdown's is.
 */
import { safe as markdown } from './markdown.mjs';

const LINED = new Set(['paragraph', 'heading', 'code', 'list_item']);

function blocks(text, first, side) {
  const source = text || '';
  const out = [];
  let pos = 0;
  let line = first;
  for (const token of markdown.lexer(source)) {
    const at = source.indexOf(token.raw, pos);
    if (at < 0) {
      if (token.type !== 'space') out.push(token);
      continue;
    }
    line += newlines(source, pos, at);
    token.lines = span(token, line, side);
    if (token.type === 'list') {
      let cursor = at;
      let cursorLine = line;
      for (const item of token.items) {
        const i = source.indexOf(item.raw, cursor);
        if (i < 0) break;
        const itemLine = cursorLine + newlines(source, cursor, i);
        item.lines = span(item, itemLine, side);
        cursor = i + item.raw.length;
        cursorLine = itemLine + newlines(source, i, cursor);
      }
    }
    line += newlines(source, at, at + token.raw.length);
    pos = at + token.raw.length;
    if (token.type !== 'space') out.push(token);
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
  return { side, from, to: from + newlines(body, 0, body.length), text: LINED.has(token.type) ? from + (fenced ? 1 : 0) : null };
}

function key(token) {
  return token.raw.replace(/\s+$/, '');
}

// Longest common subsequence over the two sequences of keys: the steps that
// turn `before` into `after`, same, del or add, in reading order.
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

function lineData(lines, left) {
  if (!lines) return '';
  return ' data-side="' + lines.side + '" data-from="' + lines.from + '" data-to="' + lines.to + '"' +
    (lines.text !== null ? ' data-text="' + lines.text + '"' : '') +
    (left ? ' data-left-from="' + left.from + '" data-left-to="' + left.to + '"' : '');
}

function block(kind, html, lines, left) {
  const title = kind === 'add' ? ' title="Added"' : kind === 'del' ? ' title="Removed"' : '';
  return '<div class="mdx-block' + (kind === 'plain' ? '' : ' mdx-' + kind) + '"' + title + lineData(lines, left) + '>' + html + '</div>';
}

function oneItemList(list, item, number) {
  return markdown.parser([{
    type: 'list', raw: item.raw, ordered: list.ordered, start: list.ordered ? number : '',
    loose: list.loose, items: [item],
  }]);
}

function start(list) {
  return list.ordered && list.start !== '' ? Number(list.start) : 1;
}

function drawn(kind, token, left) {
  if (token.type !== 'list') return block(kind, markdown.parser([token]), token.lines, left && left.lines);
  return '<div class="mdx-list">' + token.items.map((item, i) =>
    block(kind, oneItemList(token, item, start(token) + i), item.lines, left && left.items[i] && left.items[i].lines)).join('') + '</div>';
}

// Two versions of one list: mark each item rather than the whole list.
function listChange(oldList, newList) {
  const path = steps(oldList.items.map(key), newList.items.map(key));
  return '<div class="mdx-list">' + path.map(step => {
    if (step.kind === 'add') return block('add', oneItemList(newList, newList.items[step.b], start(newList) + step.b), newList.items[step.b].lines);
    if (step.kind === 'del') return block('del', oneItemList(oldList, oldList.items[step.a], start(oldList) + step.a), oldList.items[step.a].lines);
    return block('same', oneItemList(newList, newList.items[step.b], start(newList) + step.b), newList.items[step.b].lines, oldList.items[step.a].lines);
  }).join('') + '</div>';
}

function lists(tokens) {
  return tokens.filter(token => token.type === 'list');
}

// One run of removed blocks and added blocks. When the run holds exactly one
// list on each side, of the same kind, it is one list edited: mark its items.
function run(dels, adds) {
  const oldLists = lists(dels), newLists = lists(adds);
  const edited = oldLists.length === 1 && newLists.length === 1 && oldLists[0].ordered === newLists[0].ordered;
  const out = [];
  dels.forEach(token => {
    if (!edited || token !== oldLists[0]) out.push(drawn('del', token));
  });
  adds.forEach(token => {
    out.push(edited && token === newLists[0] ? listChange(oldLists[0], token) : drawn('add', token));
  });
  return out.join('');
}

function changes(oldBlocks, newBlocks) {
  const path = steps(oldBlocks.map(key), newBlocks.map(key));
  const out = [];
  let k = 0;
  while (k < path.length) {
    const step = path[k];
    if (step.kind === 'same') { out.push(drawn('same', newBlocks[step.b], oldBlocks[step.a])); k++; continue; }
    const dels = [], adds = [];
    while (k < path.length && path[k].kind !== 'same') {
      if (path[k].kind === 'del') dels.push(oldBlocks[path[k].a]); else adds.push(newBlocks[path[k].b]);
      k++;
    }
    out.push(run(dels, adds));
  }
  return out.join('');
}

function legend(text) {
  return '<p class="mdx-note">' + text + '<span class="mdx-key mdx-add"></span>added<span class="mdx-key mdx-del"></span>removed</p>';
}

function hasMarks(body) {
  return body.indexOf('mdx-block mdx-add') >= 0 || body.indexOf('mdx-block mdx-del') >= 0;
}

export function render(before, after) {
  if (before == null) return '<div class="mdx"><p class="mdx-note">New file.</p>' + blocks(after, 1, 'RIGHT').map(t => drawn('plain', t)).join('') + '</div>';
  if (after == null) return '<div class="mdx"><p class="mdx-note">Deleted file. It read:</p>' + blocks(before, 1, 'LEFT').map(t => drawn('plain', t)).join('') + '</div>';
  const body = changes(blocks(before, 1, 'LEFT'), blocks(after, 1, 'RIGHT'));
  const note = hasMarks(body) ? legend('Marked:') : '<p class="mdx-note">The change does not alter how the file reads.</p>';
  return '<div class="mdx">' + note + body + '</div>';
}

export function renderParts(parts) {
  const bodies = parts.map(part => changes(blocks(part.before, part.oldStart || 1, 'LEFT'), blocks(part.after, part.newStart || 1, 'RIGHT')));
  return '<div class="mdx">' + legend('Only the changed parts: the app holds the diff of this file, not its whole text. Marked:') +
    bodies.join('<p class="mdx-gap">&middot; &middot; &middot;</p>') + '</div>';
}
