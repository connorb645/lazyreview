/**
 * Asks Claude one question, through Claude Code run once in print mode as
 * the user's own: their settings, CLAUDE.md, skills, MCP servers and
 * default model, with every tool.
 *
 * new Claude({command, model, spawn}) runs `command` (Claude Code), with
 * `model` over the user's default when given. ask({system, prompt, folder,
 * session, resume, settings, onStep, signal}) runs it in `folder` with
 * `prompt` as the question, `system` added to Claude Code's own
 * instructions, and `settings` over the user's, which they outrank. Nothing
 * that would ask for approval runs, and no settings come from `folder`
 * itself, as the code there may be anyone's. The conversation is saved as
 * `session`, a UUID, or goes on from it when `resume`. onStep({id, tool,
 * input}) hears each tool call as Claude makes it, and onStep({id, refused})
 * each one refused, with why: the guard's words after its mark, that it
 * needed approval, or that the sandbox stopped it, even part of a command
 * that went on. Aborting `signal` stops Claude Code and everything it
 * started. It resolves with the answer's text, or throws an error saying why
 * there is none; one whose `noSession` is true says the session to go on
 * from was not found. Each call has a time limit.
 */
import { spawn as spawnProcess } from 'node:child_process';
import { english } from '../../i18n/english.mjs';

const TIMEOUT = 30 * 60_000;
const GRACE = 3000;

export class Claude {
  #command;
  #model;
  #spawn;

  constructor({ command = 'claude', model = null, spawn = spawnProcess } = {}) {
    this.#command = command;
    this.#model = model;
    this.#spawn = spawn;
  }

  ask({ system, prompt, folder, session, resume = false, settings, onStep = () => {}, signal }) {
    const args = ['-p', '--output-format', 'stream-json', '--verbose', '--setting-sources', 'user', '--settings', JSON.stringify(settings),
      '--permission-mode', 'default', '--permission-prompts', 'none', '--append-system-prompt', system,
      ...(resume ? ['--resume', session] : ['--session-id', session]), ...(this.#model ? ['--model', this.#model] : [])];
    return new Promise((resolve, reject) => {
      const child = this.#spawn(this.#command, args, { cwd: folder, detached: true, stdio: ['pipe', 'pipe', 'pipe'] });
      let buffered = '', stderr = '', result = null, why = null, settled = false;
      const stop = reason => {
        if (why || child.exitCode !== null) return;
        why = reason;
        killGroup(child, 'SIGTERM');
        setTimeout(() => killGroup(child, 'SIGKILL'), GRACE).unref();
      };
      const timer = setTimeout(() => stop('slow'), TIMEOUT);
      const aborted = () => stop('stopped');
      signal?.addEventListener('abort', aborted, { once: true });
      if (signal?.aborted) aborted();
      const finish = error => {
        if (settled) return;
        settled = true;
        clearTimeout(timer);
        signal?.removeEventListener('abort', aborted);
        if (error) reject(error); else resolve(result.result.trim());
      };
      child.stdout.setEncoding('utf8');
      child.stdout.on('data', text => {
        buffered += text;
        const lines = buffered.split('\n');
        buffered = lines.pop();
        for (const line of lines) {
          const event = readEvent(line);
          if (event?.type === 'result') result = event;
          else if (event) heard(event, onStep);
        }
      });
      child.stderr.setEncoding('utf8');
      child.stderr.on('data', text => { stderr += text; });
      child.once('error', error => finish(new Error(english.claude.noAnswer(error.code === 'ENOENT' ? english.claude.notFound(this.#command) : error.message))));
      child.once('close', code => {
        const tail = buffered && readEvent(buffered);
        if (tail?.type === 'result') result = tail;
        if (why === 'stopped') return finish(new Error(english.claude.stopped));
        if (why === 'slow') return finish(new Error(english.claude.tooSlow(TIMEOUT / 60_000)));
        if (!result) {
          const last = stderr.trim().split('\n').pop();
          const error = new Error(english.claude.noAnswer(last || english.claude.exitCode(code)));
          error.noSession = /no conversation found/i.test(stderr);
          return finish(error);
        }
        if (result.subtype === 'error_max_turns') return finish(new Error(english.claude.tooManyTurns(result.num_turns)));
        if (result.is_error || typeof result.result !== 'string') {
          const error = new Error(english.claude.couldNotAnswer(result.result || (result.errors || []).join('; ') || result.subtype || english.claude.nothing));
          error.noSession = /no conversation found/i.test(error.message);
          return finish(error);
        }
        finish(null);
      });
      child.stdin.on('error', () => {});
      child.stdin.end(prompt);
    });
  }
}

function heard(event, onStep) {
  for (const block of event.message?.content || []) {
    if (event.type === 'assistant' && block.type === 'tool_use') onStep({ id: block.id, tool: block.name, input: block.input || {} });
    if (event.type === 'user' && block.type === 'tool_result') {
      const text = [block.content].flat().map(c => (typeof c === 'string' ? c : c?.text || '')).join('\n');
      const at = text.indexOf(english.guard.marker);
      if (block.is_error && at >= 0) onStep({ id: block.tool_use_id, refused: text.slice(at + english.guard.marker.length).replace(/^[:,]\s*/, '').trim() });
      else if (block.is_error && /permission .*denied|was denied/i.test(text)) onStep({ id: block.tool_use_id, refused: english.claude.notApproved });
      else if (/<sandbox_violations>|operation not permitted/i.test(text)) onStep({ id: block.tool_use_id, refused: english.claude.sandboxed });
    }
  }
}

function readEvent(line) {
  try {
    return JSON.parse(line);
  } catch {
    return null;
  }
}

function killGroup(child, signal) {
  try {
    process.kill(-child.pid, signal);
  } catch {
    child.kill(signal);
  }
}
