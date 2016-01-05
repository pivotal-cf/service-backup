package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

const (
	awsAccessKeyIDEnvKey     = "AWS_ACCESS_KEY_ID"
	awsSecretAccessKeyEnvKey = "AWS_SECRET_ACCESS_KEY"

	existingBucketName = "service-backup-integration-test"
	awsTimeout         = "20s"

	endpointURL  = "https://s3.amazonaws.com"
	cronSchedule = "*/5 * * * * *" // every 5 seconds of every minute of every day etc
)

var (
	pathToServiceBackupBinary string
	awsAccessKeyID            string
	awsSecretAccessKey        string
	destPath                  string
)

type config struct {
	AWSAccessKeyID     string `json:"awsAccessKeyID"`
	AWSSecretAccessKey string `json:"awsSecretAccessKey"`
	PathToBackupBinary string `json:"pathToBackupBinary"`
}

func TestServiceBackupBinary(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Backup Binary Suite")
}

func beforeSuiteFirstNode() []byte {
	awsAccessKeyID = os.Getenv(awsAccessKeyIDEnvKey)
	awsSecretAccessKey = os.Getenv(awsSecretAccessKeyEnvKey)

	if awsAccessKeyID == "" || awsSecretAccessKey == "" {
		Fail(fmt.Sprintf("Specify valid AWS credentials using the env variables %s and %s", awsAccessKeyIDEnvKey, awsSecretAccessKeyEnvKey))
	}

	var err error
	pathToServiceBackupBinary, err = gexec.Build("github.com/pivotal-cf-experimental/service-backup")
	Expect(err).ToNot(HaveOccurred())

	c := config{
		AWSAccessKeyID:     awsAccessKeyID,
		AWSSecretAccessKey: awsSecretAccessKey,
		PathToBackupBinary: pathToServiceBackupBinary,
	}

	data, err := json.Marshal(c)
	Expect(err).ToNot(HaveOccurred())

	createBucketIfNeeded()

	return data
}

func createBucketIfNeeded() {
	_, err := listRemotePath(existingBucketName, "")

	if err != nil {
		errOut := err.Error()

		if !strings.Contains(errOut, "NoSuchBucket") {
			Fail("Unable to list bucket: " + existingBucketName + " - error: " + errOut)
		}

		params := &s3.CreateBucketInput{
			Bucket: aws.String(existingBucketName),
		}

		_, err := s3Client().CreateBucket(params)
		Expect(err).ToNot(HaveOccurred())
	}
}

func assetPath(filename string) string {
	path, err := filepath.Abs(filepath.Join("assets", filename))
	Expect(err).ToNot(HaveOccurred())
	return path
}

func beforeSuiteOtherNodes(b []byte) {
	var c config
	err := json.Unmarshal(b, &c)
	Expect(err).ToNot(HaveOccurred())

	awsAccessKeyID = c.AWSAccessKeyID
	awsSecretAccessKey = c.AWSSecretAccessKey
	pathToServiceBackupBinary = c.PathToBackupBinary
}

var _ = SynchronizedBeforeSuite(beforeSuiteFirstNode, beforeSuiteOtherNodes)

var _ = SynchronizedAfterSuite(func() {
	return
}, func() {
	gexec.CleanupBuildArtifacts()
})

func s3Client() *s3.S3 {
	s3Config := &aws.Config{
		Region:     "us-east-1",
		MaxRetries: 50,
	}
	return s3.New(s3Config)
}

func downloadRemoteDirectory(bucketName, remotePath, localPath string) error {
	listResp, err := listRemotePath(bucketName, remotePath)
	if err != nil {
		return err
	}

	downloader := s3manager.NewDownloader(&s3manager.DownloadOptions{
		S3: s3Client(),
	})

	for _, remoteFile := range listResp.Contents {
		filePath, fileName := filepath.Split(*remoteFile.Key)

		pathWithoutTimestamp := strings.Join(strings.Split(filePath, "/")[4:], "/")
		destPath := filepath.Join(localPath, pathWithoutTimestamp)
		os.MkdirAll(destPath, 0777)

		output, err := os.Create(filepath.Join(destPath, fileName))
		if err != nil {
			return err
		}

		getObjectInput := &s3.GetObjectInput{
			Bucket: aws.String(bucketName),
			Key:    remoteFile.Key,
		}

		_, err = downloader.Download(output, getObjectInput)
		if err != nil {
			return err
		}
	}
	return nil
}

func listRemotePath(bucketName, remotePath string) (*s3.ListObjectsOutput, error) {
	params := &s3.ListObjectsInput{
		Bucket: aws.String(bucketName),
		Prefix: aws.String(remotePath),
	}

	return s3Client().ListObjects(params)
}

func isRemotePathEmpty(bucketName, remotePath string) bool {
	resp, err := listRemotePath(bucketName, remotePath)
	Expect(err).ToNot(HaveOccurred())

	return len(resp.Contents) == 0
}

func deleteRemotePath(bucketName, remotePath string) error {
	listResp, err := listRemotePath(bucketName, remotePath)
	objectsToDelete := []*s3.ObjectIdentifier{}

	for _, key := range listResp.Contents {
		objectsToDelete = append(objectsToDelete, &s3.ObjectIdentifier{
			Key: key.Key,
		})
	}

	deleteParams := &s3.DeleteObjectsInput{
		Bucket: aws.String(bucketName),
		Delete: &s3.Delete{Objects: objectsToDelete},
	}

	_, err = s3Client().DeleteObjects(deleteParams)
	return err
}

func deleteBucket(bucketName string) {
	err := deleteRemotePath(bucketName, "")
	Expect(err).ToNot(HaveOccurred())
	params := &s3.DeleteBucketInput{Bucket: aws.String(bucketName)}
	_, err = s3Client().DeleteBucket(params)
	Expect(err).ToNot(HaveOccurred())
}
