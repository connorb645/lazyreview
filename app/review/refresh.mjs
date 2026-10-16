/**
 * The bar's Refresh button, in every view of a review. A press loads the
 * review again, a pull request's from GitHub and a branch's from its
 * repository, then opens the page again at the view and the place it was.
 *
 * createRefresh(ctx) shows the button, unless the review moved, and returns
 * reloadHere(), which opens the page again at the view and the place it is,
 * and restore(), which scrolls back to where the page was when it was opened
 * again so, and does nothing otherwise.
 */
import { send, reviewPath, reviewAddress } from '../lib/api.mjs';
import { when } from '../lib/text.mjs';

const RETURN = 'review-desk:return';

export function createRefresh(ctx) {
  const button = document.getElementById('refresh');
  const label = button.querySelector('.refresh-label');
  const branch = ctx.state.kind === 'branch';
  const what = branch ? 'Load the branch again' : 'Load again from GitHub';
  button.hidden = !!ctx.state.moved;
  button.title = what + '. Loaded ' + when(ctx.state.loaded_at) + '.';
  button.setAttribute('aria-label', what);

  button.addEventListener('click', async () => {
    button.disabled = true;
    button.classList.add('busy');
    label.textContent = 'Refreshing';
    try {
      const done = await send('POST', reviewPath(ctx.state.id, '/refresh'));
      remember({ id: done.id, hash: location.hash, y: window.scrollY, line: firstLineShown() });
      if (done.id === ctx.state.id) location.reload();
      else location.assign(reviewAddress(done.id) + location.hash);
    } catch (error) {
      button.disabled = false;
      button.classList.remove('busy');
      label.textContent = 'Refresh';
      ctx.flash('Not refreshed: ' + error.message);
    }
  });

  function reloadHere() {
    remember({ id: ctx.state.id, hash: location.hash, y: window.scrollY, line: firstLineShown() });
    location.reload();
  }

  function restore() {
    history.scrollRestoration = 'manual';
    const back = recall();
    if (!back || back.id !== ctx.state.id || back.hash !== location.hash) return;
    const row = back.line && document.querySelector('#main details.file[data-path="' + CSS.escape(back.line.path) + '"] tr[data-l="' + CSS.escape(back.line.key) + '"]');
    if (!row) { window.scrollTo(0, back.y); return; }
    row.closest('details.file').style.contentVisibility = 'visible';
    // Files further down are drawn as they come into view, so the row is placed again once they have their size.
    const place = () => window.scrollBy(0, row.getBoundingClientRect().top - back.line.top);
    place();
    requestAnimationFrame(() => requestAnimationFrame(place));
  }

  return { reloadHere, restore };
}

function firstLineShown() {
  const bar = document.getElementById('bar').getBoundingClientRect().bottom;
  const row = Array.prototype.find.call(document.querySelectorAll('#main:not([hidden]) tr[data-l]'), r => r.getBoundingClientRect().top >= bar);
  if (!row || document.getElementById('diff-view').hidden) return null;
  return { path: row.closest('details.file').getAttribute('data-path'), key: row.getAttribute('data-l'), top: row.getBoundingClientRect().top };
}

function remember(place) {
  try {
    sessionStorage.setItem(RETURN, JSON.stringify(place));
  } catch {
    // Without storage the page opens at the top of its view, which is where a reload puts it anyway.
  }
}

function recall() {
  try {
    const text = sessionStorage.getItem(RETURN);
    sessionStorage.removeItem(RETURN);
    return text ? JSON.parse(text) : null;
  } catch {
    return null;
  }
}
