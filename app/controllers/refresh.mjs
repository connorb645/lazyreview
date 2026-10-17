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
import { send, reviewPath } from '../api/server.mjs';
import { views } from '../views.mjs';
import { reviewAddress } from '../lib/addresses.mjs';
import { when } from '../lib/text.mjs';
import { remember, recall } from '../context/return-place.mjs';
import { english } from '/i18n/english.mjs';

const words = english.web.refresh;

export function createRefresh(ctx) {
  const button = views.refresh;
  const label = button.querySelector('.refresh-label');
  const branch = ctx.state.kind === 'branch';
  const what = branch ? words.branch : words.pulls;
  button.hidden = !!ctx.state.moved;
  button.title = words.title(what, when(ctx.state.loaded_at));
  button.setAttribute('aria-label', what);

  button.addEventListener('click', async () => {
    button.disabled = true;
    button.classList.add('busy');
    label.textContent = words.refreshing;
    try {
      const done = await send('POST', reviewPath(ctx.state.id, '/refresh'));
      remember({ id: done.id, hash: location.hash, y: window.scrollY, line: firstLineShown() });
      if (done.id === ctx.state.id) location.reload();
      else location.assign(reviewAddress(done.id) + location.hash);
    } catch (error) {
      button.disabled = false;
      button.classList.remove('busy');
      label.textContent = words.refresh;
      ctx.flash(words.failed(error.message));
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
  const bar = views.bar.getBoundingClientRect().bottom;
  const row = Array.prototype.find.call(document.querySelectorAll('#main:not([hidden]) tr[data-l]'), r => r.getBoundingClientRect().top >= bar);
  if (!row || views.diffView.hidden) return null;
  return { path: row.closest('details.file').getAttribute('data-path'), key: row.getAttribute('data-l'), top: row.getBoundingClientRect().top };
}
