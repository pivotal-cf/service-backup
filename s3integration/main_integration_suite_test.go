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
	awsAccessKeyIDEnvKey               = "AWS_ACCESS_KEY_ID"
	awsSecretAccessKeyEnvKey           = "AWS_SECRET_ACCESS_KEY"
	awsAccessKeyIDEnvKeyRestricted     = "AWS_ACCESS_KEY_ID_RESTRICTED"
	awsSecretAccessKeyEnvKeyRestricted = "AWS_SECRET_ACCESS_KEY_RESTRICTED"
	cephAccessKeyIDEnvKey              = "CEPH_ACCESS_KEY_ID"
	cephSecretAccessKeyEnvKey          = "CEPH_SECRET_ACCESS_KEY"
	cephEndpointURLEnvKey              = "CEPH_ENDPOINT_URL"

	existingBucketName = "service-backup-integration-test2"
	awsTimeout         = "20s"

	endpointURL  = ""
	cronSchedule = "*/5 * * * * *" // every 5 seconds of every minute of every day etc
)

var (
	pathToServiceBackupBinary    string
	pathToManualBackupBinary     string
	awsAccessKeyID               string
	awsSecretAccessKey           string
	awsAccessKeyIDRestricted     string
	awsSecretAccessKeyRestricted string
	cephAccessKeyID              string
	cephSecretAccessKey          string
	cephEndpointURL              string
	destPath                     string

	s3TestClient   *s3testclient.S3TestClient
	cephTestClient *s3testclient.S3TestClient
)

type config struct {
	AWSAccessKeyID               string `json:"awsAccessKeyID"`
	AWSSecretAccessKey           string `json:"awsSecretAccessKey"`
	AWSAccessKeyIDRestricted     string `json:"awsAccessKeyIDRestricted"`
	AWSSecretAccessKeyRestricted string `json:"awsSecretAccessKeyRestricted"`
	CephAccessKeyID              string `json:"cephAccessKeyID"`
	CephSecretAccessKey          string `json:"cephSecretAccessKey"`
	CephEndpointURL              string `json:"cephEndpointURL"`
	PathToBackupBinary           string `json:"pathToBackupBinary"`
	PathToManualBinary           string `json:"pathToManualBinary"`
}

func TestServiceBackupBinary(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "S3 integration Suite")
}

func beforeSuiteFirstNode() []byte {
	awsAccessKeyID = os.Getenv(awsAccessKeyIDEnvKey)
	awsSecretAccessKey = os.Getenv(awsSecretAccessKeyEnvKey)
	awsAccessKeyIDRestricted = os.Getenv(awsAccessKeyIDEnvKeyRestricted)
	awsSecretAccessKeyRestricted = os.Getenv(awsSecretAccessKeyEnvKeyRestricted)
	cephAccessKeyID = os.Getenv(cephAccessKeyIDEnvKey)
	cephSecretAccessKey = os.Getenv(cephSecretAccessKeyEnvKey)
	cephEndpointURL = os.Getenv(cephEndpointURLEnvKey)

	if awsAccessKeyID == "" || awsSecretAccessKey == "" {
		Fail(fmt.Sprintf("Specify valid AWS credentials using the env variables %s and %s", awsAccessKeyIDEnvKey, awsSecretAccessKeyEnvKey))
	}
	if awsAccessKeyIDRestricted == "" || awsSecretAccessKeyRestricted == "" {
		Fail(fmt.Sprintf("Specify valid AWS credentials using the env variables %s and %s", awsAccessKeyIDEnvKeyRestricted, awsSecretAccessKeyEnvKeyRestricted))
	}
	if cephAccessKeyID == "" || cephSecretAccessKey == "" || cephEndpointURL == "" {
		Fail(fmt.Sprintf("Specify valid Ceph credentials and endpoint using the env variables %s, %s and %s", cephAccessKeyIDEnvKey, cephSecretAccessKeyEnvKey, cephEndpointURLEnvKey))
	}

	var err error
	pathToServiceBackupBinary, err = gexec.Build("github.com/pivotal-cf-experimental/service-backup")
	Expect(err).ToNot(HaveOccurred())
	pathToManualBackupBinary, err = gexec.Build("github.com/pivotal-cf-experimental/service-backup/cmd/manual-backup")
	Expect(err).ToNot(HaveOccurred())

	c := config{
		AWSAccessKeyID:               awsAccessKeyID,
		AWSSecretAccessKey:           awsSecretAccessKey,
		AWSAccessKeyIDRestricted:     awsAccessKeyIDRestricted,
		AWSSecretAccessKeyRestricted: awsSecretAccessKeyRestricted,
		CephAccessKeyID:              cephAccessKeyID,
		CephSecretAccessKey:          cephSecretAccessKey,
		CephEndpointURL:              cephEndpointURL,
		PathToBackupBinary:           pathToServiceBackupBinary,
		PathToManualBinary:           pathToManualBackupBinary,
	}

	data, err := json.Marshal(c)
	Expect(err).ToNot(HaveOccurred())

	s3TestClient = s3testclient.New(endpointURL, awsAccessKeyID, awsSecretAccessKey)
	Expect(s3TestClient.CreateRemotePathIfNeeded(existingBucketName)).To(Succeed())

	cephTestClient = s3testclient.New(cephEndpointURL, cephAccessKeyID, cephSecretAccessKey)
	Expect(cephTestClient.CreateRemotePathIfNeeded(existingBucketName)).To(Succeed())

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
	awsAccessKeyIDRestricted = c.AWSAccessKeyIDRestricted
	awsSecretAccessKeyRestricted = c.AWSSecretAccessKeyRestricted
	cephAccessKeyID = c.CephAccessKeyID
	cephSecretAccessKey = c.CephSecretAccessKey
	cephEndpointURL = c.CephEndpointURL
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
