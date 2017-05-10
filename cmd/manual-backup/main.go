package main

import (
	"os"
	"os/exec"

	"code.cloudfoundry.org/lager"
	"github.com/pivotal-cf/service-backup/backup"
	"github.com/pivotal-cf/service-backup/config"
	"github.com/pivotal-cf/service-backup/executor"
)

func main() {
	logger := lager.NewLogger("ServiceBackup")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.INFO))

	configPath := os.Args[1]
	backupConfig, err := config.Parse(configPath, logger)
	if err != nil {
		os.Exit(2)
	}

	backuperFactory := config.NewBackuperCreator(&backupConfig)

	backupers, err := config.ParseDestinations(backupConfig, backuperFactory, logger)
	if err != nil {
		os.Exit(2)
	}

	var backupExecutor backup.Executor
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
			backup.NewMultiBackuper(backupers),
			backupConfig.SourceFolder,
			backupConfig.SourceExecutable,
			backupConfig.CleanupExecutable,
			backupConfig.ServiceIdentifierExecutable,
			backupConfig.ExitIfInProgress,
			logger,
			exec.Command,
			&backup.FileSystemSizeCalculator{},
		)
	}

	if err := backupExecutor.RunOnce(); err != nil {
		logger.Error("Error running backup", err)
		os.Exit(2)
	}
}
