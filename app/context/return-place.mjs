/**
 * Where a review's page was when it was opened again, by Refresh or by a
 * reload after a change elsewhere, kept for this browser tab only.
 *
 * remember(place) keeps {id, hash, y, line}; recall() gives it back once,
 * and forgets it, or null when there is none or no storage.
 */
const RETURN = 'lazyreview:return';

export function remember(place) {
  try {
    sessionStorage.setItem(RETURN, JSON.stringify(place));
  } catch {
    // Without storage the page opens at the top of its view, which is where a reload puts it anyway.
  }
}

export function recall() {
  try {
    const text = sessionStorage.getItem(RETURN);
    sessionStorage.removeItem(RETURN);
    return text ? JSON.parse(text) : null;
  } catch {
    return null;
  }
}
