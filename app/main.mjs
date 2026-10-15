/**
 * Starts the app: the list of reviews at /, and one review at
 * /review/<host>/<owner>/<repo>/<pull>.
 */
import { openHome } from './home-view.mjs';
import { openReview } from './review/review-app.mjs';
import { esc } from './lib/text.mjs';

const bar = document.getElementById('bar');
function measureBar() {
  document.documentElement.style.setProperty('--bar-h', Math.ceil(bar.getBoundingClientRect().height) + 'px');
}
measureBar();
new ResizeObserver(measureBar).observe(bar);

function problem(title, detail) {
  const view = document.getElementById('empty-view');
  view.querySelector('h2').textContent = title;
  view.querySelector('p').innerHTML = detail;
  view.hidden = false;
}

const reviewId = /^\/review\/(.+)$/.exec(location.pathname);
const loading = document.getElementById('loading');
try {
  // The diagram fits its names to their boxes by measuring them, so it is drawn in its own font, never the fallback.
  await document.fonts.ready;
  if (reviewId) await openReview(decodeURIComponent(reviewId[1]).replace(/\/+$/, ''));
  else await openHome();
} catch (error) {
  problem(reviewId ? 'This review could not open' : 'The list of reviews could not load', esc(error.message));
} finally {
  loading.hidden = true;
}
