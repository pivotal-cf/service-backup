package processterminator

import (
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
)

type ProcessTerminator struct {
	wg sync.WaitGroup
}

func New() *ProcessTerminator {
	return &ProcessTerminator{}
}

func (pt *ProcessTerminator) Start(cmd *exec.Cmd) error {
	sigUsr1Chan := make(chan os.Signal, 1)
	processExitChan := make(chan error, 1)
	signal.Notify(sigUsr1Chan, syscall.SIGUSR1)

	err := cmd.Start()
	if err != nil {
		signal.Stop(sigUsr1Chan)
		return err
	}
	pt.wg.Add(1)

	go func() {
		processExitChan <- cmd.Wait()
		pt.wg.Done()
	}()

	select {
	case <-sigUsr1Chan:
		cmd.Process.Signal(syscall.SIGTERM)
		return nil
	case retVal := <-processExitChan:
		signal.Stop(sigUsr1Chan)
		return retVal
	}
}

func (pt *ProcessTerminator) Terminate() {
	p, _ := os.FindProcess(os.Getpid())
	p.Signal(syscall.SIGUSR1)
	pt.wg.Wait()
}
