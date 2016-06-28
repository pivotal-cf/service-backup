package multiintegration_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/nu7hatch/gouuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

const (
	awsAccessKeyIDEnvKey     = "AWS_ACCESS_KEY_ID"
	awsSecretAccessKeyEnvKey = "AWS_SECRET_ACCESS_KEY"
)

var _ = Describe("Multiple destinations backup", func() {
	Context("When SCP and S3 destinations are correctly configured", func() {
		var (
			runningBin  *gexec.Session
			baseDir     string
			destPathSCP string
			destPathS3  string
		)

		BeforeEach(func() {
			var err error
			baseDir, err = ioutil.TempDir("", "multiple-destinations-integration-tests")
			Expect(err).NotTo(HaveOccurred())
			dirToBackup := filepath.Join(baseDir, "source")
			destPathSCP = filepath.Join(baseDir, "target")
			Expect(os.Mkdir(dirToBackup, 0755)).To(Succeed())
			Expect(os.Mkdir(destPathSCP, 0755)).To(Succeed())

			Expect(ioutil.WriteFile(filepath.Join(dirToBackup, "1.txt"), []byte("1"), 0644)).To(Succeed())
			Expect(os.Mkdir(filepath.Join(dirToBackup, "subdir"), 0755)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(dirToBackup, "subdir", "2.txt"), []byte("2"), 0644)).To(Succeed())

			destS3UUID, err := uuid.NewV4()
			Expect(err).ToNot(HaveOccurred())
			destPathS3 = destS3UUID.String()

			runningBin = runBackup(createConfigFile(`---
destinations:
- type: scp
  config:
    server: localhost
    user: %s
    destination: %s
    key: |
      %s
    port: 22
- type: s3
  config:
    endpoint_url: 'https://s3.amazonaws.com'
    bucket_name: %s
    bucket_path: %s
    access_key_id: %s
    secret_access_key: %s
source_folder: %s
source_executable: true
aws_cli_path: aws
exit_if_in_progress: true
cron_schedule: '*/5 * * * * *'
cleanup_executable: true
missing_properties_message: custom message`, unixUser.Username, destPathSCP, padWithSpaces(string(privateKeyContents), 6),
				existingBucketInDefaultRegion, destPathS3, awsAccessKeyID, awsSecretAccessKey,
				dirToBackup))
		})

		AfterEach(func() {
			Expect(os.RemoveAll(baseDir)).To(Succeed())
			Eventually(runningBin.Terminate()).Should(gexec.Exit())
		})

		It("copies files with SCP", func() {
			Eventually(runningBin.Out, time.Second*10).Should(gbytes.Say("scp completed"))
			runningBin.Terminate().Wait()

			content1, err := ioutil.ReadFile(pathWithDateForSCP(destPathSCP, "1.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(content1).To(Equal([]byte("1")))

			content2, err := ioutil.ReadFile(pathWithDateForSCP(destPathSCP, "subdir", "2.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(content2).To(Equal([]byte("2")))
		})

		It("copies files to S3", func() {
			Eventually(runningBin.Out, time.Second*10).Should(gbytes.Say("s3 completed"))
			runningBin.Terminate().Wait()

			downloadFolder, err := ioutil.TempDir("", "backup-tests")
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove(downloadFolder)

			err = s3TestClient.DownloadRemoteDirectory(
				existingBucketInDefaultRegion,
				destPathS3,
				downloadFolder,
			)
			Expect(err).ToNot(HaveOccurred())

			content1, err := ioutil.ReadFile(downloadedS3Path(downloadFolder, "1.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(content1).To(Equal([]byte("1")))

			content2, err := ioutil.ReadFile(downloadedS3Path(downloadFolder, "subdir", "2.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(content2).To(Equal([]byte("2")))
		})
	})
})

func padWithSpaces(data string, len int) string {
	spaces := make([]rune, len)
	for i, _ := range spaces {
		spaces[i] = ' '
	}

	parts := strings.Split(data, "\n")
	return strings.Join(parts, "\n"+string(spaces))
}

func runBackup(params ...string) *gexec.Session {
	backupCmd := exec.Command(pathToServiceBackupBinary, params...)
	session, err := gexec.Start(backupCmd, GinkgoWriter, GinkgoWriter)
	Expect(err).ToNot(HaveOccurred())
	return session
}

func createConfigFile(format string, a ...interface{}) string {
	file, err := ioutil.TempFile("", "config.yml")
	Expect(err).NotTo(HaveOccurred())
	file.Write([]byte(fmt.Sprintf(format, a...)))
	return file.Name()
}

func downloadedS3Path(downloadFolder string, endParts ...string) string {
	today := time.Now()
	dateComponents := []string{fmt.Sprintf("%d", today.Year()), fmt.Sprintf("%02d", today.Month()), fmt.Sprintf("%02d", today.Day())}
	args := []string{downloadFolder}
	args = append(args, dateComponents...)
	args = append(args, endParts...)
	return filepath.Join(args...)
}

func pathWithDateForSCP(destPathSCP string, endParts ...string) string {
	today := time.Now()
	dateComponents := []string{fmt.Sprintf("%d", today.Year()), fmt.Sprintf("%02d", today.Month()), fmt.Sprintf("%02d", today.Day())}
	args := []string{destPathSCP}
	args = append(args, dateComponents...)
	args = append(args, endParts...)
	return filepath.Join(args...)
}
