/**
 * fromHTML(html) is the element the HTML text of one element makes, ready
 * to put in the page. A table's row makes a row.
 */
export function fromHTML(html) {
  const template = document.createElement('template');
  template.innerHTML = html;
  return template.content.firstElementChild;
}
