/**
 * Keeps an open review up to date with changes made elsewhere: in another
 * window, in the terminal app, or by the command.
 *
 * createLiveUpdates(ctx, reloadHere) listens to the review's change notices.
 * A new state is read and drawn at once, and every draft stays as it is. A
 * new page build opens the page again in place, through `reloadHere`, when
 * no draft is open; with one open, a note says the review was loaded again
 * and offers the reload, so the draft is kept until then.
 */
import { getJSON, reviewPath } from '../lib/api.mjs';

export function createLiveUpdates(ctx, reloadHere) {
  const note = document.getElementById('reload-note');
  let wanted = null;
  let reading = false;

  async function catchUp() {
    if (reading) return;
    reading = true;
    try {
      while (wanted && wanted !== ctx.state.version) {
        const asked = wanted;
        const next = await getJSON(reviewPath(ctx.state.id));
        if (next.version !== ctx.state.version) ctx.setState(next);
        if (wanted === asked) break;
      }
    } catch (error) {
      ctx.flash('A change made elsewhere could not be read: ' + error.message);
    } finally {
      reading = false;
    }
  }

  function loadedAgain() {
    if (!ctx.notes.drafting() && !ctx.queuePanel.drafting()) { reloadHere(); return; }
    note.hidden = false;
  }

  note.querySelector('button').addEventListener('click', reloadHere);
  const events = new EventSource(reviewPath(ctx.state.id, '/events'));
  events.addEventListener('message', e => {
    const notice = JSON.parse(e.data);
    if (notice.build && notice.build !== ctx.data.build) { loadedAgain(); return; }
    if (notice.state) {
      wanted = notice.state;
      catchUp();
    }
  });
}
