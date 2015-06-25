package s3

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/pivotal-golang/lager"
)

type S3Client interface {
	BucketExists(bucketName string) (bool, error)
	CreateBucket(bucketName string) error
	Sync(localPath, remotePath string) error
}

type awsCLIClient struct {
	awsAccessKeyID     string
	awsSecretAccessKey string
	awsCLIPath         string
	endpointURL        string
	logger             lager.Logger
}

func NewAWSCLIClient(
	awsAccessKeyID string,
	awsSecretAccessKey string,
	endpointURL string,
	awsCLIPath string,
	logger lager.Logger,
) S3Client {
	return &awsCLIClient{
		awsAccessKeyID:     awsAccessKeyID,
		awsSecretAccessKey: awsSecretAccessKey,
		endpointURL:        endpointURL,
		awsCLIPath:         awsCLIPath,
		logger:             logger,
	}
}

func (c awsCLIClient) createS3Command(args ...string) *exec.Cmd {
	cmd := exec.Command(
		c.awsCLIPath,
		append([]string{
			"s3",
			"--region",
			"us-east-1",
		}, args...)...,
	)
	cmd.Env = []string{}
	cmd.Env = append(cmd.Env, fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", c.awsAccessKeyID))
	c.logger.Debug("S3 command debug info", lager.Data{"command": cmd})
	cmd.Env = append(cmd.Env, fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", c.awsSecretAccessKey))

	return cmd
}

func (c awsCLIClient) BucketExists(bucketName string) (bool, error) {
	cmd := c.createS3Command(
		"ls",
		bucketName,
	)

	out, err := cmd.CombinedOutput()
	if err == nil {
		c.logger.Info("Checking for bucket - bucket exists")
		return true, nil
	}

	errOut := string(out)

	if !strings.Contains(errOut, "NoSuchBucket") {
		c.logger.Error(
			"Checking for bucket failed",
			err,
			lager.Data{"bucketName": bucketName},
		)
		return false, err
	}
	return false, nil
}

func (c awsCLIClient) CreateBucket(bucketName string) error {
	cmd := c.createS3Command(
		"mb",
		fmt.Sprintf("s3://%s", bucketName),
	)

	out, err := cmd.CombinedOutput()

	if err != nil {
		c.logger.Error(
			"Create bucket failed",
			err,
			lager.Data{"bucketName": bucketName, "out": string(out)},
		)
		return err
	}
	return nil
}

func (c awsCLIClient) Sync(localPath, remotePath string) error {
	cmd := c.createS3Command(
		"sync",
		localPath,
		fmt.Sprintf("s3://%s", remotePath),
		"--endpoint-url",
		c.endpointURL,
	)

	out, err := cmd.CombinedOutput()

	if err != nil {
		c.logger.Error(
			"Syncing failed",
			err,
			lager.Data{"localPath": localPath, "remotePath": remotePath, "out": string(out)},
		)
		return err
	}
	return nil
}
