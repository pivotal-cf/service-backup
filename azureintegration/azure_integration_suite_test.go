package azureintegration_test

import (
	"encoding/json"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

const (
	azureTimeout           = "20s"
	azureAccountNameEnvKey = "AZURE_STORAGE_ACCOUNT"
	azureAccountKeyEnvKey  = "AZURE_STORAGE_ACCESS_KEY"
	azureCmd               = ""
)

var (
	azureAccountName = os.Getenv(azureAccountNameEnvKey)
	azureAccountKey  = os.Getenv(azureAccountKeyEnvKey)
)

func TestAzureIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AzureIntegration Suite")
}

type TestData struct {
	PathToServiceBackupBinary string
}

var (
	pathToServiceBackupBinary string
)

var _ = SynchronizedBeforeSuite(func() []byte {
	var err error

	pathToServiceBackupBinary, err = gexec.Build("github.com/pivotal-cf/service-backup")
	Expect(err).ToNot(HaveOccurred())

	forOtherNodes, err := json.Marshal(TestData{
		PathToServiceBackupBinary: pathToServiceBackupBinary,
	})
	Expect(err).ToNot(HaveOccurred())
	return forOtherNodes
}, func(data []byte) {
	var t TestData
	Expect(json.Unmarshal(data, &t)).To(Succeed())

	pathToServiceBackupBinary = t.PathToServiceBackupBinary
})

var _ = SynchronizedAfterSuite(func() {
}, func() {
	gexec.CleanupBuildArtifacts()
})
