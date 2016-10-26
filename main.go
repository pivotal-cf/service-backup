package main

import (
	"log"
	"os"
	"os/exec"

	"code.cloudfoundry.org/lager"

	"github.com/pivotal-cf-experimental/service-backup/backup"
	"github.com/pivotal-cf-experimental/service-backup/config"
	"github.com/pivotal-cf-experimental/service-backup/executor"
	"github.com/pivotal-cf-experimental/service-backup/scheduler"
	alerts "github.com/pivotal-cf/service-alerts-client/client"
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

	logFlags := log.Ldate | log.Ltime | log.Lmicroseconds | log.LUTC
	alertsLogger := log.New(os.Stderr, "[ServiceBackup] ", logFlags)

	var alertsClient *alerts.ServiceAlertsClient
	if backupConfig.Alerts != nil {
		alertsClient = alerts.New(backupConfig.Alerts.Config, alertsLogger)
	}

	scheduler := scheduler.NewScheduler(backupExecutor, backupConfig, alertsClient, logger)
	scheduler.Run()
}
