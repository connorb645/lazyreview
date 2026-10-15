import { test } from 'node:test';
import assert from 'node:assert/strict';
import { GitHub } from '../../src/github/github.mjs';

test('an answer from GitHub that is not JSON is named in the error', async () => {
  const gh = new GitHub({ run: async () => ({ code: 0, stdout: '{"login":', stderr: '', timedOut: false }) });
  await assert.rejects(gh.login(), { message: 'could not read GitHub\'s answer to GET user: Unexpected end of JSON input' });
});
