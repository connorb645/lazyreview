/**
 * Where lazyreview's config file is, as lazygit finds its own.
 *
 * findConfigFile({env, platform, home, exists}) returns {path, exists}:
 *
 *   1. LAZYREVIEW_CONFIG_FILE, the file itself, which must exist;
 *   2. config.yml in LAZYREVIEW_CONFIG_DIR, used as it is;
 *   3. the first lazyreview/config.yml in the platform's config folders, in
 *      the order the XDG base folders give them;
 *   4. otherwise the platform's default, where a new file goes.
 *
 * On macOS the default is ~/Library/Application Support/lazyreview/config.yml,
 * on Linux ~/.config/lazyreview/config.yml and on Windows
 * %LOCALAPPDATA%\lazyreview\config.yml; XDG_CONFIG_HOME moves each, and
 * XDG_CONFIG_DIRS replaces the folders searched after it. `exists` says
 * whether a path is a file; it is the only way the lookup reads the disk.
 */
import { existsSync, statSync } from 'node:fs';
import { posix, win32 } from 'node:path';
import { english } from '../i18n/english.mjs';

const NAME = 'config.yml';
const FOLDER = 'lazyreview';

export function findConfigFile({ env, platform, home, exists = isFile }) {
  const path = platform === 'win32' ? win32 : posix;
  if (env.LAZYREVIEW_CONFIG_FILE) {
    const file = path.resolve(expandHome(env.LAZYREVIEW_CONFIG_FILE, home, path));
    if (!exists(file)) throw new Error(english.config.fileMissing(file));
    return { path: file, exists: true };
  }
  if (env.LAZYREVIEW_CONFIG_DIR) {
    const file = path.join(path.resolve(expandHome(env.LAZYREVIEW_CONFIG_DIR, home, path)), NAME);
    return { path: file, exists: exists(file) };
  }
  const folders = configFolders({ env, platform, home, path });
  const found = folders.map(folder => path.join(folder, FOLDER, NAME)).find(exists);
  if (found) return { path: found, exists: true };
  return { path: path.join(folders[0], FOLDER, NAME), exists: false };
}

// The platform's config folders, the one a new file goes in first, as the XDG base folders give them.
function configFolders({ env, platform, home, path }) {
  const one = (name, fallback) => absolute([env[name]], home, path)[0] || fallback;
  const list = (name, fallbacks) => {
    const given = absolute((env[name] || '').split(path.delimiter), home, path);
    return given.length ? given : absolute(fallbacks, home, path);
  };
  if (platform === 'win32') {
    const local = env.LOCALAPPDATA || path.join(home, 'AppData', 'Local');
    const roaming = env.APPDATA || path.join(home, 'AppData', 'Roaming');
    const shared = env.PROGRAMDATA || 'C:\\ProgramData';
    return unique([one('XDG_CONFIG_HOME', local), ...list('XDG_CONFIG_DIRS', [shared, roaming])]);
  }
  if (platform === 'darwin') {
    const support = path.join(home, 'Library', 'Application Support');
    return unique([one('XDG_CONFIG_HOME', support), ...list('XDG_CONFIG_DIRS', [
      path.join(home, 'Library', 'Preferences'), '/Library/Application Support', '/Library/Preferences', path.join(home, '.config'),
    ])]);
  }
  return unique([one('XDG_CONFIG_HOME', path.join(home, '.config')), ...list('XDG_CONFIG_DIRS', ['/etc/xdg'])]);
}

function absolute(paths, home, path) {
  return paths.filter(Boolean).map(p => expandHome(p, home, path)).filter(p => path.isAbsolute(p));
}

function expandHome(given, home, path) {
  if (given === '~') return home;
  if (given.startsWith('~/') || given.startsWith('~\\')) return path.join(home, given.slice(2));
  return given;
}

function unique(paths) {
  return [...new Set(paths)];
}

function isFile(path) {
  return existsSync(path) && statSync(path).isFile();
}
