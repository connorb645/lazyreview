package layout

type packagePair struct {
	from, to     string
	count, tests int
	labels       []string
	links        [][2]string
}
