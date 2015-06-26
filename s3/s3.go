package s3

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/pivotal-golang/lager"
)

type S3Client interface {
	BucketExists(bucketName string) (bool, error)
	CreateBucket(bucketName string) error
	Sync(localPath, bucketName, remotePath string) error
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

func (c awsCLIClient) Sync(localPath, bucketName, remotePath string) error {
	s3Config := &aws.Config{
		Region:     "us-east-1",
		MaxRetries: 50,
	}

	s3Client := s3.New(s3Config)

	uploadOptions := &s3manager.UploadOptions{
		S3: s3Client,
	}
	uploader := s3manager.NewUploader(uploadOptions)

	return c.uploadDirectory(localPath, bucketName, remotePath, uploader)
}

func (c awsCLIClient) uploadDirectory(localPath, bucketName, remotePath string, uploader *s3manager.Uploader) error {
	directoryContents, err := filepath.Glob(localPath + "/*")
	if err != nil {
		return err
	}

	for _, filePath := range directoryContents {
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			return err
		}

		if fileInfo.IsDir() {
			remotePath := remotePath + "/" + filepath.Base(filePath)
			err := c.uploadDirectory(filePath, bucketName, remotePath, uploader)
			if err != nil {
				return err
			}
		} else {
			err := c.uploadFile(filePath, bucketName, remotePath, uploader)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (c awsCLIClient) uploadFile(filePath, bucketName, remotePath string, uploader *s3manager.Uploader) error {
	source, err := os.Open(filePath)
	if err != nil {
		return err
	}

	key := remotePath + "/" + filepath.Base(filePath)

	uploadInput := &s3manager.UploadInput{
		Bucket: &bucketName,
		Key:    &key,
		Body:   source,
	}

	out, err := uploader.Upload(uploadInput)
	if err != nil {
		c.logger.Error(
			"Uploading failed",
			err,
			lager.Data{"localPath": filePath, "remotePath": remotePath, "out": out},
		)
		return err
	}
	return nil
}
