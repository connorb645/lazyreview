import { test } from 'node:test';
import assert from 'node:assert/strict';
import { mkdtempSync, mkdirSync, writeFileSync, existsSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { stopAnswers } from '../../src/ask/answering.mjs';
import { DataHome } from '../../src/store/data-home.mjs';

test('stopping asks every answer still running to end through its lock, and waits for it, leaving a lock whose holder is gone', async () => {
  const root = mkdtempSync(join(tmpdir(), 'lazyreview-answering-'));
  try {
    const home = new DataHome(root);
    const lock = (path, pid) => {
      mkdirSync(join(root, 'reviews', path), { recursive: true });
      writeFileSync(join(root, 'reviews', path, 'pid'), String(pid));
      return join(root, 'reviews', path);
    };
    const running = lock('github.com/octo/widgets/21/threads/2.answering', process.pid);
    const gone = lock('local/work-1a2b3c4d/fix~x/threads/1.answering', 2 ** 22 + 7);
    const ended = setInterval(() => {
      if (existsSync(join(running, 'stop'))) rmSync(running, { recursive: true, force: true });
    }, 20);
    try {
      assert.equal(await stopAnswers(home, { wait: 5000 }), 1);
    } finally {
      clearInterval(ended);
    }
    assert.equal(existsSync(running), false, 'the answer ended');
    assert.equal(existsSync(join(gone, 'stop')), false, 'a lock with no holder running is not asked');
    assert.equal(await stopAnswers(new DataHome(join(root, 'none'))), 0);
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});
