/**
 * The answers the terminal app runs, each in a process of its own that holds
 * its conversation's answering lock, threads/<n>.answering, while it works.
 *
 * stopAnswers(home, {wait}) asks every answer still running under the data
 * home `home` to end, as a stop from the app does, by the stop file in its
 * lock, and resolves with how many it asked once each has ended, or `wait`
 * milliseconds have passed. It signals no process.
 */
import { existsSync, readdirSync, readFileSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { setTimeout as sleep } from 'node:timers/promises';

const ANSWERING = /^\d+\.answering$/;

export async function stopAnswers(home, { wait = 10_000 } = {}) {
  const asked = running(home.reviews);
  for (const lock of asked) writeFileSync(join(lock, 'stop'), '');
  const deadline = Date.now() + wait;
  while (asked.some(held) && Date.now() < deadline) await sleep(50);
  return asked.length;
}

function running(folder) {
  if (!existsSync(folder)) return [];
  const out = [];
  for (const entry of readdirSync(folder, { withFileTypes: true })) {
    if (!entry.isDirectory()) continue;
    const path = join(folder, entry.name);
    if (ANSWERING.test(entry.name)) {
      if (held(path)) out.push(path);
    } else out.push(...running(path));
  }
  return out;
}

function held(lock) {
  let pid;
  try {
    pid = Number(readFileSync(join(lock, 'pid'), 'utf8').trim());
  } catch {
    return false;
  }
  if (!pid) return false;
  try {
    process.kill(pid, 0);
    return true;
  } catch (error) {
    return error.code === 'EPERM';
  }
}
