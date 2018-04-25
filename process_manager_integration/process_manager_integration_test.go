package process_manager_integration_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("process manager", func() {
	Context("Backup command", func() {
		It("propagates a TERM signal to child backup commands", func() {
			evidenceFile, err := ioutil.TempFile("", "")
			Expect(err).ToNot(HaveOccurred())
			err = os.Remove(evidenceFile.Name())
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove(evidenceFile.Name())

			sleepyTime := 20

			session, cmd, err := performBackup(pathToServiceBackupBinary, fmt.Sprintf("%s %s %d", assetPath("term_trapper"), evidenceFile.Name(), sleepyTime))

			Expect(err).NotTo(HaveOccurred())
			time.Sleep(time.Second)

			cmd.Process.Signal(syscall.SIGTERM)
			Eventually(session, 2).Should(gexec.Exit())
			Expect(evidenceFile.Name()).To(BeAnExistingFile())
			Eventually(session.Out).Should(gbytes.Say("All backup processes terminated"))
		})

		It("stops cron before terminating backup commands", func() {
			fileName := "/tmp/log-to-me-please"
			os.Remove(fileName)

			executableCommand := fmt.Sprintf("%s %s %d", assetPath("slowly_logs_on_start"), fileName, 2)
			session, cmd, err := performBackup(pathToServiceBackupBinary, executableCommand)
			time.Sleep(1010 * time.Millisecond)

			cmd.Process.Signal(syscall.SIGTERM)
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

			sleepyTime := 20

			session, cmd, err := performBackup(pathToServiceBackupBinary, fmt.Sprintf("%s %s %d", assetPath("term_trapper"), evidencePath, sleepyTime))

			Expect(err).NotTo(HaveOccurred())
			time.Sleep(time.Second)

			cmd.Process.Signal(syscall.SIGTERM)
			Eventually(session, 2).Should(gexec.Exit())
			Expect(evidencePath).To(BeAnExistingFile())
			Eventually(session.Out).Should(gbytes.Say("All backup processes terminated"))
		})
	})
})

func performBackup(pathToServiceBackupBinary string, executable string) (*gexec.Session, *exec.Cmd, error) {
	configFile, err := ioutil.TempFile("", "config.yml")
	Expect(err).NotTo(HaveOccurred())

	configContent := fmt.Sprintf(`---
destinations:
- type: s3
  config:
    endpoint_url: www.example.com
    bucket_name: a_bucket
    bucket_path: a_bucket_path
    access_key_id: some_access_key
    secret_access_key: some_secret
source_folder: /tmp
source_executable: %s
exit_if_in_progress: false
cron_schedule: '* * * * * *'
cleanup_executable: ''
missing_properties_message: custom message
deployment_name: 'service-backup'
add_deployment_name_to_backup_path: true`, executable)
	configFile.Write([]byte(configContent))
	configFile.Close()

	backupCmd := exec.Command(pathToServiceBackupBinary, configFile.Name())
	session, err := gexec.Start(backupCmd, GinkgoWriter, GinkgoWriter)

	return session, backupCmd, err
}

func assetPath(filename string) string {
	path, err := filepath.Abs(filepath.Join("assets", filename))
	Expect(err).ToNot(HaveOccurred())
	return path
}
