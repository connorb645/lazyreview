package oscommands

import (
	"os/exec"
	"testing"
)

func TestAServerProcessIsAskedToEndAndWaitedFor(t *testing.T) {
	server := exec.Command("sleep", "60")
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	ended := make(chan struct{})
	go func() {
		_ = server.Wait()
		close(ended)
	}()
	defer server.Process.Kill()
	processes := ServerProcess{}
	if !processes.Alive(server.Process.Pid) {
		t.Fatal("a running process is not alive")
	}
	processes.Stop(server.Process.Pid)
	<-ended
	if processes.Alive(server.Process.Pid) {
		t.Fatal("the process lives on after its stop")
	}
}
