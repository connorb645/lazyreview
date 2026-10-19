/**
 * The open pull requests that involve the accounts in the config, for the
 * terminal app's home screen.
 *
 * new OpenPulls({store, config, github, exists}) reads the config with
 * `config()` at each list, and asks GitHub as each of its accounts by the
 * account's own token through `github({account})`. While the data home
 * still holds settings.json, which the config replaced, a list fails,
 * saying how to move it; `exists` says whether a file exists. list()
 * resolves with {config, accounts, repos}: `config` is the config file's
 * path, `accounts` each account as {login, error}, where `error` says why GitHub
 * could not be read as it, or is null, and `repos` each repository with
 * pull requests that involve an account, by name:
 *
 *   {host, name, account, hidden_by, units: [{review, members: [pull]}]}
 *
 * `hidden_by` holds the entries of the config's github.hiddenRepositories
 * that hide the repository from the home screen, or none.
 *
 * A pull request involves an account when its review is asked of the
 * account, or of one of its teams, while it is ready for review; when the
 * account opened it; or when the account commented on it or reviewed it.
 * A pull request in a stack is listed with the whole stack, bottom first,
 * as one unit, even its members that involve no account; any other is a
 * unit alone. A repository is read as the first account that found
 * something in it, and its units come by their lowest number. Each pull is
 * {number, title, author, url, draft, base, head, description, asked,
 * yours, commented, review}: `asked` names whom its review is asked of, the
 * account or its team, `review` the review holding it, or null, and a unit's
 * `review` is the one review holding every member, or null.
 */
import { existsSync } from 'node:fs';
import { PullRef } from '../models/pull-ref.mjs';
import { GitHub } from '../commands/github/github.mjs';
import { readUserConfig } from '../config/user-config.mjs';
import { english } from '../i18n/english.mjs';
import { stacks } from './stacks.mjs';
import { hiddenBy } from './hidden-by.mjs';

const OPEN = 'is:pr is:open archived:false';
const HOST = 'github.com';

export class OpenPulls {
  #store;
  #config;
  #github;
  #exists;

  constructor({ store, config = () => readUserConfig(), github = options => new GitHub(options), exists = existsSync }) {
    this.#store = store;
    this.#config = config;
    this.#github = github;
    this.#exists = exists;
  }

  async list() {
    const config = this.#config();
    const settings = this.#store.home.settingsFile;
    if (this.#exists(settings)) throw new Error(english.config.notMoved(settings, config.path));
    const { accounts, hiddenRepositories } = config.values.github;
    const read = await Promise.all(accounts.map(account => this.#read(account).then(
      repos => ({ login: account.login, error: null, repos }),
      error => ({ login: account.login, error: error.message, repos: [] }),
    )));
    const repos = new Map();
    for (const { repos: found } of read) {
      for (const repo of found) {
        const kept = repos.get(repo.name);
        if (!kept) { repos.set(repo.name, repo); continue; }
        for (const [number, involved] of repo.involved) kept.involved.set(number, joined(kept.involved.get(number), involved));
      }
    }
    return {
      config: config.path,
      accounts: read.map(({ login, error }) => ({ login, error })),
      repos: [...repos.values()].sort((a, b) => a.name.toLowerCase().localeCompare(b.name.toLowerCase()))
        .map(repo => ({ host: HOST, name: repo.name, account: repo.account, hidden_by: hiddenBy(hiddenRepositories, repo.name), units: this.#units(repo) })),
    };
  }

  async #read(account) {
    const gh = this.#github({ host: HOST, account: account.login });
    const searches = [
      [`${OPEN} draft:false user-review-requested:@me`, involved => involved.asked.push(account.login)],
      ...account.teams.map(team => [
        `${OPEN} draft:false team-review-requested:${team.name}${team.since ? ` created:>=${team.since}` : ''}`,
        involved => involved.asked.push(team.name),
      ]),
      [`${OPEN} author:@me`, involved => { involved.yours = true; }],
      [`${OPEN} commenter:@me`, involved => { involved.commented = true; }],
      [`${OPEN} reviewed-by:@me`, involved => { involved.commented = true; }],
    ];
    const found = await Promise.all(searches.map(([query]) => gh.search(query)));
    const byRepo = new Map();
    found.forEach((items, i) => {
      for (const item of items) {
        const name = item.repository_url.split('/').slice(-2).join('/');
        if (!byRepo.has(name)) byRepo.set(name, new Map());
        const involved = byRepo.get(name);
        if (!involved.has(item.number)) involved.set(item.number, { asked: [], yours: false, commented: false });
        searches[i][1](involved.get(item.number));
      }
    });
    const names = [...byRepo.keys()];
    const lists = await Promise.all(names.map(name => gh.openPulls(name)));
    return names.map((name, i) => ({ name, account: account.login, pulls: lists[i], involved: byRepo.get(name) }));
  }

  #units(repo) {
    const open = new Map(repo.pulls.map(p => [p.number, p]));
    const grouped = stacks(repo.pulls, repo.name);
    const units = new Map();
    for (const number of [...repo.involved.keys()].sort((a, b) => a - b)) {
      if (!open.has(number)) continue;
      const members = grouped.get(number);
      const key = members.map(m => m.number).join('+');
      if (!units.has(key)) units.set(key, members.map(p => this.#pull(repo, p)));
    }
    return [...units.values()].sort((a, b) => a[0].number - b[0].number).map(members => {
      const held = new Set(members.map(m => m.review));
      return { review: held.size === 1 && members[0].review ? members[0].review : null, members };
    });
  }

  #pull(repo, p) {
    const involved = repo.involved.get(p.number) || { asked: [], yours: false, commented: false };
    const found = this.#store.find(new PullRef(HOST, repo.name, p.number));
    return {
      number: p.number, title: p.title, author: p.user.login, url: p.html_url, draft: !!p.draft, base: p.base.ref, head: p.head.ref,
      description: (p.body || '').replace(/\r\n?/g, '\n'), asked: involved.asked, yours: involved.yours, commented: involved.commented,
      review: found && !found.review.moved ? found.ref.id : null,
    };
  }
}

function joined(a = { asked: [], yours: false, commented: false }, b) {
  return { asked: [...new Set([...a.asked, ...b.asked])], yours: a.yours || b.yours, commented: a.commented || b.commented };
}
