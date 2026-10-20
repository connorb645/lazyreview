/**
 * The files of one review, inside its folder, that the command reads or
 * writes:
 *
 *   review.json        the review: its pull requests, queued comments and
 *                      the reviews sent from it
 *   written.md         a written review, if one was attached
 *   .lock              held while the review changes
 *
 * The terminal app keeps the rest: page.json, config.json, threads/ and
 * github.json. ReviewFolder only names paths. It reads and writes nothing.
 */
import { join } from 'node:path';

export class ReviewFolder {
  constructor(path) {
    this.path = path;
    Object.freeze(this);
  }

  get record() { return join(this.path, 'review.json'); }
  get written() { return join(this.path, 'written.md'); }
  get lock() { return join(this.path, '.lock'); }
}
