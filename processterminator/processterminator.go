package processterminator

import (
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

type ProcessTerminator struct {
}

func New() *ProcessTerminator {
	return &ProcessTerminator{}
}

func (pt *ProcessTerminator) Start(cmd *exec.Cmd) error {
	sigTermChan := make(chan os.Signal, 1)
	processExitChan := make(chan error, 1)
	signal.Notify(sigTermChan, syscall.SIGUSR1)

	err := cmd.Start()
	if err != nil {
		signal.Stop(sigTermChan)
		return err
	}

	go func() {
		processExitChan <- cmd.Wait()
	}()

	select {
	case <-sigTermChan:
		cmd.Process.Signal(syscall.SIGTERM)
		return nil
	case retVal := <-processExitChan:
		signal.Stop(sigTermChan)
		return retVal
	}
}

func (pt *ProcessTerminator) Terminate() {
	p, _ := os.FindProcess(os.Getpid())
	p.Signal(syscall.SIGUSR1)
}
