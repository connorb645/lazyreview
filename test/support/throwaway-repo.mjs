/**
 * A throwaway repository for tests, with an origin that serves pull request
 * heads the way GitHub does, as refs/pull/<n>/head.
 *
 * throwawayRepo() makes both in a new temporary folder and returns:
 *   work            the working clone's path
 *   origin          the bare origin's path, the address a clone fetches from
 *   commit(branch, files, message)
 *                   writes `files` ({path: text}, null deletes) on `branch`,
 *                   made from the current branch when new, and commits
 *   checkout(branch)
 *   sha(ref)        the commit a ref names in the working clone
 *   publish(pulls)  pushes every branch to origin and points
 *                   refs/pull/<n>/head at each {number: branch}
 *   done()          removes everything
 */
import { execFileSync } from 'node:child_process';
import { mkdtempSync, mkdirSync, writeFileSync, rmSync, unlinkSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, dirname } from 'node:path';

export function throwawayRepo() {
  const root = mkdtempSync(join(tmpdir(), 'review-desk-repo-'));
  const work = join(root, 'work');
  const origin = join(root, 'origin.git');
  const git = (cwd, ...args) => execFileSync('git', ['-C', cwd, '-c', 'user.name=Test', '-c', 'user.email=test@example.com', '-c', 'commit.gpgsign=false', ...args], { encoding: 'utf8' }).trim();
  mkdirSync(work);
  git(work, 'init', '-q', '-b', 'main');
  execFileSync('git', ['init', '-q', '--bare', origin]);

  function commit(branch, files, message = 'change') {
    const current = git(work, 'symbolic-ref', '--short', 'HEAD');
    const exists = git(work, 'branch', '--list', branch);
    if (current !== branch) git(work, 'checkout', '-q', ...(exists ? [branch] : ['-b', branch]));
    for (const [path, text] of Object.entries(files)) {
      const full = join(work, path);
      if (text === null) { unlinkSync(full); continue; }
      mkdirSync(dirname(full), { recursive: true });
      writeFileSync(full, text);
    }
    git(work, 'add', '-A');
    git(work, 'commit', '-q', '--allow-empty', '-m', message);
    return git(work, 'rev-parse', 'HEAD');
  }

  return {
    work,
    origin,
    commit,
    checkout: branch => git(work, 'checkout', '-q', branch),
    sha: ref => git(work, 'rev-parse', ref),
    publish(pulls = {}) {
      git(work, 'push', '-q', '--force', origin, '--all');
      for (const [number, branch] of Object.entries(pulls)) git(origin, 'update-ref', `refs/pull/${number}/head`, git(work, 'rev-parse', branch));
    },
    git: (...args) => git(work, ...args),
    done: () => rmSync(root, { recursive: true, force: true }),
  };
}
