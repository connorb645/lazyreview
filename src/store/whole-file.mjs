/**
 * Reads and writes the files lazyreview keeps, each one whole.
 *
 * writeWhole(path, text, {mode}) writes `text` to a new file beside `path`,
 * made with `mode`, 0644 less the umask unless given, then renames that over
 * `path`: another process reading it finds the file as it was or as it is
 * now, never empty or half written. A failure leaves `path` as it was. readText(path) returns a file's text and readJSON(path) its
 * value, each null when there is no such file; a file that is not JSON
 * throws an error naming it. parseJSON(path, text) reads the value of text
 * already read from `path`, with the same error.
 */
import { readFileSync, writeFileSync, renameSync, rmSync } from 'node:fs';
import { randomBytes } from 'node:crypto';

export function writeWhole(path, text, { mode = 0o644 } = {}) {
  const partial = `${path}.${process.pid}.${randomBytes(4).toString('hex')}.tmp`;
  try {
    writeFileSync(partial, text, { mode });
    renameSync(partial, path);
  } catch (error) {
    rmSync(partial, { force: true });
    throw error;
  }
}

export function readText(path) {
  try {
    return readFileSync(path, 'utf8');
  } catch (error) {
    if (error.code === 'ENOENT') return null;
    throw error;
  }
}

export function readJSON(path) {
  const text = readText(path);
  return text === null ? null : parseJSON(path, text);
}

export function parseJSON(path, text) {
  try {
    return JSON.parse(text);
  } catch (error) {
    throw new Error(`could not read ${path}: ${error.message}`);
  }
}
