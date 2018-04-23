package process

import (
	"os/exec"
	"sync"
	"syscall"
)

type Manager struct {
	wg sync.WaitGroup
	killAll chan struct{}
}

func NewManager() *Manager {
	pt := &Manager{}
	pt.killAll = make(chan struct{})
	return pt
}

func (m *Manager) Start(cmd *exec.Cmd, started chan struct{}) error {
	processExitChan := make(chan error, 1)

	err := cmd.Start()
	if err != nil {
		return err
	}
	m.wg.Add(1)
	close(started)

	go func() {
		defer m.wg.Done()
		processExitChan <- cmd.Wait()
	}()

	select {
	case <-m.killAll:
		cmd.Process.Signal(syscall.SIGTERM)
		return nil
	case retVal := <-processExitChan:
		return retVal
	}
}

func (m *Manager) Terminate() {
	close(m.killAll)
	m.wg.Wait()
}
