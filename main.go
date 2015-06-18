package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"

	"github.com/cloudfoundry-incubator/cf-lager"
	"github.com/pivotal-golang/lager"
)

func main() {

	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	awsCLIBinaryPath := flags.String("aws-cli", "", "Path to AWS CLI")
	sourceFolder := flags.String("source-folder", "", "Local path to upload from (e.g.: /var/vcap/data)")
	destFolder := flags.String("dest-folder", "", "Remote path to upload to (e.g.: s3://bucket-name/path/to/loc)")
	endpointURL := flags.String("endpoint-url", "", "S3 endpoint URL")
	awsAccessKeyID := flags.String("aws-access-key-id", "", "S3 endpoint URL")
	awsSecretAccessKey := flags.String("aws-secret-access-key", "", "S3 endpoint URL")

	cf_lager.AddFlags(flags)
	flags.Parse(os.Args[1:])

	logger, _ := cf_lager.New("ServiceBackup")

	cmd := exec.Command(
		*awsCLIBinaryPath,
		"s3",
		"sync",
		*sourceFolder,
		*destFolder,
		"--endpoint-url",
		*endpointURL,
	)

	env := []string{}
	env = append(env, fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", *awsAccessKeyID))
	env = append(env, fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", *awsSecretAccessKey))
	cmd.Env = env

	logger.Info("command", lager.Data{"command": cmd})

	out, err := cmd.CombinedOutput()
	logger.Info("Upload", lager.Data{"out": string(out)})
	if err != nil {
		logger.Fatal("", err)
	}
	logger.Info("backup uploaded ok")
}
