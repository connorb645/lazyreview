import { test } from 'node:test';
import assert from 'node:assert/strict';
import { questionFor, SYSTEM } from '../../src/ask/question.mjs';
import { Claude } from '../../src/ask/claude.mjs';
import { ThreadService } from '../../src/ask/thread-service.mjs';
import { fakeClaude } from '../support/fake-claude.mjs';
import { workbench } from '../support/workbench.mjs';
import { PullRef } from '../../src/github/pull-ref.mjs';

const longFile = Array.from({ length: 900 }, (_, i) => `line ${i + 1}`);

function page() {
  return {
    model: { files: [{ path: 'src/a.ts', status: 'M', oldPath: 'src/a.ts' }, { path: 'src/new.ts', status: 'A', oldPath: null }] },
    diff: {
      'src/a.ts': 'diff --git a/src/a.ts b/src/a.ts\n@@ -10,2 +10,3 @@ top\n line 10\n+added 11\n line 12\n@@ -500,2 +501,2 @@\n-old 500\n+line 501\n line 502\n',
      'src/new.ts': 'diff --git a/src/new.ts b/src/new.ts\nnew file mode 100644\n@@ -0,0 +1,2 @@\n+export const a = 1;\n+export const b = 2;\n',
    },
    sources: { 'src/a.ts': longFile },
    context: { 'src/other.ts': ['export class Other {}'] },
    stack: null,
  };
}
const single = { members: [{ pull: 12, title: 'Tidy the parser', description: 'Makes the parser shorter.' }] };

test('a question carries the pull request, the hunk around its line, the code near it, and the conversation', () => {
  const thread = { path: 'src/a.ts', side: 'RIGHT', line: 11, messages: [{ from: 'you', text: 'Why add this?' }, { from: 'claude', text: 'To log.' }, { from: 'claude', error: 'timed out' }, { from: 'you', text: 'Is it tested?' }] };
  const { system, prompt } = questionFor({ review: single, page: page(), thread });
  assert.equal(system, SYSTEM);
  assert.match(prompt, /^Pull request #12: Tidy the parser\n\nIts description:\nMakes the parser shorter\./);
  assert.match(prompt, /File: src\/a\.ts \(changed\)/);
  assert.match(prompt, /They ask about line 11\./);
  assert.match(prompt, /```diff\n@@ -10,2 \+10,3 @@ top\n line 10\n\+added 11\n line 12\n```/);
  assert.ok(!prompt.includes('old 500'), 'only the hunk around the line goes');
  assert.match(prompt, /The file at the head, lines 1-91 of 900:\n```\n 1 \| line 1\n/);
  assert.ok(!prompt.includes('line 200'), 'a long file sends only the lines around the question');
  assert.match(prompt, /The conversation so far:\nThey asked: Why add this\?\n\nYou answered: To log\.\n\nTheir question:\nIs it tested\?$/);
});

test('a new file sends its whole text, a removed line sends no head text, and an unchanged file its context code', () => {
  const ask = (path, side, line, start_line) => questionFor({ review: single, page: page(), thread: { path, side, line, start_line, messages: [{ from: 'you', text: 'Q?' }] } }).prompt;
  assert.match(ask('src/new.ts', 'RIGHT', 2, 1), /They ask about lines 1-2\.[\s\S]*The file at the head:\n```\n1 \| export const a = 1;\n2 \| export const b = 2;\n```/);
  const removed = ask('src/a.ts', 'LEFT', 500);
  assert.match(removed, /removed line 500/);
  assert.match(removed, /-old 500/);
  assert.ok(!removed.includes('The file at the head'));
  assert.match(ask('src/other.ts', 'RIGHT', 1), /File: src\/other\.ts \(unchanged by this change\)[\s\S]*1 \| export class Other \{\}/);
  const across = questionFor({ review: single, page: page(), thread: { path: 'src/a.ts', side: 'RIGHT', line: 11, start_line: 11, start_side: 'LEFT', messages: [{ from: 'you', text: 'Q?' }] } }).prompt;
  assert.match(across, /They ask about removed line 11 to line 11\./, 'a range from a removed line to an added one names both sides');
  assert.match(across, /\+added 11/);
});

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
    await assert.rejects(new Claude({ command: '/no/such/claude' }).ask({ system: 's', prompt: 'p' }), /Claude Code did not answer/);
  } finally {
    fake.done();
  }
});

test('a conversation keeps the question, then Claude\'s answer, or why there is none', async () => {
  const w = workbench();
  const fake = fakeClaude();
  try {
    await w.loader.load({ links: [w.link(30)] });
    const ref = new PullRef('github.com', 'octo/widgets', 30);
    const threads = new ThreadService({ store: w.store, claude: new Claude({ command: fake.command }) });
    const build = w.store.pageBuild(ref);
    const thread = await threads.start(ref, { path: 'docs/guide.md', side: 'RIGHT', line: 3, code: 'The catalog lists every item.', text: ' Every? ', build });
    assert.deepEqual(thread.messages.map(m => [m.from, m.text]), [['you', 'Every?'], ['claude', 'It returns early when the list is empty.']]);
    assert.match(fake.asked()[0].prompt, /Pull request #30: Fix the guide wording[\s\S]*File: docs\/guide\.md \(changed\)/);
    fake.answer(null);
    const again = await threads.reply(ref, thread.id, { text: 'And then?', build });
    assert.match(again.messages[3].error, /could not answer/);
    assert.equal(w.store.threads(ref).length, 1);
    await assert.rejects(threads.start(ref, { path: 'docs/guide.md', side: 'RIGHT', line: 3, text: '  ', build }), /ask something first/);
    await assert.rejects(threads.reply(ref, 9, { text: 'Hello?', build }), /there is no conversation 9/);

    w.repo.commit('fix/wording', { 'docs/guide.md': '# Guide\n\nRead this first.\n' }, 'Again');
    w.repo.checkout('main');
    w.routes();
    await w.loader.load({ links: [w.link(30)] });
    const refused = /not asked: the review was loaded again after this window opened/;
    await assert.rejects(threads.start(ref, { path: 'docs/guide.md', side: 'RIGHT', line: 3, text: 'Why?', build }), refused);
    await assert.rejects(threads.reply(ref, thread.id, { text: 'Still?', build }), refused);
    assert.equal(w.store.threads(ref)[0].messages.length, 4, 'a refused question is not kept');
  } finally {
    fake.done();
    w.done();
  }
});
