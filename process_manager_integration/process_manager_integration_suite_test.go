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

var err error
var pathToServiceBackupBinary string
var pathToTermTrapperBinary string

var _ = BeforeSuite(func() {
	pathToServiceBackupBinary, err = gexec.Build("github.com/pivotal-cf/service-backup")
	Expect(err).ToNot(HaveOccurred())

	pathToTermTrapperBinary, err = gexec.Build("github.com/pivotal-cf/service-backup/process_manager_integration/fixtures/term-trapper")
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})
