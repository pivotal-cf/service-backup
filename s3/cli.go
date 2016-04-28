package s3

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/pivotal-golang/lager"
)

type S3CliClient struct {
	awsCmdPath    string
	accessKey     string
	secretKey     string
	endpointURL   string
	logger        lager.Logger
	sessionLogger lager.Logger
}

func NewCliClient(awsCmdPath, endpointURL, accessKey, secretKey string, logger lager.Logger) *S3CliClient {
	return &S3CliClient{
		awsCmdPath:    awsCmdPath,
		endpointURL:   endpointURL,
		accessKey:     accessKey,
		secretKey:     secretKey,
		logger:        logger,
		sessionLogger: logger,
	}
}

func (c *S3CliClient) S3Cmd(args ...string) *exec.Cmd {
	cmdArgs := []string{"s3"}
	cmdArgs = append(cmdArgs, args...)
	cmd := exec.Command(c.awsCmdPath, cmdArgs...)
	cmd.Env = append(cmd.Env, fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", c.accessKey))
	cmd.Env = append(cmd.Env, fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", c.secretKey))
	return cmd
}

func (c *S3CliClient) CreateRemotePathIfNeeded(remotePath string) error {
	c.sessionLogger.Info("Checking for remote path", lager.Data{"remotePath": remotePath})
	remotePathExists, err := c.remotePathExists(remotePath)
	if err != nil {
		return err
	}

	if remotePathExists {
		return nil
	}

	c.sessionLogger.Info("Checking for remote path - remote path does not exist - making it now")
	err = c.createRemotePath(remotePath)
	if err != nil {
		return err
	}
	c.sessionLogger.Info("Checking for remote path - remote path created ok")
	return nil
}

func (c *S3CliClient) remotePathExists(remotePath string) (bool, error) {
	bucketName := strings.Split(remotePath, "/")[0]

	cmd := c.S3Cmd("ls", bucketName)

	if out, err := cmd.CombinedOutput(); err != nil {
		if bytes.Contains(out, []byte("NoSuchBucket")) {
			return false, nil
		}

		wrappedErr := fmt.Errorf("unknown s3 error occurred: '%s' with output: '%s'", err, string(out))
		c.sessionLogger.Error("error checking if bucket exists", wrappedErr)
		return false, wrappedErr
	}

	return true, nil
}

func (c *S3CliClient) createRemotePath(remotePath string) error {
	bucketName := strings.Split(remotePath, "/")[0]
	cmd := c.S3Cmd("mb", fmt.Sprintf("s3://%s", bucketName))
	return c.RunCommand(cmd, "create bucket")
}

func (c *S3CliClient) Upload(localPath, remotePath string) error {
	err := c.CreateRemotePathIfNeeded(remotePath)
	if err != nil {
		return err
	}

	cmd := c.S3Cmd("sync", localPath, fmt.Sprintf("s3://%s", remotePath))
	return c.RunCommand(cmd, "sync")
}

func (c *S3CliClient) RunCommand(cmd *exec.Cmd, stepName string) error {
	c.sessionLogger.Info(fmt.Sprintf("Running command: %+v\n", cmd))
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("error in %s: %s, output: %s", stepName, err, string(out))
	}

	return nil
}

//SetLogSession adds an identifier to all log messages for the duration of the session
func (c *S3CliClient) SetLogSession(sessionName, sessionIdentifier string) {
	c.sessionLogger = c.logger.Session(
		sessionName,
		lager.Data{"identifier": sessionIdentifier},
	)
}

//CloseLogSession removes any previously added identifier from future log messages
func (c *S3CliClient) CloseLogSession() {
	c.sessionLogger = c.logger
}
