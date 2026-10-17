package context

// HomePanel is a panel of the home screen and its rows.
type HomePanel struct {
	Title   string
	Number  int
	Repo    int
	Waiting int
	Rows    SideContext
}
