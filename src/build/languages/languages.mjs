/**
 * The languages the map reads types from, chosen by file extension.
 *
 * Each language has a reader (source text in, declarations and imports out)
 * and a family: types resolve across files of one family only, so Kotlin and
 * Java share one.
 */
import { readSwift } from './swift.mjs';
import { readKotlin } from './kotlin.mjs';
import { readJava } from './java.mjs';
import { readTypeScript } from './typescript.mjs';
import { readPython } from './python.mjs';

/** Per family: the words that declare a type, and the files that hold its code. */
export const FAMILIES = {
  swift: { keywords: ['class', 'struct', 'enum', 'protocol', 'actor'], globs: ['*.swift'] },
  jvm: { keywords: ['class', 'interface', 'object', 'enum', 'record'], globs: ['*.kt', '*.kts', '*.java'] },
  js: { keywords: ['class', 'interface', 'enum', 'type'], globs: ['*.ts', '*.tsx', '*.mts', '*.cts', '*.js', '*.jsx', '*.mjs', '*.cjs'] },
  python: { keywords: ['class'], globs: ['*.py', '*.pyi'] },
};

const LANGUAGES = [
  { name: 'Swift', family: 'swift', extensions: ['.swift'], read: readSwift },
  { name: 'Kotlin', family: 'jvm', extensions: ['.kt', '.kts'], read: readKotlin },
  { name: 'Java', family: 'jvm', extensions: ['.java'], read: readJava },
  { name: 'TypeScript', family: 'js', extensions: ['.ts', '.tsx', '.mts', '.cts', '.js', '.jsx', '.mjs', '.cjs'], read: readTypeScript },
  { name: 'Python', family: 'python', extensions: ['.py', '.pyi'], read: readPython },
];

/** The language a path is written in, or null when the map cannot read its types. */
export function languageFor(path) {
  const lower = path.toLowerCase();
  return LANGUAGES.find(l => l.extensions.some(e => lower.endsWith(e))) || null;
}
