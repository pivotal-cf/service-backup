package scpintegration_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

var pathToServiceBackupBinary string

var _ = SynchronizedBeforeSuite(func() []byte {
	var err error
	pathToServiceBackupBinary, err = gexec.Build("github.com/pivotal-cf-experimental/service-backup")
	Expect(err).ToNot(HaveOccurred())
	return []byte(pathToServiceBackupBinary)
}, func(binPath []byte) {
	pathToServiceBackupBinary = string(binPath)
})

var _ = SynchronizedAfterSuite(func() {
}, func() {
	gexec.CleanupBuildArtifacts()
})

func TestScpintegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SCP Suite")
}
