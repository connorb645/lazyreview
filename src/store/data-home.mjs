/**
 * Where lazyreview keeps its files, under one folder, the data home:
 *
 *   clones/<host>/<owner>/<repo>.git          a bare clone per repository
 *   reviews/<host>/<owner>/<repo>/<pull>/     one folder per pull request review
 *   reviews/local/<repo key>/<branch>/        one folder per branch review
 *   server.json, server.log                   the running app, while it runs
 *   settings.json                             the accounts, before the config held them;
 *                                             no longer read, and moved by
 *                                             lazyreview config move
 *
 * DataHome only names paths. It reads and writes nothing.
 */
import { join } from 'node:path';
import { ReviewFolder } from './review-folder.mjs';

export class DataHome {
  constructor(root) {
    this.root = root;
    Object.freeze(this);
  }

  /** The bare clone of `repo` (owner/name) on `host`. */
  clone(host, repo) {
    return join(this.root, 'clones', host, `${repo}.git`);
  }

  get reviews() {
    return join(this.root, 'reviews');
  }

  /** The folder of the review a PullRef's or a BranchRef's id names. */
  review(ref) {
    return new ReviewFolder(join(this.reviews, ...ref.id.split('/')));
  }

  /** The folder holding every branch review. */
  get branchReviews() {
    return join(this.reviews, 'local');
  }

  /** The folder holding every review of one repository. */
  repository(host, repo) {
    return join(this.reviews, host, ...repo.split('/'));
  }

  get serverFile() {
    return join(this.root, 'server.json');
  }

  get serverLog() {
    return join(this.root, 'server.log');
  }

  get settingsFile() {
    return join(this.root, 'settings.json');
  }
}
