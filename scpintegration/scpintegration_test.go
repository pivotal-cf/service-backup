package scpintegration_test

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

var _ = Describe("SCP Backup", func() {
	Context("When SCP server is correctly configured with flags", func() {
		var (
			runningBin *gexec.Session
			baseDir    string
			destPath   string
		)

		pathWithDate := func(endParts ...string) string {
			today := time.Now()
			dateComponents := []string{fmt.Sprintf("%d", today.Year()), fmt.Sprintf("%02d", today.Month()), fmt.Sprintf("%02d", today.Day())}
			args := []string{destPath}
			args = append(args, dateComponents...)
			args = append(args, endParts...)
			return filepath.Join(args...)
		}

		BeforeEach(func() {
			var err error
			baseDir, err = ioutil.TempDir("", "scp-integration-tests")
			Expect(err).NotTo(HaveOccurred())
			dirToBackup := filepath.Join(baseDir, "source")
			destPath = filepath.Join(baseDir, "target")
			Expect(os.Mkdir(dirToBackup, 0755)).To(Succeed())
			Expect(os.Mkdir(destPath, 0755)).To(Succeed())

			Expect(ioutil.WriteFile(filepath.Join(dirToBackup, "1.txt"), []byte("1"), 0644)).To(Succeed())
			Expect(os.Mkdir(filepath.Join(dirToBackup, "subdir"), 0755)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(dirToBackup, "subdir", "2.txt"), []byte("2"), 0644)).To(Succeed())

			runningBin = performBackup("localhost", unixUser.Username, destPath, string(privateKeyContents), 22, dirToBackup)
		})

		AfterEach(func() {
			Expect(os.RemoveAll(baseDir)).To(Succeed())
			Eventually(runningBin.Terminate()).Should(gexec.Exit())
		})

		It("copies files over SCP", func() {
			Eventually(runningBin.Out, time.Second*10).Should(gbytes.Say("scp completed"))
			runningBin.Terminate().Wait()
			content1, err := ioutil.ReadFile(pathWithDate("1.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(content1).To(Equal([]byte("1")))
			content2, err := ioutil.ReadFile(pathWithDate("subdir", "2.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(content2).To(Equal([]byte("2")))
		})
	})
})

func runBackup(params ...string) *gexec.Session {
	backupCmd := exec.Command(pathToServiceBackupBinary, params...)
	session, err := gexec.Start(backupCmd, GinkgoWriter, GinkgoWriter)
	Expect(err).ToNot(HaveOccurred())
	return session
}

func performBackup(scpServer, scpUser, scpDestination, scpKey string, scpPort int, sourceFolder string) *gexec.Session {
	file, err := ioutil.TempFile("", "config.yml")
	Expect(err).NotTo(HaveOccurred())

	parts := strings.Split(scpKey, "\n")
	scpKey = strings.Join(parts, "\n      ")

	file.Write([]byte(fmt.Sprintf(`---
destinations:
- type: scp
  config:
    server: %s
    user: %s
    destination: %s
    key: |
      %s
    port: %d
source_folder: %s
source_executable: true
exit_if_in_progress: true
cron_schedule: '*/5 * * * * *'
cleanup_executable: true
missing_properties_message: custom message`, scpServer, scpUser, scpDestination, scpKey, scpPort, sourceFolder,
	)))
	file.Close()

	return runBackup(file.Name())
}
