import { test } from 'node:test';
import assert from 'node:assert/strict';
import { findConfigFile } from '../../src/config/config-file.mjs';

const only = (...files) => path => files.includes(path);

test('on macOS the config is in Application Support, unless XDG_CONFIG_HOME moves it', () => {
  const home = '/Users/lee';
  assert.deepEqual(findConfigFile({ env: {}, platform: 'darwin', home, exists: only() }),
    { path: '/Users/lee/Library/Application Support/lazyreview/config.yml', exists: false });
  assert.deepEqual(findConfigFile({ env: { XDG_CONFIG_HOME: '/Users/lee/.xdg' }, platform: 'darwin', home, exists: only() }),
    { path: '/Users/lee/.xdg/lazyreview/config.yml', exists: false });
  assert.deepEqual(findConfigFile({ env: { XDG_CONFIG_HOME: '~/dots' }, platform: 'darwin', home, exists: only() }),
    { path: '/Users/lee/dots/lazyreview/config.yml', exists: false }, 'a folder may start with ~');
  assert.deepEqual(findConfigFile({ env: { XDG_CONFIG_HOME: 'relative' }, platform: 'darwin', home, exists: only() }),
    { path: '/Users/lee/Library/Application Support/lazyreview/config.yml', exists: false }, 'a folder that is not a full path is passed over, as lazygit does');
});

test('the first config found wins, in the order the XDG base folders give, and the default is where a new file goes', () => {
  const home = '/Users/lee';
  const order = [
    '/Users/lee/Library/Application Support/lazyreview/config.yml',
    '/Users/lee/Library/Preferences/lazyreview/config.yml',
    '/Library/Application Support/lazyreview/config.yml',
    '/Library/Preferences/lazyreview/config.yml',
    '/Users/lee/.config/lazyreview/config.yml',
  ];
  for (let i = 0; i < order.length; i++) {
    assert.deepEqual(findConfigFile({ env: {}, platform: 'darwin', home, exists: only(...order.slice(i)) }), { path: order[i], exists: true });
  }
  assert.deepEqual(findConfigFile({ env: { XDG_CONFIG_DIRS: '/etc/one:relative:/etc/two' }, platform: 'darwin', home, exists: only('/etc/two/lazyreview/config.yml', order[4]) }),
    { path: '/etc/two/lazyreview/config.yml', exists: true }, 'XDG_CONFIG_DIRS replaces the folders searched after the default');
});

test('on Linux and Windows the config is in their own config folders', () => {
  assert.deepEqual(findConfigFile({ env: {}, platform: 'linux', home: '/home/lee', exists: only() }), { path: '/home/lee/.config/lazyreview/config.yml', exists: false });
  assert.deepEqual(findConfigFile({ env: {}, platform: 'linux', home: '/home/lee', exists: only('/etc/xdg/lazyreview/config.yml') }), { path: '/etc/xdg/lazyreview/config.yml', exists: true });
  const windows = { LOCALAPPDATA: 'C:\\Users\\lee\\AppData\\Local', APPDATA: 'C:\\Users\\lee\\AppData\\Roaming', PROGRAMDATA: 'C:\\ProgramData' };
  assert.deepEqual(findConfigFile({ env: windows, platform: 'win32', home: 'C:\\Users\\lee', exists: only() }), { path: 'C:\\Users\\lee\\AppData\\Local\\lazyreview\\config.yml', exists: false });
  assert.deepEqual(findConfigFile({ env: windows, platform: 'win32', home: 'C:\\Users\\lee', exists: only('C:\\Users\\lee\\AppData\\Roaming\\lazyreview\\config.yml') }),
    { path: 'C:\\Users\\lee\\AppData\\Roaming\\lazyreview\\config.yml', exists: true });
});

test('LAZYREVIEW_CONFIG_DIR names the folder, and LAZYREVIEW_CONFIG_FILE the file, which must exist', () => {
  const home = '/Users/lee';
  assert.deepEqual(findConfigFile({ env: { LAZYREVIEW_CONFIG_DIR: '/tmp/lr' }, platform: 'darwin', home, exists: only() }), { path: '/tmp/lr/config.yml', exists: false });
  assert.deepEqual(findConfigFile({ env: { LAZYREVIEW_CONFIG_DIR: '/tmp/lr' }, platform: 'darwin', home, exists: only('/tmp/lr/config.yml') }), { path: '/tmp/lr/config.yml', exists: true });
  const file = { LAZYREVIEW_CONFIG_FILE: '~/work.yml', LAZYREVIEW_CONFIG_DIR: '/tmp/lr' };
  assert.deepEqual(findConfigFile({ env: file, platform: 'darwin', home, exists: only('/Users/lee/work.yml') }), { path: '/Users/lee/work.yml', exists: true });
  assert.throws(() => findConfigFile({ env: file, platform: 'darwin', home, exists: only() }), /^Error: LAZYREVIEW_CONFIG_FILE names \/Users\/lee\/work\.yml, which does not exist$/);
});
