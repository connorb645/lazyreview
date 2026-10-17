/**
 * Names one branch of a repository on this machine, for a review of a change
 * that has no pull request yet.
 *
 * BranchRef.of(gitFolder, branch) names `branch` of the repository whose
 * shared git folder is `gitFolder`, an absolute path: every worktree of one
 * repository gives the same folder, so a branch keeps one review whichever
 * worktree loads it. The repository's key is its name with a short hash of
 * that folder, so two repositories of the same name never share a review.
 * BranchRef.parse(id) reads a review id back, and throws an error saying what
 * it expected when the text is not one.
 *
 * `id` is local/<repository key>/<branch>, with each / of the branch written
 * as ~, which no branch name holds. It is the review's folder under the data
 * home and its address in the app.
 */
import { createHash } from 'node:crypto';
import { basename, dirname } from 'node:path';
import { english } from '../i18n/english.mjs';

const KEY = /^[A-Za-z0-9_-][A-Za-z0-9_.-]*-[0-9a-f]{8}$/;

export class BranchRef {
  constructor(repoKey, branch) {
    if (!KEY.test(repoKey) || !isBranchName(branch)) throw new Error(english.refs.notABranch(repoKey, branch));
    this.repoKey = repoKey;
    this.branch = branch;
    Object.freeze(this);
  }

  static of(gitFolder, branch) {
    const hash = createHash('sha256').update(gitFolder).digest('hex').slice(0, 8);
    return new BranchRef(`${repositoryName(gitFolder)}-${hash}`, branch);
  }

  static parse(text) {
    const id = /^local\/([^/]+)\/([^/]+)$/.exec(String(text));
    if (!id) throw new Error(english.refs.notABranchId(text));
    return new BranchRef(id[1], id[2].replaceAll('~', '/'));
  }

  get id() {
    return `local/${this.repoKey}/${this.branch.replaceAll('/', '~')}`;
  }
}

/** The name a repository shows, from its shared git folder: the folder holding .git, or a bare repository's own name. */
export function repositoryName(gitFolder) {
  const name = basename(gitFolder) === '.git' ? basename(dirname(gitFolder)) : basename(gitFolder).replace(/\.git$/, '');
  return name.replace(/[^A-Za-z0-9_.-]/g, '-').replace(/^\.+/, '') || 'repository';
}

function isBranchName(name) {
  return typeof name === 'string' && name !== '' && name !== '@' &&
    !/[\x00-\x20~^:?*[\\\x7f]|\.\.|@\{|\/\/|^\/|\/$|\.$/.test(name) &&
    name.split('/').every(part => !part.startsWith('.') && !part.endsWith('.lock'));
}
