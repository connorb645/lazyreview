package oscommands

import (
	"os"
	"path/filepath"
)

// WritePage writes a page for the browser to a file of its own, named by
// name, in lazyreview's folder of the machine's temporary folder, whole:
// the browser finds it as it was or as it is now. It returns the file's
// path.
func WritePage(name, page string) (string, error) {
	folder := filepath.Join(os.TempDir(), "lazyreview")
	if err := os.MkdirAll(folder, 0o700); err != nil {
		return "", err
	}
	path := filepath.Join(folder, name+".html")
	partial, err := os.CreateTemp(folder, name+".*.tmp")
	if err != nil {
		return "", err
	}
	_, err = partial.WriteString(page)
	if closed := partial.Close(); err == nil {
		err = closed
	}
	if err == nil {
		err = os.Rename(partial.Name(), path)
	}
	if err != nil {
		_ = os.Remove(partial.Name())
		return "", err
	}
	return path, nil
}
