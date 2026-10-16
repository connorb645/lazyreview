/**
 * Tells every app open on a review when it changes, from any process: the
 * server's own changes, and a load or an attach from the command.
 *
 * new ReviewEvents({store, watch, settle}) watches a review's folder while
 * anything listens to it. subscribe(ref, send) calls send({state, build})
 * at once with the review's state version and page build, then again within
 * `settle` milliseconds of each change to its folder, with only the one that
 * changed: {state} for anything in its state, {build} for a new page. It
 * returns a function that stops the calls, and the watch ends with the last
 * one. close() stops them all. A change that cannot be read, or a watch
 * that fails, is logged, and a failed watch ends.
 */
import { watch as watchFolder } from 'node:fs';
import { reviewState } from './review-state.mjs';

const SETTLE = 100;

export class ReviewEvents {
  #store;
  #watch;
  #settle;
  #reviews = new Map();

  constructor({ store, watch = (path, changed) => watchFolder(path, { recursive: true }, changed), settle = SETTLE }) {
    this.#store = store;
    this.#watch = watch;
    this.#settle = settle;
  }

  subscribe(ref, send) {
    let review = this.#reviews.get(ref.id);
    if (!review) {
      review = { ref, sends: new Set(), last: this.#now(ref), timer: null, watcher: null };
      review.watcher = this.#watch(this.#store.home.review(ref).path, () => this.#changed(review));
      review.watcher.on('error', error => {
        console.error(`review-desk: stopped watching ${ref.id}: ${error.message}`);
        this.#end(review);
      });
      this.#reviews.set(ref.id, review);
    }
    review.sends.add(send);
    send(review.last);
    return () => {
      review.sends.delete(send);
      if (!review.sends.size) this.#end(review);
    };
  }

  close() {
    for (const review of [...this.#reviews.values()]) this.#end(review);
  }

  #changed(review) {
    clearTimeout(review.timer);
    review.timer = setTimeout(() => {
      let now;
      try {
        now = this.#now(review.ref);
      } catch (error) {
        console.error(`review-desk: could not read ${review.ref.id} after a change: ${error.message}`);
        return;
      }
      const notice = {};
      if (now.state !== review.last.state) notice.state = now.state;
      if (now.build !== review.last.build) notice.build = now.build;
      review.last = now;
      if (Object.keys(notice).length) for (const send of review.sends) send(notice);
    }, this.#settle);
  }

  #now(ref) {
    const state = reviewState(this.#store, ref);
    return { state: state ? state.version : null, build: this.#store.pageBuild(ref) };
  }

  #end(review) {
    clearTimeout(review.timer);
    review.watcher.close();
    if (this.#reviews.get(review.ref.id) === review) this.#reviews.delete(review.ref.id);
  }
}
