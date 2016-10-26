package main

import (
	"os"
	"os/exec"

	"code.cloudfoundry.org/lager"

	"github.com/pivotal-cf-experimental/service-backup/backup"
	"github.com/pivotal-cf-experimental/service-backup/config"
	"github.com/pivotal-cf-experimental/service-backup/dummy"
	"github.com/pivotal-cf-experimental/service-backup/executor"
	"github.com/pivotal-cf-experimental/service-backup/scheduler"
)

func main() {
	logger := lager.NewLogger("ServiceBackup")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.INFO))

	configPath := os.Args[1]
	backupConfig, err := config.Parse(configPath, logger)
	if err != nil {
		os.Exit(2)
	}
	backupers := config.ParseDestinations(backupConfig, logger)

	var backupExecutor backup.Executor
	if backupConfig.NoDestinations() {
		logger.Info("No destination provided - skipping backup")
		// Default cronSchedule to monthly if not provided when destination is also not provided
		// This is needed to successfully run the dummy executor and not exit
		if backupConfig.CronSchedule == "" {
			backupConfig.CronSchedule = "@monthly"
		}
		backupExecutor = dummy.NewDummyExecutor(logger)
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

	scheduler := scheduler.NewScheduler(backupExecutor, backupConfig, logger)
	scheduler.Run()
}
