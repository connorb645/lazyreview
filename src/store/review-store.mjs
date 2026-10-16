/**
 * Reads and writes the reviews kept under the data home.
 *
 * Reading never locks. A change goes through change(refs, work): it locks
 * the folders of every review in `refs`, runs `work`, and writes the files
 * `work` returns, as [[path, text]] where null text removes the file. The
 * writes are all-or-nothing: a failure partway puts back every file it had
 * written, and each file is replaced whole, never left half written.
 */
import { rmSync, mkdirSync, readdirSync, existsSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { withLocks } from './file-lock.mjs';
import { writeWhole, readText, readJSON, parseJSON } from './whole-file.mjs';
import { pageBuild } from './page-build.mjs';
import { PullRef } from '../github/pull-ref.mjs';
import { BranchRef } from '../local/branch-ref.mjs';

export class ReviewStore {
  #home;

  constructor(home) {
    this.#home = home;
  }

  get home() {
    return this.#home;
  }

  /** The review whose id `ref` gives, or null when there is none. */
  read(ref) {
    return readJSON(this.#home.review(ref).record);
  }

  /** The page data of a review with its build id, as {build, page}, or null. */
  readPage(ref) {
    const path = this.#home.review(ref).page;
    const text = readText(path);
    return text === null ? null : { build: pageBuild(text), page: parseJSON(path, text) };
  }

  /** The build id of a review's page, or null when it has none. */
  pageBuild(ref) {
    const text = readText(this.#home.review(ref).page);
    return text === null ? null : pageBuild(text);
  }

  readWritten(ref) {
    return readText(this.#home.review(ref).written);
  }

  readConfig(ref) {
    return readJSON(this.#home.review(ref).config);
  }

  /** Every conversation on the review's lines, oldest first. */
  threads(ref) {
    const folder = this.#home.review(ref);
    if (!existsSync(folder.threads)) return [];
    return readdirSync(folder.threads)
      .filter(name => /^\d+\.json$/.test(name))
      .map(name => readJSON(join(folder.threads, name)))
      .filter(Boolean)
      .sort((a, b) => a.id - b.id);
  }

  /** The review threads on GitHub as the last load read them, none for a review that keeps none. */
  githubThreads(ref) {
    return readJSON(this.#home.review(ref).github) || [];
  }

  /** Every review kept, of pull requests and of branches, each {ref, review}, newest load first. */
  list() {
    const out = [];
    for (const repoKey of folders(this.#home.branchReviews)) {
      for (const branch of folders(join(this.#home.branchReviews, repoKey))) {
        const ref = BranchRef.parse(`local/${repoKey}/${branch}`);
        const review = this.read(ref);
        if (review) out.push({ ref, review });
      }
    }
    for (const host of folders(this.#home.reviews)) {
      if (join(this.#home.reviews, host) === this.#home.branchReviews) continue;
      for (const owner of folders(join(this.#home.reviews, host))) {
        for (const name of folders(join(this.#home.reviews, host, owner))) {
          out.push(...this.#reviewsIn(host, `${owner}/${name}`));
        }
      }
    }
    return out.sort((a, b) => String(b.review.loaded_at).localeCompare(String(a.review.loaded_at)));
  }

  /**
   * The review a pull request is read in, as {ref, review}: the one holding
   * it as a live pull request, or else its own, or null when it has none.
   */
  find(ref) {
    const holding = this.#reviewsIn(ref.host, ref.repo).find(e => !e.review.moved && e.review.members.some(m => m.pull === ref.pull));
    if (holding) return holding;
    const own = this.read(ref);
    return own ? { ref, review: own } : null;
  }

  /** Every review of one repository, each {ref, review}. */
  inRepository(host, repo) {
    return this.#reviewsIn(host, repo);
  }

  #reviewsIn(host, repo) {
    return folders(this.#home.repository(host, repo))
      .filter(name => /^\d+$/.test(name))
      .map(name => new PullRef(host, repo, Number(name)))
      .map(ref => ({ ref, review: this.read(ref) }))
      .filter(entry => entry.review);
  }

  /** A file write that puts `value` in a review's record. */
  recordWrite(ref, value) {
    return [this.#home.review(ref).record, JSON.stringify(value, null, 2) + '\n'];
  }

  async change(refs, work) {
    const locks = refs.map(ref => this.#home.review(ref).lock);
    for (const lock of locks) mkdirSync(dirname(lock), { recursive: true });
    return withLocks(locks, async () => {
      const { files, result } = await work();
      writeAll(files);
      return result;
    });
  }
}

function writeAll(files) {
  const done = [];
  try {
    for (const [path, text] of files) {
      const before = readText(path);
      replace(path, text);
      done.push([path, before]);
    }
  } catch (error) {
    for (const [path, before] of done.reverse()) replace(path, before);
    throw new Error(`could not write the review, so every file is as it was: ${error.message}`);
  }
}

function replace(path, text) {
  if (text === null) { rmSync(path, { force: true }); return; }
  mkdirSync(dirname(path), { recursive: true });
  writeWhole(path, text);
}

function folders(path) {
  try {
    return readdirSync(path, { withFileTypes: true }).filter(e => e.isDirectory() && !e.name.startsWith('.')).map(e => e.name);
  } catch (error) {
    if (error.code === 'ENOENT') return [];
    throw error;
  }
}
