package scp_test

import (
	"os"
	"time"

	"github.com/pivotal-cf/service-backup/testhelpers"

	"code.cloudfoundry.org/lager/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/service-backup/process"
	"github.com/pivotal-cf/service-backup/scp"
)

var _ = Describe("scp", func() {

	It("terminates the child process when the process manager gets the terminate call", func() {
		fakeScpCmd := pathToBackupFixture

		startedPath := testhelpers.GetTempFilePath()
		evidencePath := testhelpers.GetTempFilePath()
		defer os.Remove(evidencePath)
		defer os.Remove(startedPath)

		fakeRemotePathFn := func() string { return startedPath }

		processManager := process.NewManager()

		scpClient := scp.New("foo", "foo", 1, evidencePath, evidencePath, "somefgp", fakeRemotePathFn)
		scpClient.SCPCommand = fakeScpCmd
		scpClient.SSHCommand = "true"

		go func() {
			defer GinkgoRecover()

			err := scpClient.Upload("/tmp", lager.NewLogger("foo"), processManager)
			Expect(err).To(MatchError(ContainSubstring("SIGTERM propagated to child process")))
		}()

		Eventually(startedPath).Should(BeAnExistingFile())

		processManager.Terminate()
		SetDefaultEventuallyTimeout(2 * time.Second)
		Eventually(evidencePath).Should(BeAnExistingFile())
	})
})
