package multiintegration_test

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
	"github.com/satori/go.uuid"
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

			destS3UUID := uuid.NewV4()
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
			Eventually(runningBin.Out, time.Second*10).Should(gbytes.Say("s3 completed"))
			runningBin.Terminate().Wait()

			content1, err := ioutil.ReadFile(pathWithDateForSCP(destPathSCP, "1.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(content1).To(Equal([]byte("1")))

			content2, err := ioutil.ReadFile(pathWithDateForSCP(destPathSCP, "subdir", "2.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(content2).To(Equal([]byte("2")))
		})

		It("copies files to S3", func() {
			Eventually(runningBin.Out, time.Second*10).Should(gbytes.Say("scp completed"))
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

	Context("When two SCP destinations are correctly configured", func() {
		var (
			runningBin      *gexec.Session
			baseDir         string
			destPathSCP1    string
			destPathSCP2    string
			privateKey2Path string
		)

		BeforeEach(func() {
			var err error
			baseDir, err = ioutil.TempDir("", "multiple-destinations-integration-tests")
			Expect(err).NotTo(HaveOccurred())
			dirToBackup := filepath.Join(baseDir, "source")
			destPathSCP1 = filepath.Join(baseDir, "target1")
			destPathSCP2 = filepath.Join(baseDir, "target2")
			Expect(os.Mkdir(dirToBackup, 0755)).To(Succeed())
			Expect(os.Mkdir(destPathSCP1, 0755)).To(Succeed())
			Expect(os.Mkdir(destPathSCP2, 0755)).To(Succeed())

			Expect(ioutil.WriteFile(filepath.Join(dirToBackup, "1.txt"), []byte("1"), 0644)).To(Succeed())
			Expect(os.Mkdir(filepath.Join(dirToBackup, "subdir"), 0755)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(dirToBackup, "subdir", "2.txt"), []byte("2"), 0644)).To(Succeed())

			var publicKey2Path string
			var privateKey2Contents []byte
			publicKey2Path, privateKey2Path, privateKey2Contents = createSSHKey()
			addToAuthorizedKeys(publicKey2Path)

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
- type: scp
  config:
    server: localhost
    user: %s
    destination: %s
    key: |
      %s
    port: 22
source_folder: %s
source_executable: true
exit_if_in_progress: true
cron_schedule: '*/5 * * * * *'
cleanup_executable: true
missing_properties_message: custom message`, unixUser.Username, destPathSCP1, padWithSpaces(string(privateKeyContents), 6),
				unixUser.Username, destPathSCP2, padWithSpaces(string(privateKey2Contents), 6),
				dirToBackup))
		})

		AfterEach(func() {
			Expect(os.RemoveAll(baseDir)).To(Succeed())
			Expect(os.RemoveAll(filepath.Dir(privateKey2Path))).To(Succeed())
			Eventually(runningBin.Terminate()).Should(gexec.Exit())
		})

		It("copies files with SCP to the first destination", func() {
			Eventually(runningBin.Out, time.Second*10).Should(gbytes.Say("scp completed"))
			Eventually(runningBin.Out, time.Second*10).Should(gbytes.Say("scp completed"))
			runningBin.Terminate().Wait()

			content1, err := ioutil.ReadFile(pathWithDateForSCP(destPathSCP1, "1.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(content1).To(Equal([]byte("1")))

			content2, err := ioutil.ReadFile(pathWithDateForSCP(destPathSCP1, "subdir", "2.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(content2).To(Equal([]byte("2")))
		})

		It("copies files with SCP to the second destination", func() {
			Eventually(runningBin.Out, time.Second*10).Should(gbytes.Say("scp completed"))
			Eventually(runningBin.Out, time.Second*10).Should(gbytes.Say("scp completed"))
			runningBin.Terminate().Wait()

			content1, err := ioutil.ReadFile(pathWithDateForSCP(destPathSCP2, "1.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(content1).To(Equal([]byte("1")))

			content2, err := ioutil.ReadFile(pathWithDateForSCP(destPathSCP2, "subdir", "2.txt"))
			Expect(err).NotTo(HaveOccurred())
			Expect(content2).To(Equal([]byte("2")))
		})
	})

	Context("when two S3 destinations are correctly configured", func() {
		var (
			runningBin  *gexec.Session
			baseDir     string
			dest1PathS3 string
			dest2PathS3 string
		)

		BeforeEach(func() {
			var err error
			baseDir, err = ioutil.TempDir("", "multiple-destinations-integration-tests")
			Expect(err).NotTo(HaveOccurred())
			dirToBackup := filepath.Join(baseDir, "source")
			Expect(os.Mkdir(dirToBackup, 0755)).To(Succeed())

			Expect(ioutil.WriteFile(filepath.Join(dirToBackup, "1.txt"), []byte("1"), 0644)).To(Succeed())
			Expect(os.Mkdir(filepath.Join(dirToBackup, "subdir"), 0755)).To(Succeed())
			Expect(ioutil.WriteFile(filepath.Join(dirToBackup, "subdir", "2.txt"), []byte("2"), 0644)).To(Succeed())

			dest1PathS3 = uuid.NewV4().String()
			dest2PathS3 = uuid.NewV4().String()

			runningBin = runBackup(createConfigFile(`---
destinations:
- type: s3
  config:
    endpoint_url: 'https://s3.amazonaws.com'
    bucket_name: %s
    bucket_path: %s
    access_key_id: %s
    secret_access_key: %s
- type: s3
  config:
    endpoint_url: ''
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
missing_properties_message: custom message`, existingBucketInDefaultRegion, dest1PathS3, awsAccessKeyID, awsSecretAccessKey,
				existingBucketInNonDefaultRegion, dest2PathS3, awsAccessKeyID, awsSecretAccessKey,
				dirToBackup))
		})

		AfterEach(func() {
			Expect(os.RemoveAll(baseDir)).To(Succeed())
			Eventually(runningBin.Terminate()).Should(gexec.Exit())
		})

		It("copies files to the first S3 destination", func() {
			Eventually(runningBin.Out, time.Second*10).Should(gbytes.Say("s3 completed"))
			Eventually(runningBin.Out, time.Second*10).Should(gbytes.Say("s3 completed"))
			runningBin.Terminate().Wait()

			downloadFolder, err := ioutil.TempDir("", "backup-tests")
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove(downloadFolder)

			err = s3TestClient.DownloadRemoteDirectory(
				existingBucketInDefaultRegion,
				dest1PathS3,
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

		It("copies files to the second S3 destination", func() {
			Eventually(runningBin.Out, time.Second*10).Should(gbytes.Say("s3 completed"))
			Eventually(runningBin.Out, time.Second*10).Should(gbytes.Say("s3 completed"))
			runningBin.Terminate().Wait()

			downloadFolder, err := ioutil.TempDir("", "backup-tests")
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove(downloadFolder)

			err = s3TestClient.DownloadRemoteDirectory(
				existingBucketInNonDefaultRegion,
				dest2PathS3,
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