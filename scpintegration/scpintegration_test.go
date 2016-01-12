package scpintegration_test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("SCP Backup", func() {
	Context("When SCP server is correctly configured with flags", func() {
		const (
			sshKeyUsername = "service-backup-tmp-key"
		)

		var (
			runningBin *gexec.Session
			baseDir    string
			destPath   string
			unixUser   *user.User
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
			sshKeys := filepath.Join(baseDir, "keys")
			Expect(os.Mkdir(dirToBackup, 0755)).To(Succeed())
			Expect(os.Mkdir(sshKeys, 0755)).To(Succeed())
			Expect(os.Mkdir(destPath, 0755)).To(Succeed())

			unixUser, err = user.Current()
			Expect(err).NotTo(HaveOccurred())

			privateKeyPath := filepath.Join(sshKeys, "id_rsa")
			Expect(exec.Command("ssh-keygen", "-t", "rsa", "-b", "4096", "-C", sshKeyUsername,
				"-N", "", "-f", privateKeyPath).Run()).To(Succeed())

			Expect(os.MkdirAll(filepath.Join(unixUser.HomeDir, ".ssh"), 0700)).To(Succeed())
			authKeys, err := os.OpenFile(filepath.Join(unixUser.HomeDir, ".ssh", "authorized_keys"), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
			Expect(err).ToNot(HaveOccurred())
			pubKey, err := os.Open(filepath.Join(sshKeys, "id_rsa.pub"))
			Expect(err).ToNot(HaveOccurred())
			defer authKeys.Close()
			defer pubKey.Close()
			_, err = io.Copy(authKeys, pubKey)
			Expect(err).ToNot(HaveOccurred())

			Expect(ioutil.WriteFile(filepath.Join(dirToBackup, "1.txt"), []byte("1"), 0644)).To(Succeed())
			Expect(os.Mkdir(filepath.Join(dirToBackup, "subdir"), 0755)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(dirToBackup, "subdir", "2.txt"), []byte("2"), 0644)).To(Succeed())

			cmd := exec.Command(
				pathToServiceBackupBinary, "-source-folder", dirToBackup,
				"-backup-creator-cmd", "ls", "-cleanup-cmd", "", "-cron-schedule", "*/5 * * * * *",
				"-ssh-host", "localhost", "-ssh-port", "22", "-ssh-user", unixUser.Username,
				"-ssh-private-key-path", privateKeyPath, "-dest-path", destPath,
			)

			runningBin, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			Expect(os.RemoveAll(baseDir)).To(Succeed())
			Eventually(runningBin.Terminate()).Should(gexec.Exit())
			authKeysFilePath := filepath.Join(unixUser.HomeDir, ".ssh", "authorized_keys")
			authKeysContent, err := ioutil.ReadFile(authKeysFilePath)
			Expect(err).NotTo(HaveOccurred())

			trimmedAuthKeysLines := [][]byte{}
			for _, line := range bytes.Split(authKeysContent, []byte("\n")) {
				if !bytes.Contains(line, []byte(sshKeyUsername)) {
					trimmedAuthKeysLines = append(trimmedAuthKeysLines, line)
				}
			}

			trimmedAuthKeysContent := bytes.Join(trimmedAuthKeysLines, []byte("\n"))
			err = ioutil.WriteFile(authKeysFilePath, trimmedAuthKeysContent, 0600)
			Expect(err).NotTo(HaveOccurred())
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
