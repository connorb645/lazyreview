package app

// stretch is a route's run across one gap below a row: the route, which of
// its gaps this is, the columns where it comes in from above and goes on
// below, and whether it turns back into the row it left.
type stretch struct {
	route, at, upper, lower int
	turn                    bool
}
