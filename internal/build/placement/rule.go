package placement

type rule struct {
	pattern   Pattern
	component string
	pkg       string
	order     int
}
