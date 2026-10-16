package app

import (
	"os/exec"
	"runtime"
)

func openInBrowser(address string) error {
	command := "xdg-open"
	if runtime.GOOS == "darwin" {
		command = "open"
	}
	return exec.Command(command, address).Start()
}
