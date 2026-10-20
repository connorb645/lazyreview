import { test } from 'node:test';
import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { writeFileSync, chmodSync, unlinkSync, readFileSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { Landing, describe } from '../../../scripts/land/landing.mjs';
import { throwawayRepo } from '../../support/throwaway-repo.mjs';

const git = (repo, ...args) => execFileSync('git', ['-C', repo, ...args], { encoding: 'utf8' }).trim();

// A checkout on main with two remotes, mirror and origin, each with main at its first commit.
function landingRepo() {
  const repo = throwawayRepo();
  const bare = { mirror: join(dirname(repo.origin), 'mirror.git'), origin: repo.origin };
  execFileSync('git', ['init', '-q', '--bare', bare.mirror]);
  repo.git('remote', 'add', 'mirror', bare.mirror);
  repo.git('remote', 'add', 'origin', bare.origin);
  const base = repo.commit('main', { 'README.md': 'widgets\n' }, 'Start');
  const publish = () => { for (const remote of Object.keys(bare)) repo.git('push', '-q', remote, '--all'); };
  publish();
  const hook = remote => join(bare[remote], 'hooks', 'pre-receive');
  return {
    ...repo,
    bare,
    base,
    publish,
    short: commit => repo.git('rev-parse', '--short', commit),
    mains: () => [git(bare.mirror, 'rev-parse', 'main'), git(bare.origin, 'rev-parse', 'main'), repo.sha('main')],
    state: () => ({
      head: repo.git('rev-parse', '--symbolic-full-name', 'HEAD', 'HEAD'),
      here: repo.git('for-each-ref', 'refs/heads'),
      mirror: git(bare.mirror, 'for-each-ref'),
      origin: git(bare.origin, 'for-each-ref'),
    }),
    refuse: remote => { writeFileSync(hook(remote), '#!/bin/sh\necho "error: pushes are closed" >&2\nexit 1\n'); chmodSync(hook(remote), 0o755); },
    allow: remote => unlinkSync(hook(remote)),
  };
}

// The branch feature, one commit on main, on both remotes, with the checkout back on main.
function published(repo) {
  const feature = repo.commit('feature', { 'a.txt': 'a\n' }, 'Add a');
  repo.publish();
  repo.checkout('main');
  return feature;
}

test('a branch lands on main on every remote and here, after saying what it brings and what it will do', () => {
  const repo = landingRepo();
  try {
    const feature = published(repo);
    const short = repo.short(feature);
    const landing = new Landing(repo.work);
    const plan = landing.plan('feature');
    assert.deepEqual(describe(plan), [
      `Landing feature at ${short} brings 1 commit to main:`,
      `  ${short} Add a`,
      'It will:',
      `  push ${short} to main on mirror, without force`,
      `  push ${short} to main on origin, without force`,
      `  move main here to ${short}, fast-forward only`,
    ]);
    assert.equal(landing.land(plan), `main is at ${short} on mirror, origin and here.`);
    assert.deepEqual(repo.mains(), [feature, feature, feature]);
    assert.equal(readFileSync(join(repo.work, 'a.txt'), 'utf8'), 'a\n');
    assert.equal(repo.git('status', '--porcelain'), '');
  } finally {
    repo.done();
  }
});

test('a stack lands by its top branch, bringing every branch below it', () => {
  const repo = landingRepo();
  try {
    const bottom = repo.commit('stack-bottom', { 'a.txt': 'a\n' }, 'Add a');
    const top = repo.commit('stack-top', { 'b.txt': 'b\n' }, 'Add b');
    repo.publish();
    repo.checkout('main');
    const landing = new Landing(repo.work);
    const plan = landing.plan('stack-top');
    assert.deepEqual(describe(plan).slice(0, 3), [
      `Landing stack-top at ${repo.short(top)} brings 2 commits to main:`,
      `  ${repo.short(top)} Add b`,
      `  ${repo.short(bottom)} Add a`,
    ]);
    landing.land(plan);
    assert.deepEqual(repo.mains(), [top, top, top]);
    assert.equal(repo.git('merge-base', '--is-ancestor', bottom, 'main'), '', 'the bottom of the stack is on main too');
  } finally {
    repo.done();
  }
});

const refusals = [
  {
    when: 'no branch is given',
    setup: repo => { published(repo); return { branch: '', says: 'say which branch to land: make land BRANCH=<branch>' }; },
  },
  {
    when: 'the checkout is not on main',
    setup: repo => { published(repo); repo.checkout('feature'); return { branch: 'feature', says: 'this checkout is on feature, not main' }; },
  },
  {
    when: 'the checkout is on no branch',
    setup: repo => { published(repo); repo.git('checkout', '-q', '--detach', 'main'); return { branch: 'feature', says: 'this checkout is on no branch, not main' }; },
  },
  {
    when: 'the checkout has uncommitted changes',
    setup: repo => { published(repo); writeFileSync(join(repo.work, 'notes.txt'), 'later\n'); return { branch: 'feature', says: 'this checkout has uncommitted changes' }; },
  },
  {
    when: 'there is no remote',
    setup: repo => { published(repo); repo.git('remote', 'remove', 'mirror'); repo.git('remote', 'remove', 'origin'); return { branch: 'feature', says: 'this repository has no remote to land on' }; },
  },
  {
    when: 'a remote cannot be fetched',
    setup: repo => {
      published(repo);
      repo.git('remote', 'set-url', 'mirror', join(dirname(repo.origin), 'gone.git'));
      return { branch: 'feature', says: /^could not fetch mirror: '.*gone\.git' does not appear to be a git repository$/ };
    },
  },
  {
    when: 'a remote does not have the branch',
    setup: repo => {
      repo.commit('feature', { 'a.txt': 'a\n' }, 'Add a');
      repo.git('push', '-q', 'origin', 'feature');
      repo.checkout('main');
      return { branch: 'feature', says: 'feature is not on mirror' };
    },
  },
  {
    when: 'two remotes have the branch at different commits',
    setup: repo => {
      const one = repo.commit('feature', { 'a.txt': 'one\n' }, 'Add a');
      repo.git('push', '-q', 'mirror', 'feature');
      repo.checkout('main');
      repo.git('branch', '-q', '-D', 'feature');
      const two = repo.commit('feature', { 'a.txt': 'two\n' }, 'Add a');
      repo.git('push', '-q', 'origin', 'feature');
      repo.checkout('main');
      return { branch: 'feature', says: `feature is at different commits: ${repo.short(one)} on mirror, ${repo.short(two)} on origin` };
    },
  },
  {
    when: 'a remote has no main',
    setup: repo => { published(repo); git(repo.bare.origin, 'update-ref', '-d', 'refs/heads/main'); return { branch: 'feature', says: 'there is no main on origin' }; },
  },
  {
    when: 'main on a remote has commits the branch lacks',
    setup: repo => {
      published(repo);
      repo.commit('elsewhere', { 'b.txt': 'b\n' }, 'Add b');
      repo.git('push', '-q', 'origin', 'elsewhere:main');
      repo.checkout('main');
      return { branch: 'feature', says: 'main on origin has commits feature lacks, so landing it would not be a fast-forward' };
    },
  },
  {
    when: 'main here has commits the branch lacks',
    setup: repo => {
      published(repo);
      repo.commit('main', { 'b.txt': 'b\n' }, 'Add b');
      return { branch: 'feature', says: 'main here has commits feature lacks, so landing it would not be a fast-forward' };
    },
  },
  {
    when: 'a remote will not take a push',
    setup: repo => {
      published(repo);
      repo.git('remote', 'set-url', '--push', 'origin', join(dirname(repo.origin), 'gone.git'));
      return { branch: 'feature', says: /^origin would not take the push: '.*gone\.git' does not appear to be a git repository$/ };
    },
  },
];

for (const { when, setup } of refusals) {
  test(`it refuses with one plain line, and changes nothing, when ${when}`, () => {
    const repo = landingRepo();
    try {
      const { branch, says } = setup(repo);
      const before = repo.state();
      assert.throws(() => new Landing(repo.work).plan(branch), { message: says });
      assert.deepEqual(repo.state(), before);
    } finally {
      repo.done();
    }
  });
}

test('a push the second remote refuses leaves the remotes apart and says how to finish, which running it again does', () => {
  const repo = landingRepo();
  try {
    const feature = published(repo);
    const short = repo.short(feature);
    repo.refuse('origin');
    const landing = new Landing(repo.work);
    assert.throws(() => landing.land(landing.plan('feature')), {
      message: [
        'origin refused the push: pre-receive hook declined, saying: pushes are closed',
        `the remotes are apart: main is at ${short} on mirror, but still at ${repo.short(repo.base)} on origin`,
        'fix that, then finish with: make land BRANCH=feature',
      ].join('\n'),
    });
    assert.deepEqual(repo.mains(), [feature, repo.base, repo.base], 'the push to mirror stays, and main here waits');

    repo.allow('origin');
    const again = landing.plan('feature');
    assert.deepEqual(describe(again).slice(2), [
      'It will:',
      `  push ${short} to main on origin, without force`,
      `  move main here to ${short}, fast-forward only`,
    ]);
    assert.equal(landing.land(again), `main is at ${short} on mirror, origin and here.`);
    assert.deepEqual(repo.mains(), [feature, feature, feature]);
  } finally {
    repo.done();
  }
});

test('a push the first remote refuses changes nothing', () => {
  const repo = landingRepo();
  try {
    published(repo);
    repo.refuse('mirror');
    const landing = new Landing(repo.work);
    const before = repo.state();
    assert.throws(() => landing.land(landing.plan('feature')), { message: 'mirror refused the push, so nothing has changed: pre-receive hook declined, saying: pushes are closed' });
    assert.deepEqual(repo.state(), before);
  } finally {
    repo.done();
  }
});

test('with main at the branch everywhere already, nothing is pushed or moved', () => {
  const repo = landingRepo();
  try {
    const feature = published(repo);
    const short = repo.short(feature);
    const landing = new Landing(repo.work);
    landing.land(landing.plan('feature'));
    const plan = landing.plan('feature');
    assert.deepEqual(describe(plan), [`Landing feature at ${short}, which main here is at already.`]);
    assert.equal(landing.land(plan), `main is at ${short} on mirror, origin and here.`);
  } finally {
    repo.done();
  }
});
