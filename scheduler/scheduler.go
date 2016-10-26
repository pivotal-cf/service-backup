package scheduler

import (
	"os"

	"code.cloudfoundry.org/lager"
	"github.com/pivotal-cf-experimental/service-backup/backup"
	"github.com/pivotal-cf-experimental/service-backup/config"
	"github.com/tedsuo/ifrit"
	cron "gopkg.in/robfig/cron.v2"
)

type Scheduler struct {
	cronSchedule *cron.Cron
	logger       lager.Logger
}

func NewScheduler(executor backup.Executor, backupConfig config.BackupConfig, logger lager.Logger) Scheduler {
	scheduler := cron.New()

	_, err := scheduler.AddFunc(backupConfig.CronSchedule, func() {
		backupErr := executor.RunOnce()
		if backupErr != nil {
			logger.Info("Alerts not configured.", lager.Data{})
		}
	})
	if err != nil {
		logger.Error("Error scheduling job", err)
		os.Exit(2)
	}

	return Scheduler{cronSchedule: scheduler, logger: logger}
}

func (s Scheduler) Run() {
	runner := ifrit.RunFunc(func(signals <-chan os.Signal, ready chan<- struct{}) error {
		s.cronSchedule.Start()
		close(ready)

		<-signals
		s.cronSchedule.Stop()
		return nil
	})

	process := ifrit.Invoke(runner)
	s.logger.Info("Service-backup Started")

	err := <-process.Wait()
	if err != nil {
		s.logger.Error("Error running", err)
		os.Exit(2)
	}
}
