/**
 * Finds, starts and stops the app's server running in the background.
 *
 * The server records itself in the data home's server.json while it runs:
 * {pid, address}. running(home) returns its address when that server
 * answers, or null. start({home, bin, env}) starts one in the background,
 * logging to server.log, and resolves with its address once it answers.
 * stop(home) stops the server server.json names, by that process id only,
 * and says whether one was running.
 */
import { spawn } from 'node:child_process';
import { openSync, readFileSync, rmSync, closeSync } from 'node:fs';
import { setTimeout as sleep } from 'node:timers/promises';

const START_WAIT = 15_000;

export async function running(home) {
  const recorded = readRecord(home);
  if (!recorded || !alive(recorded.pid)) return null;
  return (await answers(recorded.address)) ? recorded.address : null;
}

export async function start({ home, bin, env }) {
  const log = openSync(home.serverLog, 'a');
  try {
    const child = spawn(process.execPath, [bin, 'serve'], { detached: true, stdio: ['ignore', log, log], env });
    child.unref();
  } finally {
    closeSync(log);
  }
  const deadline = Date.now() + START_WAIT;
  while (Date.now() < deadline) {
    const address = await running(home);
    if (address) return address;
    await sleep(100);
  }
  throw new Error(`the server did not start; see ${home.serverLog}`);
}

export async function stop(home) {
  const recorded = readRecord(home);
  if (!recorded || !alive(recorded.pid)) {
    rmSync(home.serverFile, { force: true });
    return false;
  }
  process.kill(recorded.pid, 'SIGTERM');
  for (let i = 0; i < 50 && alive(recorded.pid); i++) await sleep(100);
  return true;
}

function readRecord(home) {
  try {
    return JSON.parse(readFileSync(home.serverFile, 'utf8'));
  } catch (error) {
    if (error.code === 'ENOENT') return null;
    throw error;
  }
}

function alive(pid) {
  try {
    process.kill(pid, 0);
    return true;
  } catch (error) {
    return error.code === 'EPERM';
  }
}

async function answers(address) {
  try {
    const response = await fetch(`${address}/api/health`, { signal: AbortSignal.timeout(2000) });
    return response.ok;
  } catch {
    return false;
  }
}
