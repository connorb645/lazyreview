import { test } from 'node:test';
import assert from 'node:assert/strict';
import { pathPattern } from '../../src/build/placement/path-pattern.mjs';
import { groupByFolder } from '../../src/build/placement/folder-grouping.mjs';
import { placeFiles, humanize, commonFolder, whereFiles } from '../../src/build/placement/placement.mjs';
import { readConfig } from '../../src/build/config/config.mjs';

test('path patterns: literals cover their folder, wildcards stay within a folder', () => {
  assert.equal(pathPattern('src/app').test('src/app/a.ts'), true);
  assert.equal(pathPattern('src/app').test('src/app'), true);
  assert.equal(pathPattern('src/app').test('src/apple.ts'), false);
  assert.equal(pathPattern('src/*.ts').test('src/a.ts'), true);
  assert.equal(pathPattern('src/*.ts').test('src/x/a.ts'), false);
  assert.equal(pathPattern('src/**/*.ts').test('src/a.ts'), true);
  assert.equal(pathPattern('src/**/*.ts').test('src/x/y/a.ts'), true);
  assert.equal(pathPattern('**').test('anything/at/all'), true);
  assert.equal(pathPattern('a?c.md').test('abc.md'), true);
  assert.ok(pathPattern('src/app/Main.swift').specificity > pathPattern('src/app/').specificity);
  assert.ok(pathPattern('src/app/').specificity > pathPattern('src/**').specificity);
});

test('folder grouping keeps a small diff together and splits a big one by folder', () => {
  assert.deepEqual(groupByFolder(['a/b/c.swift']).map(g => g.folder), ['a/b']);
  const small = ['src/x/a.ts', 'src/x/b.ts', 'test/a.test.ts'];
  assert.deepEqual(groupByFolder(small, { splitAbove: 8 }).map(g => g.folder), ['']);
  const big = [...Array(20)].map((_, i) => `app/f${i}.ts`).concat([...Array(12)].map((_, i) => `lib/g${i}.ts`), ['README.md']);
  assert.deepEqual(groupByFolder(big, { splitAbove: 8, balanced: true }).map(g => [g.folder, g.paths.length, g.direct]),
    [['', 1, true], ['app', 20, false], ['lib', 12, false]]);
});

test('balanced grouping does not split a folder just to peel off a stray file', () => {
  const paths = [...Array(30)].map((_, i) => `big/main/f${i}.ts`).concat(['big/other/one.ts']);
  assert.deepEqual(groupByFolder(paths, { splitAbove: 8, balanced: true }).map(g => g.folder), ['big']);
});

test('a folder one subfolder dominates splits into that subfolder\'s parts, with the rest set apart', () => {
  const paths = [...Array(14)].map((_, i) => `Modules/Search/Sources/f${i}.swift`)
    .concat([...Array(8)].map((_, i) => `Modules/Basket/Sources/g${i}.swift`), ['App/App.xcodeproj/project.pbxproj', 'App/App/Main.swift']);
  assert.deepEqual(groupByFolder(paths, { splitAbove: 8, balanced: true }).map(g => [g.folder, g.paths.length]),
    [['Modules/Basket/Sources', 8], ['Modules/Search/Sources', 14], [null, 2]]);
  const { components } = placeFiles(paths, readConfig(undefined), { rootTitle: 'app' });
  assert.deepEqual(components.map(c => [c.title, c.size]), [['Search Sources', 14], ['Basket Sources', 8], ['Other files', 2]]);
  const flat = [...Array(30)].map((_, i) => `src/f${i}.ts`).concat(['README.md', 'docs/a.md']);
  assert.deepEqual(groupByFolder(flat, { splitAbove: 8, balanced: true }).map(g => g.folder), [''], 'a dominant folder with nothing to split into stays whole');
});

test('a small diff with no config is one component named after the repository', () => {
  const { components } = placeFiles(['Sources/Core/A.swift', 'Tests/CoreTests/ATests.swift'], readConfig(undefined), { rootTitle: 'core' });
  assert.deepEqual(components.map(c => [c.id, c.title]), [['root', 'core']]);
  assert.deepEqual(components[0].packages.map(p => p.title), ['Sources/Core', 'Tests/Core Tests'], 'a package of folders is titled as their path, with no spaces round its slashes');
});

test('without a config, files are grouped by folder into components and packages', () => {
  const paths = ['Sources/Core/Log/A.swift', 'Sources/Core/Log/B.swift', 'Sources/Core/Log/E.swift', 'Sources/Core/Net/C.swift', 'Sources/Core/Net/D.swift',
    'Tests/CoreTests/ATests.swift', 'Tests/CoreTests/BTests.swift', 'Tests/CoreTests/CTests.swift', 'Tests/CoreTests/DTests.swift'];
  const { components, placement } = placeFiles(paths, readConfig(undefined));
  assert.deepEqual(components.map(c => [c.id, c.title, c.tag]), [['sources-core', 'Core', 'Sources/Core/'], ['tests-coretests', 'Core Tests', 'Tests/CoreTests/']]);
  assert.deepEqual(components[0].packages.map(p => [p.title, p.path]), [['Log', 'Sources/Core/Log/'], ['Net', 'Sources/Core/Net/']]);
  assert.deepEqual(components[1].packages.map(p => p.title), ['Core Tests']);
  assert.equal(placement.get('Sources/Core/Net/D.swift').package, components[0].packages[1].id);
  assert.equal(components[0].packages.length, 2);
});

test('config patterns place files, the most specific pattern winning', () => {
  const config = readConfig({
    components: [
      { id: 'app', title: 'App', packages: [
        { id: 'screens', title: 'Screens', paths: ['App/'] },
        { id: 'settings', title: 'Settings', paths: ['App/SettingsScreen.swift'] },
      ] },
      { id: 'rest', title: 'Everything else', paths: ['**'] },
    ],
  });
  const { components, placement } = placeFiles(['App/Home.swift', 'App/SettingsScreen.swift', 'README.md', 'tools/x/y.py'], config);
  assert.deepEqual(placement.get('App/Home.swift'), { component: 'app', package: 'screens' });
  assert.deepEqual(placement.get('App/SettingsScreen.swift'), { component: 'app', package: 'settings' });
  assert.equal(placement.get('README.md').component, 'rest');
  assert.equal(placement.get('tools/x/y.py').component, 'rest');
  assert.deepEqual(components.map(c => c.id), ['app', 'rest']);
});

test('files the config does not name are grouped by folder after the named components', () => {
  const config = readConfig({ components: [{ id: 'app', title: 'App', paths: ['App/'] }] });
  const paths = ['App/a.ts', ...[...Array(5)].map((_, i) => `lib/x${i}.ts`)];
  const { components } = placeFiles(paths, config);
  assert.deepEqual(components.map(c => [c.id, c.declared, c.size]), [['app', true, 1], ['lib', false, 5]]);
});

test('context files go to context components, and the rest to one Unchanged code component', () => {
  const config = readConfig({
    components: [
      { id: 'core', title: 'Core', paths: ['src/'] },
      { id: 'sdk', title: 'SDK around it', context: true, packages: [{ id: 'net', title: 'Network', paths: ['src/net/'] }] },
    ],
  });
  const { components, placement } = placeFiles(['src/net/Client.ts', 'src/db/Store.ts'], config, { context: true });
  assert.deepEqual(placement.get('src/net/Client.ts'), { component: 'sdk', package: 'net' });
  assert.equal(placement.get('src/db/Store.ts').component, 'context');
  assert.deepEqual(components.map(c => [c.id, c.title, c.context]), [['sdk', 'SDK around it', true], ['context', 'Unchanged code', true]]);
});

test('names and folders for headings', () => {
  assert.equal(humanize('ShopFront'), 'Shop Front');
  assert.equal(humanize('log_upload-v2'), 'log upload v2');
  assert.equal(humanize('HTTPClient'), 'HTTP Client');
  assert.equal(commonFolder(['a/b/c.ts', 'a/b/d/e.ts']), 'a/b/');
  assert.equal(commonFolder(['a.ts', 'b/c.ts']), '');
  assert.equal(whereFiles(['Tests/A/x.swift', 'Tests/A/y.swift', 'Package.swift', 'scripts/run.sh']), 'Tests/ · Package.swift · scripts/');
});
