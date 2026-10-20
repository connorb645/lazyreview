package oscommands

import (
	"errors"
	"syscall"
	"time"
)

const (
	serverStopWait = 5 * time.Second
	serverStopPoll = 100 * time.Millisecond
)

// ServerProcess is a server's process on this machine, by its process id.
type ServerProcess struct{}

// Alive says whether the process lives.
func (ServerProcess) Alive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

// Stop asks the process to end, and waits up to five seconds for it to go.
func (p ServerProcess) Stop(pid int) {
	_ = syscall.Kill(pid, syscall.SIGTERM)
	for deadline := time.Now().Add(serverStopWait); p.Alive(pid) && time.Now().Before(deadline); {
		time.Sleep(serverStopPoll)
	}
}
