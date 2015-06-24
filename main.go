package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/cloudfoundry-incubator/cf-lager"
	"github.com/pivotal-cf-experimental/service-backup/backup"
	"github.com/pivotal-golang/lager"
	"github.com/tedsuo/ifrit"
	"gopkg.in/robfig/cron.v2"
)

const (
	awsCLIFlagName           = "aws-cli"
	sourceFolderFlagName     = "source-folder"
	destFolderFlagName       = "dest-folder"
	endpointURLFlagName      = "endpoint-url"
	awsAccessKeyFlagName     = "aws-access-key-id"
	awsSecretKeyFlagName     = "aws-secret-access-key"
	backupCreatorCmdFlagName = "backup-creator-cmd"
	cleanupCmdFlagName       = "cleanup-cmd"
	cronScheduleFlagName     = "cron-schedule"
)

var (
	logger lager.Logger
)

func main() {
	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	awsCLIBinaryPath := flags.String(awsCLIFlagName, "", "Path to AWS CLI")
	sourceFolder := flags.String(sourceFolderFlagName, "", "Local path to upload from (e.g.: /var/vcap/data)")
	destFolder := flags.String(destFolderFlagName, "", "Remote path to upload to (e.g.: s3://bucket-name/path/to/loc)")
	endpointURL := flags.String(endpointURLFlagName, "", "S3 endpoint URL")
	awsAccessKeyID := flags.String(awsAccessKeyFlagName, "", "AWS access key ID")
	awsSecretAccessKey := flags.String(awsSecretKeyFlagName, "", "AWS secret access key")
	backupCreatorCmd := flags.String(backupCreatorCmdFlagName, "", "Command for creating backup")
	cleanupCmd := flags.String(cleanupCmdFlagName, "", "Command for cleaning backup")
	cronSchedule := flags.String(cronScheduleFlagName, "", "Cron schedule for running backup. Leave empty to run only once.")

	cf_lager.AddFlags(flags)
	flags.Parse(os.Args[1:])

	logger, _ = cf_lager.New("ServiceBackup")

	if *awsAccessKeyID == "" && *awsSecretAccessKey == "" {
		logger.Info("AWS credentials not provided - skipping backup")
		os.Exit(0)
	}

	validateFlag(awsAccessKeyID, awsAccessKeyFlagName)
	validateFlag(awsSecretAccessKey, awsSecretKeyFlagName)
	validateFlag(awsCLIBinaryPath, awsCLIFlagName)
	validateFlag(sourceFolder, sourceFolderFlagName)
	validateFlag(destFolder, destFolderFlagName)
	validateFlag(endpointURL, endpointURLFlagName)
	validateFlag(backupCreatorCmd, backupCreatorCmdFlagName)
	validateFlag(cronSchedule, cronScheduleFlagName)

	executor := backup.NewExecutor(
		*awsCLIBinaryPath,
		*sourceFolder,
		*destFolder,
		*awsAccessKeyID,
		*awsSecretAccessKey,
		*endpointURL,
		*backupCreatorCmd,
		*cleanupCmd,
		logger,
	)

	scheduler := cron.New()

	_, err := scheduler.AddFunc(*cronSchedule, func() {
		executor.RunOnce()
	})

	if err != nil {
		logger.Fatal("Error scheduling job", err)
	}

	schedulerRunner := ifrit.RunFunc(func(signals <-chan os.Signal, ready chan<- struct{}) error {
		scheduler.Start()
		close(ready)

		<-signals
		scheduler.Stop()
		return nil
	})

	process := ifrit.Invoke(schedulerRunner)
	logger.Info("Service-backup Started")

	err = <-process.Wait()
	if err != nil {
		logger.Fatal("Error running", err)
	}
}

func validateFlag(value *string, flagName string) {
	if *value == "" {
		logger.Fatal("main.validation", fmt.Errorf("Flag %s not provided", flagName))
	}
}
