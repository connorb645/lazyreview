/**
 * The layout engine the diagram is laid out with: ELK 0.9.3, vendored.
 *
 * defaultElk() returns a new ELK instance for layoutModel.
 */
import { createRequire } from 'node:module';
import { fileURLToPath } from 'node:url';
import { join, dirname } from 'node:path';

const VENDOR = join(dirname(fileURLToPath(import.meta.url)), '..', '..', '..', 'vendor');

export function defaultElk() {
  const ELK = createRequire(import.meta.url)(join(VENDOR, 'elk-0.9.3.bundled.js'));
  return new ELK();
}
