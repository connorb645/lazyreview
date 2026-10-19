import { test } from 'node:test';
import assert from 'node:assert/strict';
import { mkdtempSync, writeFileSync, readFileSync, rmSync, existsSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { readUserConfig } from '../../src/config/user-config.mjs';
import { HiddenRepositories } from '../../src/home/hidden-repositories.mjs';

const NAMES = ['octo/gadgets', 'octo/widgets', 'reviewer-home/notes'];

function folder(env) {
  const dir = mkdtempSync(join(tmpdir(), 'lazyreview-hidden-'));
  const vars = env(dir);
  return { dir, hidden: new HiddenRepositories({ config: () => readUserConfig({ env: vars, home: dir }) }), done: () => rmSync(dir, { recursive: true, force: true }) };
}

test('a change is saved to the file the config was read from, and says what now hides each repository', async () => {
  for (const [name, env, file] of [
    ['the config folder', dir => ({ LAZYREVIEW_CONFIG_DIR: dir }), dir => join(dir, 'config.yml')],
    ['the config file', dir => ({ LAZYREVIEW_CONFIG_FILE: join(dir, 'mine.yml') }), dir => join(dir, 'mine.yml')],
  ]) {
    const f = folder(env);
    try {
      writeFileSync(file(f.dir), '# Mine.\ngithub:\n  accounts: []\n');
      assert.deepEqual(f.hidden.entries(), [], name);
      assert.deepEqual(await f.hidden.change({ hide: 'octo/*', repos: NAMES }), {
        config: file(f.dir),
        repos: [{ name: 'octo/gadgets', hidden_by: ['octo/*'] }, { name: 'octo/widgets', hidden_by: ['octo/*'] }, { name: 'reviewer-home/notes', hidden_by: [] }],
      });
      assert.equal(readFileSync(file(f.dir), 'utf8'), '# Mine.\ngithub:\n  accounts: []\n  hiddenRepositories:\n    - octo/*\n', name);
      await f.hidden.change({ show: ['octo/*'], repos: NAMES });
      assert.deepEqual(f.hidden.entries(), [], name);
    } finally {
      f.done();
    }
  }
});

test('with no config file, the first change makes one holding only the list', async () => {
  const f = folder(dir => ({ LAZYREVIEW_CONFIG_DIR: join(dir, 'lazyreview') }));
  try {
    const answer = await f.hidden.change({ hide: 'reviewer-home/notes', repos: NAMES });
    assert.equal(readFileSync(answer.config, 'utf8'), 'github:\n  hiddenRepositories:\n    - reviewer-home/notes\n');
    assert.deepEqual(answer.repos.map(r => r.hidden_by), [[], [], ['reviewer-home/notes']]);
  } finally {
    f.done();
  }
});

test('changes asked at once are made one at a time, so none is lost', async () => {
  const f = folder(dir => ({ LAZYREVIEW_CONFIG_DIR: dir }));
  try {
    await Promise.all(['octo/gadgets', 'octo/widgets', 'reviewer-home/*'].map(hide => f.hidden.change({ hide, repos: NAMES })));
    assert.deepEqual(f.hidden.entries(), ['octo/gadgets', 'octo/widgets', 'reviewer-home/*']);
  } finally {
    f.done();
  }
});

test('an entry that is not a repository or an owner\'s every repository is refused, and nothing is written', async () => {
  const f = folder(dir => ({ LAZYREVIEW_CONFIG_DIR: dir }));
  try {
    for (const hide of ['octo', '*/widgets', 'octo/wid gets', 'octo/widgets/more']) {
      await assert.rejects(f.hidden.change({ hide, repos: NAMES }), new RegExp(`^Error: "${hide.replace(/[*/]/g, '\\$&')}" is not a repository as owner/name, or owner/\\* for every repository of the owner$`));
    }
    assert.ok(!existsSync(join(f.dir, 'config.yml')));
    writeFileSync(join(f.dir, 'config.yml'), 'github:\n  accounts: 3\n');
    await assert.rejects(f.hidden.change({ hide: 'octo/*', repos: NAMES }), /gives github\.accounts 3, which is not a list of accounts/, 'a config that cannot be read is not changed');
    assert.equal(readFileSync(join(f.dir, 'config.yml'), 'utf8'), 'github:\n  accounts: 3\n');
  } finally {
    f.done();
  }
});
