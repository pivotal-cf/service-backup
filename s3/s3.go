package s3

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/pivotal-cf-experimental/service-backup/backup"
	"github.com/pivotal-golang/lager"
)

type S3CliClient struct {
	awsCmdPath    string
	accessKey     string
	secretKey     string
	endpointURL   string
	basePath      string
	logger        lager.Logger
	sessionLogger lager.Logger
}

func New(awsCmdPath, endpointURL, accessKey, secretKey, basePath string, logger lager.Logger) *S3CliClient {
	return &S3CliClient{
		awsCmdPath:    awsCmdPath,
		endpointURL:   endpointURL,
		accessKey:     accessKey,
		secretKey:     secretKey,
		basePath:      basePath,
		logger:        logger,
		sessionLogger: logger,
	}
}

func (c *S3CliClient) S3Cmd(args ...string) *exec.Cmd {
	var cmdArgs []string

	if c.endpointURL != "" {
		cmdArgs = append(cmdArgs, "--endpoint-url", c.endpointURL)
	}
	cmdArgs = append(cmdArgs, "s3")
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
		if strings.Contains(err.Error(), "AccessDenied") {
			c.sessionLogger.Error("Configured S3 user unable to create buckets", err)
		}

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

func (c *S3CliClient) Upload(localPath string) error {
	defer c.sessionLogger.Info("s3 completed")

	remotePathGenerator := backup.RemotePathGenerator{}
	remotePath := remotePathGenerator.RemotePathWithDate(c.basePath)

	c.sessionLogger.Info(fmt.Sprintf("about to upload %s to S3 remote path %s", localPath, remotePath))
	cmd := c.S3Cmd("sync", localPath, fmt.Sprintf("s3://%s", remotePath))

	out, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	if !bytes.Contains(out, []byte("NoSuchBucket")) {
		return fmt.Errorf("error in sync: %s, output: %s", err, string(out))
	}

	err = c.CreateRemotePathIfNeeded(remotePath)
	if err != nil {
		return err
	}

	cmd = c.S3Cmd("sync", localPath, fmt.Sprintf("s3://%s", remotePath))
	return c.RunCommand(cmd, "sync")
}

func (c *S3CliClient) RunCommand(cmd *exec.Cmd, stepName string) error {
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
