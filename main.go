package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/cloudfoundry-incubator/cf-lager"
	"github.com/pivotal-cf-experimental/service-backup/azure"
	"github.com/pivotal-cf-experimental/service-backup/backup"
	"github.com/pivotal-cf-experimental/service-backup/s3"
	"github.com/pivotal-cf-experimental/service-backup/scp"
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

	// SCP specific
	sshHostFlagName           = "ssh-host"
	sshPortFlagName           = "ssh-port"
	sshUserFlagName           = "ssh-user"
	sshPrivateKeyPathFlagName = "ssh-private-key-path"

	// Azure specific
	azureStorageAccountFlagName   = "azure-storage-account"
	azureStorageAccessKeyFlagName = "azure-storage-access-key"
	azureContainerFlagName        = "azure-container"
	azureBlobStoreBaseUrlFlagName = "azure-blob-store-base-url"
)

var (
	logger lager.Logger
)

func main() {
	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	backupType := os.Args[1]

	sourceFolder := flags.String(sourceFolderFlagName, "", "Local path to upload from (e.g.: /var/vcap/data)")
	backupCreatorCmd := flags.String(backupCreatorCmdFlagName, "", "Command for creating backup")
	cleanupCmd := flags.String(cleanupCmdFlagName, "", "Command for cleaning backup")
	cronSchedule := flags.String(cronScheduleFlagName, "", "Cron schedule for running backup. Leave empty to run only once.")
	destPath := flags.String(destPathFlagName, "", "Remote directory path inside bucket to upload to. No preceding or trailing slashes. E.g. remote/path/inside/bucket")

	// S3 specific
	destBucket := flags.String(destBucketFlagName, "", "Remote bucket to upload. No preceding or trailing slashes. E.g. my-remote-bucket")
	endpointURL := flags.String(endpointURLFlagName, "", "S3 endpoint URL")
	awsAccessKeyID := flags.String(awsAccessKeyFlagName, "", "AWS access key ID")
	awsSecretAccessKey := flags.String(awsSecretKeyFlagName, "", "AWS secret access key")
	awsCmdPath := flags.String(awsCmdPathFlagName, "aws", "Path to AWS CLI binary. Optional. Defaults to looking on $PATH.")

	// SCP specific
	sshHost := flags.String(sshHostFlagName, "", "SCP destination hostname")
	sshPort := flags.Int(sshPortFlagName, 22, "SCP destination port")
	sshUser := flags.String(sshUserFlagName, "", "SCP destination user")
	sshPrivateKey := flags.String(sshPrivateKeyPathFlagName, "", "SCP destination user identity file")

	// Azure specific
	azureStorageAccount := flags.String(azureStorageAccountFlagName, "", "Azure storage account name")
	azureStorageAccessKey := flags.String(azureStorageAccessKeyFlagName, "", "Azure storage account access key")
	azureContainer := flags.String(azureContainerFlagName, "", "Azure storage account container")
	azureBlobStoreBaseUrl := flags.String(azureBlobStoreBaseUrlFlagName, "", "Azure blob store base URL (optional)")

	cf_lager.AddFlags(flags)
	flags.Parse(os.Args[2:])

	logger, _ = cf_lager.New("ServiceBackup")

	var backuper backup.Backuper
	var remotePath string
	switch backupType {
	case "s3":
		validateFlag(awsAccessKeyID, awsAccessKeyFlagName)
		validateFlag(awsSecretAccessKey, awsSecretKeyFlagName)
		validateFlag(destBucket, destBucketFlagName)

		remotePath = fmt.Sprintf("%s/%s", *destBucket, *destPath)
		backuper = s3.NewCliClient(
			*awsCmdPath,
			*endpointURL,
			*awsAccessKeyID,
			*awsSecretAccessKey,
			logger,
		)

	case "scp":
		validateFlag(sshHost, sshHostFlagName)
		validateIntFlag(sshPort, sshPortFlagName)
		validateFlag(sshUser, sshUserFlagName)
		validateFlag(sshPrivateKey, sshPrivateKeyPathFlagName)

		remotePath = *destPath
		backuper = scp.New(*sshHost, *sshPort, *sshUser, *sshPrivateKey, logger)

	case "azure":
		validateFlag(azureStorageAccessKey, azureStorageAccessKeyFlagName)
		validateFlag(azureStorageAccount, azureStorageAccountFlagName)
		validateFlag(azureContainer, azureContainerFlagName)

		remotePath = *destPath
		backuper = azure.New(*azureStorageAccessKey, *azureStorageAccount, *azureContainer, *azureBlobStoreBaseUrl, logger)

	case "skip":
		logger.Info("No destination provided - skipping backup")
		return

	default:
		logger.Fatal(fmt.Sprintf("Unknown destination type: %s", backupType), nil)
	}

	validateFlag(sourceFolder, sourceFolderFlagName)
	validateFlag(backupCreatorCmd, backupCreatorCmdFlagName)
	validateFlag(cronSchedule, cronScheduleFlagName)

	executor := backup.NewExecutor(
		backuper,
		*sourceFolder,
		remotePath,
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

func validateIntFlag(value *int, flagName string) {
	if *value == 0 {
		logger.Fatal("main.validation", fmt.Errorf("Flag %s not provided", flagName))
	}
}
