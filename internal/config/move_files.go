package config

// MoveFiles is how the move reads and writes the disk: whether a path is a
// file, a file's text, writing a new file that must not exist yet, renaming
// and removing a file, and making a folder with its parents.
type MoveFiles interface {
	Exists(path string) bool
	Read(path string) (string, error)
	Write(path, text string) error
	Rename(from, to string) error
	Remove(path string) error
	Folder(path string) error
}
