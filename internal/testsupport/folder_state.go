package testsupport

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"testing"
)

// FolderState is what a folder holds, so a test can show nothing in it was
// written: each file and folder under root, its own .git folder included,
// by path, with its mode, size and time of change, without following
// links. Any write, even one that leaves the bytes as they were, changes it.
func FolderState(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || path == root {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		out[rel] = fmt.Sprintf("%v %d %d", info.Mode(), info.Size(), info.ModTime().UnixNano())
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}
