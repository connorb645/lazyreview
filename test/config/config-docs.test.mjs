import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { documented, configSchema } from '../../src/config/config-docs.mjs';
import { defaultConfigText } from '../../src/config/config-text.mjs';
import { parseYAML } from '../../src/config/yaml.mjs';
import { defaults } from '../../src/config/schema.mjs';

const ROOT = join(dirname(fileURLToPath(import.meta.url)), '..', '..');

test('docs/Config.md and schema/config.json say what the schema says; npm run docs writes them again', () => {
  const docs = readFileSync(join(ROOT, 'docs', 'Config.md'), 'utf8');
  assert.equal(docs, documented(docs), 'docs/Config.md is out of date with src/config/schema.mjs; run npm run docs');
  assert.equal(readFileSync(join(ROOT, 'schema', 'config.json'), 'utf8'), JSON.stringify(configSchema(), null, 2) + '\n', 'schema/config.json is out of date with src/config/schema.mjs; run npm run docs');
});

test('the default config, as the docs show it, reads back as the defaults', () => {
  assert.deepEqual(parseYAML(defaultConfigText()), defaults());
});
