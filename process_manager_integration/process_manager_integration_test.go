package process_manager_integration_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("process manager", func() {
	var (
		evidenceFile string
		startedFile  string
		configFile   *os.File
	)

	BeforeEach(func() {
		evidenceFile = getTempFilePath()
		startedFile = getTempFilePath()

		var err error
		configFile, err = ioutil.TempFile("", "config.yml")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		os.Remove(evidenceFile)
		os.Remove(startedFile)
		os.Remove(configFile.Name())
	})

	performBackup := func(backupMock, uploadMock string) (*gexec.Session, error) {
		_, err := fmt.Fprintf(configFile, `---
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

			backupCmd := exec.Command(pathToTermTrapperBinary, evidenceFile, startedFile, sleepyTime)
			session, err := gexec.Start(backupCmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(startedFile).Should(BeAnExistingFile())
			Eventually(session, 2).Should(gexec.Exit(1))
		})

		It("should exit on SIGTERM and create evidence file", func() {
			sleepyTime := "1000"

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

			backupScriptMock := fmt.Sprintf("%s %s %s %d", pathToTermTrapperBinary, evidenceFile, startedFile, sleepyTime)
			session, err := performBackup(backupScriptMock, "not-needed")
			Expect(err).NotTo(HaveOccurred())

			By("waiting for the backup command to create the started file", func() {
				Eventually(startedFile, 3).Should(BeAnExistingFile())
			})

			session.Terminate()

			Eventually(session, 2).Should(gexec.Exit())
			Eventually(evidenceFile, 5).Should(BeAnExistingFile())
			Eventually(session.Out).Should(gbytes.Say("All backup processes terminated"))
		})

		It("stops cron before terminating backup commands", func() {
			fileName := "/tmp/log-to-me-please"
			os.Remove(fileName)

			executableCommand := fmt.Sprintf("%s %s %d", assetPath("slowly_logs_on_start"), fileName, 2)
			session, err := performBackup(executableCommand, "not-needed")
			Expect(err).NotTo(HaveOccurred())
			time.Sleep(1010 * time.Millisecond)

			session.Terminate()
			Eventually(session, 16).Should(gexec.Exit())

			logFileContent, err := ioutil.ReadFile(fileName)
			Expect(err).NotTo(HaveOccurred())
			strippedContent := strings.TrimSpace(string(logFileContent))
			logFileLines := strings.Split(strippedContent, "\n")
			Expect(logFileLines).To(HaveLen(1))
			Eventually(session.Out).Should(gbytes.Say("All backup processes terminated"))
		})
	})

	Context("file upload", func() {
		It("propagates a TERM signal to the upload process", func() {
			evidencePath := "/tmp/process_manager_integration_test_sigterm_received.txt"
			os.Remove(evidencePath)

			session, err := performBackup("true", pathToAWSTermTrapperBinary)
			Expect(err).NotTo(HaveOccurred())

			By("waiting for the backup command to create the started file", func() {
				Eventually(startedFile, 3).Should(BeAnExistingFile())
			})

			session.Terminate()

			Eventually(evidenceFile, 5).Should(BeAnExistingFile())
			Eventually(session, 5).Should(gexec.Exit())
			Eventually(session.Out).Should(gbytes.Say("All backup processes terminated"))
		})
	})

})

func assetPath(filename string) string {
	path, err := filepath.Abs(filepath.Join("assets", filename))
	Expect(err).ToNot(HaveOccurred())
	return path
}

func getTempFilePath() string {
	f, err := ioutil.TempFile("", "process_manager")
	Expect(err).ToNot(HaveOccurred())
	err = os.Remove(f.Name())
	Expect(err).ToNot(HaveOccurred())
	return f.Name()
}
