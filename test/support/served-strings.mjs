/**
 * Lets the web app's modules load in Node as they do in the browser: they
 * import the strings from /i18n/english.mjs, where the server serves them,
 * and this maps that address to the module itself. Import it before any
 * module of the app.
 */
import { registerHooks } from 'node:module';
import { join, dirname } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

const STRINGS = pathToFileURL(join(dirname(fileURLToPath(import.meta.url)), '..', '..', 'src', 'i18n', 'english.mjs')).href;

registerHooks({
  resolve: (specifier, context, next) => next(specifier === '/i18n/english.mjs' ? STRINGS : specifier, context),
});
