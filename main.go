package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"

	"github.com/cloudfoundry-incubator/cf-lager"
	"github.com/pivotal-golang/lager"
)

const (
	awsCLIFlagName       = "aws-cli"
	sourceFolderFlagName = "source-folder"
	destFolderFlagName   = "dest-folder"
	endpointURLFlagName  = "endpoint-url"
	awsAccessKeyFlagName = "aws-access-key-id"
	awsSecretKeyFlagName = "aws-secret-access-key"
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

	err := performBackup(
		*awsCLIBinaryPath,
		*sourceFolder,
		*destFolder,
		*awsAccessKeyID,
		*awsSecretAccessKey,
		*endpointURL,
		logger,
	)
	if err != nil {
		logger.Fatal("performBackup", err)
	}

	logger.Info("Backup uploaded successfully.")
}

func validateFlag(value *string, flagName string) {
	if *value == "" {
		logger.Fatal("main.validation", fmt.Errorf("Flag %s not provided", flagName))
	}
}

//TODO refactor this out into a separate unit-testable package
func performBackup(
	awsCLIBinaryPath,
	sourceFolder,
	destFolder,
	awsAccessKeyID,
	awsSecretAccessKey,
	endpointURL string,
	logger lager.Logger,
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

	logger.Info("performBackup", lager.Data{"command": cmd})

	out, err := cmd.CombinedOutput()
	logger.Debug("performBackup", lager.Data{"out": string(out)})
	if err != nil {
		return err
	}

	logger.Info("backup uploaded ok")
	return nil
}
