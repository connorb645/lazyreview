/**
 * The terminal app: a Go program in the repository's terminal folder, built
 * on this machine the first time it runs and again whenever its code
 * changes, then run in this terminal.
 *
 * new TerminalApp({folder, run}) names the folder; `run` runs a command and
 * resolves {code, stderr}. launch({serverFile, review, env, say}) builds the
 * app when its program is missing or older than any of its code, saying so
 * first with `say`, then runs it on this terminal, telling it where the
 * server records its address and token, and which review to open, or none
 * for the home screen; it reads the config itself, from `env`. It resolves
 * once the app quits, and throws when it cannot be built or fails.
 */
import { spawn, execFile } from 'node:child_process';
import { readdirSync, statSync, existsSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { english } from '../../i18n/english.mjs';

const FOLDER = join(dirname(fileURLToPath(import.meta.url)), '..', '..', '..', 'terminal');
const BUILD_TIMEOUT = 10 * 60_000;

export class TerminalApp {
  #folder;
  #run;

  constructor({ folder = FOLDER, run = runCommand } = {}) {
    this.#folder = folder;
    this.#run = run;
  }

  get program() {
    return join(this.#folder, 'bin', 'lazyreview-terminal');
  }

  async launch({ serverFile, review, env, say = () => {} }) {
    if (this.stale()) {
      say(english.terminalApp.building);
      await this.#build(env);
    }
    const code = await new Promise((resolve, reject) => {
      const child = spawn(this.program, ['--server-file', serverFile, ...(review ? [review] : [])], { stdio: 'inherit', env });
      child.once('error', reject);
      child.once('exit', (exit, signal) => resolve(signal ? 1 : exit));
    });
    if (code !== 0) throw new Error(english.terminalApp.stopped(code));
  }

  /** Whether the program is missing, or older than any of its code. */
  stale() {
    if (!existsSync(this.program)) return true;
    const built = statSync(this.program).mtimeMs;
    return sources(this.#folder).some(path => statSync(path).mtimeMs > built);
  }

  async #build(env) {
    const done = await this.#run('go', ['build', '-mod=vendor', '-o', this.program, '.'], { cwd: this.#folder, env, timeout: BUILD_TIMEOUT });
    if (done.code === 'ENOENT') throw new Error(english.terminalApp.needsGo);
    if (done.code !== 0) throw new Error(english.terminalApp.buildFailed(done.stderr.trim()));
  }
}

// The files a build reads: the module's own Go code, and what pins its libraries.
function sources(folder) {
  const out = ['go.mod', 'go.sum', 'vendor/modules.txt'].map(name => join(folder, name)).filter(existsSync);
  const walk = dir => {
    for (const entry of readdirSync(dir, { withFileTypes: true })) {
      const path = join(dir, entry.name);
      if (entry.isDirectory()) { if (!['bin', 'vendor', 'testdata'].includes(entry.name)) walk(path); continue; }
      if (entry.name.endsWith('.go') && !entry.name.endsWith('_test.go')) out.push(path);
    }
  };
  walk(folder);
  return out;
}

function runCommand(command, args, { cwd, env, timeout }) {
  return new Promise(resolve => {
    execFile(command, args, { cwd, env, timeout, maxBuffer: 1 << 24 }, (error, stdout, stderr) => {
      resolve({ code: error ? error.code ?? 1 : 0, stderr: String(stderr || (error ? error.message : '')) });
    });
  });
}
