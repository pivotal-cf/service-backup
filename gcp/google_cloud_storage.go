package gcp

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"cloud.google.com/go/storage"
	"github.com/pivotal-golang/lager"
	"google.golang.org/api/option"
)

type StorageClient struct {
	serviceAccountFilePath string
	gcpProjectID           string
	bucketName             string
}

func New(serviceAccountFilePath, gcpProjectID, bucketName string) *StorageClient {
	return &StorageClient{
		serviceAccountFilePath: serviceAccountFilePath,
		gcpProjectID:           gcpProjectID,
		bucketName:             bucketName,
	}
}

func (s *StorageClient) Upload(dirToUpload string, logger lager.Logger) error {
	errs := func(action string, err error) error {
		wrappedErr := fmt.Errorf("error %s: %s", action, err)
		logger.Error("error uploading to Google Cloud Storage", wrappedErr, nil)
		return wrappedErr
	}

	logger.Info(fmt.Sprintf("will upload %s to Google Cloud Storage", dirToUpload), nil)

	ctx := context.Background()
	gcpClient, err := storage.NewClient(ctx, option.WithServiceAccountFile(s.serviceAccountFilePath))
	if err != nil {
		return errs("creating Google Cloud Storage client", err) // TODO test
	}
	defer gcpClient.Close()

	bucket, err := s.ensureBucketExists(gcpClient, ctx)
	if err != nil {
		return errs("creating bucket", err)
	}

	today := time.Now()
	if err := filepath.Walk(dirToUpload, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		if err := s.uploadFile(dirToUpload, path, today, ctx, bucket, logger); err != nil {
			return errs("uploading file", err)
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

func (s *StorageClient) uploadFile(baseDir, fileAbsPath string, timeNow time.Time, ctx context.Context, bucket *storage.BucketHandle, logger lager.Logger) error {
	relativePath, err := filepath.Rel(baseDir, fileAbsPath)
	if err != nil {
		return err
	}

	nameInBucket := fmt.Sprintf("%d/%02d/%02d/%s", timeNow.Year(), timeNow.Month(), timeNow.Day(), relativePath)
	logger.Info(fmt.Sprintf("will upload %s to bucket %s", nameInBucket, s.bucketName), nil)
	obj := bucket.Object(nameInBucket)

	bucketWriter := obj.NewWriter(ctx)
	defer bucketWriter.Close()

	file, err := os.Open(fileAbsPath)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := io.Copy(bucketWriter, file); err != nil {
		return err
	}

	return nil
}

func (s *StorageClient) ensureBucketExists(gcpClient *storage.Client, ctx context.Context) (*storage.BucketHandle, error) {
	bucket := gcpClient.Bucket(s.bucketName)
	if err := bucket.Create(ctx, s.gcpProjectID, nil); err != nil {
		if err.Error() == "googleapi: Error 409: You already own this bucket. Please select another name., conflict" {
			return bucket, nil
		}
		return nil, err
	}
	return bucket, nil
}

func (c *StorageClient) Name() string {
	return "gcs"
}
