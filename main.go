// Copyright (C) 2016-Present Pivotal Software, Inc. All rights reserved.
// This program and the accompanying materials are made available under the terms of the under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"code.cloudfoundry.org/lager"

	alerts "github.com/pivotal-cf/service-alerts-client/client"
	"github.com/pivotal-cf/service-backup/config"
	"github.com/pivotal-cf/service-backup/executor"
	"github.com/pivotal-cf/service-backup/process"
	"github.com/pivotal-cf/service-backup/scheduler"
	"github.com/pivotal-cf/service-backup/upload"
)

func main() {
	sigterms := make(chan os.Signal, 1)
	signal.Notify(sigterms, syscall.SIGTERM)

	logger := lager.NewLogger("ServiceBackup")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.INFO))
	configPath := os.Args[1]
	backupConfig, err := config.Parse(configPath, logger)
	if err != nil {
		os.Exit(2)
	}

	uploader, err := upload.Initialize(&backupConfig, logger)
	if err != nil {
		os.Exit(2)
	}

	manager := process.NewManager()

	var backupExecutor executor.Executor
	if backupConfig.NoDestinations() {
		logger.Info("No destination provided - skipping backup")
		// Default cronSchedule to monthly if not provided when destination is also not provided
		// This is needed to successfully run the dummy executor and not exit
		if backupConfig.CronSchedule == "" {
			backupConfig.CronSchedule = "@monthly"
		}
		backupExecutor = executor.NewDummyExecutor(logger)
	} else {
		backupExecutor = executor.NewExecutor(
			uploader,
			backupConfig.SourceFolder,
			backupConfig.SourceExecutable,
			backupConfig.CleanupExecutable,
			backupConfig.ServiceIdentifierExecutable,
			backupConfig.ExitIfInProgress,
			logger,
			manager,
		)
	}

	logFlags := log.Ldate | log.Ltime | log.Lmicroseconds | log.LUTC
	alertsLogger := log.New(os.Stderr, "[ServiceBackup] ", logFlags)

	var alertsClient *alerts.ServiceAlertsClient
	if backupConfig.Alerts != nil {
		alertsClient = alerts.New(backupConfig.Alerts.Config, alertsLogger)
	}

	scheduler := scheduler.NewScheduler(backupExecutor, backupConfig, alertsClient, logger)
	go func() {
		<-sigterms
		scheduler.Stop()
		manager.Terminate()
		logger.Info("All backup processes terminated. Exiting")
		os.Exit(1)
	}()
	scheduler.Run()
}
