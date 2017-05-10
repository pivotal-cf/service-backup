package gcp

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"cloud.google.com/go/storage"
	"code.cloudfoundry.org/lager"
	"github.com/pivotal-cf/service-backup/backup"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type StorageClient struct {
	serviceAccountFilePath string
	gcpProjectID           string
	bucketName             string
	name                   string
}

func New(name, serviceAccountFilePath, gcpProjectID, bucketName string) *StorageClient {
	return &StorageClient{
		serviceAccountFilePath: serviceAccountFilePath,
		gcpProjectID:           gcpProjectID,
		bucketName:             bucketName,
		name:                   name,
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
		return errs("creating Google Cloud Storage client", err)
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
	pathGenerator := backup.RemotePathGenerator{}
	nameInBucket := fmt.Sprintf("%s/%s", pathGenerator.RemotePathWithDate(""), relativePath)
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
	bucketIterator := gcpClient.Buckets(ctx, s.gcpProjectID)
	bucket := gcpClient.Bucket(s.bucketName)
	for {
		attr, err := bucketIterator.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		if attr.Name == s.bucketName {
			return bucket, nil
		}
	}

	if err := bucket.Create(ctx, s.gcpProjectID, nil); err != nil {
		return nil, err
	}
	return bucket, nil
}

func (s *StorageClient) Name() string {
	return s.name
}
