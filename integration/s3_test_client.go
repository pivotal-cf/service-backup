package integration

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/pivotal-cf-experimental/service-backup/s3"

	"github.com/cloudfoundry-incubator/cf-lager"
	. "github.com/onsi/gomega"
)

type S3TestClient struct {
	*s3.S3CliClient
}

func NewS3TestClient(endpointURL, accessKeyID, secretAccessKey string) *S3TestClient {
	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	cf_lager.AddFlags(flags)
	logger, _ := cf_lager.New("s3-test-client")

	return &S3TestClient{
		S3CliClient: s3.NewCliClient("aws", endpointURL, accessKeyID, secretAccessKey, logger),
	}
}

func (c *S3TestClient) listRemotePath(bucketName, remotePath string) ([]string, error) {
	cmd := c.S3Cmd()
	cmd.Args = append(cmd.Args, "ls", "--recursive", fmt.Sprintf("s3://%s/%s", bucketName, remotePath))
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

func (c *S3TestClient) remotePathExistsInBucket(bucketName, remotePath string) bool {
	keys, err := c.listRemotePath(bucketName, "")
	Expect(err).ToNot(HaveOccurred())

	for _, key := range keys {
		if strings.Contains(key, remotePath) {
			return true
		}
	}
	return false
}

func (c *S3TestClient) createBucketIfNeeded(bucketName string) {
	exists, err := c.BucketExists(bucketName)
	Expect(err).NotTo(HaveOccurred())
	if !exists {
		Expect(c.CreateBucket(bucketName)).To(Succeed())
	}
}

func (c *S3TestClient) downloadRemoteDirectory(bucketName, remotePath, localPath string) error {
	err := os.MkdirAll(localPath, 0777)
	if err != nil {
		return err
	}

	cmd := c.S3Cmd()
	cmd.Args = append(cmd.Args, "sync", fmt.Sprintf("s3://%s/%s", bucketName, remotePath), localPath)
	return c.RunCommand(cmd, "download remote")
}

func (c *S3TestClient) deleteRemotePath(bucketName, remotePath string) error {
	cmd := c.S3Cmd()
	cmd.Args = append(cmd.Args, "rm", "--recursive", fmt.Sprintf("s3://%s/%s", bucketName, remotePath))
	return c.RunCommand(cmd, "delete remote path")
}

func (c *S3TestClient) deleteBucket(bucketName string) {
	cmd := c.S3Cmd()
	cmd.Args = append(cmd.Args, "rb", "--force", fmt.Sprintf("s3://%s", bucketName))

	err := c.RunCommand(cmd, "delete bucket")
	Expect(err).ToNot(HaveOccurred())
}
