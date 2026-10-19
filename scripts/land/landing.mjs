/**
 * Lands a branch on main as a fast-forward, on every remote and in this
 * checkout. Nothing here forces a push or undoes one.
 *
 * new Landing(repo, run) works on the checkout at `repo`; `run` runs git
 * there. plan(branch) fetches every remote, checks this checkout and each
 * remote, and returns what landing `branch` takes, as {branch, commit,
 * short, adds, remotes, here}, or throws one plain line saying why it cannot
 * land. The checkout must be on main with nothing uncommitted, the branch at
 * one commit on every remote, and main, here and on each remote, behind it.
 * describe(plan) says what land(plan) will do. land(plan) pushes the commit
 * to main on each remote that lacks it, in turn, then moves main here to it,
 * fast-forward only, and returns the line saying where main is. When a step
 * fails after another remote took the push, it throws the lines saying which
 * remotes are apart and the one command that finishes the job.
 */
import { execFileSync } from 'node:child_process';

const FINISH = branch => `make land BRANCH=${branch}`;

const words = {
  noBranch: `say which branch to land: ${FINISH('<branch>')}`,
  notOnMain: current => `this checkout is on ${current || 'no branch'}, not main`,
  uncommitted: 'this checkout has uncommitted changes',
  noRemote: 'this repository has no remote to land on',
  noFetch: (remote, why) => `could not fetch ${remote}: ${why}`,
  missing: (branch, where) => `${branch} is not on ${where}`,
  apart: (branch, at) => `${branch} is at different commits: ${at}`,
  noMain: where => `there is no main on ${where}`,
  notForward: (branch, where) => `main ${where} has commits ${branch} lacks, so landing it would not be a fast-forward`,
  wouldRefuse: (remote, why) => `${remote} would not take the push: ${why}`,
  landing: (branch, commit, adds) => (adds.length
    ? `Landing ${branch} at ${commit} brings ${adds.length === 1 ? '1 commit' : `${adds.length} commits`} to main:`
    : `Landing ${branch} at ${commit}, which main here is at already.`),
  steps: 'It will:',
  push: (commit, remote) => `  push ${commit} to main on ${remote}, without force`,
  move: commit => `  move main here to ${commit}, fast-forward only`,
  restart: '  restart the app',
  refused: (remote, why) => `${remote} refused the push, so nothing has changed: ${why}`,
  refusedPartway: (remote, why) => `${remote} refused the push: ${why}`,
  remotesApart: (commit, took, left) => `the remotes are apart: main is at ${commit} on ${took}, but still at ${left}`,
  notMoved: (commit, why) => `the remotes have main at ${commit}, but main here could not move to it: ${why}`,
  finish: branch => `fix that, then finish with: ${FINISH(branch)}`,
  landed: (commit, where) => `main is at ${commit} on ${where}.`,
};

export class Landing {
  #repo;
  #run;

  constructor(repo, run = runGit) {
    this.#repo = repo;
    this.#run = run;
  }

  plan(branch) {
    if (!branch) throw new Error(words.noBranch);
    const current = this.#git(['symbolic-ref', '--quiet', '--short', 'HEAD']).out;
    if (current !== 'main') throw new Error(words.notOnMain(current));
    if (this.#git(['--no-optional-locks', 'status', '--porcelain']).out) throw new Error(words.uncommitted);
    const names = this.#git(['remote']).out.split('\n').filter(Boolean);
    if (!names.length) throw new Error(words.noRemote);
    for (const name of names) {
      const fetched = this.#git(['fetch', '--prune', '--quiet', name]);
      if (!fetched.ok) throw new Error(words.noFetch(name, reason(fetched)));
    }

    const remotes = names.map(name => ({ name, branch: this.#commit(`refs/remotes/${name}/${branch}`), main: this.#commit(`refs/remotes/${name}/main`) }));
    const lacking = remotes.filter(r => !r.branch);
    if (lacking.length) throw new Error(words.missing(branch, list(lacking.map(r => r.name))));
    const commit = remotes[0].branch;
    if (remotes.some(r => r.branch !== commit)) throw new Error(words.apart(branch, remotes.map(r => `${this.#short(r.branch)} on ${r.name}`).join(', ')));
    const mainless = remotes.filter(r => !r.main);
    if (mainless.length) throw new Error(words.noMain(list(mainless.map(r => r.name))));
    const here = this.#commit('refs/heads/main');
    const ahead = [...remotes.filter(r => !this.#behind(r.main, commit)).map(r => `on ${r.name}`), ...(this.#behind(here, commit) ? [] : ['here'])];
    if (ahead.length) throw new Error(words.notForward(branch, list(ahead)));
    for (const remote of remotes.filter(r => r.main !== commit)) {
      const tried = this.#git(['push', '--dry-run', '--porcelain', remote.name, `${commit}:refs/heads/main`]);
      if (!tried.ok) throw new Error(words.wouldRefuse(remote.name, reason(tried)));
    }

    const adds = this.#git(['log', '--format=%h %s', `${here}..${commit}`]).out.split('\n').filter(Boolean);
    return { branch, commit, short: this.#short(commit), adds, remotes: remotes.map(({ name, main }) => ({ name, main })), here };
  }

  land(plan) {
    const pushes = plan.remotes.filter(r => r.main !== plan.commit);
    for (const [i, remote] of pushes.entries()) {
      const pushed = this.#git(['push', '--porcelain', remote.name, `${plan.commit}:refs/heads/main`]);
      if (pushed.ok) continue;
      const left = pushes.slice(i);
      const took = plan.remotes.filter(r => !left.includes(r)).map(r => r.name);
      if (!took.length) throw new Error(words.refused(remote.name, reason(pushed)));
      throw new Error([
        words.refusedPartway(remote.name, reason(pushed)),
        words.remotesApart(plan.short, list(took), left.map(r => `${this.#short(r.main)} on ${r.name}`).join(', ')),
        words.finish(plan.branch),
      ].join('\n'));
    }
    if (plan.here !== plan.commit) {
      const moved = this.#git(['merge', '--ff-only', '--quiet', plan.commit]);
      if (!moved.ok) throw new Error([words.notMoved(plan.short, reason(moved)), words.finish(plan.branch)].join('\n'));
    }
    return words.landed(this.#short(this.#commit('refs/heads/main')), list([...plan.remotes.map(r => r.name), 'here']));
  }

  #git(args) {
    return this.#run(this.#repo, args);
  }

  #commit(ref) {
    const found = this.#git(['rev-parse', '--verify', '--quiet', `${ref}^{commit}`]);
    return found.ok ? found.out : null;
  }

  #short(commit) {
    return this.#git(['rev-parse', '--short', commit]).out;
  }

  #behind(main, commit) {
    return this.#git(['merge-base', '--is-ancestor', main, commit]).ok;
  }
}

export function describe(plan) {
  return [
    words.landing(plan.branch, plan.short, plan.adds),
    ...plan.adds.map(line => `  ${line}`),
    words.steps,
    ...plan.remotes.filter(r => r.main !== plan.commit).map(r => words.push(plan.short, r.name)),
    ...(plan.here === plan.commit ? [] : [words.move(plan.short)]),
    words.restart,
  ];
}

function runGit(repo, args) {
  try {
    return { ok: true, out: execFileSync('git', ['-C', repo, ...args], { encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] }).trim() };
  } catch (error) {
    return { ok: false, out: String(error.stdout || '').trim(), err: String(error.stderr || error.message).trim() };
  }
}

function reason({ out, err }) {
  const rejected = /^!\t[^\t]*\t\[([^\]]*)\](?: \((.*)\))?$/m.exec(out);
  if (!rejected) return plain(err.split('\n')[0] || 'git failed');
  const said = /^remote: (.*\S)/m.exec(err);
  return [rejected[2] || rejected[1], ...(said ? [plain(said[1])] : [])].join(', saying: ');
}

function plain(line) {
  return line.replace(/^(?:fatal|error): /, '');
}

function list(names) {
  return names.length < 2 ? names.join('') : `${names.slice(0, -1).join(', ')} and ${names.at(-1)}`;
}
