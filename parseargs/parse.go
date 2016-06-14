package parseargs

import (
	"flag"
	"fmt"
	"os/exec"
	"strconv"

	"github.com/cloudfoundry-incubator/cf-lager"
	"github.com/pivotal-cf-experimental/service-backup/azure"
	"github.com/pivotal-cf-experimental/service-backup/backup"
	"github.com/pivotal-cf-experimental/service-backup/dummy"
	"github.com/pivotal-cf-experimental/service-backup/s3"
	"github.com/pivotal-cf-experimental/service-backup/scp"
	"github.com/pivotal-golang/lager"
)

const (
	sourceFolderFlagName         = "source-folder"
	destBucketFlagName           = "dest-bucket"
	destPathFlagName             = "dest-path"
	endpointURLFlagName          = "endpoint-url"
	awsAccessKeyFlagName         = "aws-access-key-id"
	awsSecretKeyFlagName         = "aws-secret-access-key"
	backupCreatorCmdFlagName     = "backup-creator-cmd"
	cleanupCmdFlagName           = "cleanup-cmd"
	serviceIdentifierCmdFlagName = "service-identifier-cmd"
	exitIfInProgressFlagName     = "exit-if-in-progress"
	//CronScheduleFlagName ...
	CronScheduleFlagName = "cron-schedule"
	awsCmdPathFlagName   = "aws-cli-path"

	// SCP specific
	sshHostFlagName           = "ssh-host"
	sshPortFlagName           = "ssh-port"
	sshUserFlagName           = "ssh-user"
	sshPrivateKeyPathFlagName = "ssh-private-key-path"

	// Azure specific
	azureStorageAccountFlagName   = "azure-storage-account"
	azureStorageAccessKeyFlagName = "azure-storage-access-key"
	azureContainerFlagName        = "azure-container"
	azureCliPath                  = "azure-cli-path"
	azureBlobStoreBaseURLFlagName = "azure-blob-store-base-url"
)

var logger lager.Logger

//Parse ...
func Parse(osArgs []string) (backup.Executor, *string, lager.Logger) {
	flags := flag.NewFlagSet(osArgs[0], flag.ExitOnError)

	backupType := osArgs[1]

	sourceFolder := flags.String(sourceFolderFlagName, "", "Local path to upload from (e.g.: /var/vcap/data)")
	backupCreatorCmd := flags.String(backupCreatorCmdFlagName, "", "Command for creating backup")
	cleanupCmd := flags.String(cleanupCmdFlagName, "", "Command for cleaning backup")
	serviceIdentifierCmd := flags.String(serviceIdentifierCmdFlagName, "", "Optional command for identifying service for backup")
	cronSchedule := flags.String(CronScheduleFlagName, "", "Cron schedule for running backup. Leave empty to run only once.")
	destPath := flags.String(destPathFlagName, "", "Remote directory path inside bucket to upload to. No preceding or trailing slashes. E.g. remote/path/inside/bucket")
	exitIfBackupInProgress := flags.String(exitIfInProgressFlagName, "false", "Optional command to reject subsequent backup requests if a backup is already in progress. Defaults to false.")

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
	azureCmdPath := flags.String(azureCliPath, "blobxfer", "Azure storage account container")

	azureBlobStoreBaseURL := flags.String(azureBlobStoreBaseURLFlagName, "", "Azure blob store base URL (optional)")

	cf_lager.AddFlags(flags)
	flags.Parse(osArgs[2:])

	exitIfBackupInProgressBooleanValue, err := strconv.ParseBool(*exitIfBackupInProgress)

	if err != nil {
		logger.Fatal("Invalid boolean value for --exit-if-in-progress. Please set to true, false, or leave empty for default (false).", err)
	}

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
		backuper = azure.New(*azureStorageAccessKey, *azureStorageAccount, *azureContainer, *azureBlobStoreBaseURL, *azureCmdPath, logger)

	case "skip":
		logger.Info("No destination provided - skipping backup")
		dummyExecutor := dummy.NewDummyExecutor(logger)
		// Default cronSchedule to monthly if not provided when destination is also not provided
		// This is needed to successfully run the dummy executor and not exit
		if *cronSchedule == "" {
			*cronSchedule = "@monthly"
		}
		return dummyExecutor, cronSchedule, logger

	default:
		logger.Fatal(fmt.Sprintf("Unknown destination type: %s", backupType), nil)
	}

	validateFlag(sourceFolder, sourceFolderFlagName)
	validateFlag(backupCreatorCmd, backupCreatorCmdFlagName)
	var calculator = &backup.FileSystemSizeCalculator{}

	executor := backup.NewExecutor(
		backuper,
		*sourceFolder,
		remotePath,
		*backupCreatorCmd,
		*cleanupCmd,
		*serviceIdentifierCmd,
		exitIfBackupInProgressBooleanValue,
		logger,
		exec.Command,
		calculator,
	)

	return executor, cronSchedule, logger
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
