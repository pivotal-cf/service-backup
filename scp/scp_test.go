package scp_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/service-backup/process"
	"github.com/pivotal-cf/service-backup/scp"
)

var _ = Describe("scp", func() {

	It("terminates the child process when the process manager gets the terminate call", func() {
		fakeScpCmd := assetPath("term_trapper")
		fakeRemotePathFn := func() string { return "hi" }

		evidenceFile, err := ioutil.TempFile("", "scp-test")
		Expect(err).ToNot(HaveOccurred())
		evidencePath := evidenceFile.Name()
		err = os.Remove(evidencePath)
		Expect(err).ToNot(HaveOccurred())
		defer os.Remove(evidencePath)

		processManager := process.NewManager()

		scpClient := scp.New("foo", "foo", 1, evidencePath, evidencePath, "somefgp", fakeRemotePathFn)
		scpClient.SCPCommand = fakeScpCmd
		scpClient.SSHCommand = "true"

		go func() {
			defer GinkgoRecover()
			err := scpClient.Upload("/tmp", lager.NewLogger("foo"), processManager)
			Expect(err).To(MatchError(ContainSubstring("SIGTERM propagated to child process")))
		}()

		time.Sleep(100 * time.Millisecond)
		processManager.Terminate()
		SetDefaultEventuallyTimeout(2 * time.Second)
		Eventually(evidencePath).Should(BeAnExistingFile())
	})
})

func assetPath(filename string) string {
	path, err := filepath.Abs(filepath.Join("assets", filename))
	Expect(err).ToNot(HaveOccurred())
	return path
}
