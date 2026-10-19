import { test } from 'node:test';
import assert from 'node:assert/strict';
import { mkdtempSync, realpathSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { Claude } from '../../../src/commands/claude/claude.mjs';
import { fakeClaude } from '../../support/fake-claude.mjs';

const SESSION = '3b241101-e2bb-4255-8caf-4136c566a962';
const settings = { hooks: { PreToolUse: [] } };

test('Claude is asked once in print mode as the user\'s own Claude Code, in the folder given, with the settings given over the user\'s, and nothing that needs approval', async () => {
  const fake = fakeClaude();
  const folder = mkdtempSync(join(tmpdir(), 'lazyreview-code-'));
  try {
    writeFileSync(join(folder, 'used.test.ts'), 'splitPence(1000, 3);\n');
    const claude = new Claude({ command: fake.command, model: 'claude-sonnet-5-5' });
    assert.equal(await claude.ask({ system: 'Be brief.', prompt: 'Why?', folder, session: SESSION, settings }), 'It returns early when the list is empty.');
    const [call] = fake.asked();
    assert.equal(call.prompt, 'Why?');
    assert.deepEqual(call.args, ['-p', '--output-format', 'stream-json', '--verbose', '--setting-sources', 'user', '--settings', JSON.stringify(settings),
      '--permission-mode', 'default', '--permission-prompts', 'none', '--append-system-prompt', 'Be brief.', '--session-id', SESSION, '--model', 'claude-sonnet-5-5']);
    assert.equal(call.folder, realpathSync(folder), 'Claude Code runs in the folder given');
    assert.deepEqual(call.files, { 'used.test.ts': 'splitPence(1000, 3);\n' });

    await claude.ask({ system: 's', prompt: 'And?', folder, session: SESSION, resume: true, settings });
    assert.deepEqual(fake.asked()[1].args.slice(14, 16), ['--resume', SESSION], 'a follow-up goes on from the session');
    const lost = await claude.ask({ system: 's', prompt: 'p', folder, session: '00000000-0000-4000-8000-000000000000', resume: true, settings }).catch(error => error);
    assert.equal(lost.noSession, true, 'a session that is gone says so');
  } finally {
    fake.done();
    rmSync(folder, { recursive: true, force: true });
  }
});

test('each tool call is heard as Claude makes it, and each refusal with the guard\'s words', async () => {
  const fake = fakeClaude();
  const folder = mkdtempSync(join(tmpdir(), 'lazyreview-code-'));
  try {
    fake.answer({ text: 'Only the test uses it.', steps: [
      { tool: 'Grep', input: { pattern: 'splitPence' } },
      { tool: 'Bash', input: { command: 'git push' }, refused: 'lazyreview refused this: it would push.' },
      { tool: 'Bash', input: { command: 'rm -rf /' }, refused: 'Permission for this tool use was denied.' },
      { tool: 'Bash', input: { command: 'echo x > ~/x' }, refused: 'Exit code 1\n(eval):1: operation not permitted: /Users/x/x' },
    ] });
    const heard = [];
    const text = await new Claude({ command: fake.command }).ask({ system: 's', prompt: 'p', folder, session: SESSION, settings, onStep: step => heard.push(step) });
    assert.equal(text, 'Only the test uses it.');
    assert.deepEqual(heard, [
      { id: 'toolu_0', tool: 'Grep', input: { pattern: 'splitPence' } },
      { id: 'toolu_1', tool: 'Bash', input: { command: 'git push' } },
      { id: 'toolu_1', refused: 'it would push.' },
      { id: 'toolu_2', tool: 'Bash', input: { command: 'rm -rf /' } },
      { id: 'toolu_2', refused: 'it needed approval, and nobody is there to give it' },
      { id: 'toolu_3', tool: 'Bash', input: { command: 'echo x > ~/x' } },
      { id: 'toolu_3', refused: 'the sandbox stopped it: shell commands change only the copy, and reach only the hosts allowed' },
    ]);
  } finally {
    fake.done();
    rmSync(folder, { recursive: true, force: true });
  }
});

test('stopping Claude ends it at once, and every way it can fail says why', async () => {
  const fake = fakeClaude();
  const folder = mkdtempSync(join(tmpdir(), 'lazyreview-code-'));
  try {
    const claude = new Claude({ command: fake.command });
    const ask = signal => claude.ask({ system: 's', prompt: 'p', folder, session: SESSION, settings, signal });
    fake.answer({ text: 'Too late.', wait: 30_000 });
    const controller = new AbortController();
    const started = Date.now();
    setTimeout(() => controller.abort(), 300);
    await assert.rejects(ask(controller.signal), { message: 'you stopped it' });
    assert.ok(Date.now() - started < 10_000, 'it stopped without waiting for the answer');

    fake.answer(null);
    await assert.rejects(ask(), { message: 'Claude could not answer: the model is not available' });
    fake.answer({ subtype: 'error_max_turns', is_error: true, num_turns: 40, errors: ['Reached maximum number of turns (40)'] });
    await assert.rejects(ask(), { message: 'Claude worked for 40 turns without answering; ask about less at once' });
    await assert.rejects(new Claude({ command: '/no/such/claude' }).ask({ system: 's', prompt: 'p', folder, session: SESSION, settings }), { message: 'Claude Code did not answer: /no/such/claude was not found' });
    await assert.rejects(new Claude({ command: 'false' }).ask({ system: 's', prompt: 'p', folder, session: SESSION, settings }), { message: 'Claude Code did not answer: it stopped with exit code 1' });
  } finally {
    fake.done();
    rmSync(folder, { recursive: true, force: true });
  }
});
