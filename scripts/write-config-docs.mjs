#!/usr/bin/env node
// Writes the config's documentation from its schema: the generated parts of
// docs/Config.md, and schema/config.json. Run it as npm run docs after a
// change to src/config/schema.mjs.
import { readFileSync, writeFileSync, mkdirSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { documented, configSchema } from '../src/config/config-docs.mjs';

const ROOT = join(dirname(fileURLToPath(import.meta.url)), '..');
const DOCS = join(ROOT, 'docs', 'Config.md');
const SCHEMA = join(ROOT, 'schema', 'config.json');

writeFileSync(DOCS, documented(readFileSync(DOCS, 'utf8')));
mkdirSync(dirname(SCHEMA), { recursive: true });
writeFileSync(SCHEMA, JSON.stringify(configSchema(), null, 2) + '\n');
