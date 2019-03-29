package process_manager_integration_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

func TestProcessManagerIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ProcessManagerIntegration Suite")
}

var (
	pathToServiceBackupBinary  string
	pathToTermTrapperBinary    string
	pathToAWSTermTrapperBinary string
)

var _ = BeforeSuite(func() {
	var err error
	pathToServiceBackupBinary, err = gexec.Build("github.com/pivotal-cf/service-backup")
	Expect(err).ToNot(HaveOccurred())

	pathToTermTrapperBinary, err = gexec.Build("github.com/pivotal-cf/service-backup/process_manager_integration/fixtures/backup-term-trapper")
	Expect(err).NotTo(HaveOccurred())

	pathToAWSTermTrapperBinary, err = gexec.Build("github.com/pivotal-cf/service-backup/process_manager_integration/fixtures/aws-term-trapper")
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})
