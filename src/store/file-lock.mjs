/**
 * Holds a lock on a path while work runs, across processes and within one.
 *
 * withLocks(paths, work) takes each lock in sorted order, so two callers
 * wanting the same locks never wait on each other, runs `work`, and lets the
 * locks go whether it succeeds or throws. A lock is a folder made at the
 * path, holding the process id of its holder; a lock whose holder has died,
 * or that has stood without a holder for ten seconds, is taken over. It
 * waits up to `wait` milliseconds for a held lock, then throws.
 */
import { mkdirSync, rmSync, writeFileSync, readFileSync, statSync } from 'node:fs';
import { join } from 'node:path';
import { setTimeout as sleep } from 'node:timers/promises';

const held = new Map();

export async function withLocks(paths, work, { wait = 30_000 } = {}) {
  const sorted = [...new Set(paths)].sort();
  const taken = [];
  try {
    for (const path of sorted) {
      await take(path, wait);
      taken.push(path);
    }
    return await work();
  } finally {
    for (const path of taken.reverse()) release(path);
  }
}

async function take(path, wait) {
  const deadline = Date.now() + wait;
  while (held.has(path)) {
    if (Date.now() > deadline) throw new Error(`gave up waiting for ${path}`);
    await held.get(path).promise;
  }
  const entry = {};
  entry.promise = new Promise(done => { entry.done = done; });
  held.set(path, entry);
  try {
    for (;;) {
      try {
        mkdirSync(path);
        writeFileSync(join(path, 'pid'), String(process.pid));
        return;
      } catch (error) {
        if (error.code !== 'EEXIST') throw error;
        if (!holderAlive(path)) { rmSync(path, { recursive: true, force: true }); continue; }
        if (Date.now() > deadline) throw new Error(`another process holds ${path}`);
        await sleep(50);
      }
    }
  } catch (error) {
    held.delete(path);
    entry.done();
    throw error;
  }
}

function release(path) {
  rmSync(path, { recursive: true, force: true });
  const entry = held.get(path);
  held.delete(path);
  entry?.done();
}

function holderAlive(path) {
  let pid;
  try {
    pid = Number(readFileSync(join(path, 'pid'), 'utf8'));
  } catch {
    return youngerThan(path, 10_000);
  }
  if (!pid) return youngerThan(path, 10_000);
  try {
    process.kill(pid, 0);
    return true;
  } catch (error) {
    return error.code === 'EPERM';
  }
}

function youngerThan(path, age) {
  try {
    return Date.now() - statSync(path).mtimeMs < age;
  } catch {
    return false;
  }
}
