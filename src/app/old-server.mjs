/**
 * A server an older lazyreview left running, which recorded itself in the
 * data home's server.json as {pid, address, token}.
 *
 * new OldServer({home, processes}) names the data home. stop() stops that
 * server, by the process id it recorded only, removes its record, and
 * resolves with whether one was running.
 */
import { rmSync } from 'node:fs';
import { readJSON } from '../store/whole-file.mjs';
import { ServerProcess } from '../commands/os/server-process.mjs';

export class OldServer {
  #home;
  #processes;

  constructor({ home, processes = new ServerProcess() }) {
    this.#home = home;
    this.#processes = processes;
  }

  async stop() {
    const recorded = readJSON(this.#home.serverFile);
    const running = !!recorded && this.#processes.alive(recorded.pid);
    if (running) await this.#processes.stop(recorded.pid);
    rmSync(this.#home.serverFile, { force: true });
    return running;
  }
}
