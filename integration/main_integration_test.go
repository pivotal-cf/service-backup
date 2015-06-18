package integration

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nu7hatch/gouuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Service Backup Binary", func() {
	var (
		awsCLIPath  = "aws"
		destFolder  string
		endpointURL = "https://s3.amazonaws.com"
	)

	BeforeEach(func() {
		destPath, err := uuid.NewV4()
		Expect(err).ToNot(HaveOccurred())
		destFolder = fmt.Sprintf("s3://%s/%s", bucketName, destPath.String())
	})

	Context("when credentials are provided", func() {

		Context("when credentials are valid", func() {
			var (
				sourceFolder   string
				sourceFileName string
				env            []string
			)

			BeforeEach(func() {
				var err error
				sourceFolder, err = ioutil.TempDir("", "")
				Expect(err).ToNot(HaveOccurred())

				sourceFile, err := ioutil.TempFile(sourceFolder, "temp-file.txt")
				defer sourceFile.Close()
				Expect(err).ToNot(HaveOccurred())

				_, err = sourceFile.WriteString("hi")
				Expect(err).ToNot(HaveOccurred())

				sourceFilepathSplit := strings.Split(sourceFile.Name(), "/")
				sourceFileName = sourceFilepathSplit[len(sourceFilepathSplit)-1]

				env = []string{}
				env = append(env, fmt.Sprintf("%s=%s", awsAccessKeyIDEnvKey, awsAccessKeyID))
				env = append(env, fmt.Sprintf("%s=%s", awsSecretAccessKeyEnvKey, awsSecretAccessKey))
			})

			AfterEach(func() {
				_ = os.Remove(sourceFolder)
				deleteCmd := exec.Command(
					awsCLIPath,
					"s3",
					"rm",
					destFolder+"/"+sourceFileName,
				)
				deleteCmd.Env = env

				deleteSession, err := gexec.Start(deleteCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).ToNot(HaveOccurred())
				Eventually(deleteSession, awsTimeout).Should(gexec.Exit(0))
			})

			It("uploads a directory successfully if the access and secret access keys are defined", func() {

				backupCmd := exec.Command(
					pathToServiceBackupBinary,
					"--aws-cli", awsCLIPath,
					"--aws-access-key-id", awsAccessKeyID,
					"--aws-secret-access-key", awsSecretAccessKey,
					"--source-folder", sourceFolder,
					"--dest-folder", destFolder,
					"--endpoint-url", endpointURL,
				)
				session, err := gexec.Start(backupCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).ToNot(HaveOccurred())

				Eventually(session, awsTimeout).Should(gexec.Exit(0))

				downloadedFilepath := filepath.Join(sourceFolder, "downloaded_file")
				verifyBackupCmd := exec.Command(
					awsCLIPath,
					"s3",
					"cp",
					destFolder+"/"+sourceFileName,
					downloadedFilepath,
				)
				verifyBackupCmd.Env = env

				verifySession, err := gexec.Start(verifyBackupCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).ToNot(HaveOccurred())
				Eventually(verifySession, awsTimeout).Should(gexec.Exit(0))

				downloadedFile, err := os.Open(downloadedFilepath)
				Expect(err).ToNot(HaveOccurred())
				defer downloadedFile.Close()

				sourceFile, err := os.Open(filepath.Join(sourceFolder, sourceFileName))
				Expect(err).ToNot(HaveOccurred())
				defer sourceFile.Close()

				actualData := make([]byte, 100)
				_, err = sourceFile.Read(actualData)
				Expect(err).ToNot(HaveOccurred())

				expectedData := make([]byte, 100)
				_, err = downloadedFile.Read(expectedData)
				Expect(err).ToNot(HaveOccurred())

				Expect(actualData).To(Equal(expectedData))
			})
		})
	})
})
