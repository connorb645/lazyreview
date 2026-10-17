/**
 * Which notes on a review are folded and which are open, as the person left
 * them, so a note drawn again, or the page opened again by Refresh, stays
 * that way. They are kept for this browser tab only.
 *
 * createFolds(reviewId) returns {isOpen(key, fallback), set(key, open)}:
 * isOpen gives the note's last state, or `fallback` for a note never folded
 * or opened by hand. Without storage they last as long as the page.
 */
export function createFolds(reviewId) {
  const name = 'lazyreview:folds:' + reviewId;
  let kept = {};
  try {
    kept = JSON.parse(sessionStorage.getItem(name) || '{}');
  } catch {
    kept = {};
  }
  return {
    isOpen: (key, fallback) => (key in kept ? kept[key] : fallback),
    set(key, open) {
      kept[key] = open;
      try {
        sessionStorage.setItem(name, JSON.stringify(kept));
      } catch {
        // The page still remembers them until it closes.
      }
    },
  };
}
