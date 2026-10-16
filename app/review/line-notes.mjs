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
 *   decorate(file)   draw a file's notes under their lines, again
 *   countBadge(path) the file header's count of its notes
 *   refresh()        draw every open file's notes again, after a change
 *
 * From the composer, Ask Claude starts a conversation on the lines and
 * Queue comment queues the text for the pull request. A conversation takes
 * follow-up questions, and any answer can be taken as the comment's text. A
 * branch's review has no queue, so there only Ask Claude is offered.
 *
 * GitHub threads and conversations fold to one line and open again from it.
 * A GitHub thread that is resolved or outdated starts folded, as on GitHub;
 * one on no line the diff shows sits at the top of its file.
 */
import { esc, plural } from '../lib/text.mjs';
import { send, reviewPath } from '../lib/api.mjs';
import { createFolds } from '../lib/folds.mjs';
import { pointOf, rangeOf, rowFor, rowsOf, linesWords } from './line-range.mjs';
import { createNoteCards } from './note-cards.mjs';

export function createLineNotes(ctx) {
  const cards = createNoteCards(ctx);
  const folds = createFolds(ctx.state.id);
  let composer = null;

  function threadsOn(path) {
    return ctx.state.threads.filter(t => t.path === path);
  }
  function githubOn(path) {
    return ctx.data.threads.filter(t => (t.at ? t.at.path : t.path) === path);
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

  // Where a comment on the composer's lines would go, or why none can.
  function destination(c) {
    if (!ctx.state.queue) return { words: '' };
    if (ctx.state.moved) return { off: 'This review moved, so nothing can be queued here.' };
    if (!ctx.data.files[c.path]) return { off: 'This file is not in the change, so a comment cannot go on it. Ask about it instead.' };
    const end = ctx.diffs.hunkAt(c.path, c.side, c.line);
    if (end < 0) return { off: 'GitHub takes comments only on lines of the diff, and this one is not. Ask Claude about it instead.' };
    if (c.start_line && ctx.diffs.hunkAt(c.path, c.start_side || c.side, c.start_line) !== end) {
      return { off: 'GitHub takes a range only inside one hunk of the diff, and this one leaves it. Pick lines inside one hunk, or ask Claude about these.' };
    }
    if (!ctx.pulls.members.length) {
      const m = ctx.state.queue.members[0];
      if (m && m.state && m.state !== 'open') return { off: '#' + m.pull + ' is ' + m.state + ', so nothing can be queued for it.' };
      return { words: '' };
    }
    const t = ctx.pulls.lineTarget(c.path, c.side, c.line);
    if (!t) return { off: 'This line is not in the combined diff, so a comment cannot go on it.' };
    if (c.start_line) {
      const s0 = ctx.pulls.lineTarget(c.path, c.start_side || c.side, c.start_line);
      const owner = x => (x && x.pull) || (x && x.candidates && x.candidates.length === 1 && x.candidates[0].pull) || null;
      if (!owner(t) || owner(s0) !== owner(t)) return { off: 'These lines do not all go to one pull request of the stack. Comment on fewer lines, or one comment per pull request.' };
    }
    if (t.none) return { off: 'No pull request in the stack can take a comment here: ' + t.none + '.' };
    if (t.pull) return { words: 'Goes on #' + t.pull + ', line ' + t.line + (t.path !== c.path ? ' of ' + t.path : '') + '.' };
    if (!t.candidates.length) return { off: 'No pull request in the stack shows this line, so it takes no comment.' };
    if (t.candidates.length === 1) return { words: 'Goes on #' + t.candidates[0].pull + ', line ' + t.candidates[0].line + '.' };
    return { choose: t.candidates };
  }

  function composerHTML(c) {
    const to = destination(c);
    const choose = to.choose ? '<label class="composer-choose">Comment on <select data-pull>' + to.choose.map(x =>
      '<option value="' + x.pull + '"' + (c.pull === x.pull ? ' selected' : '') + '>#' + x.pull + ', line ' + x.line + '</option>').join('') + '</select></label>' : '';
    return '<div class="composer">' +
      '<p class="composer-where"><b>' + esc(linesWords(c)) + '</b>' + (to.words ? ' &middot; ' + esc(to.words) : '') +
      (to.off ? ' &middot; ' + esc(to.off) : '') + ' <span class="hint-inline">' + (touch ? 'Tap another line number for a range.' : 'Drag across line numbers, or shift-select another, for a range.') + '</span></p>' + choose +
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
    file.querySelectorAll('tr.note-row').forEach(tr => {
      const area = tr.querySelector('.thread textarea');
      if (area && area.value) typed[tr.querySelector('.thread').getAttribute('data-thread')] = area.value;
      tr.remove();
    });
    file.querySelectorAll('tr.picked').forEach(tr => tr.classList.remove('picked'));
    const badge = file.querySelector('summary .ncount');
    const fresh = countBadge(path);
    if (badge) badge.outerHTML = fresh || '';
    else if (fresh) file.querySelector('summary .fstats').insertAdjacentHTML('beforebegin', fresh);
    const first = file.querySelector('tr[data-l]');
    if (!first) return;

    const away = [];
    const github = githubOn(path).map(t => ({ ...t.at, html: githubCard(t), cls: 'gh-row', thread: t }));
    const notes = github.filter(n => n.line)
      .concat(threadsOn(path).map(t => ({ side: t.side, line: t.line, html: cards.conversation(t, folds.isOpen('t' + t.id, true)), cls: 'thread-row' })))
      .concat(queuedOn(path).map(n => ({ side: n.side, line: n.line, html: cards.queued(n), cls: 'queued-row' })));
    away.push(...github.filter(n => !n.line).map(n => n.thread));
    notes.forEach(n => {
      let row = rowFor(file, n.side, n.line);
      if (!row && n.side === 'RIGHT') { ctx.diffs.expandFor(file, n.line); row = rowFor(file, n.side, n.line); }
      if (!row) { if (n.thread) away.push(n.thread); return; }
      let after = row;
      while (after.nextElementSibling && after.nextElementSibling.classList.contains('note-row')) after = after.nextElementSibling;
      after.after(noteRow(row, n.cls, n.html));
    });
    if (away.length) first.parentElement.prepend(noteRow(first, 'gh-row gh-away-row', awayBand(away)));
    Object.keys(typed).forEach(id => {
      const area = file.querySelector('.thread[data-thread="' + id + '"] textarea');
      if (area) area.value = typed[id];
    });
    if (composer && composer.path === path) placeComposer(file);
  }

  function githubCard(t) {
    return cards.github(t, folds.isOpen('g' + t.id, !t.resolved && !t.outdated));
  }

  function awayBand(threads) {
    return '<div class="gh-away"><p class="gh-away-h">On GitHub, on no line shown here</p>' + threads.map(githubCard).join('') + '</div>';
  }

  function placeComposer(file) {
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

  function compose(file, from, to) {
    const path = file.getAttribute('data-path');
    const range = rangeOf(file, from, to, ctx.diffs.isWide()) || { side: to.side, line: to.line, start_line: null, start_side: null };
    composer = { path, ...range, anchor: from, text: composer ? composer.text : '', pull: composer && composer.path === path ? composer.pull : null };
    clearPicked();
    placeComposer(file);
  }

  function pick(cell, e) {
    if (dragged) return;
    const file = cell.closest('details.file');
    const point = pointOf(cell);
    const open = composer && composer.path === file.getAttribute('data-path') && rowFor(file, composer.anchor.side, composer.anchor.line);
    const extend = e.shiftKey || (e.pointerType === 'touch' && !!open);
    compose(file, extend && open ? composer.anchor : point, point);
  }

  function clearPicked() {
    document.querySelectorAll('tr.composer-row').forEach(tr => tr.remove());
    document.querySelectorAll('tr.picked').forEach(tr => tr.classList.remove('picked'));
  }

  function codeOf(file, n) {
    const wide = ctx.diffs.isWide();
    return rowsOf(file, n).map(row => {
      const cells = row.querySelectorAll('td.code');
      const cell = cells.length > 1 && wide ? cells[n.side === 'LEFT' ? 0 : cells.length - 1] : cells[0];
      return cell && !cell.classList.contains('void') ? cell.textContent : null;
    }).filter(text => text !== null).join('\n');
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
        ? await send('POST', reviewPath(ctx.state.id, '/threads/' + threadId + '/messages'), { text })
        : await send('POST', reviewPath(ctx.state.id, '/threads'), { path: anchor.path, side: anchor.side, line: anchor.line, start_line: anchor.start_line || null, start_side: anchor.start_side || null, code: anchor.code, text });
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
        path: anchor.path, side: anchor.side, line: anchor.line, start_line: anchor.start_line || null, start_side: anchor.start_side || null, body: text, code: anchor.code, pull: pull || null,
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
    if (composer && e.target.closest('.composer-row') && e.target.tagName === 'TEXTAREA') composer.text = e.target.value;
  });
  main.addEventListener('change', e => {
    if (composer && e.target.matches('.composer select[data-pull]')) composer.pull = +e.target.value;
  });
  main.addEventListener('click', e => {
    const fold = e.target.closest('.note-row summary.note-head');
    if (fold) {
      const card = fold.parentElement;
      folds.set(card.matches('.thread') ? 't' + card.getAttribute('data-thread') : 'g' + card.getAttribute('data-gh'), !card.open);
      return;
    }
    const button = e.target.closest('.note-row button[data-act]');
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
      const anchor = { path, side: composer.side, line: composer.line, start_line: composer.start_line, start_side: composer.start_side, code: codeOf(file, composer) };
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
      if (act === 'queue') queue(reply, { ...thread, code: thread.code || codeOf(file, thread) }, text, null);
      return;
    }

    const queued = button.closest('.queued-note');
    if (queued) ctx.queueActions[act](+queued.getAttribute('data-comment'), queued);
  });

  return { pick, decorate, countBadge, refresh };
}
