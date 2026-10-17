/**
 * The app's server as a process of this machine.
 *
 * start({bin, env, log}) starts `bin serve` in the background, apart from
 * this process, adding its output to the file `log`. alive(pid) says
 * whether a process lives. stop(pid) asks that process to end, and waits up
 * to five seconds for it to go. answers(address) says whether a server
 * answers at `address` within two seconds.
 */
import { spawn } from 'node:child_process';
import { openSync, closeSync } from 'node:fs';
import { setTimeout as sleep } from 'node:timers/promises';

export class ServerProcess {
  start({ bin, env, log }) {
    const out = openSync(log, 'a');
    try {
      const child = spawn(process.execPath, [bin, 'serve'], { detached: true, stdio: ['ignore', out, out], env });
      child.unref();
    } finally {
      closeSync(out);
    }
  }

  alive(pid) {
    try {
      process.kill(pid, 0);
      return true;
    } catch (error) {
      return error.code === 'EPERM';
    }
  }

  async stop(pid) {
    process.kill(pid, 'SIGTERM');
    for (let i = 0; i < 50 && this.alive(pid); i++) await sleep(100);
  }

  async answers(address) {
    try {
      const response = await fetch(`${address}/api/health`, { signal: AbortSignal.timeout(2000) });
      return response.ok;
    } catch {
      return false;
    }
  }
}
