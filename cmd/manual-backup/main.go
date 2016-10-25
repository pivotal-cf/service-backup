package main

import (
	"os"

	"code.cloudfoundry.org/lager"
	"github.com/pivotal-cf-experimental/service-backup/backup"
	"github.com/pivotal-cf-experimental/service-backup/config"
	"github.com/pivotal-cf-experimental/service-backup/executor"
)

var (
	logger lager.Logger
)

func main() {
	logger := lager.NewLogger("ServiceBackup")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.INFO))

	configPath := os.Args[1]
	backupConfig := config.Parse(configPath, logger)
	backupers := config.ParseDestinations(backupConfig, logger)

	executorFactory := executor.NewExecutoryFactory(backupConfig, backup.NewMultiBackuper(backupers), logger)
	executor := executorFactory.NewExecutor()

	if err := executor.RunOnce(); err != nil {
		logger.Error("Error running backup", err)
		os.Exit(2)
	}
}
