import { test } from 'node:test';
import assert from 'node:assert/strict';
import { mkdtempSync, mkdirSync, symlinkSync, rmSync, realpathSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import { decide, guardSettings, REGISTRIES } from '../../src/ask/guard.mjs';
import { hookAnswer } from '../../src/ask/hook-answer.mjs';
import { stepWords } from '../../src/ask/step-words.mjs';

const copy = '/data/copies/github.com/octo/ledger/72/1';
const call = (tool, input) => decide({ tool_name: tool, tool_input: input, cwd: copy }, { copy, real: resolve });
const allowed = (tool, input) => assert.deepEqual(call(tool, input), { allow: true }, `${tool} ${JSON.stringify(input)}`);
const refused = (tool, input, words) => assert.match(call(tool, input).refuse || '', words, `${tool} ${JSON.stringify(input)}`);

test('reading is allowed, and so is changing the copy, but no file outside it', () => {
  for (const tool of ['Read', 'Grep', 'Glob', 'WebFetch', 'WebSearch', 'Task', 'Agent', 'Skill', 'TodoWrite', 'ToolSearch', 'ReadMcpResourceTool']) allowed(tool, {});
  allowed('Write', { file_path: `${copy}/src/split.ts` });
  allowed('Edit', { file_path: 'src/money.ts' });
  allowed('NotebookEdit', { notebook_path: `${copy}/notes.ipynb` });
  refused('Write', { file_path: '/etc/hosts' }, /^lazyreview refused this: the file is outside the copy of the code, and only the copy may change$/);
  refused('Edit', { file_path: `${copy}/../2/src/money.ts` }, /outside the copy/);
  refused('MultiEdit', { file_path: '/data/copies/github.com/octo/ledger/72/10/a.ts' }, /outside the copy/, 'a sibling whose name starts the same');
});

test('a link inside the copy that leads outside it is outside', () => {
  const home = realpathSync(mkdtempSync(join(tmpdir(), 'lazyreview-guard-')));
  try {
    mkdirSync(join(home, 'copy'));
    mkdirSync(join(home, 'elsewhere'));
    symlinkSync(join(home, 'elsewhere'), join(home, 'copy', 'link'));
    const ask = path => JSON.parse(hookAnswer(JSON.stringify({ tool_name: 'Write', tool_input: { file_path: path }, cwd: join(home, 'copy') }), { copy: join(home, 'copy') }).output).hookSpecificOutput;
    assert.equal(ask(join(home, 'copy', 'new', 'file.ts')).permissionDecision, 'allow');
    assert.equal(ask(join(home, 'copy', 'link', 'file.ts')).permissionDecision, 'deny');
  } finally {
    rmSync(home, { recursive: true, force: true });
  }
});

test('shell commands run, but not one that pushes, posts, publishes or reaches another machine', () => {
  for (const command of ['npm test', 'go test ./...', 'git log --oneline | head', 'git commit -am wip', 'grep -rn "git push" .', 'curl -s https://example.com', 'grep -rn open src']) allowed('Bash', { command });
  refused('Bash', { command: 'git push origin HEAD:main' }, /^lazyreview refused this: it would push, and nobody is there to approve it$/);
  refused('Bash', { command: 'cd src && git -C .. push --force' }, /it would push,/);
  refused('Bash', { command: 'curl -X POST https://hooks.example.com -d text=hi' }, /it would send a change to a web service/);
  refused('Bash', { command: 'curl --json \'{"a":1}\' https://api.example.com' }, /web service/);
  refused('Bash', { command: 'http PATCH https://api.example.com/x a=1' }, /web service/);
  refused('Bash', { command: 'sudo -E npm publish' }, /publish a package/);
  refused('Bash', { command: 'docker push example/image' }, /publish a package/);
  refused('Bash', { command: 'echo ok; /usr/bin/ssh build.example.com' }, /reach a program or a machine/);
  refused('Bash', { command: 'claude -p "push it"' }, /reach a program or a machine/);
  refused('Monitor', { command: 'git push' }, /it would push,/);
});

test('gh only reads, one command at a time', () => {
  for (const command of ['gh pr view 72', 'gh pr diff 72', 'gh issue list', 'gh api repos/octo/ledger/pulls/72', 'gh api -X GET repos/octo/ledger', 'gh search code splitPence']) allowed('Bash', { command });
  refused('Bash', { command: 'gh pr comment 72 --body "Looks good"' }, /^lazyreview refused this: gh may only read/);
  refused('Bash', { command: 'gh pr review 72 --approve' }, /gh may only read/);
  refused('Bash', { command: 'gh api repos/octo/ledger/issues/72/comments -f body=hi' }, /gh may only read/);
  refused('Bash', { command: 'gh api --method POST repos/octo/ledger/issues' }, /gh may only read/);
  refused('Bash', { command: 'gh pr view 72 && rm -rf ~' }, /gh may only read/);
  refused('Bash', { command: 'gh pr view 72 > /tmp/x' }, /gh may only read/);
});

test('an MCP tool is allowed only when its name says it reads, and a tool the guard does not know is refused', () => {
  for (const tool of ['mcp__slack__slack_read_channel', 'mcp__atlassian__getJiraIssue', 'mcp__notes__search_notes', 'mcp__datadog__get_datadog_metric']) allowed(tool, {});
  refused('mcp__slack__slack_send_message', {}, /^lazyreview refused this: slack_send_message on slack may change something/);
  refused('mcp__atlassian__addCommentToJiraIssue', {}, /may change something/);
  refused('mcp__notes__post_note', {}, /may change something/);
  refused('mcp__jobs__check_job', {}, /may change something/, 'a name that does not say it reads');
  for (const tool of ['PushNotification', 'RemoteTrigger', 'CronCreate', 'SendMessage', 'PublishPlugin', 'SomethingNew']) refused(tool, {}, /is not allowed while nobody is there to approve it/);
});

test('the guard\'s settings turn its hook on over the user\'s, and keep shell commands in the sandbox', () => {
  const settings = guardSettings({ hook: '\'/usr/bin/node\' \'/app/bin/lazyreview-guard\' \'/copy\'', domains: ['example.com'] });
  assert.deepEqual(settings.hooks, { PreToolUse: [{ matcher: '*', hooks: [{ type: 'command', command: '\'/usr/bin/node\' \'/app/bin/lazyreview-guard\' \'/copy\'' }] }] });
  assert.equal(settings.disableAllHooks, false);
  assert.equal(settings.enableAllProjectMcpServers, false);
  assert.deepEqual(settings.sandbox, {
    enabled: true, autoAllowBashIfSandboxed: true, allowUnsandboxedCommands: false, failIfUnavailable: true, excludedCommands: ['gh'],
    network: { strictAllowlist: true, allowedDomains: [...REGISTRIES, 'example.com'] },
  });
});

test('the hook answers Claude Code in its own words, and refuses what it cannot read', () => {
  const answer = hookAnswer(JSON.stringify({ tool_name: 'Bash', tool_input: { command: 'git push' }, cwd: '/' }), { copy: '/' });
  assert.equal(answer.code, 0);
  assert.deepEqual(JSON.parse(answer.output).hookSpecificOutput.permissionDecision, 'deny');
  assert.deepEqual(JSON.parse(hookAnswer(JSON.stringify({ tool_name: 'Read', tool_input: {}, cwd: '/' }), { copy: '/' }).output), { hookSpecificOutput: { hookEventName: 'PreToolUse', permissionDecision: 'allow' } });
  const broken = hookAnswer('not json', { copy: '/' });
  assert.equal(broken.code, 2, 'exit code 2 is a refusal to Claude Code');
  assert.match(broken.output, /^lazyreview refused this, as its guard could not read the call/);
});

test('each step says what it did, naming files from the copy\'s top', () => {
  const folder = '/data/copies/x/1';
  assert.equal(stepWords('Read', { file_path: '/data/copies/x/1/test/split.test.ts' }, folder), 'Read test/split.test.ts');
  assert.equal(stepWords('Grep', { pattern: 'splitPence', path: '/data/copies/x/1/src' }, folder), 'Searched for splitPence in src');
  assert.equal(stepWords('Glob', { pattern: '**/*.test.ts' }, folder), 'Looked for files named **/*.test.ts');
  assert.equal(stepWords('Bash', { command: 'npm test\nnpm run lint' }, folder), 'Ran npm test');
  assert.equal(stepWords('Bash', { command: 'cd "/data/copies/x/1" && gh pr view 72 > /data/copies/x/1/pr.txt' }, folder), 'Ran gh pr view 72 > pr.txt');
  assert.equal(stepWords('Write', { file_path: '/elsewhere/a.ts' }, folder), 'Wrote /elsewhere/a.ts');
  assert.equal(stepWords('WebSearch', { query: 'splitting a bill fairly' }, folder), 'Searched the web for splitting a bill fairly');
  assert.equal(stepWords('Task', { description: 'Find every caller' }, folder), 'Asked a helper to find every caller');
  assert.equal(stepWords('mcp__notes__post_note', {}, folder), 'Used post_note on notes');
  assert.equal(stepWords('Bash', { command: 'x'.repeat(400) }, folder).length, 'Ran '.length + 160);
});
