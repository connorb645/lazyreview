// Command land lands a branch on main, on every remote and in the checkout
// its working folder is in, as a fast-forward. Run it as make land
// BRANCH=<branch>. It checks everything first, says what it will do, and
// ends saying where main is.
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/connorb645/lazyreview/internal/land"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	branch := ""
	if len(args) > 0 {
		branch = args[0]
	}
	landing := land.NewLanding(".", nil)
	plan, err := landing.Plan(branch)
	if err == nil {
		for _, line := range land.Describe(plan) {
			fmt.Fprintln(stdout, line)
		}
		var landed string
		if landed, err = landing.Land(plan); err == nil {
			fmt.Fprintln(stdout, landed)
			return 0
		}
	}
	for _, line := range strings.Split(err.Error(), "\n") {
		fmt.Fprintln(stderr, "land: "+line)
	}
	return 1
}
