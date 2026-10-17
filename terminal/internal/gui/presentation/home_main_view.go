package presentation

import "github.com/connorb645/lazyreview/terminal/internal/commands/server"

// HomeMainView is what the home screen's main panel tells of: the list read
// from GitHub, or why it could not be read, the row selected and, when it is
// a pull request, its repository, its stack and itself, the stack being
// loaded, and the user's home folder, which a path under it is shown from.
type HomeMainView struct {
	List    *server.Pulls
	Failed  string
	Loading string
	Home    string
	Row     SideRow
	Repo    server.PullRepo
	Unit    server.Unit
	Pull    server.OpenPull
	Chosen  bool
}
