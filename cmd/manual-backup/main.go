// Copyright (C) 2016-Present Pivotal Software, Inc. All rights reserved.
// This program and the accompanying materials are made available under the terms of the under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package main

import (
	"os"
	"os/signal"
	"syscall"

	"code.cloudfoundry.org/lager/v3"
	"github.com/pivotal-cf/service-backup/config"
	"github.com/pivotal-cf/service-backup/executor"
	"github.com/pivotal-cf/service-backup/process"
	"github.com/pivotal-cf/service-backup/upload"
)

func main() {
	sigterms := make(chan os.Signal, 1)
	signal.Notify(sigterms, syscall.SIGTERM, syscall.SIGINT)

	logger := lager.NewLogger("ServiceBackup")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.INFO))

	configPath := os.Args[1]
	backupConfig, err := config.Parse(configPath, logger)
	if err != nil {
		logger.Error("failed to parse config", err)
		os.Exit(2)
	}

	backuper, err := upload.Initialize(&backupConfig, logger)
	if err != nil {
		logger.Error("failed to initialize uploader", err)
		os.Exit(2)
	}

	terminator := process.NewManager()
	go func() {
		<-sigterms
		terminator.Terminate()
		logger.Info("All backup processes terminated. Exiting")
		os.Exit(1)
	}()

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
			backuper,
			backupConfig.SourceFolder,
			backupConfig.SourceExecutable,
			backupConfig.CleanupExecutable,
			backupConfig.ServiceIdentifierExecutable,
			backupConfig.ExitIfInProgress,
			logger,
			terminator,
		)
	}
	if err := backupExecutor.Execute(); err != nil {
		logger.Error("Error running backup", err)
		os.Exit(2)
	}
}
