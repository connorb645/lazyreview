/**
 * Passages of a Markdown file's Rendered view. Selecting one shows a button
 * beside it that opens the composer on the source lines the passage came
 * from, to queue a comment on them or ask Claude about them.
 *
 * createPassages(ctx) watches the selection in the diff view: once the
 * pointer is up, it shows the button for the part of the selection inside
 * the Rendered view it starts in, and a press hands the lines to
 * ctx.notes.composeLines.
 *
 * passageLines(view, range) gives the lines a selected range of a Rendered
 * view covers, {side, line, start_line, start_side}, as the Rendered view's
 * blocks name them, or null when it covers none. Inside a paragraph, a
 * heading, a code block or a list item it counts the source's own line
 * breaks, so a sentence maps to the lines it is on; elsewhere a passage
 * maps to its blocks' lines whole.
 */
import { esc } from '../lib/text.mjs';
import { linesWords } from './line-range.mjs';

export function passageLines(view, range) {
  const blocks = Array.prototype.filter.call(view.querySelectorAll('.mdx-block[data-side]'), b => range.intersectsNode(b));
  if (blocks.length > 1 && endsAtStart(blocks[blocks.length - 1], range)) blocks.pop();
  if (!blocks.length) return null;
  const first = blocks[0], last = blocks[blocks.length - 1];
  const side = last.getAttribute('data-side');
  let startSide = first.getAttribute('data-side');
  let start = lineIn(first, range.startContainer, range.startOffset, 'from');
  const end = lineIn(last, range.endContainer, range.endOffset, 'to');
  if (startSide !== side && first.hasAttribute('data-left-from') && side === 'LEFT') {
    start = +first.getAttribute('data-left-from') + start - +first.getAttribute('data-from');
    startSide = 'LEFT';
  }
  if (startSide === side && start >= end) return { side, line: Math.max(start, end), start_line: start === end ? null : Math.min(start, end), start_side: start === end ? null : side };
  return { side, line: end, start_line: start, start_side: startSide };
}

function lineIn(block, node, offset, edge) {
  const from = +block.getAttribute('data-from');
  const to = +block.getAttribute('data-to');
  const root = block.querySelector('li') || block.firstElementChild;
  if (!block.hasAttribute('data-text') || !root || !root.contains(node)) return edge === 'from' ? from : to;
  const before = document.createRange();
  before.setStart(root, 0);
  before.setEnd(node, offset);
  const line = +block.getAttribute('data-text') + (before.toString().match(/\n/g) || []).length;
  return Math.min(to, Math.max(from, line));
}

function endsAtStart(block, range) {
  const before = document.createRange();
  before.setStart(block, 0);
  before.setEnd(range.endContainer, range.endOffset);
  return block.contains(range.endContainer) && !before.toString().trim();
}

export function createPassages(ctx) {
  const main = document.getElementById('main');
  const button = document.createElement('button');
  button.type = 'button';
  button.className = 'passage-pick';
  button.hidden = true;
  document.body.appendChild(button);
  let held = false;
  let offer = null;

  function hide() {
    button.hidden = true;
    offer = null;
  }

  function look() {
    const selection = getSelection();
    if (held || !selection.rangeCount || selection.isCollapsed) { if (!held) hide(); return; }
    const range = selection.getRangeAt(0).cloneRange();
    const element = node => (node.nodeType === Node.ELEMENT_NODE ? node : node.parentElement);
    const view = element(range.startContainer).closest('#main .mdview') || element(range.endContainer).closest('#main .mdview');
    if (view && !view.contains(range.startContainer)) range.setStart(view, 0);
    if (view && !view.contains(range.endContainer)) range.setEnd(view, view.childNodes.length);
    const lines = view && passageLines(view, range);
    if (!lines) { hide(); return; }
    offer = { file: view.closest('details.file'), lines, quote: range.toString().trim() };
    button.innerHTML = (ctx.state.queue ? 'Comment on ' : 'Ask Claude about ') + esc(linesWords(lines));
    button.hidden = false;
    const rects = range.getClientRects();
    const box = rects.length ? rects[rects.length - 1] : range.getBoundingClientRect();
    const width = button.offsetWidth;
    const bounds = view.getBoundingClientRect();
    button.style.left = Math.round(Math.max(bounds.left, Math.min(box.left, bounds.right - width)) + window.scrollX) + 'px';
    button.style.top = Math.round(box.bottom + 8 + window.scrollY) + 'px';
  }

  document.addEventListener('selectionchange', () => requestAnimationFrame(look));
  main.addEventListener('pointerdown', e => { if (e.button === 0) held = true; });
  document.addEventListener('pointerup', () => {
    if (!held) return;
    held = false;
    requestAnimationFrame(look);
  });
  window.addEventListener('resize', hide);
  button.addEventListener('pointerdown', e => e.preventDefault());
  button.addEventListener('click', () => {
    if (!offer) return;
    const { file, lines, quote } = offer;
    getSelection().removeAllRanges();
    hide();
    ctx.notes.composeLines(file, lines, quote);
  });
}
