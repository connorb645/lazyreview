/**
 * The hunks of one file's diff, row by row, as the apps draw them.
 *
 * readHunks(diff) is pure. It takes a file's part of a unified diff, with or
 * without its header, and returns its hunks, each {header, old_start,
 * old_count, new_start, new_count, rows}, where each row is {kind, old, new,
 * text}: `kind` is add, del, same or note, a note being a line such as "\ No
 * newline at end of file"; `old` and `new` are its line numbers on each
 * side, null on the side it is not on.
 */
const HUNK = /^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@/;

export function readHunks(diff) {
  const hunks = [];
  let hunk = null;
  let old = 0;
  let now = 0;
  for (const line of (diff || '').split('\n')) {
    const start = HUNK.exec(line);
    if (start) {
      old = +start[1];
      now = +start[3];
      hunk = { header: line, old_start: old, old_count: start[2] === undefined ? 1 : +start[2], new_start: now, new_count: start[4] === undefined ? 1 : +start[4], rows: [] };
      hunks.push(hunk);
      continue;
    }
    if (!hunk) continue;
    const sign = line.charAt(0);
    if (sign === '+') hunk.rows.push({ kind: 'add', old: null, new: now++, text: line.slice(1) });
    else if (sign === '-') hunk.rows.push({ kind: 'del', old: old++, new: null, text: line.slice(1) });
    else if (sign === '\\') hunk.rows.push({ kind: 'note', old: null, new: null, text: line });
    else if (sign === ' ' || (line === '' && !(now >= hunk.new_start + hunk.new_count && old >= hunk.old_start + hunk.old_count))) {
      hunk.rows.push({ kind: 'same', old: old++, new: now++, text: line.slice(1) });
    }
  }
  return hunks;
}
