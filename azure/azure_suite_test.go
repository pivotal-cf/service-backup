package azure_test

import (
	"testing"

	"github.com/onsi/gomega/gexec"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAzure(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Azure Suite")
}

var (
	pathToTermTrapper string
)

var _ = BeforeSuite(func() {
	var err error
	pathToTermTrapper, err = gexec.Build("github.com/pivotal-cf/service-backup/azure/fixtures/azure-term-trapper")
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})
