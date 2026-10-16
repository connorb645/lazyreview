/**
 * A stand-in for Claude Code, so tests ask no real model.
 *
 * fakeClaude() makes an executable that answers one question in print mode
 * the way Claude Code does, and returns {command, asked, answer, done}:
 * `command` is its path, `asked()` reads back each call as {args, prompt},
 * `answer(text)` sets what it says next, or `answer(null)` makes it fail,
 * and `done()` removes it.
 */
import { mkdtempSync, writeFileSync, readFileSync, existsSync, rmSync, chmodSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

export function fakeClaude() {
  const dir = mkdtempSync(join(tmpdir(), 'lazyreview-claude-'));
  const command = join(dir, 'claude');
  const log = join(dir, 'asked.jsonl');
  const answerFile = join(dir, 'answer.json');
  writeFileSync(answerFile, JSON.stringify('It returns early when the list is empty.'));
  writeFileSync(command, `#!/usr/bin/env node
const fs = require('fs');
const prompt = fs.readFileSync(0, 'utf8');
fs.appendFileSync(${JSON.stringify(log)}, JSON.stringify({ args: process.argv.slice(2), prompt }) + '\\n');
const answer = JSON.parse(fs.readFileSync(${JSON.stringify(answerFile)}, 'utf8'));
if (answer === null) { process.stdout.write(JSON.stringify({ type: 'result', is_error: true, result: 'the model is not available' })); process.exit(1); }
process.stdout.write(JSON.stringify({ type: 'result', subtype: 'success', is_error: false, result: answer }));
`);
  chmodSync(command, 0o755);
  return {
    command,
    asked: () => (existsSync(log) ? readFileSync(log, 'utf8').split('\n').filter(Boolean).map(l => JSON.parse(l)) : []),
    answer: text => writeFileSync(answerFile, JSON.stringify(text)),
    done: () => rmSync(dir, { recursive: true, force: true }),
  };
}
