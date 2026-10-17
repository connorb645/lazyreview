/**
 * Loads a branch of a repository on this machine into a review, for a change
 * that has no pull request yet, such as a branch about to be pushed.
 *
 * load({repo, base, head, title, config, context}) reads the change from the
 * merge base of `base` and `head` to `head`, as a pull request would show it,
 * straight from the repository in the folder `repo`, builds what the app
 * draws, and writes the review. `head` must name a branch: the review is
 * kept by repository and branch, so loading the same branch again, from any
 * worktree of the repository, updates it in place. `title` names the review,
 * `config` is a components config and `context` whether to draw the
 * unchanged code the change touches; each left out keeps what the review
 * had, or the default, and the title's default is the branch's name.
 *
 * A branch review is read only: it has no queue and posts nothing. Every file
 * is written or none, and load returns {ref, review, counts, notes}: `counts`
 * of the files, lines, components and types it drew. refresh(ref) loads a
 * branch review again, from the folder and the base it was last loaded with.
 */
import { realpathSync } from 'node:fs';
import { resolve } from 'node:path';
import { BranchRef, repositoryName } from '../models/branch-ref.mjs';
import { Git } from '../commands/git/git.mjs';
import { readConfig } from '../build/config/config.mjs';
import { buildPage } from '../build/page-builder.mjs';
import { branchSubject } from '../build/page/heading.mjs';
import { defaultElk } from '../build/layout/elk.mjs';

export class BranchLoader {
  #store;
  #git;
  #elk;
  #now;

  constructor({ store, git = repo => new Git(repo), elk = defaultElk(), now = () => new Date().toISOString() }) {
    this.#store = store;
    this.#git = git;
    this.#elk = elk;
    this.#now = now;
  }

  async load({ repo, base, head, title, config, context }) {
    const path = resolve(repo);
    const git = this.#git(path);
    const shared = realpathSync(git.sharedFolder());
    const branch = git.branchOf(head);
    if (!branch) throw new Error(`"${head}" is not a branch of ${path}; give a branch, so loading it again updates the same review`);
    const ref = BranchRef.of(shared, branch);
    const previous = this.#store.read(ref);
    const settled = {
      title: title !== undefined ? title : previous?.given_title ?? null,
      context: context !== undefined ? context : previous?.context ?? false,
      config: config !== undefined ? config : this.#store.readConfig(ref) || undefined,
    };
    const name = repositoryName(shared);
    const built = await buildPage({
      git,
      base,
      members: [{ head }],
      baseBranch: base,
      title: settled.title || branch,
      subject: branchSubject(branch),
      config: readConfig(settled.config),
      context: settled.context,
      rootTitle: name,
      elk: this.#elk,
    });

    return this.#store.change([ref], async () => {
      const review = {
        id: ref.id, kind: 'branch', repo: name, path, title: built.page.model.heading.title, given_title: settled.title,
        base: { ref: base, sha: git.resolve(base) }, head: { ref: branch, sha: built.page.model.heading.head }, start: built.starts[0],
        context: settled.context, loaded_at: this.#now(), notes: built.notes,
      };
      const folder = this.#store.home.review(ref);
      return {
        files: [
          [folder.page, JSON.stringify(built.page)],
          this.#store.recordWrite(ref, review),
          ...(settled.config !== undefined ? [[folder.config, JSON.stringify(settled.config, null, 2) + '\n']] : []),
        ],
        result: { ref, review, counts: built.counts, notes: built.notes },
      };
    });
  }

  async refresh(ref) {
    const review = this.#store.read(ref);
    if (!review) throw new Error(`there is no review ${ref.id}`);
    return this.load({ repo: review.path, base: review.base.ref, head: review.head.ref });
  }
}
