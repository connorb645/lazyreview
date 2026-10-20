package github

// Run runs gh with args, input on its standard input and env added to this
// process's environment.
type Run func(args []string, input []byte, env []string) Ran
