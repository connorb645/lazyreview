// Package oscommands runs the programs of the machine the terminal app runs
// on, and writes the pages it opens in the browser.
package oscommands

import (
	"os/exec"
	"runtime"
)

// OpenInBrowser opens an address in the browser, and returns once the
// program that opens it has started.
func OpenInBrowser(address string) error {
	command := "xdg-open"
	if runtime.GOOS == "darwin" {
		command = "open"
	}
	return exec.Command(command, address).Start()
}
