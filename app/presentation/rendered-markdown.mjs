/**
 * The Rendered view of a Markdown file in a diff, drawn from the blocks the
 * server reads, each marked added, removed or unchanged.
 *
 * drawRendered(rendered, status, leftOut) takes a file's {blocks, partial},
 * as the file route gives them, its status, and leftOut(block), whether the
 * stack's filter leaves a block out, and returns HTML. The blocks of an
 * added or a deleted file are drawn plain, under a line saying so. A block
 * the filter leaves out fades, as its lines do in the diff. The items of one
 * list are drawn together, and the changed parts of a file the server holds
 * only the diff of are drawn apart.
 *
 * Every block names the source lines it was read from, so a passage selected
 * in it can be traced back to them:
 *   data-side             RIGHT for the file after the change, LEFT before
 *   data-from, data-to    its first and last line on that side
 *   data-text             the line its text starts on, where a line break
 *                         in its text is a line break in the source
 *   data-left-from, data-left-to
 *                         an unchanged block's lines before the change
 *
 * The output is safe to put in a page, as safeMarkdown's is.
 */
import { safeMarkdown } from '../lib/markdown.mjs';
import { english } from '/i18n/english.mjs';

const words = english.web.rendered;

function lineData(b) {
  if (b.start_line === null) return '';
  return ' data-side="' + b.side + '" data-from="' + b.start_line + '" data-to="' + b.line + '"' +
    (b.text_line !== null ? ' data-text="' + b.text_line + '"' : '') +
    (b.left ? ' data-left-from="' + b.left.start_line + '" data-left-to="' + b.left.line + '"' : '');
}

function block(b, plain, leftOut) {
  const kind = plain ? '' : ' mdx-' + b.state + (leftOut(b) ? ' pr-other' : '');
  const title = plain ? '' : b.state === 'add' ? ' title="' + words.added + '"' : b.state === 'del' ? ' title="' + words.removed + '"' : '';
  return '<div class="mdx-block' + kind + '"' + title + lineData(b) + '>' + safeMarkdown(b.markdown) + '</div>';
}

// The blocks in order, the items of one list held together.
function body(blocks, plain, leftOut) {
  let out = '';
  for (let i = 0; i < blocks.length; i++) {
    if (blocks[i].list === null) { out += block(blocks[i], plain, leftOut); continue; }
    let items = '';
    const list = blocks[i].list;
    for (; i < blocks.length && blocks[i].list === list; i++) items += block(blocks[i], plain, leftOut);
    i--;
    out += '<div class="mdx-list">' + items + '</div>';
  }
  return out;
}

function legend(text) {
  return '<p class="mdx-note">' + text + '<span class="mdx-key mdx-add"></span>' + words.addedKey + '<span class="mdx-key mdx-del"></span>' + words.removedKey + '</p>';
}

export function drawRendered(rendered, status, leftOut = () => false) {
  const { blocks, partial } = rendered;
  if (status === 'A') return '<div class="mdx"><p class="mdx-note">' + words.newFile + '</p>' + body(blocks, true, leftOut) + '</div>';
  if (status === 'D') return '<div class="mdx"><p class="mdx-note">' + words.deletedFile + '</p>' + body(blocks, true, leftOut) + '</div>';
  if (partial) {
    const parts = [...new Set(blocks.map(b => b.part))].map(part => body(blocks.filter(b => b.part === part), false, leftOut));
    return '<div class="mdx">' + legend(words.changedParts) +
      parts.join('<p class="mdx-gap">&middot; &middot; &middot;</p>') + '</div>';
  }
  const marked = blocks.some(b => b.state !== 'same');
  return '<div class="mdx">' + (marked ? legend(words.marked) : '<p class="mdx-note">' + words.readsTheSame + '</p>') + body(blocks, false, leftOut) + '</div>';
}
