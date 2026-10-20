package oscommands

import (
	"os/exec"
	"syscall"
)

// RunDetached runs program with args in a session of its own, with no
// terminal, so it goes on when the app that started it quits, and returns
// once it ends.
func RunDetached(program string, args []string) error {
	cmd := exec.Command(program, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	return cmd.Run()
}
