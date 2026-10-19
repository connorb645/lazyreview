/**
 * The front page's markup.
 *
 * brandTitle() is the app's name as the bar draws it. homeList(reviews)
 * is the list of every review that did not move, each a link to it, or a
 * line saying there are none. A review of a repository the config hides is
 * left out, and a line says how many were, and of which repositories.
 */
import { esc, when } from '../lib/text.mjs';
import { reviewAddress } from '../lib/addresses.mjs';
import { english } from '/i18n/english.mjs';

const words = english.web.home;

export function brandTitle() {
  return 'lazy<span class="accent-word">review</span>';
}

export function homeList(reviews) {
  const hidden = reviews.filter(r => !r.moved && r.hidden_by && r.hidden_by.length);
  const live = reviews.filter(r => !r.moved && !hidden.includes(r));
  const leftOut = hidden.length
    ? '<li class="home-hidden">' + words.hiddenLeftOut(hidden.length, [...new Set(hidden.map(r => r.repo))].map(repo => '<b>' + esc(repo) + '</b>')) + '</li>'
    : '';
  if (!live.length) return (hidden.length ? '' : '<li class="home-empty">' + words.none + '</li>') + leftOut;
  return live.map(r => {
    const branch = r.kind === 'branch';
    const what = branch ? english.review.branchName(r.branch, r.base) : english.review.pullsName(r.members.map(m => m.pull));
    const ended = branch ? [] : r.members.filter(m => m.state && m.state !== 'open').map(m => words.ended(m.pull, m.state));
    return '<li><a class="home-item" href="' + esc(reviewAddress(r.id)) + '">' +
      '<span class="home-title">' + esc(r.title) + '</span>' +
      '<span class="home-meta"><b>' + esc(r.repo) + '</b> ' + esc(what) + ' &middot; ' + words.loaded(esc(when(r.loaded_at))) +
      (r.queued ? ' &middot; ' + words.queued(r.queued) : '') + (ended.length ? ' &middot; <span class="st-ended">' + esc(ended.join(', ')) + '</span>' : '') + '</span>' +
      '</a></li>';
  }).join('') + leftOut;
}
