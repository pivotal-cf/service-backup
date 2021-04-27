// Copyright (C) 2016-Present Pivotal Software, Inc. All rights reserved.
// This program and the accompanying materials are made available under the terms of the under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package scheduler

import (
	"fmt"
	"os"

	"code.cloudfoundry.org/lager"
	alerts "github.com/pivotal-cf/service-alerts-client/client"
	"github.com/pivotal-cf/service-backup/config"
	"github.com/pivotal-cf/service-backup/executor"
	cron "github.com/robfig/cron/v3"
	"github.com/tedsuo/ifrit"
)

type Scheduler struct {
	cronSchedule *cron.Cron
	logger       lager.Logger
}

func NewScheduler(e executor.Executor, backupConfig config.BackupConfig, alertsClient *alerts.ServiceAlertsClient, logger lager.Logger) Scheduler {
	scheduler := cron.New()

	_, err := scheduler.AddFunc(backupConfig.CronSchedule, func() {
		backupErr := e.Execute()
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

		// ifrit does not call Notify on this channel
		// it will wait indefinitely here
		<-signals
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

func (s Scheduler) Stop() {
	s.cronSchedule.Stop()
}
