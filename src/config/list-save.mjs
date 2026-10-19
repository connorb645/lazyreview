/**
 * Saves a change to one list setting in the config file, in place.
 *
 * saveListChange(path, key, change, {files}) edits the file at `path` as
 * listEdited does, and writes it whole: to a new file in the same folder,
 * renamed over the old one, so a failure leaves the old file as it was. A
 * path that is a link keeps the link, and the file it points to changes,
 * keeping its mode. With no file at `path`, it makes one, and its folder,
 * holding only the list. A change that changes nothing writes nothing. It
 * throws an error naming the file, and why, when it cannot.
 */
import { existsSync, mkdirSync, realpathSync, statSync } from 'node:fs';
import { dirname } from 'node:path';
import { english } from '../i18n/english.mjs';
import { readText, writeWhole } from '../store/whole-file.mjs';
import { listEdited } from './list-edit.mjs';

const nodeFiles = {
  exists: path => existsSync(path),
  real: path => realpathSync(path),
  mode: path => statSync(path).mode & 0o777,
  read: path => readText(path),
  write: (path, text, mode) => writeWhole(path, text, { mode }),
  folder: path => mkdirSync(path, { recursive: true }),
};

export function saveListChange(path, key, change, { files = nodeFiles } = {}) {
  try {
    const exists = files.exists(path);
    const target = exists ? files.real(path) : path;
    const text = exists ? files.read(target) : '';
    const edited = listEdited(text, key, change);
    if (edited === text) return;
    if (!exists) files.folder(dirname(target));
    files.write(target, edited, exists ? files.mode(target) : 0o644);
  } catch (error) {
    throw new Error(english.config.edit.failed(path, error.message));
  }
}
