package gcp_test

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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pborman/uuid"
	"github.com/pivotal-cf-experimental/service-backup/gcp"
	"github.com/pivotal-golang/lager"
	"google.golang.org/api/option"
)

var _ = Describe("backups to Google Cloud Storage", func() {
	var (
		bucketName  string
		bucket      *storage.BucketHandle
		dirToBackup string
		ctx         context.Context

		backuper *gcp.StorageClient
	)

	BeforeEach(func() {
		bucketName = fmt.Sprintf("service-backup-test-%s", uuid.New())

		serviceAccountFilePathKey := "SERVICE_BACKUP_TESTS_GCP_SERVICE_ACCOUNT_FILE"
		gcpServiceAccountFilePath := os.Getenv(serviceAccountFilePathKey)
		Expect(gcpServiceAccountFilePath).NotTo(BeEmpty(), fmt.Sprintf("must set %s", serviceAccountFilePathKey))

		gcpProjectNameKey := "SERVICE_BACKUP_TESTS_GCP_PROJECT_NAME"
		gcpProjectName := os.Getenv(gcpProjectNameKey)
		Expect(gcpProjectName).NotTo(BeEmpty(), fmt.Sprintf("must set %s", gcpProjectNameKey))

		var err error
		dirToBackup, err = ioutil.TempDir("", "gcp-backup-tests")
		Expect(err).NotTo(HaveOccurred())
		Expect(createFile("GCP FTW", dirToBackup, "should-back-up.txt"))

		ctx = context.Background()
		gcpClient, err := storage.NewClient(ctx, option.WithServiceAccountFile(gcpServiceAccountFilePath))
		Expect(err).NotTo(HaveOccurred())
		bucket = gcpClient.Bucket(bucketName)

		backuper = gcp.New(gcpServiceAccountFilePath, gcpProjectName, bucketName)
		logger := lager.NewLogger("[GCP tests] ")
		logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))
		Expect(backuper.Upload(dirToBackup, logger)).To(Succeed())
	})

	AfterEach(func() {
		Expect(os.RemoveAll(dirToBackup)).To(Succeed())

		objectsInBucket := bucket.Objects(ctx, nil)
		for {
			obj, err := objectsInBucket.Next()
			if err == storage.Done {
				break
			}
			Expect(err).NotTo(HaveOccurred())
			Expect(bucket.Object(obj.Name).Delete(ctx)).To(Succeed())
		}
		Expect(bucket.Delete(ctx)).To(Succeed())
	})

	It("backs up files", func() {
		today := time.Now()
		expectedObjectName := fmt.Sprintf("%d/%02d/%02d/%s", today.Year(), today.Month(), today.Day(), "should-back-up.txt")
		bucketObj := bucket.Object(expectedObjectName)
		objReader, err := bucketObj.NewReader(ctx)
		Expect(err).NotTo(HaveOccurred())
		defer objReader.Close()
		remoteContents := new(bytes.Buffer)
		_, err = io.Copy(remoteContents, objReader)
		Expect(err).NotTo(HaveOccurred())
		Expect(remoteContents.String()).To(Equal("GCP FTW"))
	})
})

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
