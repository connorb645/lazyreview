import { test } from 'node:test';
import assert from 'node:assert/strict';
import { GitHub } from '../../../src/commands/github/github.mjs';

test('an answer from GitHub that is not JSON is named in the error', async () => {
  const gh = new GitHub({ run: async () => ({ code: 0, stdout: '{"login":', stderr: '', timedOut: false }) });
  await assert.rejects(gh.login(), { message: 'could not read GitHub\'s answer to GET user: Unexpected end of JSON input' });
});

test('review threads are read from GitHub\'s GraphQL API page by page, and its errors are named', async () => {
  const asked = [];
  const pages = [
    { data: { repository: { pullRequest: { reviewThreads: { pageInfo: { hasNextPage: true, endCursor: 'c1' }, nodes: [{ id: 'T1' }] } } } } },
    { data: { repository: { pullRequest: { reviewThreads: { pageInfo: { hasNextPage: false, endCursor: 'c2' }, nodes: [{ id: 'T2' }] } } } } },
  ];
  const gh = new GitHub({ run: async (args, { input }) => {
    asked.push({ args, body: JSON.parse(input) });
    return { code: 0, stdout: JSON.stringify(pages[asked.length - 1]), stderr: '', timedOut: false };
  } });
  assert.deepEqual((await gh.reviewThreads('octo/widgets', 30)).map(t => t.id), ['T1', 'T2']);
  assert.deepEqual(asked.map(a => a.args), [['api', '-X', 'POST', 'graphql', '--input', '-'], ['api', '-X', 'POST', 'graphql', '--input', '-']]);
  assert.deepEqual(asked.map(a => a.body.variables), [{ owner: 'octo', name: 'widgets', number: 30, after: null }, { owner: 'octo', name: 'widgets', number: 30, after: 'c1' }]);
  assert.match(asked[0].body.query, /reviewThreads\(first: 50, after: \$after\)[\s\S]*isResolved isOutdated/);

  const refused = new GitHub({ run: async () => ({ code: 0, stdout: JSON.stringify({ errors: [{ message: 'Resource not accessible by integration' }] }), stderr: '', timedOut: false }) });
  await assert.rejects(refused.reviewThreads('octo/widgets', 30), { message: 'GitHub refused the review threads of octo/widgets#30: Resource not accessible by integration' });
});

test('a search is read page by page, up to the thousand results GitHub gives', async () => {
  const asked = [];
  const items = n => Array.from({ length: n }, (_, i) => ({ number: i + 1 }));
  const gh = new GitHub({ run: async args => {
    asked.push(args[1]);
    return { code: 0, stdout: JSON.stringify({ items: items(asked.length < 3 ? 100 : 7) }), stderr: '', timedOut: false };
  } });
  assert.equal((await gh.search('is:pr author:@me')).length, 207);
  assert.deepEqual(asked, [1, 2, 3].map(page => `search/issues?q=is%3Apr+author%3A%40me&per_page=100&page=${page}`));
  const full = new GitHub({ run: async () => ({ code: 0, stdout: JSON.stringify({ items: items(100) }), stderr: '', timedOut: false }) });
  assert.equal((await full.search('is:pr')).length, 1000);
});

test('a host gh cannot reach is named as unreachable, not as refusing', async () => {
  const gh = new GitHub({ run: async () => ({ code: 1, stdout: '', stderr: 'error connecting to api.github.com\ncheck your internet connection or https://githubstatus.com\n', timedOut: false }) });
  await assert.rejects(gh.login(), { message: 'could not reach github.com: error connecting to api.github.com' });
});
