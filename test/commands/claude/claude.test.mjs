import { test } from 'node:test';
import assert from 'node:assert/strict';
import { Claude } from '../../../src/commands/claude/claude.mjs';
import { fakeClaude } from '../../support/fake-claude.mjs';

test('Claude is asked in one print-mode call with no tools, settings, servers or saved session', async () => {
  const fake = fakeClaude();
  try {
    const claude = new Claude({ command: fake.command, model: 'claude-sonnet-5-5' });
    assert.equal(await claude.ask({ system: 'Be brief.', prompt: 'Why?' }), 'It returns early when the list is empty.');
    const [call] = fake.asked();
    assert.equal(call.prompt, 'Why?');
    assert.deepEqual(call.args, ['-p', '--output-format', 'json', '--tools', '', '--strict-mcp-config', '--setting-sources', '',
      '--disable-slash-commands', '--no-session-persistence', '--system-prompt', 'Be brief.', '--model', 'claude-sonnet-5-5']);
    fake.answer(null);
    await assert.rejects(claude.ask({ system: 's', prompt: 'p' }), /Claude could not answer: the model is not available/);
    await assert.rejects(new Claude({ command: '/no/such/claude' }).ask({ system: 's', prompt: 'p' }), { message: 'Claude Code did not answer: /no/such/claude was not found' });
    await assert.rejects(new Claude({ command: 'false' }).ask({ system: 'A long system prompt.', prompt: 'p' }), { message: 'Claude Code did not answer: it stopped with exit code 1' });
  } finally {
    fake.done();
  }
});
