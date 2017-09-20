package process_terminator_integration_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("process terminator", func() {
	It("propagates a TERM signal to its child backup command", func() {
		pathToServiceBackupBinary, err := gexec.Build("github.com/pivotal-cf/service-backup")
		Expect(err).ToNot(HaveOccurred())
		evidencePath := "/tmp/process_terminator_integration_test_sigterm_received.txt"
		os.Remove(evidencePath)

		sleepyTime := 20

		session, cmd, err := performBackup(pathToServiceBackupBinary, evidencePath, sleepyTime)

		Expect(err).NotTo(HaveOccurred())
		time.Sleep(time.Second)

		cmd.Process.Signal(syscall.SIGTERM)
		cmd.Wait()
		Eventually(session, 2).Should(gexec.Exit())

		Expect(evidencePath).To(BeAnExistingFile())
	})
})

func performBackup(pathToServiceBackupBinary string, evidencePath string, sleepyTime int) (*gexec.Session, *exec.Cmd, error) {
	configFile, err := ioutil.TempFile("", "config.yml")
	Expect(err).NotTo(HaveOccurred())

	configContent := fmt.Sprintf(`---
destinations:
- type: s3
  config:
    endpoint_url: www.example.com
    bucket_name: a_bucket
    bucket_path: a_bucket_path
    access_key_id: AKAIADCIWI@ICFIJ
    secret_access_key: ASCDMIACDNI@UD937e9237aSCDAS
source_folder: /tmp
source_executable: %s %s %d
exit_if_in_progress: true
cron_schedule: '* * * * * *'
cleanup_executable: ''
missing_properties_message: custom message
deployment_name: 'service-backup'
add_deployment_name_to_backup_path: true`, assetPath("term_trapper"), evidencePath, sleepyTime)
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
