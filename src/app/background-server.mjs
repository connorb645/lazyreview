/**
 * Finds, starts and stops the app's server running in the background.
 *
 * The server records itself in the data home's server.json while it runs:
 * {pid, address, token}. new BackgroundServer({home, bin, env, processes})
 * starts it as `bin serve`, with the environment `env`. running() resolves
 * with its address when that server answers, or null. start() starts one in
 * the background, logging to server.log, and resolves with its address once
 * it answers. stop() stops the server server.json names, by that process id
 * only, and says whether one was running.
 */
import { rmSync } from 'node:fs';
import { setTimeout as sleep } from 'node:timers/promises';
import { readJSON } from '../store/whole-file.mjs';
import { ServerProcess } from '../commands/os/server-process.mjs';
import { english } from '../i18n/english.mjs';

const START_WAIT = 15_000;

export class BackgroundServer {
  #home;
  #bin;
  #env;
  #processes;

  constructor({ home, bin, env, processes = new ServerProcess() }) {
    this.#home = home;
    this.#bin = bin;
    this.#env = env;
    this.#processes = processes;
  }

  async running() {
    const recorded = readJSON(this.#home.serverFile);
    if (!recorded || !this.#processes.alive(recorded.pid)) return null;
    return (await this.#processes.answers(recorded.address)) ? recorded.address : null;
  }

  async start() {
    this.#processes.start({ bin: this.#bin, env: this.#env, log: this.#home.serverLog });
    const deadline = Date.now() + START_WAIT;
    while (Date.now() < deadline) {
      const address = await this.running();
      if (address) return address;
      await sleep(100);
    }
    throw new Error(english.app.didNotStart(this.#home.serverLog));
  }

  async stop() {
    const recorded = readJSON(this.#home.serverFile);
    if (!recorded || !this.#processes.alive(recorded.pid)) {
      rmSync(this.#home.serverFile, { force: true });
      return false;
    }
    await this.#processes.stop(recorded.pid);
    return true;
  }
}
