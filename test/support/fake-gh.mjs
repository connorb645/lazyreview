/**
 * A stand-in for the GitHub command line, so tests reach no network and can
 * check exactly what would have been sent.
 *
 * fakeGh() makes a folder holding an executable `gh` and returns {env, bin,
 * routes, calls, done}: `env` puts it first on PATH for a child process, and
 * `bin` is its folder, for this process's own PATH; `routes(map)` sets the
 * answers, as {"GET repos/o/r/pulls/1": body} or {"POST ...": body};
 * `calls()` reads back each call as {method, path, body}; `done()` removes
 * the folder. `gh auth token` answers with a token named for the account, and
 * a route with no answer fails as GitHub's 404 does.
 */
import { mkdtempSync, writeFileSync, readFileSync, existsSync, rmSync, chmodSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

const SCRIPT = `#!/usr/bin/env node
const fs = require('fs');
const args = process.argv.slice(2);
const log = call => fs.appendFileSync(process.env.FAKE_GH_LOG, JSON.stringify(call) + '\\n');
if (args[0] === 'auth' && args[1] === 'token') { process.stdout.write('token-' + args[args.indexOf('--user') + 1] + '\\n'); process.exit(0); }
if (args[0] !== 'api') { process.stderr.write('fake gh: only api and auth token\\n'); process.exit(2); }
let method = null, path = null, input = false;
for (let i = 1; i < args.length; i++) {
  if (args[i] === '-X') method = args[++i];
  else if (args[i] === '--hostname') i++;
  else if (args[i] === '--input') { input = true; i++; }
  else if (!path) path = args[i];
}
method = method || (input ? 'POST' : 'GET');
const key = method + ' ' + path;
const body = input ? JSON.parse(fs.readFileSync(0, 'utf8')) : null;
log({ method, path, body, token: process.env.GH_TOKEN || null });
const routes = JSON.parse(fs.readFileSync(process.env.FAKE_GH_ROUTES, 'utf8'));
if (!(key in routes)) { process.stderr.write('HTTP 404: Not Found (fake gh has no ' + key + ')\\n'); process.exit(1); }
process.stdout.write(JSON.stringify(routes[key]));
`;

export function fakeGh() {
  const dir = mkdtempSync(join(tmpdir(), 'review-desk-gh-'));
  writeFileSync(join(dir, 'gh'), SCRIPT);
  chmodSync(join(dir, 'gh'), 0o755);
  const routesFile = join(dir, 'routes.json');
  const logFile = join(dir, 'calls.jsonl');
  writeFileSync(routesFile, '{}');
  const extra = { FAKE_GH_ROUTES: routesFile, FAKE_GH_LOG: logFile };
  return {
    bin: dir,
    extra,
    env: { ...process.env, PATH: `${dir}:${process.env.PATH}`, ...extra },
    routes: map => writeFileSync(routesFile, JSON.stringify(map)),
    calls: () => (existsSync(logFile) ? readFileSync(logFile, 'utf8').split('\n').filter(Boolean).map(l => JSON.parse(l)) : []),
    done: () => rmSync(dir, { recursive: true, force: true }),
  };
}
