package executor

import (
	"code.cloudfoundry.org/lager"
	"github.com/pivotal-cf/service-backup/backup"
)

type dummyExecutor struct {
	logger lager.Logger
}

//NewDummyExecutor ...
func NewDummyExecutor(
	logger lager.Logger,
) backup.Executor {
	return &dummyExecutor{
		logger: logger,
	}
}

func (d *dummyExecutor) RunOnce() error {
	d.logger.Info("Backups Disabled")
	return nil
}
