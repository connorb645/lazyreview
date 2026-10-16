package app

// port is one end of a route on a card's top or bottom border, with the
// column of where the route goes next, which orders the ends along it.
type port struct {
	route  int
	source bool
	toward int
}
