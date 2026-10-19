/**
 * A stand-in for Claude Code, so tests ask no real model.
 *
 * fakeClaude() makes an executable that answers one question in print mode
 * with stream-json output, the way Claude Code does, and returns {command,
 * asked, answer, done}. `command` is its path. `asked()` reads back each call
 * as {args, prompt, folder, files, git}: `folder` is the folder it ran in,
 * `files` every file it could read there but .git, {path: text}, and `git`
 * the folder's branch, its branches and its remotes. `answer(text)` sets what
 * it says next; `answer({text, steps, wait})` also takes `steps`, each
 * {tool, input, refused}, and waits `wait` milliseconds first, until it is
 * stopped; `answer(null)` makes it fail, and any other object is printed as
 * the result, failing, as Claude Code does when it stops early. A session it
 * has not saved cannot be resumed. `done()` removes it.
 */
import { mkdtempSync, writeFileSync, readFileSync, existsSync, rmSync, chmodSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

export function fakeClaude() {
  const dir = mkdtempSync(join(tmpdir(), 'lazyreview-claude-'));
  const command = join(dir, 'claude');
  const log = join(dir, 'asked.jsonl');
  const answerFile = join(dir, 'answer.json');
  const sessions = join(dir, 'sessions.txt');
  writeFileSync(answerFile, JSON.stringify('It returns early when the list is empty.'));
  writeFileSync(command, `#!/usr/bin/env node
const fs = require('fs');
const path = require('path');
const { execFileSync } = require('child_process');
const args = process.argv.slice(2);
const prompt = fs.readFileSync(0, 'utf8');
const files = {};
(function read(folder, at) {
  for (const entry of fs.readdirSync(folder, { withFileTypes: true })) {
    if (entry.name === '.git') continue;
    const full = path.join(folder, entry.name), name = at + entry.name;
    if (entry.isDirectory()) read(full, name + '/');
    else files[name] = fs.readFileSync(full, 'utf8');
  }
})(process.cwd(), '');
const git = (...a) => { try { return execFileSync('git', a, { encoding: 'utf8' }).trim(); } catch { return null; } };
fs.appendFileSync(${JSON.stringify(log)}, JSON.stringify({ args, prompt, folder: process.cwd(), files,
  git: { branch: git('branch', '--show-current'), branches: git('branch', '--format=%(refname:short) %(objectname)'), remotes: git('remote') } }) + '\\n');
const say = event => process.stdout.write(JSON.stringify(event) + '\\n');
const at = name => args[args.indexOf(name) + 1];
const known = fs.existsSync(${JSON.stringify(sessions)}) ? fs.readFileSync(${JSON.stringify(sessions)}, 'utf8').split('\\n') : [];
if (args.includes('--resume') && !known.includes(at('--resume'))) {
  process.stderr.write('No conversation found with session ID: ' + at('--resume') + '\\n');
  process.exit(1);
}
const session = args.includes('--resume') ? at('--resume') : at('--session-id');
fs.appendFileSync(${JSON.stringify(sessions)}, session + '\\n');
say({ type: 'system', subtype: 'init', session_id: session, cwd: process.cwd(), tools: ['Read', 'Bash'] });
let spec = JSON.parse(fs.readFileSync(${JSON.stringify(answerFile)}, 'utf8'));
if (typeof spec === 'string') spec = { text: spec };
if (spec === null) { say({ type: 'result', subtype: 'success', is_error: true, result: 'the model is not available', session_id: session }); process.exit(1); }
if (spec.text === undefined && !spec.steps && !spec.wait) { say({ type: 'result', session_id: session, ...spec }); process.exit(1); }
(spec.steps || []).forEach((step, i) => {
  const id = 'toolu_' + i;
  say({ type: 'assistant', session_id: session, message: { role: 'assistant', content: [{ type: 'tool_use', id, name: step.tool, input: step.input || {} }] } });
  say({ type: 'user', session_id: session, message: { role: 'user', content: [{ type: 'tool_result', tool_use_id: id, is_error: !!step.refused, content: step.refused ? 'PreToolUse:' + step.tool + ' hook error: ' + step.refused : 'done' }] } });
});
setTimeout(() => {
  say({ type: 'result', subtype: 'success', is_error: false, result: spec.text || '', session_id: session, num_turns: 1 });
}, spec.wait || 0);
`);
  chmodSync(command, 0o755);
  return {
    command,
    asked: () => (existsSync(log) ? readFileSync(log, 'utf8').split('\n').filter(Boolean).map(l => JSON.parse(l)) : []),
    answer: text => writeFileSync(answerFile, JSON.stringify(text)),
    done: () => rmSync(dir, { recursive: true, force: true }),
  };
}
