package ask

type diffHunk struct {
	rows     []string
	old, now [2]int
}
