/**
 * The build id of a review's page: a short hash of page.json's text. It names
 * exactly the diff and the diagram a window was drawn from, so a load that
 * draws anything differently gives a new one, and a load that draws the same
 * page keeps it.
 *
 * pageBuild(text) returns the build id of a page's text. checkBuild(current,
 * given, refused) throws, with `refused` saying what did not happen, when a
 * change comes from a window drawn from build `given` but the review's page
 * is now build `current`: its line numbers may name other code now.
 */
import { createHash } from 'node:crypto';

export function pageBuild(text) {
  return createHash('sha256').update(text).digest('hex').slice(0, 12);
}

export function checkBuild(current, given, refused) {
  if (!given) throw new Error(`${refused}: the change does not name the page build its window was drawn from`);
  if (given !== current) throw new Error(`${refused}: the review was loaded again after this window opened, so its lines may have moved; reload it, then try again`);
}
