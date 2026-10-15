/**
 * Serves the app's own files: the page, its scripts and styles, and the
 * vendored Markdown library and font.
 *
 * staticFile(urlPath) returns {path, type} for a file the app may load, or
 * null. Only files inside the app's folder, and the library and font files it
 * uses from the vendor folder, are ever served.
 */
import { join, dirname, normalize, sep, extname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { existsSync, statSync } from 'node:fs';

const ROOT = join(dirname(fileURLToPath(import.meta.url)), '..', '..');
const APP = join(ROOT, 'app');
const VENDOR = ['marked-12.0.2.min.js', 'plus-jakarta-sans-5.3.0-latin.woff2', 'plus-jakarta-sans-5.3.0-latin-ext.woff2'];
const TYPES = { '.html': 'text/html; charset=utf-8', '.mjs': 'text/javascript; charset=utf-8', '.js': 'text/javascript; charset=utf-8', '.css': 'text/css; charset=utf-8', '.svg': 'image/svg+xml', '.woff2': 'font/woff2' };

export const PAGE = join(APP, 'index.html');

export function staticFile(urlPath) {
  const vendored = VENDOR.find(name => urlPath === `/vendor/${name}`);
  if (vendored) return { path: join(ROOT, 'vendor', vendored), type: TYPES[extname(vendored)] };
  if (!urlPath.startsWith('/app/')) return null;
  const path = normalize(join(APP, decodeURIComponent(urlPath.slice('/app/'.length))));
  if (!path.startsWith(APP + sep) || !TYPES[extname(path)] || !existsSync(path) || !statSync(path).isFile()) return null;
  return { path, type: TYPES[extname(path)] };
}
