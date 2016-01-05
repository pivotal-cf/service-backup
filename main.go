package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/cloudfoundry-incubator/cf-lager"
	"github.com/pivotal-cf-experimental/service-backup/backup"
	"github.com/pivotal-cf-experimental/service-backup/s3"
	"github.com/pivotal-golang/lager"
	"github.com/tedsuo/ifrit"
	"gopkg.in/robfig/cron.v2"
)

const (
	sourceFolderFlagName     = "source-folder"
	destBucketFlagName       = "dest-bucket"
	destPathFlagName         = "dest-path"
	endpointURLFlagName      = "endpoint-url"
	awsAccessKeyFlagName     = "aws-access-key-id"
	awsSecretKeyFlagName     = "aws-secret-access-key"
	backupCreatorCmdFlagName = "backup-creator-cmd"
	cleanupCmdFlagName       = "cleanup-cmd"
	cronScheduleFlagName     = "cron-schedule"
	awsCmdPathFlagName       = "aws-cli-path"
)

var (
	logger lager.Logger
)

func main() {
	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	sourceFolder := flags.String(sourceFolderFlagName, "", "Local path to upload from (e.g.: /var/vcap/data)")
	destBucket := flags.String(destBucketFlagName, "", "Remote bucket to upload. No preceding or trailing slashes. E.g. my-remote-bucket")
	destPath := flags.String(destPathFlagName, "", "Remote directory path inside bucket to upload to. No preceding or trailing slashes. E.g. remote/path/inside/bucket")
	endpointURL := flags.String(endpointURLFlagName, "", "S3 endpoint URL")
	awsAccessKeyID := flags.String(awsAccessKeyFlagName, "", "AWS access key ID")
	awsSecretAccessKey := flags.String(awsSecretKeyFlagName, "", "AWS secret access key")
	backupCreatorCmd := flags.String(backupCreatorCmdFlagName, "", "Command for creating backup")
	cleanupCmd := flags.String(cleanupCmdFlagName, "", "Command for cleaning backup")
	cronSchedule := flags.String(cronScheduleFlagName, "", "Cron schedule for running backup. Leave empty to run only once.")
	awsCmdPath := flags.String(awsCmdPathFlagName, "aws", "Path to AWS CLI binary. Optional. Defaults to looking on $PATH.")

	cf_lager.AddFlags(flags)
	flags.Parse(os.Args[1:])

	logger, _ = cf_lager.New("ServiceBackup")

	if *awsAccessKeyID == "" && *awsSecretAccessKey == "" {
		logger.Info("AWS credentials not provided - skipping backup")
		os.Exit(0)
	}

	validateFlag(awsAccessKeyID, awsAccessKeyFlagName)
	validateFlag(awsSecretAccessKey, awsSecretKeyFlagName)
	validateFlag(sourceFolder, sourceFolderFlagName)
	validateFlag(destBucket, destBucketFlagName)
	validateFlag(endpointURL, endpointURLFlagName)
	validateFlag(backupCreatorCmd, backupCreatorCmdFlagName)
	validateFlag(cronSchedule, cronScheduleFlagName)

	s3Client := s3.NewCliClient(
		*awsCmdPath,
		*endpointURL,
		*awsAccessKeyID,
		*awsSecretAccessKey,
		logger,
	)

	executor := backup.NewExecutor(
		s3Client,
		*sourceFolder,
		*destBucket,
		*destPath,
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
