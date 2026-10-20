#!/usr/bin/env node
// Lands a branch on main, on every remote and in this checkout, as a
// fast-forward. Run it as make land BRANCH=<branch>. It checks everything
// first, says what it will do, and ends saying where main is.
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { Landing, describe } from './land/landing.mjs';

const ROOT = join(dirname(fileURLToPath(import.meta.url)), '..');

try {
  const landing = new Landing(ROOT);
  const plan = landing.plan(process.argv[2]);
  for (const line of describe(plan)) console.log(line);
  console.log(landing.land(plan));
} catch (error) {
  for (const line of error.message.split('\n')) console.error(`land: ${line}`);
  process.exitCode = 1;
}
