package main

import (
	"log"
	"os"

	"code.cloudfoundry.org/lager"

	alerts "github.com/pivotal-cf/service-alerts-client/client"
	"github.com/pivotal-cf/service-backup/config"
	"github.com/pivotal-cf/service-backup/executor"
	"github.com/pivotal-cf/service-backup/scheduler"
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

	uploader, err := upload.Initialize(&backupConfig, logger)
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
			uploader,
			backupConfig.SourceFolder,
			backupConfig.SourceExecutable,
			backupConfig.CleanupExecutable,
			backupConfig.ServiceIdentifierExecutable,
			backupConfig.ExitIfInProgress,
			logger,
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
