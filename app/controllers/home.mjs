/**
 * The app's front page: every review loaded, of pull requests and of
 * branches, newest first, each opening its review, and how to load one.
 *
 * openHome() fetches the list and draws it into the home view.
 */
import { getJSON } from '../api/server.mjs';
import { views } from '../views.mjs';
import { brandTitle, homeList } from '../presentation/home.mjs';

export async function openHome() {
  const reviews = await getJSON('/api/reviews');
  document.title = 'lazyreview';
  views.pageTitle.innerHTML = brandTitle();
  views.homeList.innerHTML = homeList(reviews);
  views.homeView.hidden = false;
}
