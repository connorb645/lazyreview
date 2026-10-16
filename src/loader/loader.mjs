/**
 * Loads a pull request, or a stack of them, into a review.
 *
 * load({links, config, context, account}) reads each pull request from
 * GitHub, fetches their heads into the repository's clone, builds what the
 * app draws, and writes the review. `links` are pull request addresses of one
 * repository, bottom first; two or more are a stack, each based on the head
 * branch of the one before, reviewed together. `config` is a components
 * config, `context` whether to draw the unchanged code the change touches,
 * and `account` the GitHub account to act as; each left out keeps what the
 * review had, or the default.
 *
 * The review threads already on GitHub are read with the pull requests and
 * kept as their own record, each on the line the app shows it on; when
 * GitHub will not give them, the load goes on without them and a note says
 * why.
 *
 * Loading again refreshes a review in place. A stack's review lives with its
 * lowest pull request that has one, and the queues of other reviews of its
 * pull requests move into it, those reviews then pointing at it. Every file
 * is written or none, and it returns {ref, review, counts, notes}: `counts`
 * of the files, lines, components and types it drew.
 */
import { PullRef } from '../github/pull-ref.mjs';
import { GitHub } from '../github/github.mjs';
import { Git } from '../build/git/git.mjs';
import { readConfig } from '../build/config/config.mjs';
import { buildPage } from '../build/page-builder.mjs';
import { pullsSubject } from '../build/page/heading.mjs';
import { defaultElk } from '../build/layout/elk.mjs';
import { regroup } from '../queue/regroup.mjs';
import { readThreads, placeThreads } from '../review/github-threads.mjs';

const DESCRIPTION = 2000;

export class Loader {
  #store;
  #clones;
  #github;
  #elk;
  #now;

  constructor({ store, clones, github = options => new GitHub(options), elk = defaultElk(), now = () => new Date().toISOString() }) {
    this.#store = store;
    this.#clones = clones;
    this.#github = github;
    this.#elk = elk;
    this.#now = now;
  }

  async load({ links, config, context, account }) {
    const refs = readLinks(links);
    const { host, repo } = refs[0];
    const pulls = refs.map(r => r.pull);
    const { anchor, absorbed } = this.#place(host, repo, pulls);
    const previous = this.#store.read(anchor);
    const settled = {
      account: account !== undefined ? account : previous?.account || null,
      context: context !== undefined ? context : previous?.context ?? false,
      config: config !== undefined ? config : this.#store.readConfig(anchor) || undefined,
    };
    const checked = readConfig(settled.config);
    const gh = this.#github({ host, account: settled.account });

    const found = [];
    for (const pull of pulls) found.push(await gh.pull(repo, pull));
    if (found.length > 1) checkChained(found, repo);
    const me = await gh.login();
    const threads = [];
    const unread = [];
    for (const f of found) {
      try {
        threads.push(...readThreads(f.number, await gh.reviewThreads(repo, f.number)));
      } catch (error) {
        unread.push(`#${f.number}'s review threads on GitHub are not shown, as they could not be read: ${error.message}`);
      }
    }
    const leaving = {};
    for (const m of previous && !previous.moved ? previous.members : []) {
      if (!pulls.includes(m.pull)) leaving[m.pull] = stateOf(await gh.pull(repo, m.pull));
    }
    const bottom = found[0];
    const clone = await this.#clones.prepare({
      host, repo, url: (await gh.repository(repo)).clone_url, base: bottom.base.ref, pulls,
      commits: [bottom.base.sha, ...found.map(f => f.head.sha)], environment: await gh.environment(),
    });
    const built = await buildPage({
      git: new Git(clone),
      base: bottom.base.sha,
      members: found.map(f => ({ pull: f.number, head: f.head.sha })),
      baseBranch: bottom.base.ref,
      title: found.length === 1 ? bottom.title : `Stack #${pulls[0]} to #${pulls[pulls.length - 1]}`,
      subject: pullsSubject(pulls),
      config: checked,
      context: settled.context,
      rootTitle: repo.split('/')[1],
      elk: this.#elk,
    });
    const fresh = found.map((f, i) => ({
      pull: f.number, head: f.head.sha, start: built.starts[i], title: f.title, url: f.html_url, author: f.user.login, base: f.base.ref,
      description: (f.body || '').slice(0, DESCRIPTION), ...stateOf(f),
    }));

    return this.#store.change([anchor, ...absorbed], async () => {
      const now = this.#now();
      const grouped = regroup({
        previous: this.#store.read(anchor),
        fresh,
        leaving,
        absorbed: absorbed.map(ref => ({ ref, review: this.#store.read(ref) })),
        anchor: anchor.id,
        now,
      });
      const review = {
        id: anchor.id, host, repo, account: settled.account, me, title: built.page.model.heading.title,
        base: { ref: bottom.base.ref, sha: bottom.base.sha }, context: settled.context, loaded_at: now,
        notes: [...built.notes, ...unread], moved: null, members: grouped.members, earlier: grouped.earlier,
      };
      const folder = this.#store.home.review(anchor);
      return {
        files: [
          [folder.page, JSON.stringify(built.page)],
          [folder.github, JSON.stringify(placeThreads(threads, built.page.stack), null, 2) + '\n'],
          this.#store.recordWrite(anchor, review),
          ...(settled.config !== undefined ? [[folder.config, JSON.stringify(settled.config, null, 2) + '\n']] : []),
          ...grouped.moved.map(m => this.#store.recordWrite(m.ref, m.review)),
        ],
        result: { ref: anchor, review, counts: built.counts, notes: [...review.notes, ...grouped.notes] },
      };
    });
  }

  /**
   * Loads a review again from GitHub, as its pull requests now stand. A
   * stack's members that merged or closed leave it, unless none is open.
   */
  async refresh(ref) {
    const review = this.#store.read(ref);
    if (!review) throw new Error(`there is no review ${ref.id}`);
    if (review.moved) throw new Error(`this review moved to ${review.moved.to}; load that one`);
    const gh = this.#github({ host: review.host, account: review.account });
    const open = [];
    for (const m of review.members) if (stateOf(await gh.pull(review.repo, m.pull)).state === 'open') open.push(m);
    const keep = review.members.length > 1 && open.length ? open : review.members;
    return this.load({ links: keep.map(m => ref.withPull(m.pull).id) });
  }

  // A stack's review lives with its lowest pull request that has a review;
  // every other review holding one of its pull requests moves into it.
  #place(host, repo, pulls) {
    const holding = this.#store.inRepository(host, repo)
      .filter(e => !e.review.moved && e.review.members.some(m => pulls.includes(m.pull)));
    const home = pulls.map(p => holding.find(e => e.review.members.some(m => m.pull === p))).find(Boolean);
    const anchor = home ? home.ref : new PullRef(host, repo, pulls[0]);
    const absorbed = holding.filter(e => e.ref.id !== anchor.id);
    for (const e of absorbed) {
      const others = e.review.members.filter(m => !pulls.includes(m.pull)).map(m => `#${m.pull}`);
      if (others.length) {
        throw new Error(`${e.ref.id} also reviews ${others.join(', ')}, which this load does not name; load the whole stack, bottom first`);
      }
    }
    return { anchor, absorbed: absorbed.map(e => e.ref) };
  }
}

function readLinks(links) {
  if (!links.length) throw new Error('give the pull request to load, or every pull request of a stack, bottom first');
  const refs = links.map(link => PullRef.parse(link));
  const other = refs.find(r => r.host !== refs[0].host || r.repo !== refs[0].repo);
  if (other) throw new Error(`a stack is in one repository, but ${other.host}/${other.repo} is not ${refs[0].host}/${refs[0].repo}`);
  if (new Set(refs.map(r => r.pull)).size !== refs.length) throw new Error('a pull request is named twice');
  return refs;
}

function checkChained(found, repo) {
  const problems = [];
  for (let i = 1; i < found.length; i++) {
    const below = found[i - 1], above = found[i];
    if (above.base.ref !== below.head.ref || below.head.repo?.full_name !== repo) {
      problems.push(`#${above.number} is based on ${above.base.ref}, not on #${below.number}'s branch ${below.head.ref}`);
    }
  }
  if (problems.length) throw new Error(`these are not a stack, bottom first: ${problems.join('; ')}`);
}

function stateOf(pull) {
  if (pull.merged_at) return { state: 'merged', ended_at: pull.merged_at };
  return { state: pull.state, ended_at: pull.closed_at || null };
}
