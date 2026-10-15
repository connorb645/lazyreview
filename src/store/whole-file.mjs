/**
 * Writes a file whole, so another process reading it never finds it empty or
 * half written.
 *
 * writeWhole(path, text) writes `text` to a new file beside `path`, then
 * renames that over `path`: a reader finds the file as it was or as it is
 * now, nothing in between. A failure leaves `path` as it was.
 */
import { writeFileSync, renameSync, rmSync } from 'node:fs';
import { randomBytes } from 'node:crypto';

export function writeWhole(path, text) {
  const partial = `${path}.${process.pid}.${randomBytes(4).toString('hex')}.tmp`;
  try {
    writeFileSync(partial, text);
    renameSync(partial, path);
  } catch (error) {
    rmSync(partial, { force: true });
    throw error;
  }
}
