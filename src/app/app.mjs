/**
 * Starts lazyreview: reads the settings, then builds what a command uses.
 *
 * new App({env, terminal}) reads the settings from `env` and makes the data
 * home. It holds `config`, what it read, `home`, the data home, `store`, the
 * reviews kept there, `terminal`, the terminal app's program, which loads
 * and opens reviews, and `oldServer`, a server an older lazyreview may have
 * left running.
 */
import { mkdirSync } from 'node:fs';
import { readUserConfig } from '../config/user-config.mjs';
import { DataHome } from '../store/data-home.mjs';
import { ReviewStore } from '../store/review-store.mjs';
import { TerminalApp } from '../commands/os/terminal-app.mjs';
import { OldServer } from './old-server.mjs';

export class App {
  constructor({ env = process.env, terminal = new TerminalApp() }) {
    const config = readUserConfig({ env });
    const home = new DataHome(config.values.dataHome);
    mkdirSync(home.root, { recursive: true });
    this.config = config;
    this.home = home;
    this.store = new ReviewStore(home);
    this.terminal = terminal;
    this.oldServer = new OldServer({ home });
    Object.freeze(this);
  }
}
