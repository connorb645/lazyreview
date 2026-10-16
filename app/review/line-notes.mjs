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
 *   countBadge(path) the file header's count of its notes
 *   refresh()        draw every open file's notes again, after a change
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
import { esc, plural } from '../lib/text.mjs';
import { send, getJSON, reviewPath } from '../lib/api.mjs';
import { createFolds } from '../lib/folds.mjs';
import { pointOf, rangeOf, rowFor, rowsOf, linesWords } from './line-range.mjs';
import { createNoteCards } from './note-cards.mjs';

const QUOTE = 200;
const BLANK_LINES = 2;

export function createLineNotes(ctx) {
  const cards = createNoteCards(ctx);
  const folds = createFolds(ctx.state.id);
  let composer = null;

  function threadsOn(path) {
    return ctx.state.threads.filter(t => t.path === path);
  }
  function githubOn(path) {
    return ctx.state.github.filter(t => (t.at ? t.at.path : t.path) === path);
  }
  function queuedOn(path) {
    const out = [];
    const q = ctx.state.queue;
    if (!q) return out;
    q.members.concat(q.earlier).forEach(m => m.comments.forEach(c => {
      const at = c.shown || c;
      if (at.path === path) out.push({ member: m, comment: c, side: at.side, line: at.line, start_line: at.start_line, start_side: at.start_side });
    }));
    return out;
  }

  function countBadge(path) {
    const g = githubOn(path).length, t = threadsOn(path).length, q = queuedOn(path).length;
    if (!g && !t && !q) return '';
    const words = [g ? plural(g, 'GitHub thread') : '', t ? plural(t, 'conversation') : '', q ? plural(q, 'queued comment') : ''].filter(Boolean).join(', ');
    return '<span class="ncount" title="' + esc(words) + '">' + esc(words) + '</span>';
  }

  // Where a comment on the composer's lines goes, as far as the server has said.
  function whereOf(c) {
    if (!ctx.state.queue) return { words: '' };
    if (!c.where) return { words: 'Checking where a comment here goes…', off: 'Checking where a comment here goes.' };
    if (c.where.refused) return { words: c.where.refused, off: c.where.refused };
    return { words: c.where.words, choose: c.where.choose || c.choices };
  }

  function chooseHTML(c, choose) {
    if (!choose) return '';
    return '<label class="composer-choose">Comment on <select data-pull>' + choose.map(x =>
      '<option value="' + x.pull + '"' + (c.pull === x.pull ? ' selected' : '') + '>#' + x.pull + ', line ' + x.line + '</option>').join('') + '</select></label>';
  }

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
      answer = { refused: 'Where a comment here goes could not be checked: ' + error.message };
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
    const to = whereOf(c);
    box.querySelector('.where-words').textContent = to.words ? ' · ' + to.words : '';
    box.querySelector('.choose-slot').innerHTML = chooseHTML(c, to.choose);
    const button = box.querySelector('button[data-act="queue"]');
    if (button) {
      button.disabled = !!to.off;
      button.title = to.off || '';
    }
  }

  function composerHTML(c) {
    const to = whereOf(c);
    const hint = c.quote !== undefined ? 'Select another passage to change the lines.'
      : touch ? 'Tap another line number for a range.' : 'Drag across line numbers, or shift-select another, for a range.';
    return '<div class="composer">' +
      '<p class="composer-where"><b>' + esc(linesWords(c)) + '</b><span class="where-words">' + (to.words ? ' &middot; ' + esc(to.words) : '') + '</span>' +
      ' <span class="hint-inline">' + hint + '</span></p>' +
      (c.quote ? '<blockquote class="composer-quote">' + esc(c.quote.length > QUOTE ? c.quote.slice(0, QUOTE) + '…' : c.quote) + '</blockquote>' : '') +
      '<div class="choose-slot">' + chooseHTML(c, to.choose) + '</div>' +
      cards.textArea(ctx.state.queue ? 'Ask Claude about these lines, or write a comment for the pull request' : 'Ask Claude about these lines', c.text) +
      cards.actions(to.off) + '<button type="button" class="nbtn quiet close" data-act="cancel" aria-label="Close">Cancel</button></div>';
  }

  function columns(row) {
    return Array.from(row.cells).reduce((n, cell) => n + cell.colSpan, 0);
  }
  function noteRow(row, cls, html) {
    const tr = document.createElement('tr');
    tr.className = 'note-row ' + cls;
    tr.innerHTML = '<td colspan="' + columns(row) + '">' + html + '</td>';
    return tr;
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
    const fresh = countBadge(path);
    if (badge) badge.outerHTML = fresh || '';
    else if (fresh) file.querySelector('summary .fstats').insertAdjacentHTML('beforebegin', fresh);
    const view = file.querySelector('.mdview');
    if (view) drawRendered(view, path);
    else drawTable(file, path);
    Object.keys(typed).forEach(id => {
      const area = file.querySelector('.thread[data-thread="' + id + '"] textarea');
      if (area) area.value = typed[id];
    });
    if (composer && composer.path === path) placeComposer(file);
  }

  function notesOn(path) {
    return githubOn(path).map(t => ({ ...t.at, html: githubCard(t), cls: 'gh-row' }))
      .concat(threadsOn(path).map(t => ({ side: t.side, line: t.line, html: cards.conversation(t, folds.isOpen('t' + t.id, true)), cls: 'thread-row' })))
      .concat(queuedOn(path).map(n => ({ side: n.side, line: n.line, html: cards.queued(n), cls: 'queued-row' })));
  }

  function drawTable(file, path) {
    const first = file.querySelector('tr[data-l]');
    if (!first) return;
    const away = [];
    notesOn(path).forEach(n => {
      let row = n.line ? rowFor(file, n.side, n.line) : null;
      if (!row && n.line && n.side === 'RIGHT') { ctx.diffs.expandFor(file, n.line); row = rowFor(file, n.side, n.line); }
      if (!row) { if (n.cls === 'gh-row') away.push(n.html); return; }
      let after = row;
      while (after.nextElementSibling && after.nextElementSibling.classList.contains('note-row')) after = after.nextElementSibling;
      after.after(noteRow(row, n.cls, n.html));
    });
    if (away.length) first.parentElement.prepend(noteRow(first, 'gh-row gh-away-row', awayBand(away)));
  }

  function drawRendered(view, path) {
    const away = [];
    notesOn(path).forEach(n => {
      const block = n.line ? blockFor(view, n.side, n.line) : null;
      if (!block) { away.push(n.html); return; }
      afterNotes(block).after(mdNote(n.cls, n.html));
    });
    if (away.length) view.querySelector('.mdx').prepend(mdNote('gh-row', awayBand(away)));
  }

  function mdNote(cls, html) {
    const div = document.createElement('div');
    div.className = 'md-note ' + cls;
    div.innerHTML = html;
    return div;
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

  function githubCard(t) {
    return cards.github(t, folds.isOpen('g' + t.id, !t.resolved && !t.outdated));
  }

  function awayBand(html) {
    return '<div class="gh-away"><p class="gh-away-h">On no line shown here</p>' + html.join('') + '</div>';
  }

  function placeComposer(file) {
    const view = file.querySelector('.mdview');
    if (view) { placeInRendered(view); return; }
    const row = rowFor(file, composer.side, composer.line);
    if (!row) { composer = null; return; }
    rowsOf(file, composer).forEach(r => r.classList.add('picked'));
    let after = row;
    while (after.nextElementSibling && after.nextElementSibling.classList.contains('note-row') && !after.nextElementSibling.classList.contains('composer-row')) after = after.nextElementSibling;
    const tr = noteRow(row, 'composer-row', composerHTML(composer));
    after.after(tr);
    const area = tr.querySelector('textarea');
    area.focus();
    area.setSelectionRange(area.value.length, area.value.length);
  }

  function placeInRendered(view) {
    const end = blockFor(view, composer.side, composer.line);
    if (!end) { composer = null; return; }
    const start = composer.start_line ? blockFor(view, composer.start_side || composer.side, composer.start_line) || end : end;
    let picking = false;
    for (const block of view.querySelectorAll('.mdx-block[data-side]')) {
      if (block === start) picking = true;
      if (picking) block.classList.add('picked');
      if (block === end) break;
    }
    const note = mdNote('composer-row', composerHTML(composer));
    afterNotes(end).after(note);
    const area = note.querySelector('textarea');
    area.focus({ preventScroll: true });
    area.setSelectionRange(area.value.length, area.value.length);
  }

  function composeLines(file, lines, quote) {
    const path = file.getAttribute('data-path');
    composer = { path, ...lines, anchor: null, quote, text: composer ? composer.text : '', pull: composer && composer.path === path ? composer.pull : null };
    clearPicked();
    placeComposer(file);
    lookWhere(composer, false);
  }

  function compose(file, from, to) {
    const path = file.getAttribute('data-path');
    const range = rangeOf(file, from, to, ctx.diffs.isWide()) || { side: to.side, line: to.line, start_line: null, start_side: null };
    composer = { path, ...range, anchor: from, text: composer ? composer.text : '', pull: composer && composer.path === path ? composer.pull : null };
    clearPicked();
    placeComposer(file);
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

  function busy(box, words) {
    box.querySelectorAll('button, textarea, select').forEach(el => { el.disabled = true; });
    box.querySelector('.note-status').textContent = words;
  }
  function failed(box, error) {
    box.querySelectorAll('button, textarea, select').forEach(el => { el.disabled = false; });
    const status = box.querySelector('.note-status');
    status.textContent = error.message;
    status.classList.add('bad');
  }

  async function ask(box, anchor, text, threadId) {
    busy(box, 'Claude is reading the file and the diff…');
    try {
      const done = threadId
        ? await send('POST', reviewPath(ctx.state.id, '/threads/' + threadId + '/messages'), { text, build: ctx.data.build })
        : await send('POST', reviewPath(ctx.state.id, '/threads'), { path: anchor.path, side: anchor.side, line: anchor.line, start_line: anchor.start_line || null, start_side: anchor.start_side || null, text, build: ctx.data.build });
      const others = ctx.state.threads.filter(t => t.id !== done.thread.id);
      if (!threadId) composer = null;
      ctx.setState({ ...ctx.state, threads: others.concat([done.thread]).sort((a, b) => a.id - b.id) });
    } catch (error) {
      failed(box, error);
    }
  }

  async function queue(box, anchor, text, pull) {
    busy(box, 'Checking the line with GitHub…');
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

  const main = document.getElementById('main');
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

  return { pick, composeLines, decorate, countBadge, refresh };
}
