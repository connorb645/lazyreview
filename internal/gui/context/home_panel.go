package context

// HomePanel is a panel of the home screen and its rows. Hidden says the
// config hides its repository.
type HomePanel struct {
	Title   string
	Number  int
	Repo    int
	Waiting int
	Hidden  bool
	Rows    SideContext
}
