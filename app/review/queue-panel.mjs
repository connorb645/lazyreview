/**
 * The Queued comments section: the comments waiting to go on each pull
 * request, exactly as they will read, and the Add Comments, Approve With
 * Comments and Approve buttons.
 *
 * createQueuePanel(ctx) returns mount(section), which fills the section from
 * ctx.state.queue, and the edit and drop actions every view shares. A stack
 * gets one group per pull request, bottom first, each with its own comments
 * and its own buttons, and each comment shows the pull request it goes on. A
 * press asks once more in the page, then sends the action and the set id the
 * page showed; the server posts only if the queue still gives exactly that
 * set, so the press approves exactly what its button offered. An edit or a
 * drop names the version of the comment it was made on, and when another
 * window changed the comment since, it shows the comment as it now reads.
 */
import { esc, when } from '../lib/text.mjs';
import { send, getJSON, reviewPath } from '../lib/api.mjs';
import { linesWords } from './line-range.mjs';

const ORDER = ['comment', 'approve-with-comments', 'approve'];
const LABELS = { 'comment': 'Add Comments', 'approve-with-comments': 'Approve With Comments', 'approve': 'Approve' };
const DONE = { 'comment': 'Posted', 'approve-with-comments': 'Approved with', 'approve': 'Approved' };
const NOTE = 'Add Comments sends a review that only comments. Approve With Comments approves and sends the comments with it. Approve approves alone. None of them asks for changes.';

export function createQueuePanel(ctx) {
  function el(tag, cls, text) {
    const node = document.createElement(tag);
    if (cls) node.className = cls;
    if (text != null) node.textContent = text;
    return node;
  }
  const count = n => n + (n === 1 ? ' comment' : ' comments');
  const where = data => ctx.state.repo + '#' + data.pull;

  function pill(pull, prefix) {
    const span = el('span', 'qc-pill', (prefix || '') + '#' + pull);
    span.style.setProperty('--qc-pc', ctx.pulls.pullColor(pull));
    return span;
  }

  function item(comment, data, stacked) {
    const li = el('li', 'qc-item');
    li.setAttribute('data-comment', String(comment.id));
    const row = el('div', 'qc-where');
    row.appendChild(el('span', 'qc-num', String(comment.id)));
    if (stacked) row.appendChild(pill(data.pull, '→ '));
    const shown = comment.shown || comment;
    const link = el('a', 'qc-path', comment.path + ', ' + linesWords(comment));
    link.href = '#';
    link.setAttribute('data-go-file', shown.path);
    link.setAttribute('data-go-line', String(shown.line));
    link.setAttribute('data-go-side', shown.side);
    row.appendChild(link);
    if (comment.shown && (comment.shown.path !== comment.path || comment.shown.line !== comment.line)) {
      row.appendChild(el('span', 'qc-shown', 'line ' + comment.shown.line + (comment.shown.path !== comment.path ? ' of ' + comment.shown.path : '') + ' on this page'));
    }
    if (!data.ended && !ctx.state.moved) {
      const tools = el('span', 'qc-tools');
      tools.appendChild(button('Edit', 'nbtn quiet small', 'edit'));
      tools.appendChild(button('Drop', 'nbtn quiet small', 'drop'));
      row.appendChild(tools);
    }
    li.appendChild(row);
    if (comment.code) li.appendChild(el('pre', 'qc-code', comment.code));
    li.appendChild(el('p', 'qc-body', comment.body));
    if (comment.stale) li.appendChild(el('p', 'qc-stale', 'Its line may have moved: ' + comment.stale + '. No button sends it until it is queued again or dropped.'));
    return li;
  }

  function button(label, cls, act) {
    const b = el('button', cls, label);
    b.type = 'button';
    if (act) b.setAttribute('data-act', act);
    return b;
  }

  function wire(data, actionsEl, status) {
    const offered = ORDER.filter(action => data.sets && data.sets[action]);
    const buttons = offered.map(action => {
      const b = button(LABELS[action] + (action === 'approve' ? '' : ' (' + data.comments.length + ')'), 'qc-button' + (action === 'comment' ? '' : ' qc-approve'));
      b.addEventListener('click', () => confirmStep(action, b));
      return b;
    });
    function showButtons() {
      actionsEl.textContent = '';
      buttons.forEach(b => actionsEl.appendChild(b));
    }
    function confirmStep(action, opener) {
      actionsEl.textContent = '';
      const yes = button('Yes, ' + LABELS[action].toLowerCase(), 'qc-button qc-yes' + (action === 'comment' ? '' : ' qc-approve'));
      const no = button('Cancel', 'qc-quiet');
      actionsEl.appendChild(yes);
      actionsEl.appendChild(no);
      status.textContent = data.sets[action].words;
      no.addEventListener('click', () => { showButtons(); status.textContent = ''; opener.focus(); });
      yes.addEventListener('click', async () => {
        yes.disabled = no.disabled = true;
        status.textContent = 'Checking the pull request with GitHub, then sending…';
        try {
          const done = await send('POST', reviewPath(ctx.state.id, '/post'), { pull: data.pull, action, set: data.sets[action].set, build: ctx.data.build });
          ctx.setState(done.state, done.message);
        } catch (error) {
          showButtons();
          status.textContent = error.message;
          status.classList.add('bad');
        }
      });
      yes.focus();
    }
    showButtons();
  }

  function postedList(data) {
    const list = el('ul', 'qc-posted');
    data.posted.forEach(set => {
      const what = DONE[set.action] || DONE.comment;
      const li = el('li', null, (set.action === 'approve' ? what : what + ' ' + count(set.count)) + ', ' + when(set.at) + ': ');
      const link = el('a', null, 'the review');
      link.href = set.review;
      link.target = '_blank';
      link.rel = 'noopener';
      li.appendChild(link);
      list.appendChild(li);
    });
    return list;
  }

  function fill(box, data, stacked, left) {
    const shut = data.ended || left;
    if (data.comments.length) {
      const list = el('ol', 'qc-list');
      data.comments.forEach(c => list.appendChild(item(c, data, stacked)));
      box.appendChild(list);
      const stale = data.comments.some(c => c.stale);
      if (stale && !shut) box.appendChild(el('p', 'qc-note', 'Nothing queued here can be sent until each comment marked stale is queued again or dropped.'));
      else if (!shut) box.appendChild(el('p', 'qc-note', 'Each comment goes on ' + (stacked ? '#' + data.pull : where(data)) + ' exactly as shown.'));
    } else if (stacked) {
      box.appendChild(el('p', 'qc-empty', 'No comments queued for #' + data.pull + '.'));
    }
    if (data.ended) {
      const at = data.ended_at ? ' on ' + when(data.ended_at) : '';
      box.appendChild(el('p', 'qc-status', (stacked ? '#' + data.pull : where(data)) + ' was ' + data.state + at + '. Nothing can be sent ' + (stacked ? 'to it' : 'from here') + '.'));
    } else if (!left && !ctx.state.moved) {
      const actionsEl = el('div', 'qc-actions');
      const status = el('p', 'qc-status');
      status.setAttribute('aria-live', 'polite');
      box.appendChild(actionsEl);
      box.appendChild(status);
      if (!stacked) box.appendChild(el('p', 'qc-note', NOTE));
      wire(data, actionsEl, status);
    }
    if (data.posted.length) box.appendChild(postedList(data));
  }

  // The server's words for what a press did, with the review's address as a link.
  function result(message) {
    const p = el('p', 'qc-result');
    const text = message.charAt(0).toUpperCase() + message.slice(1);
    text.split(/(https:\/\/\S+)/).forEach(part => {
      if (!/^https:\/\//.test(part)) { p.appendChild(document.createTextNode(part)); return; }
      const a = el('a', null, part.replace(/[.,;:]+$/, ''));
      a.href = a.textContent;
      a.target = '_blank';
      a.rel = 'noopener';
      p.appendChild(a);
      p.appendChild(document.createTextNode(part.slice(a.textContent.length)));
    });
    return p;
  }

  function emptyHint() {
    return el('p', 'qc-empty', 'No comments queued. Select a line number in the diff, write the comment, and press Queue comment. Ask Claude there to draft one.');
  }

  function mount(section, message) {
    const q = ctx.state.queue;
    section.textContent = '';
    section.classList.add('qc');
    const head = el('div', 'qc-head');
    head.appendChild(el('h2', null, 'Queued comments'));
    section.appendChild(head);
    if (message) section.appendChild(result(message));
    if (q.moved) {
      const note = el('p', 'qc-empty', 'This review moved to ');
      const link = el('a', null, 'the stack’s review');
      link.href = '/review/' + q.moved.to;
      note.appendChild(link);
      note.appendChild(document.createTextNode('. Its queued comments moved there.'));
      section.appendChild(note);
      q.members.forEach(m => { if (m.posted.length) section.appendChild(postedList(m)); });
      return;
    }
    const views = q.members.map(m => ({ ...m, ended: m.state && m.state !== 'open' }));
    const stacked = ctx.pulls.members.length > 0;
    if (!stacked) {
      if (!views[0].comments.length) section.appendChild(emptyHint());
      fill(section, views[0], false, false);
      return;
    }
    section.appendChild(el('p', 'qc-note', 'A comment goes on the pull request that last changed its line, shown as → #number. Each pull request has its own buttons and gets its own review.'));
    if (!views.some(m => m.comments.length)) section.appendChild(emptyHint());
    views.map(v => [v, false]).concat(q.earlier.map(m => [{ ...m, ended: m.state && m.state !== 'open' }, true])).forEach(([view, left]) => {
      const box = el('section', 'qc-pr');
      box.setAttribute('data-pull', String(view.pull));
      const h = el('h3', 'qc-pr-head');
      h.appendChild(pill(view.pull));
      if (view.title) h.appendChild(el('span', 'qc-pr-title', view.title));
      box.appendChild(h);
      if (left && !view.ended) box.appendChild(el('p', 'qc-status', '#' + view.pull + ' left this stack. Nothing can be sent to it from here.'));
      fill(box, view, true, left);
      section.appendChild(box);
    });
    section.appendChild(el('p', 'qc-note', NOTE));
  }

  function queued(state, id) {
    return state.queue.members.concat(state.queue.earlier).flatMap(m => m.comments).find(c => c.id === id);
  }

  // The state as it now stands when another window changed comment `id` since its version `seen`, or null.
  async function changedSince(id, seen) {
    const now = await getJSON(reviewPath(ctx.state.id));
    const comment = now.queue && queued(now, id);
    return comment && comment.version !== seen ? now : null;
  }

  async function drop(id) {
    const seen = queued(ctx.state, id).version;
    try {
      const done = await send('DELETE', reviewPath(ctx.state.id, '/comments/' + id), { version: seen, build: ctx.data.build });
      ctx.setState(done.state);
    } catch (error) {
      ctx.flash(error.message);
      const now = await changedSince(id, seen).catch(() => null);
      if (now) ctx.setState(now);
    }
  }

  // Swaps a queued comment's text for a box to edit it in, with Save and Cancel.
  function edit(id, holder) {
    const body = holder.querySelector('.qc-body, .qn-body');
    if (!body || holder.querySelector('.edit-box')) return;
    const found = queued(ctx.state, id);
    let seen = found.version;
    const box = el('div', 'edit-box');
    box.innerHTML = '<textarea rows="3">' + esc(found.body) + '</textarea><div class="note-actions">' +
      '<button type="button" class="nbtn primary" data-edit="save">Save</button><button type="button" class="nbtn quiet" data-edit="cancel">Cancel</button>' +
      '<span class="note-status" aria-live="polite"></span></div>';
    body.replaceWith(box);
    const area = box.querySelector('textarea');
    area.focus();
    box.addEventListener('click', async e => {
      const act = e.target.getAttribute('data-edit');
      if (act === 'cancel') { box.replaceWith(body); return; }
      if (act !== 'save') return;
      box.querySelectorAll('button, textarea').forEach(x => { x.disabled = true; });
      try {
        const done = await send('PATCH', reviewPath(ctx.state.id, '/comments/' + id), { body: area.value, version: seen, build: ctx.data.build });
        ctx.setState(done.state);
      } catch (error) {
        const now = await changedSince(id, seen).catch(() => null);
        if (now) seen = queued(now, id).version;
        box.querySelectorAll('button, textarea').forEach(x => { x.disabled = false; });
        const status = box.querySelector('.note-status');
        status.textContent = error.message + (now ? '. Save again to replace it with your text.' : '');
        status.classList.add('bad');
      }
    });
  }

  return { mount, actions: { edit, drop } };
}
