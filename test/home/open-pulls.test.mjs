import { test } from 'node:test';
import assert from 'node:assert/strict';
import { writeFileSync } from 'node:fs';
import { workbench } from '../support/workbench.mjs';
import { SETTINGS, homeRoutes, accountsConfig } from '../support/home-routes.mjs';
import { OpenPulls } from '../../src/home/open-pulls.mjs';

const UNREACHABLE = 'error connecting to api.github.com\ncheck your internet connection or https://githubstatus.com';

// Each unit of a repository as its review and its members, each with what involves the accounts in it.
const units = repo => repo.units.map(u => ({
  review: u.review,
  members: u.members.map(m => [m.number, ...(m.asked.length ? [`asked of ${m.asked.join(', ')}`] : []), ...(m.yours ? ['yours'] : []),
    ...(m.commented ? ['commented'] : []), ...(m.draft ? ['draft'] : []), ...(m.review ? [`in ${m.review}`] : [])].join(' ')),
}));

function bench(extra = {}) {
  const w = workbench();
  writeFileSync(w.configFile, accountsConfig(SETTINGS.accounts));
  w.routes({ extra: { ...homeRoutes(), ...extra } });
  return w;
}

test('the home screen lists the open pull requests that involve each account, by repository, each account read by its own token', async () => {
  const w = bench();
  try {
    await w.loader.load({ links: [w.link(21), w.link(22)] });
    const list = await new OpenPulls({ store: w.store, config: w.config }).list();
    assert.equal(list.config, w.configFile);
    assert.deepEqual(list.accounts, [{ login: 'reviewer', error: null }, { login: 'reviewer-home', error: null }]);
    assert.deepEqual(list.repos.map(r => [r.host, r.name, r.account]), [
      ['github.com', 'octo/gadgets', 'reviewer'], ['github.com', 'octo/widgets', 'reviewer'], ['github.com', 'reviewer-home/notes', 'reviewer-home'],
    ]);
    assert.deepEqual(list.repos.map(units), [
      [{ review: null, members: ['7 asked of octo/reviewers'] }, { review: null, members: ['9 commented'] }],
      [
        { review: 'github.com/octo/widgets/21', members: ['21 in github.com/octo/widgets/21', '22 asked of reviewer in github.com/octo/widgets/21'] },
        { review: null, members: ['30 commented'] },
      ],
      [{ review: null, members: ['3 yours', '4 yours draft'] }],
    ], 'a stack comes whole, bottom first, with the member that involves no one, and a draft is listed but asks for no review');
    assert.deepEqual(list.repos[0].units[0].members[0], {
      number: 7, title: 'Count the gadgets in stock', author: 'kim', url: 'https://github.com/octo/gadgets/pull/7', draft: false, base: 'main', head: 'kim/stock',
      description: 'Counts each gadget once.\n\n- Adds a stock count\n- Shows it on the shelf', asked: ['octo/reviewers'], yours: false, commented: false, review: null,
    });
    const asked = w.gh.calls().filter(c => c.path.startsWith('search/') || c.path.includes('/pulls?state=open'));
    const own = { 'octo/gadgets': 'reviewer', 'octo/widgets': 'reviewer', 'reviewer-home/notes': 'reviewer-home' };
    for (const call of asked) {
      const name = /^repos\/([^/]+\/[^/]+)\//.exec(call.path)?.[1];
      if (name) assert.equal(call.token, `token-${own[name]}`, call.path);
      else assert.match(call.token, /^token-reviewer(-home)?$/, call.path);
    }
    assert.equal(asked.filter(c => c.path.startsWith('search/')).length, 9, 'four searches an account, and one more for each team');
  } finally {
    w.done();
  }
});

test('a stack counts as loaded only when one review holds every pull request of it', async () => {
  const w = bench();
  try {
    await w.loader.load({ links: [w.link(21)] });
    const widgets = list => units(list.repos.find(r => r.name === 'octo/widgets'))[0];
    assert.deepEqual(widgets(await new OpenPulls({ store: w.store, config: w.config }).list()), {
      review: null, members: ['21 in github.com/octo/widgets/21', '22 asked of reviewer'],
    });
    await w.loader.load({ links: [w.link(21), w.link(22)] });
    assert.equal(widgets(await new OpenPulls({ store: w.store, config: w.config }).list()).review, 'github.com/octo/widgets/21');
  } finally {
    w.done();
  }
});

test('an account GitHub cannot be reached as is named with why, and the others are still listed', async () => {
  const w = bench({ 'token-reviewer-home *': { $fail: UNREACHABLE } });
  try {
    const list = await new OpenPulls({ store: w.store, config: w.config }).list();
    assert.deepEqual(list.accounts, [{ login: 'reviewer', error: null }, { login: 'reviewer-home', error: 'could not reach github.com: error connecting to api.github.com' }]);
    assert.deepEqual(list.repos.map(r => r.name), ['octo/gadgets', 'octo/widgets']);
  } finally {
    w.done();
  }
});

test('with no config there are no accounts to list, a config that is wrong says where and why, and settings.json says how to move it', async () => {
  const w = workbench();
  const quoted = text => text.replace(/[.*+?^${}()|[\]\\/]/g, '\\$&');
  try {
    assert.deepEqual(await new OpenPulls({ store: w.store, config: w.config }).list(), { config: w.configFile, accounts: [], repos: [] });
    writeFileSync(w.configFile, accountsConfig([{ login: 'reviewer author:sam' }]));
    await assert.rejects(new OpenPulls({ store: w.store, config: w.config }).list(), new RegExp(`^Error: ${quoted(w.configFile)} gives account 1 no GitHub login$`));
    writeFileSync(w.configFile, accountsConfig(SETTINGS.accounts));
    writeFileSync(w.home.settingsFile, JSON.stringify(SETTINGS));
    await assert.rejects(new OpenPulls({ store: w.store, config: w.config }).list(),
      new RegExp(`^Error: ${quoted(w.home.settingsFile)} is no longer read; lazyreview config move moves its accounts into ${quoted(w.configFile)}$`));
  } finally {
    w.done();
  }
});
