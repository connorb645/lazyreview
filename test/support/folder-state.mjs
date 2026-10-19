/**
 * What a folder holds, so a test can show that nothing in it was written.
 *
 * folderState(root) returns {path: [mode, size, mtime, ctime]} for every file
 * and folder under `root`, its own .git folder included, without following
 * links. Any write, even one that leaves the bytes as they were, changes it.
 */
import { readdirSync, lstatSync } from 'node:fs';
import { join, relative } from 'node:path';

export function folderState(root) {
  const out = {};
  (function walk(folder) {
    for (const entry of readdirSync(folder, { withFileTypes: true })) {
      const path = join(folder, entry.name);
      const stat = lstatSync(path);
      out[relative(root, path)] = [stat.mode, stat.size, stat.mtimeMs, stat.ctimeMs];
      if (entry.isDirectory()) walk(path);
    }
  })(root);
  return out;
}
