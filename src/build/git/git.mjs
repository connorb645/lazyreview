/**
 * Reads a repository through git. Every command here only reads: nothing
 * checks out, fetches or writes to the repository or its config.
 */
import { execFileSync } from 'node:child_process';

function runGit(repo, args, { input, allowExit = [] } = {}) {
  try {
    return execFileSync('git', ['-C', repo, '-c', 'core.quotepath=false', ...args], {
      input, maxBuffer: 1 << 30, stdio: ['pipe', 'pipe', 'pipe'],
    });
  } catch (error) {
    if (allowExit.includes(error.status)) return Buffer.from(error.stdout || '');
    const message = String(error.stderr || error.message).trim().split('\n')[0];
    throw new Error(`git ${args[0]} failed: ${message}`);
  }
}

export class Git {
  #repo;
  #run;

  constructor(repo, run = runGit) {
    this.#repo = repo;
    this.#run = run;
  }

  #text(args, options) {
    return this.#run(this.#repo, args, options).toString('utf8');
  }

  /** The full commit id a ref names, or an error naming the ref. */
  resolve(ref) {
    try {
      return this.#text(['rev-parse', '--verify', '--quiet', `${ref}^{commit}`]).trim();
    } catch {
      throw new Error(`"${ref}" is not a commit in this repository`);
    }
  }

  mergeBase(a, b) {
    return this.#text(['merge-base', a, b]).trim();
  }

  diff(base, head) {
    return this.#text(['diff', '--no-color', '--no-ext-diff', '--find-renames', base, head]);
  }

  /** {path: {adds, dels, binary}} for every file the diff touches. */
  counts(base, head) {
    const out = new Map();
    const fields = this.#text(['diff', '--numstat', '-z', '--find-renames', base, head]).split('\0');
    for (let i = 0; i < fields.length; i++) {
      const m = /^(-|\d+)\t(-|\d+)\t(.*)$/s.exec(fields[i].replace(/^\n/, ''));
      if (!m) continue;
      let path = m[3];
      if (path === '') { path = fields[i + 2]; i += 2; }
      out.set(path, { adds: m[1] === '-' ? 0 : +m[1], dels: m[2] === '-' ? 0 : +m[2], binary: m[1] === '-' });
    }
    return out;
  }

  commitCount(base, head) {
    return +this.#text(['rev-list', '--count', `${base}..${head}`]).trim();
  }

  /** The text of each path at a commit, or null where the path is missing there. */
  read(commit, paths) {
    const out = new Map();
    if (!paths.length) return out;
    const buffer = this.#run(this.#repo, ['cat-file', '--batch'], { input: paths.map(p => `${commit}:${p}`).join('\n') + '\n' });
    let at = 0;
    for (const path of paths) {
      const end = buffer.indexOf(10, at);
      const header = buffer.subarray(at, end).toString('utf8');
      at = end + 1;
      const m = /^\S+ (\S+) (\d+)$/.exec(header);
      if (!m) { out.set(path, null); continue; }
      const size = +m[2];
      out.set(path, m[1] === 'blob' ? buffer.subarray(at, at + size).toString('utf8') : null);
      at += size + 1;
    }
    return out;
  }

  /** Lines at a commit that match an extended pattern, in files matching the globs. */
  grep(commit, pattern, globs) {
    const text = this.#text(['grep', '--null', '-I', '-E', '-e', pattern, commit, '--', ...globs], { allowExit: [1] });
    return text.split('\n').filter(Boolean).map(line => {
      const cut = line.indexOf('\0');
      return { path: line.slice(commit.length + 1, cut), text: line.slice(cut + 1) };
    });
  }

  /** How many lines in each file at a commit name any of the words, as whole words. */
  countMentions(commit, words, globs) {
    const out = new Map();
    for (let i = 0; i < words.length; i += 1000) {
      const args = ['grep', '--null', '-I', '-c', '-w', '-F'];
      for (const w of words.slice(i, i + 1000)) args.push('-e', w);
      const text = this.#text([...args, commit, '--', ...globs], { allowExit: [1] });
      for (const line of text.split('\n').filter(Boolean)) {
        const cut = line.indexOf('\0');
        const path = line.slice(commit.length + 1, cut);
        out.set(path, (out.get(path) || 0) + +line.slice(cut + 1));
      }
    }
    return out;
  }
}
