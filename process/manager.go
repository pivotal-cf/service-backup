package process

import (
	"bytes"
	"errors"
	"os/exec"
	"sync"
	"syscall"
)

//go:generate counterfeiter -o fakes/process_manager.go . ProcessManager
type ProcessManager interface {
	Start(*exec.Cmd) ([]byte, error)
}

type Manager struct {
	wg      sync.WaitGroup
	killAll chan struct{}
	lock    sync.Mutex
}

func (m *Manager) isBeingShutdown() bool {
	select {
	case <-m.killAll:
		return true
	default:
		return false
	}
}

func NewManager() *Manager {
	pt := &Manager{}
	pt.killAll = make(chan struct{})
	return pt
}

func (m *Manager) Start(cmd *exec.Cmd) ([]byte, error) {
	m.lock.Lock()
	if m.isBeingShutdown() {
		return nil, errors.New("Shutdown in progress")
	}

	processExitChan := make(chan error, 1)

	var cmdOutput bytes.Buffer

	cmd.Stdout = &cmdOutput
	cmd.Stderr = &cmdOutput

	err := cmd.Start()
	if err != nil {
		m.lock.Unlock()
		return nil, err
	}
	m.wg.Add(1)
	m.lock.Unlock()

	go func() {
		defer m.wg.Done()
		processExitChan <- cmd.Wait()
	}()

	select {
	case <-m.killAll:
		cmd.Process.Signal(syscall.SIGTERM)
		<-processExitChan
		return cmdOutput.Bytes(), errors.New("SIGTERM propagated to child process")
	case retVal := <-processExitChan:
		return cmdOutput.Bytes(), retVal
	}
}

func (m *Manager) Terminate() {
	m.lock.Lock()
	close(m.killAll)
	m.wg.Wait()
	m.lock.Unlock()
}
