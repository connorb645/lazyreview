/**
 * The Queued comments section: the comments waiting to go on each pull
 * request, exactly as they will read, and the Add Comments, Approve With
 * Comments and Approve buttons.
 *
 * createQueuePanel(ctx) returns mount(section), which fills the section from
 * ctx.state.queue, the edit and drop actions every view shares, restore(),
 * which opens again every edit a drawing of the page closed, with its text,
 * and drafting(), whether any edit is open. A stack
 * gets one group per pull request, bottom first, each with its own comments
 * and its own buttons, and each comment shows the pull request it goes on. A
 * press asks once more in the page, then sends the action and the set id the
 * page showed; the server posts only if the queue still gives exactly that
 * set, so the press approves exactly what its button offered. An edit or a
 * drop names the version of the comment it was made on, and when another
 * window changed the comment since, it shows the comment as it now reads.
 */
import { send, getJSON, reviewPath } from '../api/server.mjs';
import { fromHTML } from '../lib/html.mjs';
import { createQueueMarkup } from '../presentation/queue.mjs';
import { english } from '/i18n/english.mjs';

const words = english.web.queue;

export function createQueuePanel(ctx) {
  const markup = createQueueMarkup(ctx);

  function wire(data, actionsEl, status) {
    const buttons = markup.offered(data).map(action => {
      const b = fromHTML(markup.actionButton(action, data));
      b.addEventListener('click', () => confirmStep(action, b));
      return b;
    });
    function showButtons() {
      actionsEl.textContent = '';
      buttons.forEach(b => actionsEl.appendChild(b));
    }
    function confirmStep(action, opener) {
      actionsEl.textContent = '';
      const yes = fromHTML(markup.yesButton(action));
      const no = fromHTML(markup.noButton());
      actionsEl.appendChild(yes);
      actionsEl.appendChild(no);
      status.textContent = data.sets[action].words;
      no.addEventListener('click', () => { showButtons(); status.textContent = ''; opener.focus(); });
      yes.addEventListener('click', async () => {
        yes.disabled = no.disabled = true;
        status.textContent = words.sending;
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

  function mount(section, message) {
    section.textContent = '';
    section.classList.add('qc');
    const { html, wired } = markup.section(message);
    section.innerHTML = html;
    section.querySelectorAll('.qc-actions').forEach((actionsEl, i) => wire(wired[i], actionsEl, actionsEl.nextElementSibling));
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

  // Edits open, by where each is drawn and its comment's number, each with its text, the version it was made on, and what the last Save said.
  const editing = new Map();

  function draftKey(holder, id) {
    return (holder.classList.contains('qc-item') ? 'review:' : 'diff:') + id;
  }

  // Swaps a queued comment's text for a box to edit it in, with Save and Cancel.
  function edit(id, holder, focus = true) {
    const body = holder.querySelector('.qc-body, .qn-body');
    if (!body || holder.querySelector('.edit-box')) return;
    const key = draftKey(holder, id);
    const found = queued(ctx.state, id);
    const draft = editing.get(key) || { text: found.body, seen: found.version, said: '' };
    editing.set(key, draft);
    const box = fromHTML(markup.editBox(draft));
    body.replaceWith(box);
    const area = box.querySelector('textarea');
    if (focus) area.focus();
    area.addEventListener('input', () => { draft.text = area.value; });
    box.addEventListener('click', async e => {
      const act = e.target.getAttribute('data-edit');
      if (act === 'cancel') { editing.delete(key); box.replaceWith(body); return; }
      if (act !== 'save') return;
      box.querySelectorAll('button, textarea').forEach(x => { x.disabled = true; });
      try {
        const done = await send('PATCH', reviewPath(ctx.state.id, '/comments/' + id), { body: area.value, version: draft.seen, build: ctx.data.build });
        editing.delete(key);
        ctx.setState(done.state);
      } catch (error) {
        const now = await changedSince(id, draft.seen).catch(() => null);
        if (now) draft.seen = queued(now, id).version;
        draft.said = now ? words.saveAgain(error.message) : error.message;
        box.querySelectorAll('button, textarea').forEach(x => { x.disabled = false; });
        const status = box.querySelector('.note-status');
        status.textContent = draft.said;
        status.classList.add('bad');
      }
    });
  }

  function restore() {
    for (const [key, draft] of editing) {
      const [where, id] = key.split(':');
      if (!queued(ctx.state, +id)) {
        editing.delete(key);
        ctx.flash(words.droppedElsewhere(id, draft.text));
        continue;
      }
      const holder = document.querySelector((where === 'review' ? '.qc-item' : '.queued-note') + '[data-comment="' + id + '"]');
      if (holder) edit(+id, holder, false);
    }
  }

  return { mount, restore, drafting: () => editing.size > 0, actions: { edit, drop } };
}
