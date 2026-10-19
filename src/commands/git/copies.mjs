/**
 * Makes throwaway copies of a repository at a commit, through git, for
 * Claude Code to work in.
 *
 * new Copies({run, now}) takes how git runs and the clock, in milliseconds.
 * prepare(folder, {source, commit, branches, branch}) makes `folder` a git
 * repository cloned from `source`, one of lazyreview's own bare clones,
 * unless an earlier call did, marks it used now and returns it. The copy
 * borrows the source's objects instead of copying them, and has no remote,
 * so a push has nowhere to go. It holds a branch for each of `branches`
 * ({name: commit}) and has `branch` checked out, which must be one of them,
 * at `commit`. Making it runs no hook and fetches nothing. The folder
 * appears whole or not at all.
 *
 * prune(parent, {unused, keep}) removes each copy under `parent` not used in
 * the last `unused` milliseconds, other than `keep`, and the folders it
 * leaves empty. Neither takes a lock: the caller keeps one prepare or prune
 * at a time under `parent`.
 */
import { execFile } from 'node:child_process';
import { existsSync, mkdirSync, mkdtempSync, renameSync, rmSync, rmdirSync, utimesSync, readdirSync, statSync } from 'node:fs';
import { basename, dirname, join } from 'node:path';
import { english } from '../../i18n/english.mjs';

const TIMEOUT = 15 * 60_000;
const QUIET = ['-c', 'core.hooksPath=/dev/null', '-c', 'advice.detachedHead=false'];

export class Copies {
  #run;
  #now;

  constructor({ run = runGit, now = () => Date.now() } = {}) {
    this.#run = run;
    this.#now = now;
  }

  async prepare(folder, { source, commit, branches, branch }) {
    if (!existsSync(join(folder, '.git'))) await this.#make(folder, { source, commit, branches, branch });
    const now = new Date(this.#now());
    utimesSync(folder, now, now);
    return folder;
  }

  prune(parent, { unused, keep }) {
    if (!existsSync(parent)) return;
    const before = this.#now() - unused;
    (function walk(folder) {
      for (const entry of readdirSync(folder, { withFileTypes: true })) {
        if (!entry.isDirectory()) continue;
        const path = join(folder, entry.name);
        if (path === keep) continue;
        if (existsSync(join(path, '.git')) || entry.name.startsWith('.')) {
          if (statSync(path).mtimeMs < before) rmSync(path, { recursive: true, force: true });
        } else {
          walk(path);
          if (!readdirSync(path).length) rmdirSync(path);
        }
      }
    })(parent);
  }

  async #make(folder, { source, commit, branches, branch }) {
    if ((await this.#run(source, ['cat-file', '-e', `${commit}^{commit}`])).code !== 0) throw new Error(english.git.missing(commit, source));
    mkdirSync(dirname(folder), { recursive: true });
    const temporary = mkdtempSync(join(dirname(folder), `.${basename(folder)}-`));
    try {
      await this.#git(null, [...QUIET, 'clone', '--quiet', '--shared', '--no-checkout', source, temporary]);
      await this.#git(temporary, ['remote', 'remove', 'origin']);
      for (const [name, at] of Object.entries(branches)) await this.#git(temporary, ['update-ref', `refs/heads/${name}`, at]);
      await this.#git(temporary, [...QUIET, 'checkout', '--quiet', branch]);
      rmSync(folder, { recursive: true, force: true });
      renameSync(temporary, folder);
    } catch (error) {
      rmSync(temporary, { recursive: true, force: true });
      throw error;
    }
  }

  async #git(repo, args) {
    const done = await this.#run(repo, args);
    if (done.code !== 0) throw new Error(english.git.failed(args.find(a => !a.startsWith('-') && !a.includes('=')), done.stderr.trim().split('\n').pop()));
    return done.stdout;
  }
}

function runGit(repo, args) {
  return new Promise(done => {
    execFile('git', [...(repo ? ['-C', repo] : []), ...args], {
      timeout: TIMEOUT, maxBuffer: 1 << 24, env: { ...process.env, GIT_TERMINAL_PROMPT: '0', GIT_NO_LAZY_FETCH: '1', GIT_LFS_SKIP_SMUDGE: '1' },
    }, (error, stdout, stderr) => {
      done({ code: error ? (typeof error.code === 'number' ? error.code : 1) : 0, stdout: String(stdout), stderr: String(stderr || (error ? error.message : '')) });
    });
  });
}
