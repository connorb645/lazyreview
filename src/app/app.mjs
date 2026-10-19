/**
 * Starts lazyreview: reads the settings, then builds every service once.
 *
 * new App({env, bin, terminal}) reads the settings from `env`, makes the data
 * home, and builds the services a command uses: `store`, `loader` and
 * `branches`, which load pull requests and branches, `queue`, `threads`,
 * `pulls`, `hidden`, the repositories the home screens leave out,
 * `background`, the app's server running in the background, started as
 * `bin serve`, and `terminal`, the terminal app to launch. `settings` and
 * `home` are what it read.
 *
 * serve() runs the app's server in this process until it is stopped, and
 * resolves at once with the line saying where. The server records its
 * address and a token in server.json, which only its owner can read; a
 * client such as the terminal app sends the token with each change. When
 * stopped, it stops Claude on every conversation it is answering first.
 */
import { mkdirSync, rmSync } from 'node:fs';
import { randomBytes } from 'node:crypto';
import { readUserConfig } from '../config/user-config.mjs';
import { DataHome } from '../store/data-home.mjs';
import { ReviewStore } from '../store/review-store.mjs';
import { writeWhole, readJSON } from '../store/whole-file.mjs';
import { Loader } from '../loader/loader.mjs';
import { BranchLoader } from '../loader/branch-loader.mjs';
import { QueueService } from '../queue/queue-service.mjs';
import { ThreadService } from '../ask/thread-service.mjs';
import { OpenPulls } from '../home/open-pulls.mjs';
import { HiddenRepositories } from '../home/hidden-repositories.mjs';
import { Claude } from '../commands/claude/claude.mjs';
import { TerminalApp } from '../commands/os/terminal-app.mjs';
import { AppServer } from '../server/app-server.mjs';
import { BackgroundServer } from './background-server.mjs';
import { english } from '../i18n/english.mjs';

export class App {
  constructor({ env = process.env, bin, terminal = new TerminalApp() }) {
    const config = readUserConfig({ env });
    const { values } = config;
    const home = new DataHome(values.dataHome);
    mkdirSync(home.root, { recursive: true });
    const store = new ReviewStore(home);
    this.config = config;
    this.home = home;
    this.store = store;
    this.loader = new Loader({ store });
    this.branches = new BranchLoader({ store });
    this.queue = new QueueService({ store });
    this.threads = new ThreadService({ store, claude: new Claude({ command: values.claude.command, model: values.claude.model || null }) });
    this.pulls = new OpenPulls({ store, config: () => readUserConfig({ env }) });
    this.hidden = new HiddenRepositories({ config: () => readUserConfig({ env }) });
    this.background = new BackgroundServer({ home, bin, env });
    this.terminal = terminal;
    Object.freeze(this);
  }

  async serve() {
    if (await this.background.running()) throw new Error(english.app.runningAlready);
    const token = randomBytes(32).toString('hex');
    const server = new AppServer({
      store: this.store,
      loader: this.loader,
      branches: this.branches,
      queue: this.queue,
      threads: this.threads,
      pulls: this.pulls,
      hidden: this.hidden,
      token,
      port: this.config.values.server.port,
    });
    const address = await server.listen();
    const serverFile = this.home.serverFile;
    writeWhole(serverFile, JSON.stringify({ pid: process.pid, address, token }) + '\n', { mode: 0o600 });
    const shut = async () => {
      try {
        const recorded = readJSON(serverFile);
        if (recorded?.pid === process.pid) rmSync(serverFile, { force: true });
      } catch (error) {
        console.error(english.cli.failed(error.message));
      } finally {
        await this.threads.stopAll();
        await server.close();
        process.exit(0);
      }
    };
    process.once('SIGINT', shut);
    process.once('SIGTERM', shut);
    return [english.app.running(address)];
  }
}
