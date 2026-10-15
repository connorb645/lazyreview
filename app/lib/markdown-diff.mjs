/**
 * The Rendered view of a Markdown file in a diff.
 *
 * render(before, after) takes the whole file before and after the change
 * and returns HTML: the document as it reads, with each block marked added,
 * removed or unchanged. A list that changes is marked item by item. Pass null
 * as before for a file the change adds, and null as after for one it deletes.
 *
 * renderParts(parts) is for a file whose whole text the app does not hold:
 * parts is [{before, after}], one per hunk, and the output says it shows only
 * the changed parts.
 *
 * The output is safe to put in a page, as safeMarkdown's is.
 */
import { safe as markdown } from './markdown.mjs';

function blocks(text) {
  return markdown.lexer(text || '').filter(token => token.type !== 'space');
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

function block(kind, html) {
  const title = kind === 'add' ? ' title="Added"' : kind === 'del' ? ' title="Removed"' : '';
  return '<div class="mdx-block mdx-' + kind + '"' + title + '>' + html + '</div>';
}

function oneItemList(list, item, number) {
  return markdown.parser([{
    type: 'list', raw: item.raw, ordered: list.ordered, start: list.ordered ? number : '',
    loose: list.loose, items: [item],
  }]);
}

// Two versions of one list: mark each item rather than the whole list.
function listChange(oldList, newList) {
  const start = list => (list.ordered && list.start !== '' ? Number(list.start) : 1);
  const path = steps(oldList.items.map(key), newList.items.map(key));
  return '<div class="mdx-list">' + path.map(step => {
    if (step.kind === 'add') return block('add', oneItemList(newList, newList.items[step.b], start(newList) + step.b));
    if (step.kind === 'del') return block('del', oneItemList(oldList, oldList.items[step.a], start(oldList) + step.a));
    return block('same', oneItemList(newList, newList.items[step.b], start(newList) + step.b));
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
    if (!edited || token !== oldLists[0]) out.push(block('del', markdown.parser([token])));
  });
  adds.forEach(token => {
    out.push(edited && token === newLists[0] ? listChange(oldLists[0], token) : block('add', markdown.parser([token])));
  });
  return out.join('');
}

function changes(oldBlocks, newBlocks) {
  const path = steps(oldBlocks.map(key), newBlocks.map(key));
  const out = [];
  let k = 0;
  while (k < path.length) {
    const step = path[k];
    if (step.kind === 'same') { out.push(block('same', markdown.parser([newBlocks[step.b]]))); k++; continue; }
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
  if (before == null) return '<div class="mdx"><p class="mdx-note">New file.</p>' + markdown.parser(blocks(after)) + '</div>';
  if (after == null) return '<div class="mdx"><p class="mdx-note">Deleted file. It read:</p>' + markdown.parser(blocks(before)) + '</div>';
  const body = changes(blocks(before), blocks(after));
  const note = hasMarks(body) ? legend('Marked:') : '<p class="mdx-note">The change does not alter how the file reads.</p>';
  return '<div class="mdx">' + note + body + '</div>';
}

export function renderParts(parts) {
  const bodies = parts.map(part => changes(blocks(part.before), blocks(part.after)));
  return '<div class="mdx">' + legend('Only the changed parts: the app holds the diff of this file, not its whole text. Marked:') +
    bodies.join('<p class="mdx-gap">&middot; &middot; &middot;</p>') + '</div>';
}
