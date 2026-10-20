package diff

// Read is a change as ReadDiff reads it: the commits it is read between,
// the diff split per file, and each file ready to draw.
type Read struct {
	BaseCommit string
	HeadCommit string
	MergeBase  string
	Chunks     []Chunk
	Files      []File
}
