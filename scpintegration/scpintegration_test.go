package scpintegration_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
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
			flags      []string
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

			flags = []string{
				"scp",
				"-source-folder", dirToBackup,
				"-backup-creator-cmd", "ls",
				"-cleanup-cmd", "",
				"-cron-schedule", "*/5 * * * * *",
				"-ssh-host", "localhost",
				"-ssh-port", "22",
				"-ssh-user", unixUser.Username,
				"-ssh-private-key-path", privateKeyPath,
				"-dest-path", destPath,
			}
		})

		JustBeforeEach(func() {
			var err error
			cmd := exec.Command(pathToServiceBackupBinary, flags...)
			runningBin, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
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

		Context("when not all mandatory flags are supplied for SCP to work", func() {
			BeforeEach(func() {
				flags = []string{
					"scp",
					"-source-folder", "somedir",
					"-backup-creator-cmd", "ls",
					"-cleanup-cmd", "",
					"-cron-schedule", "*/5 * * * * *",
					"-ssh-host", "localhost",
					"-ssh-port", "22",
					"-ssh-user", unixUser.Username,
					"-dest-path", destPath,
				}
			})

			It("exits with non-zero", func() {
				Expect(runningBin.Wait(time.Second).ExitCode()).ToNot(Equal(0))
				Expect(string(runningBin.Out.Contents())).To(ContainSubstring("Flag ssh-private-key-path not provided"))
			})
		})
	})
})
