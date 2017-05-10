package s3testclient

import (
	"fmt"
	"os"
	"strings"
	"time"

	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/service-backup/config"
	"github.com/pivotal-cf/service-backup/s3"
	"github.com/pivotal-cf/service-backup/systemtruststorelocator"
)

type S3TestClient struct {
	*s3.S3CliClient
}

func New(endpointURL, accessKeyID, secretAccessKey, basePath string) *S3TestClient {
	systemTrustStorePath, err := systemtruststorelocator.New(config.RealFileSystem{}).Path()
	Expect(err).NotTo(HaveOccurred())

	generator := config.RemotePathGenerator{
		BasePath: basePath,
	}
	s3CLIClient := s3.New("s3_test_client", "aws", endpointURL, "", accessKeyID, secretAccessKey, systemTrustStorePath, generator)
	return &S3TestClient{S3CliClient: s3CLIClient}
}

func (c *S3TestClient) ListRemotePath(bucketName, region string) ([]string, error) {
	cmdArgs := []string{}
	if region != "" {
		cmdArgs = append(cmdArgs, "--region", region)
	}
	cmdArgs = append(cmdArgs, "ls", "--recursive", fmt.Sprintf("s3://%s/", bucketName))
	cmd := c.S3Cmd(cmdArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return []string{}, fmt.Errorf("command failed: %s\nwith error:%s", string(out), err)
	}

	remoteKeys := []string{}
	for _, line := range strings.Split(string(out), "\n") {
		cols := strings.Fields(line)
		if len(cols) < 4 {
			continue
		}
		remoteKeys = append(remoteKeys, cols[3])
	}
	return remoteKeys, nil
}

func (c *S3TestClient) RemotePathExistsInBucket(bucketName, remotePath string) bool {
	keys, err := c.ListRemotePath(bucketName, "")
	Expect(err).ToNot(HaveOccurred())

	for _, key := range keys {
		if strings.Contains(key, remotePath) {
			return true
		}
	}
	return false
}

func (c *S3TestClient) DownloadRemoteDirectory(bucketName, remotePath, localPath string) error {
	err := os.MkdirAll(localPath, 0777)
	if err != nil {
		return err
	}

	cmd := c.S3Cmd()
	cmd.Args = append(cmd.Args, "sync", fmt.Sprintf("s3://%s/%s", bucketName, remotePath), localPath)
	return c.RunCommand(cmd, "download remote")
}

func (c *S3TestClient) DeleteRemotePath(bucketName, remotePath, region string) error {
	cmd := c.S3Cmd()
	if region != "" {
		cmd.Args = append(cmd.Args, "--region", region)
	}
	cmd.Args = append(cmd.Args, "rm", "--recursive", fmt.Sprintf("s3://%s/%s", bucketName, remotePath))
	return c.RunCommand(cmd, "delete remote path")
}

func (c *S3TestClient) DeleteBucket(bucketName, region string) {
	err := c.DeleteRemotePath(bucketName, "", region)
	if err != nil && strings.Contains(err.Error(), "NoSuchBucket") {
		return
	}
	Expect(err).ToNot(HaveOccurred())

	rbArgs := []string{}
	if region != "" {
		rbArgs = append(rbArgs, "--region", region)
	}
	rbArgs = append(rbArgs, "rb", "--force", fmt.Sprintf("s3://%s", bucketName))

	cmd := c.S3Cmd(rbArgs...)

	err = c.RunCommand(cmd, "delete bucket")
	if err != nil {
		// Try again, because s3 is flaky
		time.Sleep(10 * time.Second)
		cmd = c.S3Cmd(rbArgs...)
		err = c.RunCommand(cmd, "retry delete bucket")
		Expect(err).ToNot(HaveOccurred())
	}
}
