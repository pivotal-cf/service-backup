package backup

import (
	"os/exec"

	"code.cloudfoundry.org/lager"
)

type Executor interface {
	RunOnce() error
}

//go:generate counterfeiter -o backupfakes/fake_provider_factory.go . ProviderFactory
type ProviderFactory interface {
	ExecCommand(string, ...string) *exec.Cmd
}

//ExecCommand fakeable exec.Command
type ExecCommand func(string, ...string) *exec.Cmd

//go:generate counterfeiter -o backupfakes/fake_backuper.go . Backuper
type Backuper interface {
	Upload(localPath string, sessionLogger lager.Logger) error
	Name() string
}

type RemotePathGenerator interface {
	RemotePathWithDate() string
}
