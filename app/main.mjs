/**
 * Starts the app: the list of reviews at /, and one review at
 * /review/<host>/<owner>/<repo>/<pull>.
 */
import { watchBar, showProblem, loaded } from './controllers/page.mjs';
import { openHome } from './controllers/home.mjs';
import { openReview } from './controllers/review.mjs';
import { english } from '/i18n/english.mjs';

watchBar();

const reviewId = /^\/review\/(.+)$/.exec(location.pathname);
try {
  // The diagram fits its names to their boxes by measuring them, so it is drawn in its own font, never the fallback.
  await document.fonts.ready;
  if (reviewId) await openReview(decodeURIComponent(reviewId[1]).replace(/\/+$/, ''));
  else await openHome();
} catch (error) {
  showProblem(reviewId ? english.web.app.reviewFailed : english.web.app.listFailed, error.message);
} finally {
  loaded();
}
