/**
 * The app's front page: every review loaded, of pull requests and of
 * branches, newest first, each opening its review, and how to load one.
 *
 * openHome() fetches the list and draws it into the home view.
 */
import { getJSON, reviewAddress } from './lib/api.mjs';
import { esc, when } from './lib/text.mjs';

export async function openHome() {
  const reviews = await getJSON('/api/reviews');
  document.title = 'Review Desk';
  document.getElementById('page-title').innerHTML = 'Review <span class="accent-word">Desk</span>';
  const list = document.getElementById('home-list');
  const live = reviews.filter(r => !r.moved);
  list.innerHTML = live.length ? live.map(r => {
    const branch = r.kind === 'branch';
    const what = branch ? r.branch + ' against ' + r.base : r.members.map(m => '#' + m.pull).join(', ');
    const ended = branch ? [] : r.members.filter(m => m.state && m.state !== 'open').map(m => '#' + m.pull + ' ' + m.state);
    return '<li><a class="home-item" href="' + esc(reviewAddress(r.id)) + '">' +
      '<span class="home-title">' + esc(r.title) + '</span>' +
      '<span class="home-meta"><b>' + esc(r.repo) + '</b> ' + esc(what) + ' &middot; loaded ' + esc(when(r.loaded_at)) +
      (r.queued ? ' &middot; ' + r.queued + ' queued' : '') + (ended.length ? ' &middot; <span class="st-ended">' + esc(ended.join(', ')) + '</span>' : '') + '</span>' +
      '</a></li>';
  }).join('') : '<li class="home-empty">No reviews yet.</li>';
  document.getElementById('home-view').hidden = false;
}
