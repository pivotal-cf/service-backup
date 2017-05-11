package main

import (
	"os"

	"code.cloudfoundry.org/lager"
	"github.com/pivotal-cf/service-backup/config"
	"github.com/pivotal-cf/service-backup/executor"
	"github.com/pivotal-cf/service-backup/upload"
)

func main() {
	logger := lager.NewLogger("ServiceBackup")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.INFO))

	configPath := os.Args[1]
	backupConfig, err := config.Parse(configPath, logger) //TODO pointer plz
	if err != nil {
		os.Exit(2)
	}

	backuper, err := upload.Initialize(&backupConfig, logger)
	if err != nil {
		os.Exit(2)
	}

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
		)
	}

	if err := backupExecutor.Execute(); err != nil {
		logger.Error("Error running backup", err)
		os.Exit(2)
	}
}
