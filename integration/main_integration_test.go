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
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

func performBackup(
	awsCLIPath,
	awsAccessKeyID,
	awsSecretAccessKey,
	sourceFolder,
	destFolder,
	endpointURL,
	backupCreatorCmd,
	cleanupCmd,
	cronSchedule string,
) (*gexec.Session, error) {

	backupCmd := exec.Command(
		pathToServiceBackupBinary,
		"--aws-cli", awsCLIPath,
		"--aws-access-key-id", awsAccessKeyID,
		"--aws-secret-access-key", awsSecretAccessKey,
		"--source-folder", sourceFolder,
		"--dest-folder", destFolder,
		"--endpoint-url", endpointURL,
		"--logLevel", "debug",
		"--backup-creator-cmd", backupCreatorCmd,
		"--cleanup-cmd", cleanupCmd,
		"--cron-schedule", cronSchedule,
	)

	return gexec.Start(backupCmd, GinkgoWriter, GinkgoWriter)
}

func downloadBackup(sourceFilePath, targetFilePath string) (*gexec.Session, error) {
	return runS3Command(
		"cp",
		sourceFilePath,
		targetFilePath,
	)
}

func deleteBackup(sourceFilePath string) (*gexec.Session, error) {
	return runS3Command(
		"rm",
		sourceFilePath,
	)
}

var _ = Describe("Service Backup Binary", func() {
	var (
		destFolder       string
		backupCreatorCmd string
		cleanupCmd       string
		fileContents     string
	)

	BeforeEach(func() {
		destPath, err := uuid.NewV4()
		Expect(err).ToNot(HaveOccurred())
		destFolder = fmt.Sprintf("s3://%s/%s", bucketName, destPath.String())
	})

	Context("when credentials are provided", func() {

		var (
			sourceFolder   string
			sourceFileName string
		)

		var createFileToUpload = func() string {
			var err error

			sourceFolder, err = ioutil.TempDir("", "")
			Expect(err).ToNot(HaveOccurred())

			sourceFile, err := ioutil.TempFile(sourceFolder, "temp-file.txt")
			Expect(err).ToNot(HaveOccurred())

			return sourceFile.Name()
		}

		var getFilenameFromPath = func(sourceFilePath string) string {
			sourceFilePathSplit := strings.Split(sourceFilePath, "/")
			return sourceFilePathSplit[len(sourceFilePathSplit)-1]
		}

		BeforeEach(func() {
			sourceFilePath := createFileToUpload()
			sourceFileName = getFilenameFromPath(sourceFilePath)

			fileContentsUUID, err := uuid.NewV4()
			Expect(err).ToNot(HaveOccurred())
			fileContents = fileContentsUUID.String()

			backupCreatorCmd = fmt.Sprintf(
				"%s %s %s",
				assetPath("create-fake-backup"),
				sourceFilePath,
				fileContents,
			)

			cleanupCmd = fmt.Sprintf(
				"rm -rf %s",
				sourceFolder,
			)
		})

		Context("when all required inputs are valid", func() {
			AfterEach(func() {
				_ = os.Remove(sourceFolder)
			})

			Context("when the bucket already exists", func() {
				AfterEach(func() {
					session, err := deleteBackup(destFolder + "/" + sourceFileName)
					Expect(err).ToNot(HaveOccurred())
					Eventually(session, awsTimeout).Should(gexec.Exit(0))
				})

				It("uploads a directory successfully", func() {
					By("Uploading the file to the blobstore")
					session, err := performBackup(
						awsCLIPath,
						awsAccessKeyID,
						awsSecretAccessKey,
						sourceFolder,
						destFolder,
						endpointURL,
						backupCreatorCmd,
						cleanupCmd,
						cronSchedule,
					)
					Expect(err).ToNot(HaveOccurred())
					Eventually(session.Out, awsTimeout).Should(gbytes.Say("Cleanup completed"))

					session.Terminate().Wait()
					Eventually(session).Should(gexec.Exit())

					targetFilePath := filepath.Join(sourceFolder, "downloaded_file")

					By("Downloading the uploaded file from the blobstore")
					verifySession, err := downloadBackup(destFolder+"/"+sourceFileName, targetFilePath)
					Expect(err).ToNot(HaveOccurred())
					Eventually(verifySession, awsTimeout).Should(gexec.Exit(0))

					By("Validating the contents of the downloaded file")
					downloadedFile, err := os.Open(targetFilePath)
					Expect(err).ToNot(HaveOccurred())
					defer downloadedFile.Close()

					actualData := make([]byte, len(fileContents))
					_, err = downloadedFile.Read(actualData)
					Expect(err).ToNot(HaveOccurred())

					actualString := string(actualData)

					Expect(actualString).To(Equal(fileContents))
				})
			})

			Context("when the bucket does not already exist", func() {
				var newBucketName string
				var strippedUUID string

				BeforeEach(func() {
					bucketUUID, err := uuid.NewV4()
					Expect(err).ToNot(HaveOccurred())
					strippedUUID = bucketUUID.String()
					strippedUUID = strippedUUID[:10]
					newBucketName = bucketName + strippedUUID

					destFolder = fmt.Sprintf("s3://%s/%s", newBucketName, strippedUUID)
				})

				AfterEach(func() {
					session, err := runS3Command(
						"rb",
						"s3://"+newBucketName,
						"--force",
					)
					Expect(err).ToNot(HaveOccurred())
					Eventually(session, awsTimeout).Should(gexec.Exit(0))
					Eventually(session.Out).Should(gbytes.Say("remove_bucket: s3://" + newBucketName))
				})

				It("makes the bucket", func() {
					By("Uploading the file to the blobstore")
					session, err := performBackup(
						awsCLIPath,
						awsAccessKeyID,
						awsSecretAccessKey,
						sourceFolder,
						destFolder,
						endpointURL,
						backupCreatorCmd,
						cleanupCmd,
						cronSchedule,
					)
					Expect(err).ToNot(HaveOccurred())
					Eventually(session.Out, awsTimeout).Should(gbytes.Say("Cleanup completed"))

					session.Terminate().Wait()
					Eventually(session).Should(gexec.Exit())

					session, err = runS3Command(
						"ls",
						"s3://"+newBucketName,
					)
					Expect(err).ToNot(HaveOccurred())
					Eventually(session, awsTimeout).Should(gexec.Exit(0))
					Eventually(session.Out).Should(gbytes.Say(strippedUUID))
				})
			})

			Context("when cleanup-cmd is provided", func() {
				AfterEach(func() {
					session, err := deleteBackup(destFolder + "/" + sourceFileName)
					Expect(err).ToNot(HaveOccurred())
					Eventually(session, awsTimeout).Should(gexec.Exit(0))
				})

				Context("when the cleanup command fails with non-zero exit code", func() {
					const failingCleanupCmd = "ls /not/a/valid/directory"

					It("logs and exits without error", func() {
						session, err := performBackup(
							awsCLIPath,
							awsAccessKeyID,
							awsSecretAccessKey,
							sourceFolder,
							destFolder,
							endpointURL,
							backupCreatorCmd,
							failingCleanupCmd,
							cronSchedule,
						)

						Expect(err).ToNot(HaveOccurred())
						Eventually(session.Out, awsTimeout).Should(gbytes.Say("Cleanup completed with error"))
						session.Terminate().Wait()
						Eventually(session).Should(gexec.Exit())
					})
				})

				Context("when the cleanup command is valid", func() {
					It("executes the cleanup command and returns without error", func() {
						By("Uploading the file to the blobstore")
						session, err := performBackup(
							awsCLIPath,
							awsAccessKeyID,
							awsSecretAccessKey,
							sourceFolder,
							destFolder,
							endpointURL,
							backupCreatorCmd,
							cleanupCmd,
							cronSchedule,
						)
						Expect(err).ToNot(HaveOccurred())
						Eventually(session.Out, awsTimeout).Should(gbytes.Say("Cleanup completed without error"))
						session.Terminate().Wait()
						Eventually(session).Should(gexec.Exit())

						By("Validating that the source directory is deleted")
						_, err = os.Stat(sourceFolder)
						Expect(err).To(HaveOccurred())
						Expect(os.IsNotExist(err)).To(BeTrue())
					})
				})
			})

			Context("when cleanup-cmd is not provided", func() {
				const emptyCleanupCmd = ""

				AfterEach(func() {
					session, err := deleteBackup(destFolder + "/" + sourceFileName)
					Expect(err).ToNot(HaveOccurred())
					Eventually(session, awsTimeout).Should(gexec.Exit(0))
				})

				It("logs and exits without error", func() {
					session, err := performBackup(
						awsCLIPath,
						awsAccessKeyID,
						awsSecretAccessKey,
						sourceFolder,
						destFolder,
						endpointURL,
						backupCreatorCmd,
						emptyCleanupCmd,
						cronSchedule,
					)
					Expect(err).ToNot(HaveOccurred())
					Eventually(session.Out, awsTimeout).Should(gbytes.Say("Cleanup command not provided"))
					session.Terminate().Wait()
					Eventually(session).Should(gexec.Exit())
				})
			})
		})

		Context("when credentials are invalid", func() {
			const (
				invalidAwsAccessKeyID     = "invalid-access-key-id"
				invalidAwsSecretAccessKey = "invalid-secret-access-key"
			)

			It("fails to upload a directory", func() {
				By("Trying to upload the file to the blobstore")
				session, err := performBackup(
					awsCLIPath,
					invalidAwsAccessKeyID,
					invalidAwsSecretAccessKey,
					sourceFolder,
					destFolder,
					endpointURL,
					backupCreatorCmd,
					cleanupCmd,
					cronSchedule,
				)
				Expect(err).ToNot(HaveOccurred())
				Eventually(session.Out).Should(gbytes.Say("Service-backup Started"))
				session.Terminate().Wait()
				Eventually(session).Should(gexec.Exit())

				By("Verifying that the file was never uploaded")
				verifySession, err := downloadBackup(destFolder+"/"+sourceFileName, filepath.Join(sourceFolder, "downloaded_file"))
				Expect(err).ToNot(HaveOccurred())
				Eventually(verifySession, awsTimeout).Should(gexec.Exit(1))
			})
		})

		Context("when the AWS CLI path flag is not provided", func() {
			const invalidAWSCLIPath = ""

			It("gracefully fails to perform the upload", func() {
				session, err := performBackup(
					invalidAWSCLIPath,
					awsAccessKeyID,
					awsSecretAccessKey,
					sourceFolder,
					destFolder,
					endpointURL,
					backupCreatorCmd,
					cleanupCmd,
					cronSchedule,
				)

				Expect(err).ToNot(HaveOccurred())
				Eventually(session, awsTimeout).Should(gexec.Exit(2))
				Eventually(session.Out).Should(gbytes.Say("Flag aws-cli not provided"))
			})
		})

		Context("when the source folder flag is not provided", func() {
			const invalidSourceFolder = ""

			It("gracefully fails to perform the upload", func() {
				session, err := performBackup(
					awsCLIPath,
					awsAccessKeyID,
					awsSecretAccessKey,
					invalidSourceFolder,
					destFolder,
					endpointURL,
					backupCreatorCmd,
					cleanupCmd,
					cronSchedule,
				)

				Expect(err).ToNot(HaveOccurred())
				Eventually(session, awsTimeout).Should(gexec.Exit(2))
				Eventually(session.Out).Should(gbytes.Say("Flag source-folder not provided"))
			})
		})

		Context("when the destination folder flag is not provided", func() {
			const emptyDestFolder = ""

			It("gracefully fails to perform the upload", func() {
				session, err := performBackup(
					awsCLIPath,
					awsAccessKeyID,
					awsSecretAccessKey,
					sourceFolder,
					emptyDestFolder,
					endpointURL,
					backupCreatorCmd,
					cleanupCmd,
					cronSchedule,
				)

				Expect(err).ToNot(HaveOccurred())
				Eventually(session, awsTimeout).Should(gexec.Exit(2))
				Eventually(session.Out).Should(gbytes.Say("Flag dest-folder not provided"))
			})
		})

		Context("when the endpoint URL flag is not provided", func() {
			const emptyEndpointURL = ""

			It("gracefully fails to perform the upload", func() {
				session, err := performBackup(
					awsCLIPath,
					awsAccessKeyID,
					awsSecretAccessKey,
					sourceFolder,
					destFolder,
					emptyEndpointURL,
					backupCreatorCmd,
					cleanupCmd,
					cronSchedule,
				)

				Expect(err).ToNot(HaveOccurred())
				Eventually(session, awsTimeout).Should(gexec.Exit(2))
				Eventually(session.Out).Should(gbytes.Say("Flag endpoint-url not provided"))
			})
		})

		Context("when backup-creator-cmd is not provided", func() {
			const invalidBackupCreatorCmd = ""

			It("gracefully fails to perform the upload", func() {
				session, err := performBackup(
					awsCLIPath,
					awsAccessKeyID,
					awsSecretAccessKey,
					sourceFolder,
					destFolder,
					endpointURL,
					invalidBackupCreatorCmd,
					cleanupCmd,
					cronSchedule,
				)

				Expect(err).ToNot(HaveOccurred())
				Eventually(session, awsTimeout).Should(gexec.Exit(2))
				Eventually(session.Out).Should(gbytes.Say("Flag backup-creator-cmd not provided"))
			})
		})

		Context("when the backup creation command fails with non-zero exit code", func() {
			const failingBackupCreatorCmd = "ls /not/a/valid/directory"

			It("gracefully fails to perform the upload", func() {
				session, err := performBackup(
					awsCLIPath,
					awsAccessKeyID,
					awsSecretAccessKey,
					sourceFolder,
					destFolder,
					endpointURL,
					failingBackupCreatorCmd,
					cleanupCmd,
					cronSchedule,
				)
				Expect(err).ToNot(HaveOccurred())
				Eventually(session.Out, awsTimeout).Should(gbytes.Say("Perform backup completed with error"))
				session.Terminate().Wait()
				Eventually(session).Should(gexec.Exit())
			})
		})

		Context("when the cron schedule is not provided", func() {
			const emptyCronSchedule = ""

			It("gracefully fails to perform the upload", func() {
				session, err := performBackup(
					awsCLIPath,
					awsAccessKeyID,
					awsSecretAccessKey,
					sourceFolder,
					destFolder,
					endpointURL,
					backupCreatorCmd,
					cleanupCmd,
					emptyCronSchedule,
				)

				Expect(err).ToNot(HaveOccurred())
				Eventually(session, awsTimeout).Should(gexec.Exit(2))
				Eventually(session.Out).Should(gbytes.Say("Flag cron-schedule not provided"))
			})
		})

		Context("when the cron schedule is not valid", func() {
			const invalidCronSchedule = "* * * * * 99"

			It("gracefully fails to perform the upload", func() {
				session, err := performBackup(
					awsCLIPath,
					awsAccessKeyID,
					awsSecretAccessKey,
					sourceFolder,
					destFolder,
					endpointURL,
					backupCreatorCmd,
					cleanupCmd,
					invalidCronSchedule,
				)

				Expect(err).ToNot(HaveOccurred())
				Eventually(session, awsTimeout).Should(gexec.Exit(2))
				Eventually(session.Out).Should(gbytes.Say("Error scheduling job"))
			})
		})
	})

	Context("when credentials are not provided", func() {
		const (
			emptyAWSAccessKeyID     = ""
			emptyAWSSecretAccessKey = ""
			sourceFolder            = "/path/to/source-folder"
		)

		It("returns without error", func() {
			session, err := performBackup(
				awsCLIPath,
				emptyAWSAccessKeyID,
				emptyAWSSecretAccessKey,
				sourceFolder,
				destFolder,
				endpointURL,
				backupCreatorCmd,
				cleanupCmd,
				cronSchedule,
			)

			Expect(err).ToNot(HaveOccurred())
			Eventually(session, awsTimeout).Should(gexec.Exit(0))
		})

		It("logs that it is skipping", func() {
			session, err := performBackup(
				awsCLIPath,
				emptyAWSAccessKeyID,
				emptyAWSSecretAccessKey,
				sourceFolder,
				destFolder,
				endpointURL,
				backupCreatorCmd,
				cleanupCmd,
				cronSchedule,
			)

			Expect(err).ToNot(HaveOccurred())
			Eventually(session, awsTimeout).Should(gexec.Exit())
			Eventually(session.Out).Should(gbytes.Say("skipping"))
		})
	})
})
