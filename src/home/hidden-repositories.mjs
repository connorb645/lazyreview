/**
 * The repositories the home screen leaves out: the config's
 * github.hiddenRepositories, read at each ask and changed in the file.
 *
 * new HiddenRepositories({config, save}) reads the config with `config()`,
 * and saves a change to the list with `save(path, key, change)`, as
 * saveListChange does. entries() is the list as the config gives it.
 * change({hide, show, repos}) puts the entry `hide` on the list, a
 * repository as owner/name or every repository of an owner as owner/*, and
 * takes each entry of `show` off it, in the file the config was read from;
 * one change runs at a time. It resolves with {config, repos}: the file's
 * path, and each repository named in `repos` with the entries that now hide
 * it, as {name, hidden_by}.
 */
import { readUserConfig } from '../config/user-config.mjs';
import { saveListChange } from '../config/list-save.mjs';
import { PATTERNS } from '../config/schema.mjs';
import { english } from '../i18n/english.mjs';
import { hiddenBy } from './hidden-by.mjs';

const KEY = 'github.hiddenRepositories';
const REPOSITORY = new RegExp(PATTERNS.repository);

export class HiddenRepositories {
  #config;
  #save;
  #last = Promise.resolve();

  constructor({ config = () => readUserConfig(), save = saveListChange } = {}) {
    this.#config = config;
    this.#save = save;
  }

  entries() {
    return this.#config().values.github.hiddenRepositories;
  }

  change({ hide = null, show = [], repos }) {
    const run = () => {
      if (hide !== null && !REPOSITORY.test(hide)) throw new Error(english.config.notAnEntry(hide));
      this.#save(this.#config().path, KEY, { add: hide === null ? [] : [hide], remove: show });
      const read = this.#config();
      return { config: read.path, repos: repos.map(name => ({ name, hidden_by: hiddenBy(read.values.github.hiddenRepositories, name) })) };
    };
    const done = this.#last.then(run, run);
    this.#last = done.catch(() => {});
    return done;
  }
}
