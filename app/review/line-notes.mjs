/**
 * Notes on the lines of a diff: the composer a line number opens, the
 * conversations with Claude, and the comments queued for the pull request.
 *
 * createLineNotes(ctx) returns:
 *   compose(file, side, line, extend, row)  open the composer on a line, or
 *                                     with `extend`, stretch it to a range
 *   decorate(file)   draw a file's notes under their lines, again
 *   countBadge(path) the file header's count of its notes
 *   refresh()        draw every open file's notes again, after a change
 *
 * From the composer, Ask Claude starts a conversation on the lines and
 * Queue comment queues the text for the pull request. A conversation takes
 * follow-up questions, and any answer can be taken as the comment's text. A
 * branch's review has no queue, so there only Ask Claude is offered.
 */
import { esc, plural } from '../lib/text.mjs';
import { send, reviewPath } from '../lib/api.mjs';
import { safeMarkdown } from '../lib/markdown.mjs';

export function createLineNotes(ctx) {
  let composer = null;

  function threadsOn(path) {
    return ctx.state.threads.filter(t => t.path === path);
  }
  function queuedOn(path) {
    const out = [];
    const q = ctx.state.queue;
    if (!q) return out;
    q.members.concat(q.earlier).forEach(m => m.comments.forEach(c => {
      const at = c.shown || c;
      if (at.path === path) out.push({ member: m, comment: c, side: at.side, line: at.line, start_line: at.start_line });
    }));
    return out;
  }

  function countBadge(path) {
    const t = threadsOn(path).length, q = queuedOn(path).length;
    if (!t && !q) return '';
    const words = [t ? plural(t, 'conversation') : '', q ? plural(q, 'queued comment') : ''].filter(Boolean).join(', ');
    return '<span class="ncount" title="' + esc(words) + '">' + esc(words) + '</span>';
  }

  function linesWords(n) {
    const side = n.side === 'LEFT' ? 'removed line' : 'line';
    return n.start_line ? side + 's ' + n.start_line + '-' + n.line : side + ' ' + n.line;
  }

  function messageHTML(m) {
    if (m.from === 'you') return '<div class="msg you"><p class="who">You</p><div class="msg-body plain">' + esc(m.text) + '</div></div>';
    if (m.error) return '<div class="msg claude failed"><p class="who">Claude</p><div class="msg-body">No answer: ' + esc(m.error) + '</div></div>';
    return '<div class="msg claude"><p class="who">Claude</p><div class="msg-body md">' + safeMarkdown(m.text) + '</div>' +
      (ctx.state.queue ? '<button type="button" class="nbtn quiet small" data-act="use">Use as comment</button>' : '') + '</div>';
  }

  function threadHTML(t) {
    const waiting = t.messages.length && t.messages[t.messages.length - 1].from === 'you';
    return '<div class="thread" data-thread="' + t.id + '">' +
      '<p class="thread-head">Conversation on ' + esc(linesWords(t)) + '</p>' +
      t.messages.map(m => messageHTML(m)).join('') +
      (waiting ? '<p class="thinking">Claude is reading the file and the diff&hellip;</p>' : '') +
      '<div class="reply">' + textArea(ctx.state.queue ? 'Ask a follow-up, or write a comment for the pull request' : 'Ask a follow-up') +
      actions(false) + '</div></div>';
  }

  function queuedHTML(n) {
    const c = n.comment;
    const stacked = ctx.pulls.members.length > 0;
    return '<div class="queued-note" data-comment="' + c.id + '">' +
      '<div class="qn-head"><span class="qn-tag">Queued ' + c.id + '</span>' +
      (stacked ? '<span class="qn-to">&rarr; ' + ctx.pulls.pullPill(n.member.pull) + '</span>' : '') +
      (c.stale ? '<span class="qn-stale" title="' + esc(c.stale) + '">stale</span>' : '') +
      '<span class="spacer"></span>' +
      '<button type="button" class="nbtn quiet small" data-act="edit">Edit</button>' +
      '<button type="button" class="nbtn quiet small" data-act="drop">Drop</button></div>' +
      '<p class="qn-body">' + esc(c.body) + '</p></div>';
  }

  function textArea(placeholder, value) {
    return '<textarea rows="3" placeholder="' + esc(placeholder) + '">' + esc(value || '') + '</textarea>';
  }
  function actions(queueOff) {
    return '<div class="note-actions">' +
      '<button type="button" class="nbtn primary" data-act="ask">Ask Claude</button>' +
      (ctx.state.queue ? '<button type="button" class="nbtn" data-act="queue"' + (queueOff ? ' disabled title="' + esc(queueOff) + '"' : '') + '>Queue comment</button>' : '') +
      '<span class="note-status" aria-live="polite"></span></div>';
  }

  // Where a comment on the composer's lines would go, or why none can.
  function destination(c) {
    if (!ctx.state.queue) return { words: '' };
    if (ctx.state.moved) return { off: 'This review moved, so nothing can be queued here.' };
    if (!ctx.data.files[c.path]) return { off: 'This file is not in the change, so a comment cannot go on it. Ask about it instead.' };
    if (!ctx.pulls.members.length) {
      const m = ctx.state.queue.members[0];
      if (m && m.state && m.state !== 'open') return { off: '#' + m.pull + ' is ' + m.state + ', so nothing can be queued for it.' };
      return { words: '' };
    }
    const t = ctx.pulls.lineTarget(c.path, c.side, c.line);
    if (!t) return { off: 'This line is not in the combined diff, so a comment cannot go on it.' };
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
      (to.off ? ' &middot; ' + esc(to.off) : '') + ' <span class="hint-inline">Shift-select another line number for a range.</span></p>' + choose +
      textArea(ctx.state.queue ? 'Ask Claude about these lines, or write a comment for the pull request' : 'Ask Claude about these lines', c.text) +
      actions(to.off) + '<button type="button" class="nbtn quiet close" data-act="cancel" aria-label="Close">Cancel</button></div>';
  }

  function columns(row) {
    return Array.from(row.cells).reduce((n, cell) => n + cell.colSpan, 0);
  }
  function rowFor(file, side, line) {
    return file.querySelector('tr[data-l~="' + (side === 'LEFT' ? 'L:' : 'R:') + line + '"]');
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
    if (!file.querySelector('tr[data-l]')) return;

    const notes = threadsOn(path).map(t => ({ side: t.side, line: t.line, html: threadHTML(t), cls: 'thread-row', key: 't' + t.id }))
      .concat(queuedOn(path).map(n => ({ side: n.side, line: n.line, html: queuedHTML(n), cls: 'queued-row', key: 'q' + n.comment.id })));
    notes.forEach(n => {
      let row = rowFor(file, n.side, n.line);
      if (!row && n.side === 'RIGHT') { ctx.diffs.expandFor(file, n.line); row = rowFor(file, n.side, n.line); }
      if (!row) return;
      let after = row;
      while (after.nextElementSibling && after.nextElementSibling.classList.contains('note-row')) after = after.nextElementSibling;
      after.after(noteRow(row, n.cls, n.html));
    });
    Object.keys(typed).forEach(id => {
      const area = file.querySelector('.thread[data-thread="' + id + '"] textarea');
      if (area) area.value = typed[id];
    });
    if (composer && composer.path === path) placeComposer(file);
  }

  function placeComposer(file) {
    const row = rowFor(file, composer.side, composer.line);
    if (!row) { composer = null; return; }
    for (let n = composer.start_line || composer.line; n <= composer.line; n++) {
      const r = rowFor(file, composer.side, n);
      if (r) r.classList.add('picked');
    }
    let after = row;
    while (after.nextElementSibling && after.nextElementSibling.classList.contains('note-row') && !after.nextElementSibling.classList.contains('composer-row')) after = after.nextElementSibling;
    const tr = noteRow(row, 'composer-row', composerHTML(composer));
    after.after(tr);
    const area = tr.querySelector('textarea');
    area.focus();
    area.setSelectionRange(area.value.length, area.value.length);
  }

  function compose(file, side, line, extend) {
    const path = file.getAttribute('data-path');
    const text = composer ? composer.text : '';
    if (extend && composer && composer.path === path && composer.side === side && line !== composer.line) {
      const first = Math.min(composer.start_line || composer.line, line), last = Math.max(composer.start_line || composer.line, composer.line, line);
      composer = { path, side, line: last, start_line: first === last ? null : first, text, pull: composer.pull };
    } else {
      composer = { path, side, line, start_line: null, text, pull: null };
    }
    document.querySelectorAll('tr.composer-row').forEach(tr => tr.remove());
    document.querySelectorAll('tr.picked').forEach(tr => tr.classList.remove('picked'));
    placeComposer(file);
  }

  function codeOf(file, n) {
    const rows = [];
    for (let k = n.start_line || n.line; k <= n.line; k++) {
      const row = rowFor(file, n.side, k);
      if (!row) continue;
      const cells = row.querySelectorAll('td.code');
      const cell = cells.length > 1 ? cells[n.side === 'LEFT' ? 0 : cells.length - 1] : cells[0];
      if (cell) rows.push(cell.textContent);
    }
    return rows.join('\n');
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
        : await send('POST', reviewPath(ctx.state.id, '/threads'), { path: anchor.path, side: anchor.side, line: anchor.line, start_line: anchor.start_line || null, code: anchor.code, text });
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
        path: anchor.path, side: anchor.side, line: anchor.line, start_line: anchor.start_line || null, body: text, code: anchor.code, pull: pull || null,
      });
      if (box.closest('.composer-row')) composer = null;
      else box.querySelector('textarea').value = '';
      ctx.setState(done.state);
    } catch (error) {
      failed(box, error);
    }
  }

  const main = document.getElementById('main');
  main.addEventListener('input', e => {
    if (composer && e.target.closest('.composer-row') && e.target.tagName === 'TEXTAREA') composer.text = e.target.value;
  });
  main.addEventListener('change', e => {
    if (composer && e.target.matches('.composer select[data-pull]')) composer.pull = +e.target.value;
  });
  main.addEventListener('click', e => {
    const button = e.target.closest('.note-row button[data-act]');
    if (!button) return;
    const act = button.getAttribute('data-act');
    const file = button.closest('details.file');
    const path = file.getAttribute('data-path');

    const composerBox = button.closest('.composer');
    if (composerBox) {
      if (act === 'cancel') { composer = null; button.closest('tr').remove(); file.querySelectorAll('tr.picked').forEach(tr => tr.classList.remove('picked')); return; }
      const text = composerBox.querySelector('textarea').value.trim();
      if (!text) { composerBox.querySelector('textarea').focus(); return; }
      const select = composerBox.querySelector('select[data-pull]');
      const anchor = { path, side: composer.side, line: composer.line, start_line: composer.start_line, code: codeOf(file, composer) };
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

  return { compose, decorate, countBadge, refresh };
}
