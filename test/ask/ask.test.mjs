import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync, realpathSync, renameSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { headAsked, copyBranches, questionFor } from '../../src/ask/question.mjs';
import { Claude } from '../../src/commands/claude/claude.mjs';
import { ThreadService } from '../../src/ask/thread-service.mjs';
import { BranchLoader } from '../../src/loader/branch-loader.mjs';
import { fakeClaude } from '../support/fake-claude.mjs';
import { workbench } from '../support/workbench.mjs';
import { folderState } from '../support/folder-state.mjs';
import { PullRef } from '../../src/models/pull-ref.mjs';
import { english } from '../../src/i18n/english.mjs';

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
const single = { base: { ref: 'main', sha: 'f'.repeat(40) }, members: [{ pull: 12, head: 'e'.repeat(40), title: 'Tidy the parser', description: 'Makes the parser shorter.' }] };

test('a question carries the pull request, where the codebase is, the hunk around its line, the code near it, and the conversation', () => {
  const thread = { path: 'src/a.ts', side: 'RIGHT', line: 11, messages: [{ from: 'you', text: 'Why add this?' }, { from: 'claude', text: 'To log.' }, { from: 'claude', error: 'timed out' }, { from: 'you', text: 'Is it tested?' }] };
  const { system, prompt } = questionFor({ review: single, page: page(), thread, folder: '/code/c0ffee' });
  assert.equal(system, english.ask.system);
  assert.match(system, /throwaway copy of the codebase/);
  assert.match(system, /is refused, as nobody is there to approve it/);
  assert.match(prompt, /^Pull request #12: Tidy the parser\n\nIts description:\nMakes the parser shorter\.\n\nThe codebase as it stands after this change is in \/code\/c0ffee, your working folder, with the branch pull\/12 checked out and the branch main at the base of the change\.\n\n/);
  assert.match(prompt, /The files this change touches:\nsrc\/a\.ts \(changed\)\nsrc\/new\.ts \(new\)\n\n/);
  assert.match(prompt, /File: src\/a\.ts \(changed\), at \/code\/c0ffee\/src\/a\.ts\n\nThey ask about line 11\.\n\nThe diff of this file/);
  assert.match(prompt, /```diff\n@@ -10,2 \+10,3 @@ top\n line 10\n\+added 11\n line 12\n```/);
  assert.ok(!prompt.includes('old 500'), 'only the hunk around the line goes');
  assert.match(prompt, /The file at the head, lines 1-91 of 900:\n```\n 1 \| line 1\n/);
  assert.ok(!prompt.includes('line 200'), 'a long file sends only the lines around the question');
  assert.match(prompt, /The conversation so far:\nThey asked: Why add this\?\n\nYou answered: To log\.\n\nTheir question:\nIs it tested\?$/);
});

test('a new file sends its whole text, a removed line sends no head text, and an unchanged file its context code', () => {
  const ask = (path, side, line, start_line) => questionFor({ review: single, page: page(), thread: { path, side, line, start_line, messages: [{ from: 'you', text: 'Q?' }] }, folder: '/code' }).prompt;
  assert.match(ask('src/new.ts', 'RIGHT', 2, 1), /They ask about lines 1-2\.[\s\S]*The file at the head:\n```\n1 \| export const a = 1;\n2 \| export const b = 2;\n```/);
  const removed = ask('src/a.ts', 'LEFT', 500);
  assert.match(removed, /removed line 500/);
  assert.match(removed, /-old 500/);
  assert.ok(!removed.includes('The file at the head'));
  assert.match(ask('src/other.ts', 'RIGHT', 1), /File: src\/other\.ts \(unchanged by this change\), at \/code\/src\/other\.ts[\s\S]*1 \| export class Other \{\}/);
  const across = questionFor({ review: single, page: page(), thread: { path: 'src/a.ts', side: 'RIGHT', line: 11, start_line: 11, start_side: 'LEFT', messages: [{ from: 'you', text: 'Q?' }] }, folder: '/code' }).prompt;
  assert.match(across, /They ask about removed line 11 to line 11\./, 'a range from a removed line to an added one names both sides');
  assert.match(across, /\+added 11/);
});

test('a conversation keeps the question, then Claude\'s answer, or why there is none', async () => {
  const w = workbench();
  const fake = fakeClaude();
  try {
    await w.loader.load({ links: [w.link(30)] });
    const ref = new PullRef('github.com', 'octo/widgets', 30);
    const threads = new ThreadService({ store: w.store, claude: new Claude({ command: fake.command }) });
    const build = w.store.pageBuild(ref);
    const thread = await threads.start(ref, { path: 'docs/guide.md', side: 'RIGHT', line: 3, text: ' Every? ', build });
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

test('on a stack, the code read is the head of the pull request that changed the lines, and the message says where they are there', () => {
  const [low, top] = ['a'.repeat(40), 'b'.repeat(40)];
  const stack = { base: { ref: 'main', sha: 'c'.repeat(40) }, members: [{ pull: 21, head: low, title: 'Add search' }, { pull: 22, head: top, title: 'Add basket' }] };
  const stacked = { ...page(), stack: { lines: { 'src/a.ts': { RIGHT: { 11: [21, 9], 12: [21, 10, 'src/old-a.ts'], 13: { c: [[21, 11], [22, 13]] } }, LEFT: { 500: [22, 500] } } } } };
  const thread = (line, start_line, side = 'RIGHT') => ({ path: 'src/a.ts', side, line, start_line, messages: [{ from: 'you', text: 'Q?' }] });
  assert.deepEqual(headAsked({ review: stack, page: stacked, thread: thread(11) }), { commit: low, pull: 21, path: 'src/a.ts', lines: [9, 9] });
  assert.deepEqual(headAsked({ review: stack, page: stacked, thread: thread(12, 11) }), { commit: low, pull: 21, path: 'src/old-a.ts', lines: [10, 10] }, 'a range that moves file keeps its last line');
  assert.deepEqual(headAsked({ review: stack, page: stacked, thread: thread(13) }), { commit: top, pull: 22, path: 'src/a.ts', lines: [13, 13] }, 'a line no one pull request changed is read at the top');
  assert.deepEqual(headAsked({ review: stack, page: stacked, thread: thread(500, null, 'LEFT') }), { commit: top, pull: 22, path: 'src/a.ts', lines: null }, 'a removed line is at no head');
  assert.deepEqual(headAsked({ review: { base: stack.base, members: [{ pull: 12, head: low }] }, page: page(), thread: thread(12, 11) }), { commit: low, pull: null, path: 'src/a.ts', lines: [11, 12] });
  assert.deepEqual(headAsked({ review: { kind: 'branch', head: { sha: top } }, page: page(), thread: thread(12, 11) }), { commit: top, pull: null, path: 'src/a.ts', lines: [11, 12] });

  const prompt = questionFor({ review: stack, page: stacked, thread: thread(11), folder: '/code' }).prompt;
  assert.match(prompt, /The codebase as it stands after #21 is in \/code, your working folder, with the branch pull\/21 checked out and the branch main at the base of the change\./);
  assert.deepEqual(copyBranches({ review: stack, page: stacked, thread: thread(11) }), { branches: { main: 'c'.repeat(40), 'pull/21': low, 'pull/22': top }, branch: 'pull/21', base: 'main' });
  assert.deepEqual(copyBranches({ review: { kind: 'branch', base: { ref: 'main', sha: low }, head: { ref: 'fix/x', sha: top } }, page: page(), thread: thread(11) }), { branches: { main: low, 'fix/x': top }, branch: 'fix/x', base: 'main' });
  assert.match(prompt, /File: src\/a\.ts \(changed\), at \/code\/src\/a\.ts\n\nThey ask about line 11, which #21 changed\.\n\nIn the codebase, after #21, they are line 9\./);
  assert.match(prompt, /The file at the head of #22, lines 1-91 of 900:/);
  const atTop = questionFor({ review: stack, page: stacked, thread: thread(13), folder: '/code' }).prompt;
  assert.match(atTop, /with the branch pull\/22 checked out/);
  assert.ok(!atTop.includes('In the codebase, after'), 'lines at the top are where the app shows them');
  assert.match(atTop, /The file at the head, lines/);
});


const copyOf = (w, ref, id) => realpathSync(w.home.copy(ref, id));

test('Claude works in a copy of the code at the pull request\'s head, one per conversation, and a follow-up goes on from the conversation\'s session', async () => {
  const w = workbench();
  const fake = fakeClaude();
  try {
    await w.loader.load({ links: [w.link(30)] });
    const ref = new PullRef('github.com', 'octo/widgets', 30);
    const threads = new ThreadService({ store: w.store, claude: new Claude({ command: fake.command }) });
    const build = w.store.pageBuild(ref);
    const thread = await threads.start(ref, { path: 'docs/guide.md', side: 'RIGHT', line: 3, text: 'Where is the catalog?', build });
    await threads.reply(ref, thread.id, { text: 'And its test?', build });
    const other = await threads.start(ref, { path: 'docs/guide.md', side: 'RIGHT', line: 5, text: 'Why a list?', build });
    const [asked, followUp, second] = fake.asked();
    assert.equal(asked.folder, copyOf(w, ref, 1));
    assert.equal(asked.files['docs/guide.md'], '# Guide\n\nThe catalog lists every item.\n\n- Browse\n- Search\n');
    assert.match(asked.files['Sources/Catalog/Catalog.swift'], /protocol Catalog/, 'files the diff does not show are there');
    assert.deepEqual(asked.git, { branch: 'pull/30', branches: `main ${w.repo.sha('main')}\npull/30 ${w.repo.sha('fix/wording')}`, remotes: '' });
    assert.ok(asked.prompt.includes(`The codebase as it stands after this change is in ${w.home.copy(ref, 1)}, your working folder, with the branch pull/30 checked out`));
    const settings = JSON.parse(asked.args[asked.args.indexOf('--settings') + 1]);
    assert.match(settings.hooks.PreToolUse[0].hooks[0].command, /bin\/lazyreview-guard' '.*\/copies\/github\.com\/octo\/widgets\/30\/1'$/, 'the guard knows the copy');
    const session = w.store.threads(ref)[0].session;
    assert.match(session, /^[0-9a-f-]{36}$/);
    assert.equal(asked.args[asked.args.indexOf('--session-id') + 1], session);
    assert.equal(followUp.folder, asked.folder);
    assert.equal(followUp.args[followUp.args.indexOf('--resume') + 1], session, 'the follow-up goes on from the session');
    assert.equal(followUp.prompt, 'Their follow-up question, on the same lines:\nAnd its test?');
    assert.equal(second.folder, copyOf(w, ref, 2), 'each conversation has its own copy');
    assert.notEqual(other.session, session);
  } finally {
    fake.done();
    w.done();
  }
});

test('on a stack, the copy is at the head of the pull request that changed the lines asked about', async () => {
  const w = workbench();
  const fake = fakeClaude();
  try {
    const { ref } = await w.loader.load({ links: [w.link(21), w.link(22)] });
    const threads = new ThreadService({ store: w.store, claude: new Claude({ command: fake.command }) });
    const build = w.store.pageBuild(ref);
    await threads.start(ref, { path: 'Sources/Catalog/Search.swift', side: 'RIGHT', line: 4, text: 'Who calls find?', build });
    await threads.start(ref, { path: 'Sources/Basket/Basket.swift', side: 'RIGHT', line: 3, text: 'Why keep a search?', build });
    const [lower, upper] = fake.asked();
    assert.equal(lower.git.branch, 'pull/21');
    assert.ok(!lower.files['Sources/Basket/Basket.swift'].includes('search'), 'the code is as #21 leaves it, before #22');
    assert.match(lower.prompt, /The codebase as it stands after #21 is in /);
    assert.equal(upper.git.branch, 'pull/22');
    assert.match(upper.files['Sources/Basket/Basket.swift'], /let search: Search/);
  } finally {
    fake.done();
    w.done();
  }
});

test('on a branch review, the copy is at the branch\'s commit, not the worktree, from lazyreview\'s own clone, and nothing in the repository is written', async () => {
  const w = workbench();
  const fake = fakeClaude();
  try {
    w.repo.checkout('fix/wording');
    const { ref } = await new BranchLoader({ store: w.store }).load({ repo: w.repo.work, base: 'main', head: 'fix/wording' });
    writeFileSync(join(w.repo.work, 'docs/guide.md'), 'Not committed.\n');
    writeFileSync(join(w.repo.work, 'notes.txt'), 'Not added.\n');
    const before = folderState(w.repo.work);
    const threads = new ThreadService({ store: w.store, claude: new Claude({ command: fake.command }) });
    const thread = await threads.start(ref, { path: 'docs/guide.md', side: 'RIGHT', line: 3, text: 'Every?', build: w.store.pageBuild(ref) });
    assert.equal(thread.messages[1].text, 'It returns early when the list is empty.');
    const [asked] = fake.asked();
    assert.equal(asked.folder, copyOf(w, ref, 1));
    assert.equal(asked.files['docs/guide.md'], '# Guide\n\nThe catalog lists every item.\n\n- Browse\n- Search\n');
    assert.equal(asked.files['notes.txt'], undefined);
    assert.deepEqual([asked.git.branch, asked.git.remotes], ['fix/wording', '']);
    assert.match(readFileSync(join(asked.folder, '.git/objects/info/alternates'), 'utf8'), /clones\/local\/work-[0-9a-f]{8}\.git\/objects/, 'the copy borrows from lazyreview\'s own clone');
    assert.deepEqual(folderState(w.repo.work), before, 'the worktree and its .git folder are as they were');
  } finally {
    fake.done();
    w.done();
  }
});

test('the answer keeps the steps Claude took, each refused one with why', async () => {
  const w = workbench();
  const fake = fakeClaude();
  try {
    await w.loader.load({ links: [w.link(30)] });
    const ref = new PullRef('github.com', 'octo/widgets', 30);
    const threads = new ThreadService({ store: w.store, claude: new Claude({ command: fake.command }) });
    const folder = join(realpathSync(w.home.root), 'copies', 'github.com', 'octo', 'widgets', '30', '1');
    fake.answer({ text: 'Only the guide says so.', steps: [
      { tool: 'Grep', input: { pattern: 'catalog', path: `${folder}/docs` } },
      { tool: 'Bash', input: { command: 'gh pr comment 30 --body "Every item?"' }, refused: 'lazyreview refused this: gh may only read, one command at a time.' },
    ] });
    const thread = await threads.start(ref, { path: 'docs/guide.md', side: 'RIGHT', line: 3, text: 'Every?', build: w.store.pageBuild(ref) });
    assert.deepEqual(thread.messages[1].steps, [
      { words: 'Searched for catalog in docs' },
      { words: 'Ran gh pr comment 30 --body "Every item?"', refused: 'gh may only read, one command at a time.' },
    ]);
    assert.equal(thread.working, undefined);
  } finally {
    fake.done();
    w.done();
  }
});

test('while Claude works the conversation shows its steps, and stopping it says so; a conversation left waiting stops too', async () => {
  const w = workbench();
  const fake = fakeClaude();
  try {
    await w.loader.load({ links: [w.link(30)] });
    const ref = new PullRef('github.com', 'octo/widgets', 30);
    const threads = new ThreadService({ store: w.store, claude: new Claude({ command: fake.command }) });
    const build = w.store.pageBuild(ref);
    fake.answer({ text: 'Too late.', steps: [{ tool: 'Bash', input: { command: 'npm test' } }], wait: 60_000 });
    const asking = threads.start(ref, { path: 'docs/guide.md', side: 'RIGHT', line: 3, text: 'Is it tested?', build });
    let working;
    for (let i = 0; i < 100 && !working?.steps?.length; i++) {
      await new Promise(done => setTimeout(done, 100));
      working = w.store.threads(ref)[0]?.working;
    }
    assert.deepEqual(working, { steps: [{ words: 'Ran npm test' }] });
    await assert.rejects(threads.reply(ref, 1, { text: 'And?', build }), /Claude is still working on this conversation/);
    const stopped = await threads.stop(ref, 1);
    assert.deepEqual(stopped.messages.map(m => m.error || m.text), ['Is it tested?', 'you stopped it']);
    assert.deepEqual(stopped.messages[1].steps, [{ words: 'Ran npm test' }]);
    assert.equal(stopped.working, undefined);
    assert.equal((await asking).messages.length, 2);

    fake.answer({ text: 'Too late.', wait: 60_000 });
    const again = threads.reply(ref, 1, { text: 'Now?', build });
    for (let i = 0; i < 100 && !w.store.threads(ref)[0].session; i++) await new Promise(done => setTimeout(done, 100));
    await new Promise(done => setTimeout(done, 500));
    await threads.stopAll();
    assert.deepEqual((await again).messages.slice(-1).map(m => m.error), ['you stopped it'], 'stopping the server stops Claude too');

    await w.store.change([ref], async () => {
      const left = { ...w.store.threads(ref)[0], id: 2, messages: [{ from: 'you', text: 'Asked before a restart', at: 'x' }], working: { steps: [] } };
      return { files: [[w.home.review(ref).thread(2), JSON.stringify(left)]], result: null };
    });
    const unstuck = await threads.stop(ref, 2);
    assert.deepEqual(unstuck.messages.map(m => m.error || m.text), ['Asked before a restart', 'you stopped it']);
    assert.equal(unstuck.working, undefined);
  } finally {
    fake.done();
    w.done();
  }
});

test('a follow-up whose session is gone starts a new one with the whole question', async () => {
  const w = workbench();
  const fake = fakeClaude();
  try {
    await w.loader.load({ links: [w.link(30)] });
    const ref = new PullRef('github.com', 'octo/widgets', 30);
    const threads = new ThreadService({ store: w.store, claude: new Claude({ command: fake.command }) });
    const build = w.store.pageBuild(ref);
    await threads.start(ref, { path: 'docs/guide.md', side: 'RIGHT', line: 3, text: 'Every?', build });
    await w.store.change([ref], async () => {
      const thread = { ...w.store.threads(ref)[0], session: '00000000-0000-4000-8000-000000000000' };
      return { files: [[w.home.review(ref).thread(1), JSON.stringify(thread)]], result: null };
    });
    const thread = await threads.reply(ref, 1, { text: 'And then?', build });
    const [, lost, fresh] = fake.asked();
    assert.equal(lost.args[lost.args.indexOf('--resume') + 1], '00000000-0000-4000-8000-000000000000');
    assert.equal(fresh.args[fresh.args.indexOf('--session-id') + 1], thread.session);
    assert.match(fresh.prompt, /The conversation so far:\nThey asked: Every\?[\s\S]*Their question:\nAnd then\?$/);
    assert.equal(thread.messages[3].text, 'It returns early when the list is empty.');
  } finally {
    fake.done();
    w.done();
  }
});

test('when no copy of the code can be made, the conversation says why, and Claude is not asked', async () => {
  const w = workbench();
  const fake = fakeClaude();
  try {
    const { ref } = await new BranchLoader({ store: w.store }).load({ repo: w.repo.work, base: 'main', head: 'fix/wording' });
    renameSync(w.repo.work, `${w.repo.work}-moved`);
    const threads = new ThreadService({ store: w.store, claude: new Claude({ command: fake.command }) });
    const thread = await threads.start(ref, { path: 'docs/guide.md', side: 'RIGHT', line: 3, text: 'Every?', build: w.store.pageBuild(ref) });
    renameSync(`${w.repo.work}-moved`, w.repo.work);
    assert.match(thread.messages[1].error, /^could not make a copy of the code for Claude: git fetch failed: .*does not appear to be a git repository/);
    assert.deepEqual(fake.asked(), []);
  } finally {
    fake.done();
    w.done();
  }
});
