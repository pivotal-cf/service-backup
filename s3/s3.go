package s3

import (
	"os"
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

type awsSDKClient struct {
	awsAccessKeyID     string
	awsSecretAccessKey string
	endpointURL        string
	s3Client           *s3.S3
	logger             lager.Logger
}

func NewAWSSDKClient(
	awsAccessKeyID string,
	awsSecretAccessKey string,
	endpointURL string,
	maxRetries int,
	logger lager.Logger,
) S3Client {

	s3Config := &aws.Config{
		Region:     "us-east-1",
		MaxRetries: maxRetries,
	}

	s3Client := s3.New(s3Config)
	return &awsSDKClient{
		awsAccessKeyID:     awsAccessKeyID,
		awsSecretAccessKey: awsSecretAccessKey,
		endpointURL:        endpointURL,
		s3Client:           s3Client,
		logger:             logger,
	}
}

func (c awsSDKClient) BucketExists(bucketName string) (bool, error) {
	params := &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	}
	resp, err := c.s3Client.HeadBucket(params)

	if err != nil {
		if strings.Contains(err.Error(), "status code: 404") {
			return false, nil
		}

		c.logger.Error(
			"Checking for bucket failed",
			err,
			lager.Data{"bucketName": bucketName, "resp": resp},
		)
		return false, err
	}

	return true, nil
}

func (c awsSDKClient) CreateBucket(bucketName string) error {
	params := &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	}
	resp, err := c.s3Client.CreateBucket(params)

	if err != nil {
		c.logger.Error(
			"Create bucket failed",
			err,
			lager.Data{"bucketName": bucketName, "resp": resp},
		)
		return err
	}
	return nil
}

func (c awsSDKClient) Sync(localPath, bucketName, remotePath string) error {
	uploadOptions := &s3manager.UploadOptions{
		S3: c.s3Client,
	}
	uploader := s3manager.NewUploader(uploadOptions)

	return c.uploadDirectory(localPath, bucketName, remotePath, uploader)
}

func (c awsSDKClient) uploadDirectory(localPath, bucketName, remotePath string, uploader *s3manager.Uploader) error {
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

func (c awsSDKClient) uploadFile(filePath, bucketName, remotePath string, uploader *s3manager.Uploader) error {
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
