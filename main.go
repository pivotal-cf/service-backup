package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/cloudfoundry-incubator/cf-lager"
	"github.com/pivotal-golang/lager"
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

	err := performBackup(
		*backupCreatorCmd,
	)
	if err != nil {
		logger.Fatal("Backup creator command failed", err)
	}

	err = uploadBackup(
		*awsCLIBinaryPath,
		*sourceFolder,
		*destFolder,
		*awsAccessKeyID,
		*awsSecretAccessKey,
		*endpointURL,
	)
	if err != nil {
		logger.Fatal("performBackup", err)
	}

	err = performCleanup(
		*cleanupCmd,
	)
	if err != nil {
		logger.Error("Cleanup command failed", err)
	}

	logger.Info("Backup uploaded successfully.")
}

func validateFlag(value *string, flagName string) {
	if *value == "" {
		logger.Fatal("main.validation", fmt.Errorf("Flag %s not provided", flagName))
	}
}

func performBackup(
	backupCreatorCmd string,
) error {

	args := strings.Split(backupCreatorCmd, " ")
	cmd := exec.Command(args[0], args[1:]...)

	out, err := cmd.CombinedOutput()
	logger.Debug("performBackup", lager.Data{"cmd": backupCreatorCmd, "out": string(out)})

	return err
}

func uploadBackup(
	awsCLIBinaryPath,
	sourceFolder,
	destFolder,
	awsAccessKeyID,
	awsSecretAccessKey,
	endpointURL string,
) error {

	cmd := exec.Command(
		awsCLIBinaryPath,
		"s3",
		"sync",
		sourceFolder,
		destFolder,
		"--endpoint-url",
		endpointURL,
	)

	env := []string{}
	env = append(env, fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", awsAccessKeyID))
	env = append(env, fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", awsSecretAccessKey))
	cmd.Env = env

	logger.Info("uploadBackup", lager.Data{"command": cmd})

	out, err := cmd.CombinedOutput()
	logger.Debug("uploadBackup", lager.Data{"out": string(out)})
	if err != nil {
		return err
	}

	logger.Info("backup uploaded ok")
	return nil
}

func performCleanup(cleanupCmd string) error {
	if cleanupCmd == "" {
		logger.Info("Cleanup command not provided")
		return nil
	}

	args := strings.Split(cleanupCmd, " ")
	cmd := exec.Command(args[0], args[1:]...)

	out, err := cmd.CombinedOutput()
	logger.Debug("performCleanup", lager.Data{"cmd": cleanupCmd, "out": string(out)})

	return err
}
