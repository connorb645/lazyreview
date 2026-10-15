/**
 * The files of one review, inside its folder:
 *
 *   review.json        the review: its pull requests, queued comments and
 *                      the reviews sent from it
 *   page.json          what the app draws, built from the clone; loading
 *                      the review again rebuilds it
 *   config.json        the components config it was loaded with, if any
 *   written.md         a written review, if one was attached
 *   threads/<n>.json   each conversation on a line
 *   .lock              held while the review changes
 *
 * ReviewFolder only names paths. It reads and writes nothing.
 */
import { join } from 'node:path';

export class ReviewFolder {
  constructor(path) {
    this.path = path;
    Object.freeze(this);
  }

  get record() { return join(this.path, 'review.json'); }
  get page() { return join(this.path, 'page.json'); }
  get config() { return join(this.path, 'config.json'); }
  get written() { return join(this.path, 'written.md'); }
  get threads() { return join(this.path, 'threads'); }
  get lock() { return join(this.path, '.lock'); }

  thread(id) {
    return join(this.threads, `${id}.json`);
  }
}
