package process

import (
	"bufio"
	"errors"
	"io"
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

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		m.lock.Unlock()
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		m.lock.Unlock()
		return nil, err
	}
	multi := io.MultiReader(stdout, stderr)
	combinedOutput := []byte{}

	err = cmd.Start()
	if err != nil {
		m.lock.Unlock()
		return nil, err
	}
	m.wg.Add(1)
	m.lock.Unlock()

	go func() {
		defer m.wg.Done()
		scanner := bufio.NewScanner(multi)
		scanner.Split(splitOn16k)
		for scanner.Scan() {
			combinedOutput = append(combinedOutput, scanner.Bytes()...)
		}
		processExitChan <- cmd.Wait()
	}()

	select {
	case <-m.killAll:
		cmd.Process.Signal(syscall.SIGTERM)
		stdout.Close()
		stderr.Close()
		<-processExitChan
		return combinedOutput, errors.New("SIGTERM propagated to child process")
	case retVal := <-processExitChan:
		return combinedOutput, retVal
	}
}

func (m *Manager) Terminate() {
	m.lock.Lock()
	close(m.killAll)
	m.wg.Wait()
	m.lock.Unlock()
}

func splitOn16k(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	if atEOF {
		return len(data), data, nil
	}

	maxBufSize := 16 * 1024

	if len(data) < maxBufSize {
		return 0, nil, nil
	}

	return maxBufSize, data[0:maxBufSize], nil
}
