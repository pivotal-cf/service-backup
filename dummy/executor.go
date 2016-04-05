package dummy

import (
	"github.com/pivotal-cf-experimental/service-backup/backup"
	"github.com/pivotal-golang/lager"
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
