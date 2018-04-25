// Copyright (C) 2016-Present Pivotal Software, Inc. All rights reserved.
// This program and the accompanying materials are made available under the terms of the under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package gcs

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"cloud.google.com/go/storage"
	"code.cloudfoundry.org/lager"
	"github.com/pivotal-cf/service-backup/process"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type StorageClient struct {
	serviceAccountFilePath string
	projectID              string
	bucketName             string
	name                   string
	remotePathFn           func() string
}

func New(name, serviceAccountFilePath, projectID, bucketName string, remotePathFn func() string) *StorageClient {
	return &StorageClient{
		serviceAccountFilePath: serviceAccountFilePath,
		projectID:              projectID,
		bucketName:             bucketName,
		name:                   name,
		remotePathFn:           remotePathFn,
	}
}

func (s *StorageClient) Upload(dirToUpload string, logger lager.Logger, _ process.ProcessManager) error {
	errs := func(action string, err error) error {
		wrappedErr := fmt.Errorf("error %s: %s", action, err)
		logger.Error("error uploading to Google Cloud Storage", wrappedErr, nil)
		return wrappedErr
	}

	logger.Info(fmt.Sprintf("will upload %s to Google Cloud Storage", dirToUpload), nil)

	ctx := context.Background()
	client, err := storage.NewClient(ctx, option.WithServiceAccountFile(s.serviceAccountFilePath))
	if err != nil {
		return errs("creating Google Cloud Storage client", err)
	}
	defer client.Close()

	bucket, err := s.ensureBucketExists(client, ctx)
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
	nameInBucket := fmt.Sprintf("%s/%s", s.remotePathFn(), relativePath)
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

func (s *StorageClient) ensureBucketExists(client *storage.Client, ctx context.Context) (*storage.BucketHandle, error) {
	bucketIterator := client.Buckets(ctx, s.projectID)
	bucket := client.Bucket(s.bucketName)
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

	if err := bucket.Create(ctx, s.projectID, nil); err != nil {
		return nil, err
	}
	return bucket, nil
}

func (s *StorageClient) Name() string {
	return s.name
}
