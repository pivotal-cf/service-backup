package integration

import (
	"fmt"
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

const (
	awsAccessKeyIDEnvKey     = "AWS_ACCESS_KEY_ID"
	awsSecretAccessKeyEnvKey = "AWS_SECRET_ACCESS_KEY"

	bucketName = "service-backup-integration-test"
	awsTimeout = "10s"
)

var (
	pathToServiceBackupBinary string
	awsAccessKeyID            string
	awsSecretAccessKey        string
)

func TestServiceBackupBinary(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Backup Binary Suite")
}

var _ = BeforeSuite(func() {
	var err error
	pathToServiceBackupBinary, err = gexec.Build("github.com/pivotal-cf-experimental/service-backup")
	Expect(err).ToNot(HaveOccurred())

	awsAccessKeyID = os.Getenv(awsAccessKeyIDEnvKey)
	awsSecretAccessKey = os.Getenv(awsSecretAccessKeyEnvKey)

	if awsAccessKeyID == "" || awsSecretAccessKey == "" {
		Fail(fmt.Sprintf("Specify valid AWS credentials using the env variables %s and %s", awsAccessKeyIDEnvKey, awsSecretAccessKeyEnvKey))
	}
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})
