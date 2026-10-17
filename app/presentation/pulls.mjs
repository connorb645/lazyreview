/**
 * A stack's pull requests as pills, each in its pull request's colour.
 *
 * pullPill(pulls, pull, href) is one pull request's pill, a link when it has
 * an address, and pullPills(pulls, list) the pills of a list of them, or
 * nothing for none. `pulls` gives the colours, as createPulls does.
 */
import { esc } from '../lib/text.mjs';

export function pullPill(pulls, pull, href) {
  const style = ' style="--pc:' + pulls.pullColor(pull) + '"';
  return href
    ? '<a class="pp" href="' + esc(href) + '" target="_blank" rel="noopener"' + style + '>#' + esc(pull) + '</a>'
    : '<span class="pp"' + style + '>#' + esc(pull) + '</span>';
}

export function pullPills(pulls, list) {
  return list && list.length ? '<span class="pps">' + list.map(p => pullPill(pulls, p)).join('') + '</span>' : '';
}
