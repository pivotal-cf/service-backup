package scheduler

import (
	"fmt"
	"os"

	"code.cloudfoundry.org/lager"
	alerts "github.com/pivotal-cf/service-alerts-client/client"
	"github.com/pivotal-cf/service-backup/backup"
	"github.com/pivotal-cf/service-backup/config"
	"github.com/pivotal-cf/service-backup/executor"
	"github.com/tedsuo/ifrit"
	cron "gopkg.in/robfig/cron.v2"
)

type Scheduler struct {
	cronSchedule *cron.Cron
	logger       lager.Logger
}

func NewScheduler(e backup.Executor, backupConfig config.BackupConfig, alertsClient *alerts.ServiceAlertsClient, logger lager.Logger) Scheduler {
	scheduler := cron.New()

	_, err := scheduler.AddFunc(backupConfig.CronSchedule, func() {
		backupErr := e.RunOnce()
		if backupErr != nil {
			if alertsClient == nil {
				logger.Info("Alerts not configured.", lager.Data{})
			} else {
				logger.Info("Sending alert.", lager.Data{})
				content := fmt.Sprintf("A backup run has failed with the following error: %s", backupErr)
				if err := alertsClient.SendServiceAlert(backupConfig.Alerts.ProductName, "Service Backup Failed", backupErr.(executor.ServiceInstanceError).ServiceInstanceID, content); err != nil {
					logger.Error("error sending service alert", err, lager.Data{})
					return
				}
				logger.Info("Sent alert.", lager.Data{})
			}
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
