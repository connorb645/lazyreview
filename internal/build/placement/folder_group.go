package placement

// FolderGroup is some paths grouped by their folder: Folder is "" for the
// repository root; Others marks everything beside a dominant subfolder;
// Direct marks the files left directly in a folder whose subfolders were
// split off.
type FolderGroup struct {
	Folder string
	Others bool
	Paths  []string
	Direct bool
}
