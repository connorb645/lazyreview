package diff

// File is one file of a change, ready to draw: its chunk's paths, status
// and hunks, its counts of added and removed lines, its text at the head
// and its text at the base where it is read, and whether it starts closed.
// On a stack, Pulls are the pull requests that changed it and ByPull the
// lines each added and removed.
type File struct {
	Path      string
	OldPath   string
	Status    string
	Binary    bool
	Hunks     []Hunk
	Adds      int
	Dels      int
	Text      *string
	BaseText  *string
	Collapsed bool
	Pulls     []int
	ByPull    map[int][2]int
}
