/**
 * Asks Claude one question, through Claude Code run once in print mode.
 *
 * new Claude({command, model, folder}) runs `command` (Claude Code) with no
 * tools, no settings, no MCP servers and no saved session, so each question
 * is one short call that carries only the message it is given; `folder` is
 * where it runs. ask({system, prompt}) returns the answer's text, or throws
 * an error saying why there is none. Each call has a time limit.
 */
import { execFile } from 'node:child_process';
import { english } from '../../i18n/english.mjs';

const TIMEOUT = 180_000;

export class Claude {
  #command;
  #model;
  #folder;
  #run;

  constructor({ command = 'claude', model = null, folder = process.cwd(), run = runCommand } = {}) {
    this.#command = command;
    this.#model = model;
    this.#folder = folder;
    this.#run = run;
  }

  async ask({ system, prompt }) {
    const args = ['-p', '--output-format', 'json', '--tools', '', '--strict-mcp-config', '--setting-sources', '',
      '--disable-slash-commands', '--no-session-persistence', '--system-prompt', system, ...(this.#model ? ['--model', this.#model] : [])];
    const done = await this.#run(this.#command, args, { input: prompt, cwd: this.#folder });
    if (done.timedOut) throw new Error(english.claude.tooSlow(TIMEOUT / 1000));
    let answer;
    try {
      answer = JSON.parse(done.stdout);
    } catch {
      throw new Error(english.claude.noAnswer((done.stderr || done.stdout).trim().split('\n').pop() || english.claude.exitCode(done.code)));
    }
    if (answer.is_error || typeof answer.result !== 'string') throw new Error(english.claude.couldNotAnswer(answer.result || answer.subtype || english.claude.nothing));
    return answer.result.trim();
  }
}

function runCommand(command, args, { input, cwd }) {
  return new Promise(done => {
    const child = execFile(command, args, { cwd, timeout: TIMEOUT, maxBuffer: 1 << 24 }, (error, stdout, stderr) => {
      const missing = error && error.code === 'ENOENT' ? english.claude.notFound(command) : '';
      done({ code: error ? (typeof error.code === 'number' ? error.code : 1) : 0, stdout: String(stdout), stderr: String(stderr || missing), timedOut: !!(error && error.killed) });
    });
    child.stdin.end(input);
  });
}
