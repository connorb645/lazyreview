/**
 * The reference a review id names: a branch's review, as local/<repository
 * key>/<branch>, or a pull request's, as <host>/<owner>/<repo>/<number>.
 *
 * reviewRef(id) is pure. It returns a BranchRef or a PullRef, and throws for
 * an id that is neither.
 */
import { PullRef } from './pull-ref.mjs';
import { BranchRef } from './branch-ref.mjs';
import { english } from '../i18n/english.mjs';

export function reviewRef(id) {
  const parts = id.split('/').map(decodeURIComponent);
  if (parts[0] === 'local' && parts.length === 3) return BranchRef.parse(parts.join('/'));
  if (parts.length !== 4 || !/^\d+$/.test(parts[3])) throw new Error(english.refs.notAReviewId(id));
  return new PullRef(parts[0], `${parts[1]}/${parts[2]}`, Number(parts[3]));
}
