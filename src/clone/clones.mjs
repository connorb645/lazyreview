/**
 * Keeps a bare clone of each repository a review reads, under the data home.
 *
 * clones.prepare({host, repo, url, base, pulls, commits, environment}) makes
 * the clone the first time, then fetches the base branch and each pull
 * request's head into it, and makes sure every commit in `commits` is there,
 * fetching any missing one by its id. It returns the clone's path. Network
 * calls go through gh as git's credential helper, as the account
 * `environment` names, so a private repository needs nothing set up beyond
 * gh itself. A clone only ever holds what Review Desk fetched into it, so
 * fetching overwrites its refs freely; one fetch at a time runs in it.
 */
import { execFile } from 'node:child_process';
import { existsSync, mkdirSync } from 'node:fs';
import { dirname } from 'node:path';
import { withLocks } from '../store/file-lock.mjs';

const TIMEOUT = 15 * 60_000;
const CREDENTIALS = ['-c', 'credential.helper=', '-c', 'credential.helper=!gh auth git-credential'];

export class Clones {
  #home;
  #run;

  constructor(home, run = runGit) {
    this.#home = home;
    this.#run = run;
  }

  async prepare({ host, repo, url, base, pulls, commits, environment = {} }) {
    const path = this.#home.clone(host, repo);
    mkdirSync(dirname(path), { recursive: true });
    return withLocks([`${path}.lock`], () => this.#fetch(path, { url, base, pulls, commits, environment }), { wait: TIMEOUT });
  }

  async #fetch(path, { url, base, pulls, commits, environment }) {
    if (!existsSync(path)) {
      await this.#git(null, ['init', '--quiet', '--bare', path]);
      await this.#git(path, ['remote', 'add', 'origin', url]);
    } else {
      await this.#git(path, ['remote', 'set-url', 'origin', url]);
    }
    const refspecs = [`+refs/heads/${base}:refs/heads/${base}`, ...pulls.map(n => `+refs/pull/${n}/head:refs/pull/${n}/head`)];
    await this.#git(path, [...CREDENTIALS, 'fetch', '--quiet', '--no-tags', 'origin', ...refspecs], environment);
    const missing = [];
    for (const commit of commits) if (!(await this.#has(path, commit))) missing.push(commit);
    if (missing.length) await this.#git(path, [...CREDENTIALS, 'fetch', '--quiet', '--no-tags', 'origin', ...missing], environment);
    return path;
  }

  async #has(path, commit) {
    const done = await this.#run(path, ['cat-file', '-e', `${commit}^{commit}`]);
    return done.code === 0;
  }

  async #git(path, args, environment = {}) {
    const done = await this.#run(path, args, environment);
    if (done.code !== 0) throw new Error(`git ${args.filter(a => !a.startsWith('credential') && a !== '-c')[0]} failed: ${done.stderr.trim().split('\n').pop()}`);
    return done.stdout;
  }
}

function runGit(path, args, environment = {}) {
  return new Promise(done => {
    execFile('git', [...(path ? ['-C', path] : []), ...args], {
      timeout: TIMEOUT, maxBuffer: 1 << 26, env: { ...process.env, GIT_TERMINAL_PROMPT: '0', ...environment },
    }, (error, stdout, stderr) => {
      done({ code: error ? (typeof error.code === 'number' ? error.code : 1) : 0, stdout: String(stdout), stderr: String(stderr || (error ? error.message : '')) });
    });
  });
}
