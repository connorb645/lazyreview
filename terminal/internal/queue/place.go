package queue

// Place is where a line of a file's patch sits: the hunk it is in and its
// row in the patch. An unchanged line has the same place on both sides, so
// two places compare in the order the diff shows them.
type Place struct {
	Hunk int
	Row  int
}
