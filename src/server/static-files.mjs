/**
 * Serves the app's own files: the page, its scripts and styles, and the
 * vendored Markdown library.
 *
 * staticFile(urlPath) returns {path, type} for a file the app may load, or
 * null. Only files inside the app's folder, and the one library it uses from
 * the vendor folder, are ever served.
 */
import { join, dirname, normalize, sep, extname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { existsSync, statSync } from 'node:fs';

const ROOT = join(dirname(fileURLToPath(import.meta.url)), '..', '..');
const APP = join(ROOT, 'app');
const VENDOR = { '/vendor/marked-12.0.2.min.js': join(ROOT, 'vendor', 'marked-12.0.2.min.js') };
const TYPES = { '.html': 'text/html; charset=utf-8', '.mjs': 'text/javascript; charset=utf-8', '.js': 'text/javascript; charset=utf-8', '.css': 'text/css; charset=utf-8', '.svg': 'image/svg+xml' };

export const PAGE = join(APP, 'index.html');

export function staticFile(urlPath) {
  if (VENDOR[urlPath]) return { path: VENDOR[urlPath], type: TYPES['.js'] };
  if (!urlPath.startsWith('/app/')) return null;
  const path = normalize(join(APP, decodeURIComponent(urlPath.slice('/app/'.length))));
  if (!path.startsWith(APP + sep) || !TYPES[extname(path)] || !existsSync(path) || !statSync(path).isFile()) return null;
  return { path, type: TYPES[extname(path)] };
}
