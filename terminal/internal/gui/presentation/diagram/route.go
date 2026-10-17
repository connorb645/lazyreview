package diagram

type route struct {
	from, to int
	words    string
	lit      bool
	faint    bool
	out      bool
	side     bool
	across   int
	gaps     []int
	lanes    []int
	via      int
	xa, xb   int
}
