import { test } from 'node:test';
import assert from 'node:assert/strict';
import { mkdtempSync, writeFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { readAccounts } from '../../src/settings/settings.mjs';

function settings(text) {
  const dir = mkdtempSync(join(tmpdir(), 'review-desk-settings-'));
  const file = join(dir, 'settings.json');
  if (text !== undefined) writeFileSync(file, typeof text === 'string' ? text : JSON.stringify(text));
  return { file, done: () => rmSync(dir, { recursive: true, force: true }) };
}

test('the accounts come from the settings file, each with its teams and the date each counts from', () => {
  const s = settings({ accounts: [{ login: 'sam-at-work', teams: [{ name: 'example-org/mobile', since: '2026-10-01' }, { name: 'example-org/web' }] }, { login: 'sam' }] });
  try {
    assert.deepEqual(readAccounts(s.file), [
      { login: 'sam-at-work', teams: [{ name: 'example-org/mobile', since: '2026-10-01' }, { name: 'example-org/web', since: null }] },
      { login: 'sam', teams: [] },
    ]);
  } finally {
    s.done();
  }
});

test('with no settings file there are no accounts, and a wrong one says what is wrong in it', () => {
  const none = settings();
  try {
    assert.deepEqual(readAccounts(none.file), []);
  } finally {
    none.done();
  }
  const wrong = [
    ['{"accounts": [', /could not be read: /],
    [{ login: 'sam' }, /needs "accounts", a list of \{"login": "\.\.\."\}$/],
    [{ accounts: [{ login: 'sam author:alex' }] }, /gives account 1 no GitHub login$/],
    [{ accounts: [{ login: 'sam' }, { login: 'sam' }] }, /names sam twice$/],
    [{ accounts: [{ login: 'sam', teams: 'mobile' }] }, /gives sam "teams" that are not a list$/],
    [{ accounts: [{ login: 'sam', teams: [{ name: 'mobile' }] }] }, /gives sam a team that is not named as organisation\/team$/],
    [{ accounts: [{ login: 'sam', teams: [{ name: 'example-org/mobile', since: 'last week' }] }] }, /gives example-org\/mobile a "since" that is not a date such as 2026-10-01$/],
  ];
  for (const [text, error] of wrong) {
    const s = settings(text);
    try {
      assert.throws(() => readAccounts(s.file), error, JSON.stringify(text));
      assert.throws(() => readAccounts(s.file), new RegExp(`^Error: ${s.file.replace(/[.*+?^${}()|[\]\\/]/g, '\\$&')} `));
    } finally {
      s.done();
    }
  }
});
