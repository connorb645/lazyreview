/**
 * Links each declaration to the one it sits inside.
 *
 * Takes declarations with a `span` ([from, to] offsets) and returns them in
 * source order with `parent` (the enclosing declaration's name, or null),
 * `owner` (the outermost enclosing declaration's name, or its own) and `top`
 * (true when nothing encloses it and `isTopLevel(declaration)` agrees).
 */
export function nestDeclarations(declarations, isTopLevel) {
  const sorted = [...declarations].sort((a, b) => a.span[0] - b.span[0] || b.span[1] - a.span[1]);
  const open = [];
  return sorted.map(declaration => {
    while (open.length && open[open.length - 1].span[1] <= declaration.span[0]) open.pop();
    const parent = open.length ? open[open.length - 1] : null;
    const outermost = open.length ? open[0] : null;
    const placed = {
      ...declaration,
      parent: parent ? parent.name : null,
      owner: outermost ? outermost.name : declaration.name,
      top: !parent && isTopLevel(declaration),
    };
    open.push(placed);
    return placed;
  });
}
