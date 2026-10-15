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
    if (done.timedOut) throw new Error(`Claude did not answer within ${TIMEOUT / 1000} seconds`);
    let answer;
    try {
      answer = JSON.parse(done.stdout);
    } catch {
      throw new Error(`Claude Code did not answer: ${(done.stderr || done.stdout).trim().split('\n').pop() || `exit ${done.code}`}`);
    }
    if (answer.is_error || typeof answer.result !== 'string') throw new Error(`Claude could not answer: ${answer.result || answer.subtype || 'no answer'}`);
    return answer.result.trim();
  }
}

function runCommand(command, args, { input, cwd }) {
  return new Promise(done => {
    const child = execFile(command, args, { cwd, timeout: TIMEOUT, maxBuffer: 1 << 24 }, (error, stdout, stderr) => {
      done({ code: error ? (typeof error.code === 'number' ? error.code : 1) : 0, stdout: String(stdout), stderr: String(stderr || (error && !stdout ? error.message : '')), timedOut: !!(error && error.killed) });
    });
    child.stdin.end(input);
  });
}
