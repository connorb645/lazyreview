/**
 * A server's process on this machine, by its process id.
 *
 * alive(pid) says whether the process lives. stop(pid) asks it to end, and
 * waits up to five seconds for it to go.
 */
import { setTimeout as sleep } from 'node:timers/promises';

export class ServerProcess {
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
}
