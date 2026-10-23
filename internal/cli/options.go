// Package cli reads lazyreview's command line: the command, its options and
// arguments, the addresses `terminal` opens, and the address of a review.
package cli

// Options is the command line as read: its command, the arguments that are
// not options, in order, and each option given, the last one given winning,
// apart from --head, which may be given once for each branch, in order. A
// valued option that is not given is nil.
type Options struct {
	Command    string
	Positional []string
	Account    *string
	Config     *string
	Repo       *string
	Base       *string
	Heads      []string
	Title      *string
	Context    bool
	NoContext  bool
	Check      bool
}
