package s3

import (
	"bytes"
	"fmt"
	"os/exec"
)

type S3CliClient struct {
	awsCmdPath  string
	accessKey   string
	secretKey   string
	endpointURL string
}

func NewCliClient(awsCmdPath, endpointURL, accessKey, secretKey string) *S3CliClient {
	return &S3CliClient{
		awsCmdPath:  awsCmdPath,
		endpointURL: endpointURL,
		accessKey:   accessKey,
		secretKey:   secretKey,
	}
}

func (c *S3CliClient) s3Cmd() *exec.Cmd {
	cmd := exec.Command(c.awsCmdPath, "--endpoint-url", c.endpointURL, "s3")
	cmd.Env = append(cmd.Env, fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", c.accessKey))
	cmd.Env = append(cmd.Env, fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", c.secretKey))
	return cmd
}

func (c *S3CliClient) BucketExists(bucketName string) (bool, error) {
	cmd := c.s3Cmd()
	cmd.Args = append(cmd.Args, "ls", bucketName)
	if out, err := cmd.CombinedOutput(); err != nil {
		if bytes.Contains(out, []byte("NoSuchBucket")) {
			return false, nil
		}

		return false, fmt.Errorf("unknown s3 error occurred: '%s' with output: '%s'", err, string(out))
	}

	return true, nil
}

func (c *S3CliClient) CreateBucket(bucketName string) error {
	cmd := c.s3Cmd()
	cmd.Args = append(cmd.Args, "mb", fmt.Sprintf("s3://%s", bucketName))
	return cmd.Run()
}
func (c *S3CliClient) Sync(localPath, bucketName, remotePath string) error {
	cmd := c.s3Cmd()
	cmd.Args = append(cmd.Args, "sync", localPath, fmt.Sprintf("s3://%s/%s", bucketName, remotePath))
	return cmd.Run()
}
