package processterminator

import (
	"os/exec"
	"sync"
	"syscall"
)

type ProcessTerminator struct {
	currentPgid int
	lock        sync.Mutex
}

func New() *ProcessTerminator {
	return &ProcessTerminator{}
}

func (pt *ProcessTerminator) Start(cmd *exec.Cmd) error {
	pt.lock.Lock()
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pgid: pt.currentPgid}
	err := cmd.Start()
	if err != nil {
		return err
	}
	if pt.currentPgid == 0 {
		pt.currentPgid = cmd.Process.Pid
	}
	pt.lock.Unlock()
	err = cmd.Wait()
	if err != nil {
		return err
	}
	return nil
}

func (pt *ProcessTerminator) Terminate() {
	if pt.currentPgid == 0 {
		return
	}
	syscall.Kill(-pt.currentPgid, syscall.SIGTERM)
}
