package s3

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/pivotal-golang/lager"
)

type S3CliClient struct {
	awsCmdPath  string
	accessKey   string
	secretKey   string
	endpointURL string
	logger      lager.Logger
}

func NewCliClient(awsCmdPath, endpointURL, accessKey, secretKey string, logger lager.Logger) *S3CliClient {
	if endpointURL == "" {
		endpointURL = "https://s3.amazonaws.com"
	}

	return &S3CliClient{
		awsCmdPath:  awsCmdPath,
		endpointURL: endpointURL,
		accessKey:   accessKey,
		secretKey:   secretKey,
		logger:      logger,
	}
}

func (c *S3CliClient) S3Cmd() *exec.Cmd {
	cmd := exec.Command(c.awsCmdPath, "--endpoint-url", c.endpointURL, "s3")
	cmd.Env = append(cmd.Env, fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", c.accessKey))
	cmd.Env = append(cmd.Env, fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", c.secretKey))
	return cmd
}

func (c *S3CliClient) CreateRemotePathIfNeeded(remotePath string) error {
	c.logger.Info("Checking for remote path", lager.Data{"remotePath": remotePath})
	remotePathExists, err := c.remotePathExists(remotePath)
	if err != nil {
		return err
	}

	if remotePathExists {
		return nil
	}

	c.logger.Info("Checking for remote path - remote path does not exist - making it now")
	err = c.createRemotePath(remotePath)
	if err != nil {
		return err
	}
	c.logger.Info("Checking for remote path - remote path created ok")
	return nil
}

func (c *S3CliClient) remotePathExists(remotePath string) (bool, error) {
	bucketName := strings.Split(remotePath, "/")[0]
	cmd := c.S3Cmd()
	cmd.Args = append(cmd.Args, "ls", bucketName)
	if out, err := cmd.CombinedOutput(); err != nil {
		if bytes.Contains(out, []byte("NoSuchBucket")) {
			return false, nil
		}

		wrappedErr := fmt.Errorf("unknown s3 error occurred: '%s' with output: '%s'", err, string(out))
		c.logger.Error("error checking if bucket exists", wrappedErr)
		return false, wrappedErr
	}

	return true, nil
}

func (c *S3CliClient) createRemotePath(remotePath string) error {
	bucketName := strings.Split(remotePath, "/")[0]
	cmd := c.S3Cmd()
	cmd.Args = append(cmd.Args, "mb", fmt.Sprintf("s3://%s", bucketName))
	return c.RunCommand(cmd, "create bucket")
}

func (c *S3CliClient) Upload(localPath, remotePath string) error {
	c.CreateRemotePathIfNeeded(remotePath)

	cmd := c.S3Cmd()
	cmd.Args = append(cmd.Args, "sync", localPath, fmt.Sprintf("s3://%s", remotePath))
	return c.RunCommand(cmd, "sync")
}

func (c *S3CliClient) RunCommand(cmd *exec.Cmd, stepName string) error {
	c.logger.Info(fmt.Sprintf("Running command: %+v\n", cmd))
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error in %s: %s, output: %s", stepName, err, string(out))
	}

	return nil
}
