/**
 * A stand-in for the GitHub command line, so tests reach no network and can
 * check exactly what would have been sent.
 *
 * fakeGh() makes a folder holding an executable `gh` and returns {env, bin,
 * routes, calls, done}: `env` puts it first on PATH for a child process, and
 * `bin` is its folder, for this process's own PATH; `routes(map)` sets the
 * answers, as {"GET repos/o/r/pulls/1": body} or {"POST ...": body}, and a
 * GraphQL query by its variables, as {"POST graphql o/r#1": body}, with
 * " after <cursor>" for a later page;
 * `calls()` reads back each call as {method, path, body}; `done()` removes
 * the folder. `gh auth token` answers with a token named for the account, and
 * a route with no answer fails as GitHub's 404 does. A route may be kept to
 * one account's calls by its token, as {"token-sam GET user": body}, or
 * {"token-sam *": body} for every call that account makes; an answer of
 * {"$fail": text} makes gh fail, saying `text`.
 *
 * Like GitHub, it refuses a review whose comment is not on a line of its
 * pull request's diff, as the files route gives it, or whose range leaves
 * one hunk or ends before it starts: HTTP 422, posting nothing.
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
const body = input ? JSON.parse(fs.readFileSync(0, 'utf8')) : null;
const asked = path === 'graphql' && body && body.variables;
const key = asked ? 'POST graphql ' + asked.owner + '/' + asked.name + '#' + asked.number + (asked.after ? ' after ' + asked.after : '') : method + ' ' + path;
log({ method, path, body, token: process.env.GH_TOKEN || null });
const routes = JSON.parse(fs.readFileSync(process.env.FAKE_GH_ROUTES, 'utf8'));
const review = /^repos\\/([^/]+\\/[^/]+)\\/pulls\\/(\\d+)\\/reviews$/.exec(path);
if (method === 'POST' && review && body && body.comments) {
  const files = routes['GET repos/' + review[1] + '/pulls/' + review[2] + '/files?per_page=100&page=1'] || [];
  const refused = body.comments.map(c => refusal(files, c)).filter(Boolean);
  if (refused.length) { process.stderr.write('gh: Unprocessable Entity (HTTP 422)\\n' + refused.join('\\n') + '\\n'); process.exit(1); }
}
const token = process.env.GH_TOKEN || '';
const route = [token + ' ' + key, token + ' *', key].find(k => k in routes);
if (route === undefined) { process.stderr.write('HTTP 404: Not Found (fake gh has no ' + key + ')\\n'); process.exit(1); }
const answer = routes[route];
if (answer && typeof answer === 'object' && '$fail' in answer) { process.stderr.write(answer.$fail + '\\n'); process.exit(1); }
process.stdout.write(JSON.stringify(answer));

function places(patch) {
  const at = { RIGHT: {}, LEFT: {} };
  let hunk = -1, old = 0, now = 0;
  (patch || '').split('\\n').forEach((row, i) => {
    const m = /^@@ -(\\d+)(?:,\\d+)? \\+(\\d+)(?:,\\d+)? @@/.exec(row);
    if (m) { hunk++; old = +m[1]; now = +m[2]; return; }
    if (hunk < 0 || row === '' || row[0] === '\\\\') return;
    if (row[0] !== '-') at.RIGHT[now++] = { hunk, i };
    if (row[0] !== '+') at.LEFT[old++] = { hunk, i };
  });
  return at;
}
function refusal(files, c) {
  const file = files.find(f => f.filename === c.path);
  if (!file) return c.path + ': Path could not be resolved';
  const at = places(file.patch);
  const end = at[c.side] && at[c.side][c.line];
  if (!end) return c.path + ':' + c.line + ': Line could not be resolved';
  if (!c.start_line) return null;
  const start = at[c.start_side] && at[c.start_side][c.start_line];
  if (!start) return c.path + ':' + c.start_line + ': Start line could not be resolved';
  if (start.hunk !== end.hunk) return c.path + ':' + c.start_line + '-' + c.line + ': start_line must be part of the same hunk as the line';
  if (start.i >= end.i) return c.path + ':' + c.start_line + '-' + c.line + ': start_line must precede the end line';
  return null;
}
`;

export function fakeGh() {
  const dir = mkdtempSync(join(tmpdir(), 'lazyreview-gh-'));
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
