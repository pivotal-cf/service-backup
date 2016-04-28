package s3integration_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/pivotal-cf-experimental/service-backup/s3testclient"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

const (
	awsAccessKeyIDEnvKey     = "AWS_ACCESS_KEY_ID"
	awsSecretAccessKeyEnvKey = "AWS_SECRET_ACCESS_KEY"

	existingBucketInNonDefaultRegion = "service-backup-integration-test"
	existingBucketInDefaultRegion    = "service-backup-integration-test2"
	awsTimeout                       = "20s"

	cronSchedule = "*/5 * * * * *" // every 5 seconds of every minute of every day etc
)

var (
	endpointURL               string
	pathToServiceBackupBinary string
	pathToManualBackupBinary  string
	awsAccessKeyID            string
	awsSecretAccessKey        string
	destPath                  string

	s3TestClient *s3testclient.S3TestClient
)

type config struct {
	AWSAccessKeyID     string `json:"awsAccessKeyID"`
	AWSSecretAccessKey string `json:"awsSecretAccessKey"`
	PathToBackupBinary string `json:"pathToBackupBinary"`
	PathToManualBinary string `json:"pathToManualBinary"`
}

func TestServiceBackupBinary(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "S3 integration Suite")
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
	pathToManualBackupBinary, err = gexec.Build("github.com/pivotal-cf-experimental/service-backup/cmd/manual-backup")
	Expect(err).ToNot(HaveOccurred())

	c := config{
		AWSAccessKeyID:     awsAccessKeyID,
		AWSSecretAccessKey: awsSecretAccessKey,
		PathToBackupBinary: pathToServiceBackupBinary,
		PathToManualBinary: pathToManualBackupBinary,
	}

	data, err := json.Marshal(c)
	Expect(err).ToNot(HaveOccurred())

	s3TestClient = s3testclient.New("", awsAccessKeyID, awsSecretAccessKey)
	Expect(s3TestClient.CreateRemotePathIfNeeded(existingBucketInDefaultRegion)).To(Succeed())
	Expect(s3TestClient.CreateRemotePathIfNeeded(existingBucketInNonDefaultRegion)).To(Succeed())

	return data
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
	pathToManualBackupBinary = c.PathToManualBinary
	s3TestClient = s3testclient.New(endpointURL, awsAccessKeyID, awsSecretAccessKey)
}

var _ = SynchronizedBeforeSuite(beforeSuiteFirstNode, beforeSuiteOtherNodes)

var _ = SynchronizedAfterSuite(func() {
	return
}, func() {
	gexec.CleanupBuildArtifacts()
})
