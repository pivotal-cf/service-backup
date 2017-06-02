// Copyright (C) 2016-Present Pivotal Software, Inc. All rights reserved.
// This program and the accompanying materials are made available under the terms of the under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package gcs_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"cloud.google.com/go/storage"
	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pborman/uuid"
	"github.com/pivotal-cf/service-backup/gcs"
	"github.com/pivotal-cf/service-backup/testhelpers"
	"github.com/pivotal-cf/service-backup/upload"
	"google.golang.org/api/option"
)

var _ = Describe("backups to Google Cloud Storage", func() {
	Describe("successful backups", func() {
		var (
			bucketName  string
			bucket      *storage.BucketHandle
			dirToBackup string
			ctx         context.Context
			projectName string
			name        string

			backuper *gcs.StorageClient
		)

		itBacksUpFiles := func() {
			It("backs up files", func() {
				Expect(readObject(ctx, bucket, "a.txt")).To(Equal("content for a.txt"))
				Expect(readObject(ctx, bucket, "d1/b.txt")).To(Equal("content for b.txt"))
				Expect(readObject(ctx, bucket, "d1/d2/c.txt")).To(Equal("content for c.txt"))
			})
		}

		BeforeEach(func() {
			gcpServiceAccountFilePath := envMustHave("SERVICE_BACKUP_TESTS_GCP_SERVICE_ACCOUNT_FILE")
			projectName = envMustHave("SERVICE_BACKUP_TESTS_GCP_PROJECT_NAME")

			var err error
			dirToBackup, err = ioutil.TempDir("", "gcs-backup-tests")
			Expect(err).NotTo(HaveOccurred())
			Expect(createFile("content for a.txt", dirToBackup, "a.txt"))
			Expect(createFile("content for b.txt", dirToBackup, "d1", "b.txt"))
			Expect(createFile("content for c.txt", dirToBackup, "d1", "d2", "c.txt"))

			ctx = context.Background()
			client, err := storage.NewClient(ctx, option.WithServiceAccountFile(gcpServiceAccountFilePath))
			Expect(err).NotTo(HaveOccurred())
			bucketName = fmt.Sprintf("service-backup-test-%s", uuid.New())
			bucket = client.Bucket(bucketName)
			name = "google_cloud_destination"

			backuper = gcs.New(name, gcpServiceAccountFilePath, projectName, bucketName, upload.RemotePathFunc("", ""))
		})

		JustBeforeEach(func() {
			logger := lager.NewLogger("[GCS tests] ")
			logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))
			Expect(backuper.Upload(dirToBackup, logger)).To(Succeed())
		})

		AfterEach(func() {
			Expect(os.RemoveAll(dirToBackup)).To(Succeed())
			testhelpers.DeleteGCSBucket(ctx, bucket)
		})

		itBacksUpFiles()

		Context("when the bucket already exists", func() {
			BeforeEach(func() {
				Expect(bucket.Create(ctx, projectName, nil)).To(Succeed())
			})

			itBacksUpFiles()
		})
	})

	Describe("failed backups", func() {
		Context("when the service account credentials are invalid", func() {
			It("returns an error", func() {
				backuper := gcs.New("icanbeanything", "idontexist", "", "", upload.RemotePathFunc("", ""))
				logger := lager.NewLogger("[GCS tests] ")
				logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))
				Expect(backuper.Upload("", logger)).To(MatchError(ContainSubstring("error creating Google Cloud Storage client")))
			})
		})
	})
})

func envMustHave(key string) string {
	val := os.Getenv(key)
	Expect(val).NotTo(BeEmpty(), fmt.Sprintf("must set %s", key))
	return val
}

func createFile(content string, nameParts ...string) error {
	fullPath, err := ensureDirExists(nameParts)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(fullPath, []byte(content), 0644)
}

func ensureDirExists(nameParts []string) (string, error) {
	fullPath := filepath.Join(nameParts...)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return "", err
	}
	return fullPath, nil
}

func readObject(ctx context.Context, bucket *storage.BucketHandle, relativePath string) string {
	bucketObj := bucket.Object(expectedNameInBucket(relativePath))
	objReader, err := bucketObj.NewReader(ctx)
	Expect(err).NotTo(HaveOccurred())
	defer objReader.Close()

	remoteContents := new(bytes.Buffer)
	_, err = io.Copy(remoteContents, objReader)
	Expect(err).NotTo(HaveOccurred())
	return remoteContents.String()
}

func expectedNameInBucket(relativePath string) string {
	today := time.Now()
	return fmt.Sprintf("%d/%02d/%02d/%s", today.Year(), today.Month(), today.Day(), relativePath)
}
