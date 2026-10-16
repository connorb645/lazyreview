/**
 * Talks to GitHub through the `gh` command, as one account on one host.
 *
 * new GitHub({host, account}) answers as `account` when one is named, with
 * that account's token given to each call, so the account gh has active for
 * everything else on the machine is never changed; without one, it answers as
 * gh's active account. Every call has a time limit, and a failure throws an
 * error saying what GitHub refused, or that the host could not be reached.
 */
import { execFile } from 'node:child_process';

const TIMEOUT = 60_000;
const UNREACHABLE = /^error connecting to /;
const SEARCH_PAGES = 10;
const THREADS = `query($owner: String!, $name: String!, $number: Int!, $after: String) {
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) {
      reviewThreads(first: 50, after: $after) {
        pageInfo { hasNextPage endCursor }
        nodes {
          id isResolved isOutdated path line startLine originalLine originalStartLine diffSide startDiffSide subjectType
          resolvedBy { login }
          comments(first: 100) {
            totalCount
            nodes { databaseId author { login __typename } body createdAt url diffHunk }
          }
        }
      }
    }
  }
}`;

export class GitHub {
  #host;
  #account;
  #run;
  #token = null;

  constructor({ host = 'github.com', account = null, run = runGh } = {}) {
    this.#host = host;
    this.#account = account;
    this.#run = run;
  }

  /** The environment a child process needs to act as this account, such as git fetching through gh. */
  async environment() {
    if (!this.#account) return {};
    if (this.#token === null) {
      const done = await this.#run(['auth', 'token', '--hostname', this.#host, '--user', this.#account]);
      if (done.code !== 0) throw new Error(`gh has no token for ${this.#account} on ${this.#host}: ${done.stderr.trim()}`);
      this.#token = done.stdout.trim();
    }
    return this.#host === 'github.com' ? { GH_TOKEN: this.#token } : { GH_ENTERPRISE_TOKEN: this.#token, GH_HOST: this.#host };
  }

  async api(path, { method = 'GET', payload } = {}) {
    const args = ['api', ...(this.#host === 'github.com' ? [] : ['--hostname', this.#host]),
      ...(method === 'GET' ? [] : ['-X', method]), path, ...(payload === undefined ? [] : ['--input', '-'])];
    const done = await this.#run(args, { input: payload === undefined ? undefined : JSON.stringify(payload), env: await this.environment() });
    if (done.timedOut) throw new Error(`GitHub did not answer ${method} ${path} in time; nothing changed`);
    if (done.code !== 0 && UNREACHABLE.test(done.stderr)) throw new Error(`could not reach ${this.#host}: ${done.stderr.trim().split('\n')[0]}`);
    if (done.code !== 0) throw new Error(`GitHub refused ${method} ${path}: ${(done.stderr || done.stdout).trim()}`);
    if (!done.stdout.trim()) return null;
    try {
      return JSON.parse(done.stdout);
    } catch (error) {
      throw new Error(`could not read GitHub's answer to ${method} ${path}: ${error.message}`);
    }
  }

  async #pages(path) {
    const out = [];
    for (let page = 1; ; page++) {
      const batch = await this.api(`${path}${path.includes('?') ? '&' : '?'}per_page=100&page=${page}`);
      out.push(...batch);
      if (batch.length < 100) return out;
    }
  }

  pull(repo, number) {
    return this.api(`repos/${repo}/pulls/${number}`);
  }

  /** Every open pull request of the repository. */
  openPulls(repo) {
    return this.#pages(`repos/${repo}/pulls?state=open`);
  }

  /**
   * Every issue and pull request a search finds, such as `is:pr is:open
   * author:@me`, as GitHub's search gives them, up to the 1,000 it gives.
   */
  async search(query) {
    const out = [];
    for (let page = 1; ; page++) {
      const found = await this.api(`search/issues?${new URLSearchParams({ q: query, per_page: '100', page: String(page) })}`);
      out.push(...found.items);
      if (found.items.length < 100 || page === SEARCH_PAGES) return out;
    }
  }

  repository(repo) {
    return this.api(`repos/${repo}`);
  }

  async login() {
    return (await this.api('user')).login;
  }

  /** Each file of the pull request's diff, as GitHub shows it, with its patch. */
  files(repo, number) {
    return this.#pages(`repos/${repo}/pulls/${number}/files`);
  }

  reviews(repo, number) {
    return this.#pages(`repos/${repo}/pulls/${number}/reviews`);
  }

  reviewComments(repo, number) {
    return this.#pages(`repos/${repo}/pulls/${number}/comments`);
  }

  /**
   * Every review thread on the pull request, as GitHub's GraphQL API gives
   * them, each with its first 100 comments and their count: the REST API
   * does not say which threads are resolved.
   */
  async reviewThreads(repo, number) {
    const [owner, name] = repo.split('/');
    const threads = [];
    let after = null;
    do {
      const answer = await this.api('graphql', { method: 'POST', payload: { query: THREADS, variables: { owner, name, number, after } } });
      if (answer?.errors?.length) throw new Error(`GitHub refused the review threads of ${repo}#${number}: ${answer.errors.map(e => e.message).join('; ')}`);
      const page = answer?.data?.repository?.pullRequest?.reviewThreads;
      if (!page) throw new Error(`GitHub gave no review threads for ${repo}#${number}`);
      threads.push(...page.nodes);
      after = page.pageInfo.hasNextPage ? page.pageInfo.endCursor : null;
    } while (after);
    return threads;
  }

  createReview(repo, number, payload) {
    return this.api(`repos/${repo}/pulls/${number}/reviews`, { method: 'POST', payload });
  }
}

function runGh(args, { input, env = {} } = {}) {
  return new Promise(done => {
    const child = execFile('gh', args, { timeout: TIMEOUT, maxBuffer: 1 << 28, env: { ...process.env, ...env } }, (error, stdout, stderr) => {
      done({ code: error ? (typeof error.code === 'number' ? error.code : 1) : 0, stdout: String(stdout), stderr: String(stderr || (error && !stdout ? error.message : '')), timedOut: !!(error && error.killed) });
    });
    if (input !== undefined) child.stdin.end(input);
  });
}
