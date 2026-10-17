/**
 * The build id of a review's page: a short hash of page.json's text. It names
 * exactly the diff and the diagram a window was drawn from, so a load that
 * draws anything differently gives a new one, and a load that draws the same
 * page keeps it.
 *
 * pageBuild(text) returns the build id of a page's text. buildProblem(current,
 * given) says why a window drawn from build `given` may not change a review
 * whose page is now build `current`, or is null when it may: its line
 * numbers may name other code now. checkBuild(current, given, refused)
 * throws that reason, as `refused` words it, saying what did not happen.
 */
import { createHash } from 'node:crypto';
import { english } from '../i18n/english.mjs';

export function pageBuild(text) {
  return createHash('sha256').update(text).digest('hex').slice(0, 12);
}

export function buildProblem(current, given) {
  if (!given) return english.store.noBuild;
  if (given !== current) return english.store.loadedAgain;
  return null;
}

export function checkBuild(current, given, refused) {
  const problem = buildProblem(current, given);
  if (problem) throw new Error(refused(problem));
}
