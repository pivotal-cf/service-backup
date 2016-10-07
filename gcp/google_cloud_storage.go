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

	bucket := gcpClient.Bucket(s.bucketName)
	if err := bucket.Create(ctx, s.gcpProjectID, nil); err != nil {
		return errs("creating bucket", err) // TODO test
	}

	today := time.Now()
	if err := filepath.Walk(dirToUpload, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relativePath, err := filepath.Rel(dirToUpload, path)
		if err != nil {
			return err
		}

		nameInBucket := fmt.Sprintf("%d/%02d/%02d/%s", today.Year(), today.Month(), today.Day(), relativePath)
		logger.Info(fmt.Sprintf("will upload %s to bucket %s", nameInBucket, s.bucketName), nil)
		obj := bucket.Object(nameInBucket)

		bucketWriter := obj.NewWriter(ctx)
		defer bucketWriter.Close()

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		if _, err := io.Copy(bucketWriter, file); err != nil {
			return errs("writing file to bucket", err)
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}
