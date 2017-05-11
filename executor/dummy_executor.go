package executor

import "code.cloudfoundry.org/lager"

type dummyExecutor struct {
	logger lager.Logger
}

//NewDummyExecutor ...
func NewDummyExecutor(
	logger lager.Logger,
) *dummyExecutor {
	return &dummyExecutor{
		logger: logger,
	}
}

func (d *dummyExecutor) Execute() error {
	d.logger.Info("Backups Disabled")
	return nil
}
