// Copyright (C) 2016-Present Pivotal Software, Inc. All rights reserved.
// This program and the accompanying materials are made available under the terms of the under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package gcsintegration_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"cloud.google.com/go/storage"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/pborman/uuid"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

var _ = Describe("gcs", func() {
	const uploadTimeout = time.Second * 20

	var bucketName string
	var gcpServiceAccountFilePath string

	Context("when the deployment name is configured", func() {
		BeforeEach(func() {
			gcpServiceAccountFilePath = envMustHave("SERVICE_BACKUP_TESTS_GCP_SERVICE_ACCOUNT_FILE")
			bucketName = fmt.Sprintf("service-backup-test-%s", uuid.New())
		})

		AfterEach(func() {
			deleteBucket(bucketName, gcpServiceAccountFilePath)
		})

		It("uploads to GCS with deployment name in the remote path", func() {
			os.Setenv("GCP_SERVICE_ACCOUNT_FILE", gcpServiceAccountFilePath)
			projectID := envMustHave("SERVICE_BACKUP_TESTS_GCP_PROJECT_NAME")
			baseDir := createBaseDir()
			sourceDir := createSourceDir(baseDir)
			deploymentName := "deployment-name"
			addDeploymentNameToPath := deploymentName != ""

			runningBin := runBackup(serviceBackupBinaryPath, createConfigFile(`---
destinations:
- type: gcs
  config:
    bucket_name: %s
    project_id: %s
source_folder: %s
source_executable: true
exit_if_in_progress: true
cron_schedule: '*/5 * * * * *'
cleanup_executable: true
missing_properties_message: custom message
deployment_name: %s
add_deployment_name_to_backup_path: %t`, bucketName, projectID, sourceDir, deploymentName, addDeploymentNameToPath))

			Eventually(runningBin.Out, uploadTimeout).Should(gbytes.Say("Cleanup completed successfully"))
			runningBin.Terminate().Wait()

			ctx := context.Background()
			bucketHandle := newBucketHandle(ctx, gcpServiceAccountFilePath, bucketName)

			Expect(readObject(ctx, bucketHandle, deploymentName, "1.txt")).To(Equal("1"))
			Expect(os.RemoveAll(sourceDir)).To(Succeed())
		})
	})
})

func readObject(ctx context.Context, bucket *storage.BucketHandle, deploymentName, relativePath string) string {
	bucketObj := bucket.Object(remotePathInBucket(deploymentName, relativePath))
	objReader, err := bucketObj.NewReader(ctx)
	Expect(err).NotTo(HaveOccurred())
	defer objReader.Close()

	remoteContents := new(bytes.Buffer)
	_, err = io.Copy(remoteContents, objReader)
	Expect(err).NotTo(HaveOccurred())
	return fmt.Sprint(remoteContents)
}

func remotePathInBucket(deploymentName, relativePath string) string {
	today := time.Now()
	datePath := fmt.Sprintf("%d/%02d/%02d", today.Year(), today.Month(), today.Day())
	if deploymentName != "" {
		return fmt.Sprintf("%s/%s/%s", deploymentName, datePath, relativePath)
	}
	return fmt.Sprintf("%s/%s", datePath, relativePath)
}

func envMustHave(key string) string {
	val := os.Getenv(key)
	Expect(val).NotTo(BeEmpty(), fmt.Sprintf("must set %s", key))
	return val
}

func createBaseDir() string {
	baseDir, err := ioutil.TempDir("", "multiple-destinations-integration-tests")
	Expect(err).NotTo(HaveOccurred())
	return baseDir
}

func createSourceDir(baseDir string) string {
	backupDir := filepath.Join(baseDir, "source")
	Expect(os.Mkdir(backupDir, 0755)).To(Succeed())

	Expect(ioutil.WriteFile(filepath.Join(backupDir, "1.txt"), []byte("1"), 0644)).To(Succeed())
	return backupDir
}

func runBackup(binaryPath string, params ...string) *gexec.Session {
	backupCmd := exec.Command(binaryPath, params...)
	session, err := gexec.Start(backupCmd, GinkgoWriter, GinkgoWriter)
	Expect(err).ToNot(HaveOccurred())
	return session
}

func createConfigFile(format string, a ...interface{}) string {
	file, err := ioutil.TempFile("", "config.yml")
	Expect(err).NotTo(HaveOccurred())
	file.Write([]byte(fmt.Sprintf(format, a...)))
	return file.Name()
}

func newBucketHandle(ctx context.Context, gcpServiceAccountFilePath, bucketName string) *storage.BucketHandle {
	client, err := storage.NewClient(ctx, option.WithServiceAccountFile(gcpServiceAccountFilePath))
	Expect(err).NotTo(HaveOccurred())
	return client.Bucket(bucketName)
}

func deleteBucket(bucketName, gcpServiceAccountFilePath string) {
	client, err := storage.NewClient(context.Background(), option.WithServiceAccountFile(gcpServiceAccountFilePath))
	Expect(err).NotTo(HaveOccurred())
	bucket := client.Bucket(bucketName)
	objsIt := bucket.Objects(context.Background(), nil)
	for {
		attrs, err := objsIt.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			Fail(err.Error())
		}
		Expect(bucket.Object(attrs.Name).Delete(context.Background())).To(Succeed())
	}

	err = bucket.Delete(context.Background())
	Expect(err).NotTo(HaveOccurred())
}
