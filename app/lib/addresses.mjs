/**
 * The app's own addresses.
 */

/** The address of a review's page in the app, from its id. */
export function reviewAddress(id) {
  return '/review/' + id.split('/').map(encodeURIComponent).join('/');
}
