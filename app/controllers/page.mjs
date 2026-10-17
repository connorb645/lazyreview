/**
 * What every page of the app shares: the bar, whose height the page's
 * layout follows, the loading line, and the view that says why a page could
 * not open.
 *
 * watchBar() keeps the page's --bar-h at the bar's height as it changes.
 * showProblem(title, message) shows the problem view with `message` as its
 * words, and loaded() hides the loading line.
 */
import { views } from '../views.mjs';
import { esc } from '../lib/text.mjs';

export function watchBar() {
  const measure = () => document.documentElement.style.setProperty('--bar-h', Math.ceil(views.bar.getBoundingClientRect().height) + 'px');
  measure();
  new ResizeObserver(measure).observe(views.bar);
}

export function showProblem(title, message) {
  views.emptyView.querySelector('h2').textContent = title;
  views.emptyView.querySelector('p').innerHTML = esc(message);
  views.emptyView.hidden = false;
}

export function loaded() {
  views.loading.hidden = true;
}
