package gcsintegration_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"

	"github.com/onsi/gomega/gexec"
)

func TestGCSIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "GCS Integration Suite")
}

var serviceBackupBinaryPath string

var _ = SynchronizedBeforeSuite(func() []byte {
	binaryPath, err := gexec.Build("github.com/pivotal-cf/service-backup")
	Expect(err).ToNot(HaveOccurred())

	return []byte(binaryPath)
}, func(binaryPath []byte) {
	serviceBackupBinaryPath = string(binaryPath)
})

var _ = SynchronizedAfterSuite(func() {}, func() {
	gexec.CleanupBuildArtifacts()
})
