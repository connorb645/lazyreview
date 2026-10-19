/**
 * Notes on the lines of a diff: the composer a line number opens, the
 * review threads already on GitHub, the conversations with Claude, and the
 * comments queued for the pull request.
 *
 * createLineNotes(ctx) returns:
 *   pick(cell, event) open the composer on a line number's line; with Shift,
 *                    or a second tap on a touch screen, make it a range from
 *                    the line picked first. Dragging across line numbers
 *                    picks a range too.
 *   composeLines(file, lines, quote)
 *                    open the composer on lines a passage of the Rendered
 *                    view came from, quoting it
 *   decorate(file)   draw a file's notes under their lines, again, or in the
 *                    Rendered view under the passage holding their lines
 *   refresh()        draw every open file's notes again, after a change
 *   drafting()       whether a question or a comment is being written
 *
 * The composer says where a comment on its lines goes, or why none can, as
 * the server answers it: the same answer the queue takes. From the composer,
 * Ask Claude starts a conversation on the lines and
 * Queue comment queues the text for the pull request. A conversation takes
 * follow-up questions, and any answer can be taken as the comment's text. A
 * branch's review has no queue, so there only Ask Claude is offered.
 *
 * GitHub threads and conversations fold to one line and open again from it.
 * A GitHub thread that is resolved or outdated starts folded, as on GitHub.
 * A note on no line the view shows sits at the top of its file.
 */
import { send, getJSON, reviewPath } from '../api/server.mjs';
import { views } from '../views.mjs';
import { createFolds } from '../context/folds.mjs';
import { pointOf, rangeOf, rowFor, rowsOf } from '../lib/line-range.mjs';
import { fromHTML } from '../lib/html.mjs';
import { createNotesMarkup, countBadge } from '../presentation/line-notes.mjs';
import { english } from '/i18n/english.mjs';

const BLANK_LINES = 2;
const words = english.web.notes;

export function createLineNotes(ctx) {
  const folds = createFolds(ctx.state.id);
  const markup = createNotesMarkup(ctx, folds);
  let composer = null;

  async function lookWhere(c, named) {
    if (!ctx.state.queue) return;
    const asked = { path: c.path, side: c.side, line: c.line, build: ctx.data.build };
    if (c.start_line) Object.assign(asked, { start_line: c.start_line, start_side: c.start_side || c.side });
    if (named) asked.pull = c.pull;
    c.asking = asked;
    let answer;
    try {
      answer = await getJSON(reviewPath(ctx.state.id, '/where?' + new URLSearchParams(asked)));
    } catch (error) {
      answer = { refused: words.whereUnchecked(error.message) };
    }
    if (composer !== c || c.asking !== asked) return;
    c.where = answer;
    if (answer.choose) {
      c.choices = answer.choose;
      if (!answer.choose.some(x => x.pull === c.pull)) c.pull = answer.choose[0].pull;
    }
    showWhere(c);
  }

  function showWhere(c) {
    const box = document.querySelector('#main .composer-row .composer');
    if (!box) return;
    const to = markup.whereOf(c);
    box.querySelector('.where-words').textContent = to.words ? ' · ' + to.words : '';
    box.querySelector('.choose-slot').innerHTML = markup.chooseHTML(c, to.choose);
    const button = box.querySelector('button[data-act="queue"]');
    if (button) {
      button.disabled = !!to.off;
      button.title = to.off || '';
    }
  }

  function columns(row) {
    return Array.from(row.cells).reduce((n, cell) => n + cell.colSpan, 0);
  }
  function noteRow(row, cls, html) {
    return fromHTML(markup.noteRow(columns(row), cls, html));
  }

  function decorate(file) {
    if (!file || !file.open) return;
    const path = file.getAttribute('data-path');
    const typed = {};
    file.querySelectorAll('.note-row, .md-note').forEach(note => {
      const area = note.querySelector('.thread textarea');
      if (area && area.value) typed[note.querySelector('.thread').getAttribute('data-thread')] = area.value;
      note.remove();
    });
    file.querySelectorAll('.picked').forEach(el => el.classList.remove('picked'));
    const badge = file.querySelector('summary .ncount');
    const fresh = countBadge(ctx.state, path);
    if (badge) badge.outerHTML = fresh || '';
    else if (fresh) file.querySelector('summary .fstats').insertAdjacentHTML('beforebegin', fresh);
    const view = file.querySelector('.mdview');
    if (view) drawRendered(view, path);
    else drawTable(file, path);
    Object.keys(typed).forEach(id => {
      const area = file.querySelector('.thread[data-thread="' + id + '"] textarea');
      if (area) area.value = typed[id];
    });
    if (composer && composer.path === path && !questionShown(composer)) placeComposer(file, false);
    ctx.queuePanel.restore();
  }

  function drawTable(file, path) {
    const first = file.querySelector('tr[data-l]');
    if (!first) return;
    const away = [];
    markup.notesOn(path).forEach(n => {
      let row = n.line ? rowFor(file, n.side, n.line) : null;
      if (!row && n.line && n.side === 'RIGHT') { ctx.diffs.expandFor(file, n.line); row = rowFor(file, n.side, n.line); }
      if (!row) { if (n.cls === 'gh-row') away.push(n.html); return; }
      let after = row;
      while (after.nextElementSibling && after.nextElementSibling.classList.contains('note-row')) after = after.nextElementSibling;
      after.after(noteRow(row, n.cls, n.html));
    });
    if (away.length) first.parentElement.prepend(noteRow(first, 'gh-row gh-away-row', markup.awayBand(away)));
  }

  function drawRendered(view, path) {
    const away = [];
    markup.notesOn(path).forEach(n => {
      const block = n.line ? blockFor(view, n.side, n.line) : null;
      if (!block) { away.push(n.html); return; }
      afterNotes(block).after(mdNote(n.cls, n.html));
    });
    if (away.length) view.querySelector('.mdx').prepend(mdNote('gh-row', markup.awayBand(away)));
  }

  function mdNote(cls, html) {
    return fromHTML(markup.mdNote(cls, html));
  }

  function afterNotes(block) {
    let after = block;
    while (after.nextElementSibling && after.nextElementSibling.classList.contains('md-note') && !after.nextElementSibling.classList.contains('composer-row')) after = after.nextElementSibling;
    return after;
  }

  function blockFor(view, side, line) {
    let near = null;
    for (const block of view.querySelectorAll('.mdx-block[data-side]')) {
      const own = block.getAttribute('data-side') === side;
      if (!own && !(side === 'LEFT' && block.hasAttribute('data-left-from'))) continue;
      const from = +block.getAttribute(own ? 'data-from' : 'data-left-from');
      const to = +block.getAttribute(own ? 'data-to' : 'data-left-to');
      if (from <= line && line <= to) return block;
      if (to < line && line - to <= BLANK_LINES) near = block;
    }
    return near;
  }

  function placeComposer(file, focus) {
    const view = file.querySelector('.mdview');
    if (view) { placeInRendered(view, focus); return; }
    const row = rowFor(file, composer.side, composer.line);
    if (!row) { composer = null; return; }
    rowsOf(file, composer).forEach(r => r.classList.add('picked'));
    let after = row;
    while (after.nextElementSibling && after.nextElementSibling.classList.contains('note-row') && !after.nextElementSibling.classList.contains('composer-row')) after = after.nextElementSibling;
    const tr = noteRow(row, 'composer-row', markup.composerHTML(composer, touch));
    after.after(tr);
    if (focus) focusEnd(tr.querySelector('textarea'));
  }

  function focusEnd(area) {
    area.focus({ preventScroll: true });
    area.setSelectionRange(area.value.length, area.value.length);
  }

  function placeInRendered(view, focus) {
    const end = blockFor(view, composer.side, composer.line);
    if (!end) { composer = null; return; }
    const start = composer.start_line ? blockFor(view, composer.start_side || composer.side, composer.start_line) || end : end;
    let picking = false;
    for (const block of view.querySelectorAll('.mdx-block[data-side]')) {
      if (block === start) picking = true;
      if (picking) block.classList.add('picked');
      if (block === end) break;
    }
    const note = mdNote('composer-row', markup.composerHTML(composer, touch));
    afterNotes(end).after(note);
    if (focus) focusEnd(note.querySelector('textarea'));
  }

  function composeLines(file, lines, quote) {
    const path = file.getAttribute('data-path');
    composer = { path, ...lines, anchor: null, quote, text: composer ? composer.text : '', pull: composer && composer.path === path ? composer.pull : null };
    clearPicked();
    placeComposer(file, true);
    lookWhere(composer, false);
  }

  function compose(file, from, to) {
    const path = file.getAttribute('data-path');
    const range = rangeOf(file, from, to, ctx.diffs.isWide()) || { side: to.side, line: to.line, start_line: null, start_side: null };
    composer = { path, ...range, anchor: from, text: composer ? composer.text : '', pull: composer && composer.path === path ? composer.pull : null };
    clearPicked();
    placeComposer(file, true);
    lookWhere(composer, false);
  }

  function pick(cell, e) {
    if (dragged) return;
    const file = cell.closest('details.file');
    const point = pointOf(cell);
    const open = composer && composer.anchor && composer.path === file.getAttribute('data-path') && rowFor(file, composer.anchor.side, composer.anchor.line);
    const extend = e.shiftKey || (e.pointerType === 'touch' && !!open);
    compose(file, extend && open ? composer.anchor : point, point);
  }

  function clearPicked() {
    document.querySelectorAll('#main .composer-row').forEach(el => el.remove());
    document.querySelectorAll('#main .picked').forEach(el => el.classList.remove('picked'));
  }

  function refresh() {
    document.querySelectorAll('#main details.file[open]').forEach(decorate);
  }

  function busy(box, said) {
    box.querySelectorAll('button, textarea, select').forEach(el => { el.disabled = true; });
    const status = box.querySelector('.note-status');
    status.textContent = said;
    status.classList.add('working');
  }
  function failed(box, error) {
    box.querySelectorAll('button, textarea, select').forEach(el => { el.disabled = false; });
    const status = box.querySelector('.note-status');
    status.textContent = error.message;
    status.classList.remove('working');
    status.classList.add('bad');
  }

  function questionShown(c) {
    return c.sending && ctx.state.threads.some(t => t.path === c.path && t.side === c.side && t.line === c.line && t.messages.length === 1 && t.messages[0].text === c.sending);
  }

  async function ask(box, anchor, text, threadId) {
    busy(box, words.reading);
    const sent = threadId ? null : composer;
    if (sent) sent.sending = text;
    try {
      const done = threadId
        ? await send('POST', reviewPath(ctx.state.id, '/threads/' + threadId + '/messages'), { text, build: ctx.data.build })
        : await send('POST', reviewPath(ctx.state.id, '/threads'), { path: anchor.path, side: anchor.side, line: anchor.line, start_line: anchor.start_line || null, start_side: anchor.start_side || null, text, build: ctx.data.build });
      const others = ctx.state.threads.filter(t => t.id !== done.thread.id);
      if (sent && composer === sent) composer = null;
      ctx.setState({ ...ctx.state, threads: others.concat([done.thread]).sort((a, b) => a.id - b.id) });
    } catch (error) {
      if (sent) delete sent.sending;
      const shown = document.querySelector('#main .composer') || box;
      failed(shown, error);
    }
  }

  async function stop(box, threadId) {
    busy(box, words.stopping);
    try {
      const done = await send('POST', reviewPath(ctx.state.id, '/threads/' + threadId + '/stop'));
      ctx.setState({ ...ctx.state, threads: ctx.state.threads.map(t => (t.id === done.thread.id ? done.thread : t)) });
    } catch (error) {
      failed(box, error);
    }
  }

  async function queue(box, anchor, text, pull) {
    busy(box, words.checkingLine);
    try {
      const done = await send('POST', reviewPath(ctx.state.id, '/comments'), {
        path: anchor.path, side: anchor.side, line: anchor.line, start_line: anchor.start_line || null, start_side: anchor.start_side || null, body: text, pull: pull || null, build: ctx.data.build,
      });
      if (box.closest('.composer-row')) composer = null;
      else box.querySelector('textarea').value = '';
      ctx.setState(done.state);
    } catch (error) {
      failed(box, error);
    }
  }

  const main = views.main;
  const touch = matchMedia('(pointer: coarse)').matches;
  let drag = null;
  let dragged = false;
  main.addEventListener('pointerdown', e => {
    const cell = e.target.closest('td.ln');
    if (!cell || e.button !== 0 || e.pointerType === 'touch') return;
    e.preventDefault();
    if (!e.shiftKey) drag = { file: cell.closest('details.file'), from: pointOf(cell), to: null };
  });
  main.addEventListener('pointerover', e => {
    const cell = drag && e.target.closest('td.ln');
    if (!cell || cell.closest('details.file') !== drag.file) return;
    const point = pointOf(cell);
    if (!drag.to && point.side === drag.from.side && point.line === drag.from.line) return;
    drag.to = point;
    document.querySelectorAll('tr.picked').forEach(tr => tr.classList.remove('picked'));
    const range = rangeOf(drag.file, drag.from, point, ctx.diffs.isWide());
    if (range) rowsOf(drag.file, range).forEach(r => r.classList.add('picked'));
  });
  document.addEventListener('pointerup', () => {
    if (!drag) return;
    const done = drag;
    drag = null;
    if (!done.to) return;
    dragged = true;
    setTimeout(() => { dragged = false; });
    compose(done.file, done.from, done.to);
  });
  document.addEventListener('pointercancel', () => {
    if (drag) { drag = null; refresh(); }
  });
  main.addEventListener('input', e => {
    if (composer && e.target.closest('.composer') && e.target.tagName === 'TEXTAREA') composer.text = e.target.value;
  });
  main.addEventListener('change', e => {
    if (composer && e.target.matches('.composer select[data-pull]')) {
      composer.pull = +e.target.value;
      lookWhere(composer, true);
    }
  });
  main.addEventListener('click', e => {
    const fold = e.target.closest('.note-row summary.note-head, .md-note summary.note-head');
    if (fold) {
      const card = fold.parentElement;
      folds.set(card.matches('.thread') ? 't' + card.getAttribute('data-thread') : 'g' + card.getAttribute('data-gh'), !card.open);
      return;
    }
    const button = e.target.closest('.note-row button[data-act], .md-note button[data-act]');
    if (!button) return;
    const act = button.getAttribute('data-act');
    const file = button.closest('details.file');
    const path = file.getAttribute('data-path');

    const composerBox = button.closest('.composer');
    if (composerBox) {
      if (act === 'cancel') { composer = null; clearPicked(); return; }
      const text = composerBox.querySelector('textarea').value.trim();
      if (!text) { composerBox.querySelector('textarea').focus(); return; }
      const select = composerBox.querySelector('select[data-pull]');
      const anchor = { path, side: composer.side, line: composer.line, start_line: composer.start_line, start_side: composer.start_side };
      if (act === 'ask') ask(composerBox, anchor, text, null);
      if (act === 'queue') queue(composerBox, anchor, text, select ? +select.value : null);
      return;
    }

    const threadBox = button.closest('.thread');
    if (threadBox) {
      const thread = ctx.state.threads.find(t => t.id === +threadBox.getAttribute('data-thread'));
      if (act === 'stop') {
        stop(threadBox.querySelector('.working'), thread.id);
        return;
      }
      const area = threadBox.querySelector('.reply textarea');
      if (act === 'use') {
        const msg = button.closest('.msg');
        const index = Array.prototype.indexOf.call(threadBox.querySelectorAll('.msg'), msg);
        area.value = thread.messages[index].text;
        area.focus();
        return;
      }
      const text = area.value.trim();
      if (!text) { area.focus(); return; }
      const reply = threadBox.querySelector('.reply');
      if (act === 'ask') ask(reply, thread, text, thread.id);
      if (act === 'queue') queue(reply, thread, text, null);
      return;
    }

    const queued = button.closest('.queued-note');
    if (queued) ctx.queueActions[act](+queued.getAttribute('data-comment'), queued);
  });

  function drafting() {
    return !!(composer && composer.text.trim()) || Array.prototype.some.call(document.querySelectorAll('#main .thread .reply textarea'), area => area.value.trim());
  }

  return { pick, composeLines, decorate, refresh, drafting };
}
