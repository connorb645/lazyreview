/**
 * A stand-in for a slow disk under the app's server, so a test can catch a
 * reader that opens a file between its writer creating it and filling it.
 *
 * Loaded into a child process with NODE_OPTIONS=--import=<this file's URL>,
 * it acts in `review-desk serve` alone: every file that process writes under
 * REVIEW_DESK_HOME with writeFileSync is created empty at once and filled
 * half a second later. Every other process runs as it would.
 */
import fs from 'node:fs';
import { syncBuiltinESMExports } from 'node:module';

const DELAY = 500;

if (process.argv.includes('serve') && process.env.REVIEW_DESK_HOME) {
  const write = fs.writeFileSync;
  fs.writeFileSync = (path, ...rest) => {
    if (typeof path === 'string' && path.startsWith(process.env.REVIEW_DESK_HOME)) {
      fs.closeSync(fs.openSync(path, 'w'));
      Atomics.wait(new Int32Array(new SharedArrayBuffer(4)), 0, 0, DELAY);
    }
    return write(path, ...rest);
  };
  syncBuiltinESMExports();
}
