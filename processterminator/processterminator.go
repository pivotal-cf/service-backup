package processterminator

import (
	"os/exec"
	"sync"
	"syscall"
)

type ProcessTerminator struct {
	wg sync.WaitGroup
	killAll chan struct{}
}

func New() *ProcessTerminator {
	pt := &ProcessTerminator{}
	pt.killAll = make(chan struct{})
	return pt
}

func (pt *ProcessTerminator) Start(cmd *exec.Cmd, started chan struct{}) error {
	processExitChan := make(chan error, 1)

	err := cmd.Start()
	if err != nil {
		return err
	}
	pt.wg.Add(1)
	close(started)

	go func() {
		processExitChan <- cmd.Wait()
		pt.wg.Done()
	}()

	select {
	case <-pt.killAll:
		cmd.Process.Signal(syscall.SIGTERM)
		return nil
	case retVal := <-processExitChan:
		return retVal
	}
}

func (pt *ProcessTerminator) Terminate() {
	close(pt.killAll)
	pt.wg.Wait()
}
