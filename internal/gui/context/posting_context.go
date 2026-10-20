package context

// PostingContext is posting's popup for a pull request: the actions
// offered and the one chosen. Its confirmation, which shows what the
// action sends, is a context of its own on the focus stack.
type PostingContext struct {
	Pull    int
	Actions []string
	Choice  int
}
