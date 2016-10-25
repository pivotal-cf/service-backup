package executor

import (
	"os/exec"

	"code.cloudfoundry.org/lager"
	"github.com/pivotal-cf-experimental/service-backup/backup"
	"github.com/pivotal-cf-experimental/service-backup/config"
	"github.com/pivotal-cf-experimental/service-backup/dummy"
)

type ExecutorFactory struct {
	backupConfig config.BackupConfig
	backuper     backup.MultiBackuper
	logger       lager.Logger
}

func NewExecutoryFactory(backupConfig config.BackupConfig, backuper backup.MultiBackuper, logger lager.Logger) ExecutorFactory {
	return ExecutorFactory{backupConfig, backuper, logger}
}

func (e ExecutorFactory) NewExecutor() backup.Executor {
	if e.backupConfig.NoDestinations() {
		return dummy.NewDummyExecutor(e.logger)
	}

	var calculator = &backup.FileSystemSizeCalculator{}

	return NewExecutor(
		e.backuper,
		e.backupConfig.SourceFolder,
		e.backupConfig.SourceExecutable,
		e.backupConfig.CleanupExecutable,
		e.backupConfig.ServiceIdentifierExecutable,
		e.backupConfig.ExitIfInProgress,
		e.logger,
		exec.Command,
		calculator,
	)
}
