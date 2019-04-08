package scp_test

import (
	"testing"

	"github.com/onsi/gomega/gexec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestScp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Scp Suite")
}

var (
	pathToBackupFixture string
)

var _ = BeforeSuite(func() {
	var err error
	pathToBackupFixture, err = gexec.Build("github.com/pivotal-cf/service-backup/scp/fixtures/scp-term-trapper")
	Expect(err).ToNot(HaveOccurred())
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})
