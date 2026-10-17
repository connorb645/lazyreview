/**
 * Names one pull request: its host, its repository and its number.
 *
 * PullRef.parse(text) reads a pull request's address, such as
 * https://github.com/octo/widgets/pull/12 (anything after the number is
 * ignored), a review id such as github.com/octo/widgets/12, or the short
 * form octo/widgets#12, which means github.com. It throws an error saying
 * what it expected when the text is none of these.
 *
 * `id` is the review id the pull request gives, which is also the review's
 * folder under the data home and its address in the app, and `url` is its
 * address on the host.
 */
import { english } from '../i18n/english.mjs';

const NAME = /^[A-Za-z0-9_.-]+$/;

export class PullRef {
  constructor(host, repo, pull) {
    const [owner, name, ...rest] = String(repo).split('/');
    if (!/^[A-Za-z0-9.-]+(:\d+)?$/.test(host) || !NAME.test(owner || '') || !NAME.test(name || '') || rest.length ||
      [owner, name].some(part => part === '.' || part === '..') || !Number.isInteger(pull) || pull <= 0) {
      throw new Error(english.refs.notAPull(host, repo, pull));
    }
    this.host = host.toLowerCase();
    this.repo = repo;
    this.pull = pull;
    Object.freeze(this);
  }

  static parse(text) {
    const given = String(text).trim();
    const link = /^https?:\/\/([^/]+)\/([^/]+)\/([^/]+)\/pull\/(\d+)(?:[/?#].*)?$/.exec(given);
    if (link) return new PullRef(link[1], `${link[2]}/${link[3]}`, Number(link[4]));
    const id = /^([^/]+\.[^/]+)\/([^/]+)\/([^/]+)\/(\d+)$/.exec(given);
    if (id) return new PullRef(id[1], `${id[2]}/${id[3]}`, Number(id[4]));
    const short = /^([^/#\s]+)\/([^/#\s]+)#(\d+)$/.exec(given);
    if (short) return new PullRef('github.com', `${short[1]}/${short[2]}`, Number(short[3]));
    throw new Error(english.refs.notAPullAddress(given));
  }

  get id() {
    return `${this.host}/${this.repo}/${this.pull}`;
  }

  get url() {
    return `https://${this.host}/${this.repo}/pull/${this.pull}`;
  }

  /** The same repository's pull request `pull`. */
  withPull(pull) {
    return new PullRef(this.host, this.repo, pull);
  }
}
