package models

// FileEntry is a file of the change, or an unchanged file of its context,
// with where it is drawn. On a stack, Pulls are the pull requests that
// changed it, and ByPull the lines each added and removed of those the
// combined diff shows. Lines counts an unchanged file's lines.
type FileEntry struct {
	Path      string         `json:"path"`
	OldPath   string         `json:"oldPath"`
	Status    string         `json:"status"`
	Adds      int            `json:"adds"`
	Dels      int            `json:"dels"`
	Binary    bool           `json:"binary"`
	Collapsed bool           `json:"collapsed"`
	Pulls     []int          `json:"pulls"`
	ByPull    map[int][2]int `json:"byPull"`
	Component string         `json:"component"`
	Package   string         `json:"package"`
	Lines     int            `json:"lines"`
}
