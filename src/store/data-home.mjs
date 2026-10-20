/**
 * Where lazyreview keeps its files, under one folder, the data home:
 *
 *   clones/<host>/<owner>/<repo>.git          a bare clone per repository
 *   clones/local/<repo key>.git               a bare clone per repository of
 *                                             this machine a branch review asks about
 *   reviews/<host>/<owner>/<repo>/<pull>/     one folder per pull request review
 *   reviews/local/<repo key>/<branch>/        one folder per branch review
 *   copies/<review id>/<conversation>/        a throwaway copy of the code,
 *                                             one per conversation with Claude
 *   server.json                               a server an older lazyreview
 *                                             left running, until stop stops it
 *   settings.json                             the accounts, before the config held them;
 *                                             no longer read, and moved by
 *                                             lazyreview config move
 *
 * The terminal app keeps the clones and copies. DataHome names only the
 * paths the command uses, and reads and writes nothing.
 */
import { join } from 'node:path';
import { ReviewFolder } from './review-folder.mjs';

export class DataHome {
  constructor(root) {
    this.root = root;
    Object.freeze(this);
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
}
