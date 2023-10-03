package process_manager_integration_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/pivotal-cf/service-backup/testhelpers"
)

var _ = Describe("process manager", func() {
	performBackup := func(backupMock, uploadMock, evidenceFile, startedFile string) (*gexec.Session, error) {
		var err error
		configFile, err := ioutil.TempFile("", "config.yml")
		Expect(err).NotTo(HaveOccurred())

		_, err = fmt.Fprintf(configFile, `---
destinations:
- type: s3
  config:
    endpoint_url: %s
    region: %s
aws_cli_path: %s
source_executable: %s
cron_schedule: '* * * * * *'
`, evidenceFile, startedFile, uploadMock, backupMock)
		Expect(err).NotTo(HaveOccurred())

		backupCmd := exec.Command(pathToServiceBackupBinary, configFile.Name())
		session, err := gexec.Start(backupCmd, GinkgoWriter, GinkgoWriter)

		return session, err
	}

	Context("inspiring confidence in our term-trapper fixture", func() {
		It("should create startedFile and then exit 0 after sleepytime", func() {
			sleepyTime := "1"

			evidenceFile := testhelpers.GetTempFilePath()
			startedFile := testhelpers.GetTempFilePath()
			defer os.Remove(startedFile)
			defer os.Remove(evidenceFile)

			backupCmd := exec.Command(pathToTermTrapperBinary, evidenceFile, startedFile, sleepyTime)
			session, err := gexec.Start(backupCmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(startedFile).Should(BeAnExistingFile())
			Eventually(session, 2).Should(gexec.Exit(1))
		})

		It("should exit on SIGTERM and create evidence file", func() {
			sleepyTime := "1000"

			evidenceFile := testhelpers.GetTempFilePath()
			startedFile := testhelpers.GetTempFilePath()
			defer os.Remove(startedFile)
			defer os.Remove(evidenceFile)

			backupCmd := exec.Command(pathToTermTrapperBinary, evidenceFile, startedFile, sleepyTime)
			session, err := gexec.Start(backupCmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			By("waiting for the child process to start", func() {
				Eventually(startedFile).Should(BeAnExistingFile())
			})

			session.Terminate()
			Eventually(session).Should(gexec.Exit(129))
			Eventually(evidenceFile).Should(BeAnExistingFile())
		})

	})

	Context("Backup command", func() {
		It("propagates a TERM signal to child backup commands", func() {
			sleepyTime := 1000

			evidenceFile := testhelpers.GetTempFilePath()
			startedFile := testhelpers.GetTempFilePath()
			defer os.Remove(startedFile)
			defer os.Remove(evidenceFile)

			backupScriptMock := fmt.Sprintf("%s %s %s %d", pathToTermTrapperBinary, evidenceFile, startedFile, sleepyTime)
			session, err := performBackup(backupScriptMock, "not-needed", evidenceFile, startedFile)
			Expect(err).NotTo(HaveOccurred())

			By("waiting for the backup command to create the started file", func() {
				Eventually(startedFile, 3).Should(BeAnExistingFile())
			})

			session.Terminate()

			Eventually(session, 2).Should(gexec.Exit())
			Eventually(evidenceFile, 5).Should(BeAnExistingFile())
			Eventually(session.Out).Should(gbytes.Say("All backup processes terminated"))
		})

		It("doesn't start a new backup after a sigterm is received", func() {
			sleepyTime := 3000
			sleepAfterSigterm := 2000

			evidenceFile := testhelpers.GetTempFilePath()
			startedFile := testhelpers.GetTempFilePath()
			defer os.Remove(startedFile)
			defer os.Remove(evidenceFile)

			backupScriptMock := fmt.Sprintf("%s %s %s %d %d", pathToTermTrapperBinary, evidenceFile, startedFile, sleepyTime, sleepAfterSigterm)
			session, err := performBackup(backupScriptMock, "not-needed", evidenceFile, startedFile)
			Expect(err).NotTo(HaveOccurred())

			By("waiting for the backup command to create the started file", func() {
				Eventually(startedFile, 3).Should(BeAnExistingFile())
			})

			session.Terminate()

			Eventually(session, 5).Should(gexec.Exit())
			Expect(evidenceFile).To(BeAnExistingFile())
			Expect(session.Out).To(gbytes.Say("All backup processes terminated"))

			backupsStarted := strings.Count(string(session.Out.Contents()), "Perform backup started")
			Expect(backupsStarted).To(Equal(1))
		})
	})

})

func assetPath(filename string) string {
	path, err := filepath.Abs(filepath.Join("assets", filename))
	Expect(err).ToNot(HaveOccurred())
	return path
}
