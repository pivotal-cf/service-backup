package s3integration_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/nu7hatch/gouuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

func performBackup(
	awsAccessKeyID,
	awsSecretAccessKey,
	sourceFolder,
	destBucket,
	destPath,
	endpointURL,
	backupCreatorCmd,
	cleanupCmd,
	cronSchedule string,
) (*gexec.Session, error) {

	backupCmd := exec.Command(
		pathToServiceBackupBinary,
		"s3",
		"--aws-access-key-id", awsAccessKeyID,
		"--aws-secret-access-key", awsSecretAccessKey,
		"--source-folder", sourceFolder,
		"--dest-bucket", destBucket,
		"--dest-path", destPath,
		"--endpoint-url", endpointURL,
		"--logLevel", "debug",
		"--backup-creator-cmd", backupCreatorCmd,
		"--cleanup-cmd", cleanupCmd,
		"--cron-schedule", cronSchedule,
	)

	return gexec.Start(backupCmd, GinkgoWriter, GinkgoWriter)
}

func performManualBackup(
	awsAccessKeyID,
	awsSecretAccessKey,
	sourceFolder,
	destBucket,
	destPath,
	endpointURL,
	backupCreatorCmd,
	cleanupCmd string,
) (*gexec.Session, error) {

	manualBackupCmd := exec.Command(
		pathToManualBackupBinary,
		"s3",
		"--aws-access-key-id", awsAccessKeyID,
		"--aws-secret-access-key", awsSecretAccessKey,
		"--source-folder", sourceFolder,
		"--dest-bucket", destBucket,
		"--dest-path", destPath,
		"--endpoint-url", endpointURL,
		"--logLevel", "debug",
		"--backup-creator-cmd", backupCreatorCmd,
		"--cleanup-cmd", cleanupCmd,
	)

	return gexec.Start(manualBackupCmd, GinkgoWriter, GinkgoWriter)

}

func performBackupWithServiceIdentifier(
	awsAccessKeyID,
	awsSecretAccessKey,
	sourceFolder,
	destBucket,
	destPath,
	endpointURL,
	backupCreatorCmd,
	cleanupCmd,
	cronSchedule,
	serviceIdentifierCmd string,
) (*gexec.Session, error) {

	backupCmd := exec.Command(
		pathToServiceBackupBinary,
		"s3",
		"--aws-access-key-id", awsAccessKeyID,
		"--aws-secret-access-key", awsSecretAccessKey,
		"--source-folder", sourceFolder,
		"--dest-bucket", destBucket,
		"--dest-path", destPath,
		"--endpoint-url", endpointURL,
		"--logLevel", "debug",
		"--backup-creator-cmd", backupCreatorCmd,
		"--cleanup-cmd", cleanupCmd,
		"--cron-schedule", cronSchedule,
		"--service-identifier-cmd", serviceIdentifierCmd,
	)

	return gexec.Start(backupCmd, GinkgoWriter, GinkgoWriter)
}

func pathWithDate(path string) string {
	today := time.Now()
	datePath := fmt.Sprintf("%d/%02d/%02d", today.Year(), today.Month(), today.Day())
	return path + "/" + datePath
}

func createFilesToUpload(sourceFolder string) map[string]string {
	createdFiles := map[string]string{}

	rootFile, contents := createFileIn(sourceFolder)
	createdFiles[rootFile] = contents

	dir1 := filepath.Join(sourceFolder, "dir1")
	err := os.Mkdir(dir1, 0777)
	Expect(err).ToNot(HaveOccurred())

	dir1File, contents := createFileIn(dir1)
	createdFiles["dir1/"+dir1File] = contents

	dir2 := filepath.Join(dir1, "dir2")
	err = os.Mkdir(dir2, 0777)
	Expect(err).ToNot(HaveOccurred())

	dir2File, contents := createFileIn(dir2)
	createdFiles["dir1/dir2/"+dir2File] = contents

	return createdFiles
}

func createFileIn(sourceFolder string) (string, string) {
	file, err := ioutil.TempFile(sourceFolder, "")
	Expect(err).ToNot(HaveOccurred())

	fileContentsUUID, err := uuid.NewV4()
	Expect(err).ToNot(HaveOccurred())

	fileContents := fileContentsUUID.String()
	_, err = file.Write([]byte(fileContents))
	Expect(err).ToNot(HaveOccurred())

	fileName := filepath.Base(file.Name())
	return fileName, fileContents
}

var _ = Describe("Service Backup Binary", func() {
	var (
		destBucket       string
		backupCreatorCmd string
		cleanupCmd       string
	)

	BeforeEach(func() {
		destBucket = existingBucketName

		destPathUUID, err := uuid.NewV4()
		Expect(err).ToNot(HaveOccurred())
		destPath = destPathUUID.String()
	})

	Context("when credentials are provided", func() {
		var (
			sourceFolder    string
			downloadFolder  string
			filesToContents map[string]string
		)

		BeforeEach(func() {
			var err error

			sourceFolder, err = ioutil.TempDir("", "")
			Expect(err).ToNot(HaveOccurred())

			downloadFolder, err = ioutil.TempDir("", "")
			Expect(err).ToNot(HaveOccurred())

			filesToContents = createFilesToUpload(sourceFolder)

			backupCreatorCmd = fmt.Sprintf(
				"%s %s",
				assetPath("create-fake-backup"),
				sourceFolder,
			)

			cleanupCmd = fmt.Sprintf(
				"rm -rf %s",
				sourceFolder,
			)
		})

		AfterEach(func() {
			os.Remove(sourceFolder)
			os.Remove(downloadFolder)
		})

		Context("when all required inputs are valid", func() {

			Context("when the bucket already exists", func() {
				AfterEach(func() {
					Expect(s3TestClient.DeleteRemotePath(destBucket, pathWithDate(destPath))).To(Succeed())
				})

				Context("using cron scheduled backup", func() {

					It("recursively uploads the contents of a directory successfully", func() {
						By("Uploading the directory contents to the blobstore")
						session, err := performBackup(
							awsAccessKeyID,
							awsSecretAccessKey,
							sourceFolder,
							destBucket,
							destPath,
							endpointURL,
							backupCreatorCmd,
							cleanupCmd,
							cronSchedule,
						)
						Expect(err).ToNot(HaveOccurred())
						Eventually(session.Out, awsTimeout).Should(gbytes.Say("Cleanup completed"))

						session.Terminate().Wait()
						Eventually(session).Should(gexec.Exit())

						By("Downloading the uploaded files from the blobstore")
						err = s3TestClient.DownloadRemoteDirectory(
							destBucket,
							pathWithDate(destPath),
							downloadFolder,
						)
						Expect(err).ToNot(HaveOccurred())

						By("Validating the contents of the downloaded files")
						for fileName, contents := range filesToContents {
							downloadedFilePath := filepath.Join(downloadFolder, fileName)

							downloadedFile, err := os.Open(downloadedFilePath)
							Expect(err).ToNot(HaveOccurred())
							defer downloadedFile.Close()

							actualData := make([]byte, len(contents))
							_, err = downloadedFile.Read(actualData)
							Expect(err).ToNot(HaveOccurred())

							actualString := string(actualData)

							Expect(actualString).To(Equal(contents))
						}
					})

					Context("when service identifier binary is provided", func() {
						var serviceIdentifierCmd string

						BeforeEach(func() {
							serviceIdentifierCmd = assetPath("fake-service-identifier")
						})

						It("logs events with the data element including an identifier", func() {
							By("Uploading the directory contents to the blobstore")
							session, err := performBackupWithServiceIdentifier(
								awsAccessKeyID,
								awsSecretAccessKey,
								sourceFolder,
								destBucket,
								destPath,
								endpointURL,
								backupCreatorCmd,
								cleanupCmd,
								cronSchedule,
								serviceIdentifierCmd,
							)

							identifier := `"identifier":"FakeIdentifier"`

							Expect(err).ToNot(HaveOccurred())
							Eventually(session.Out, awsTimeout).Should(gbytes.Say("Perform backup started"))
							Eventually(session.Out, awsTimeout).Should(gbytes.Say(identifier))
							Eventually(session.Out, awsTimeout).Should(gbytes.Say("Perform backup debug info"))
							Eventually(session.Out, awsTimeout).Should(gbytes.Say(identifier))
							Eventually(session.Out, awsTimeout).Should(gbytes.Say("Perform backup completed without error"))
							Eventually(session.Out, awsTimeout).Should(gbytes.Say(identifier))
							Eventually(session.Out, awsTimeout).Should(gbytes.Say("Upload backup started"))
							Eventually(session.Out, awsTimeout).Should(gbytes.Say(identifier))
							Eventually(session.Out, awsTimeout).Should(gbytes.Say("Running command"))
							Eventually(session.Out, awsTimeout).Should(gbytes.Say(identifier))
							Eventually(session.Out, awsTimeout).Should(gbytes.Say("Upload backup completed without error"))
							Eventually(session.Out, awsTimeout).Should(gbytes.Say(identifier))
							Eventually(session.Out, awsTimeout).Should(gbytes.Say("Cleanup completed"))
							Eventually(session.Out, awsTimeout).Should(gbytes.Say(identifier))

							session.Terminate().Wait()
							Eventually(session).Should(gexec.Exit())
						})
					})
				})

				Context("using manually triggered backup", func() {
					It("uploads a snapshot that has been manually generated", func() {
						session, err := performManualBackup(
							awsAccessKeyID,
							awsSecretAccessKey,
							sourceFolder,
							destBucket,
							destPath,
							endpointURL,
							backupCreatorCmd,
							cleanupCmd,
						)
						Expect(err).ToNot(HaveOccurred())
						Eventually(session.Out, awsTimeout).Should(gbytes.Say("Cleanup completed"))

						session.Terminate().Wait()
						Eventually(session).Should(gexec.Exit())

						By("Downloading the uploaded files from the blobstore")
						err = s3TestClient.DownloadRemoteDirectory(
							destBucket,
							pathWithDate(destPath),
							downloadFolder,
						)
						Expect(err).ToNot(HaveOccurred())

						By("Validating the contents of the downloaded files")
						for fileName, contents := range filesToContents {
							downloadedFilePath := filepath.Join(downloadFolder, fileName)

							downloadedFile, err := os.Open(downloadedFilePath)
							Expect(err).ToNot(HaveOccurred())
							defer downloadedFile.Close()

							actualData := make([]byte, len(contents))
							_, err = downloadedFile.Read(actualData)
							Expect(err).ToNot(HaveOccurred())

							actualString := string(actualData)

							Expect(actualString).To(Equal(contents))
						}
					})

					Context("when backup fails", func() {
						It("exits with non-zero status", func() {
							session, err := performManualBackup(
								"wrong-access-key",
								awsSecretAccessKey,
								sourceFolder,
								destBucket,
								destPath,
								endpointURL,
								backupCreatorCmd,
								cleanupCmd,
							)
							Expect(err).ToNot(HaveOccurred())
							session.Wait(time.Second * 3)
							Expect(session.ExitCode()).NotTo(Equal(0))
						})
					})
				})
			})

			Context("when the bucket does not already exist", func() {
				var strippedUUID string

				BeforeEach(func() {
					bucketUUID, err := uuid.NewV4()
					Expect(err).ToNot(HaveOccurred())

					strippedUUID = bucketUUID.String()
					strippedUUID = strippedUUID[:10]

					destBucket = existingBucketName + strippedUUID
					destPath = strippedUUID
				})

				AfterEach(func() {
					s3TestClient.DeleteBucket(destBucket)
				})

				It("makes the bucket", func() {
					By("Uploading the file to the blobstore")
					session, err := performBackup(
						awsAccessKeyID,
						awsSecretAccessKey,
						sourceFolder,
						destBucket,
						destPath,
						endpointURL,
						backupCreatorCmd,
						cleanupCmd,
						cronSchedule,
					)
					Expect(err).ToNot(HaveOccurred())
					Eventually(session.Out, awsTimeout).Should(gbytes.Say("Cleanup completed"))

					session.Terminate().Wait()
					Eventually(session).Should(gexec.Exit())

					keys, err := s3TestClient.ListRemotePath(destBucket, "")
					Expect(err).ToNot(HaveOccurred())
					Expect(keys).ToNot(BeEmpty())
				})
			})

			Context("when cleanup-cmd is provided", func() {
				Context("when the cleanup command fails with non-zero exit code", func() {
					const failingCleanupCmd = "ls /not/a/valid/directory"

					It("logs and exits without error", func() {
						session, err := performBackup(
							awsAccessKeyID,
							awsSecretAccessKey,
							sourceFolder,
							destBucket,
							destPath,
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
							awsAccessKeyID,
							awsSecretAccessKey,
							sourceFolder,
							destBucket,
							destPath,
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

				It("logs and exits without error", func() {
					session, err := performBackup(
						awsAccessKeyID,
						awsSecretAccessKey,
						sourceFolder,
						destBucket,
						destPath,
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

			Context("when a user does not have the CreateBucket permission", func() {
				Context("when the bucket already exists", func() {
					It("successfully uploads the backup", func() {
						By("Uploading the directory contents to the blobstore")
						session, err := performBackup(
							awsAccessKeyIDRestricted,
							awsSecretAccessKeyRestricted,
							sourceFolder,
							destBucket,
							destPath,
							endpointURL,
							backupCreatorCmd,
							cleanupCmd,
							cronSchedule,
						)
						Expect(err).ToNot(HaveOccurred())
						Eventually(session.Out, awsTimeout).Should(gbytes.Say("Cleanup completed"))

						session.Terminate().Wait()
						Eventually(session).Should(gexec.Exit())

						By("Downloading the uploaded files from the blobstore")
						err = s3TestClient.DownloadRemoteDirectory(
							destBucket,
							pathWithDate(destPath),
							downloadFolder,
						)
						Expect(err).ToNot(HaveOccurred())

						By("Validating the contents of the downloaded files")
						for fileName, contents := range filesToContents {
							downloadedFilePath := filepath.Join(downloadFolder, fileName)

							downloadedFile, err := os.Open(downloadedFilePath)
							Expect(err).ToNot(HaveOccurred())
							defer downloadedFile.Close()

							actualData := make([]byte, len(contents))
							_, err = downloadedFile.Read(actualData)
							Expect(err).ToNot(HaveOccurred())

							actualString := string(actualData)

							Expect(actualString).To(Equal(contents))
						}
					})
				})

				Context("when the bucket does not exist", func() {
					BeforeEach(func() {
						bucketUUID, err := uuid.NewV4()
						Expect(err).ToNot(HaveOccurred())
						destBucket = "doesnotexist" + bucketUUID.String()
					})

					It("logs an error", func() {
						session, err := performBackup(
							awsAccessKeyIDRestricted,
							awsSecretAccessKeyRestricted,
							sourceFolder,
							destBucket,
							destPath,
							endpointURL,
							backupCreatorCmd,
							cleanupCmd,
							cronSchedule,
						)
						Expect(err).ToNot(HaveOccurred())
						Eventually(session.Out, awsTimeout).Should(gbytes.Say("Checking for remote path - remote path does not exist - making it now"))
						Eventually(session.Out, awsTimeout).Should(gbytes.Say("Configured S3 user unable to create buckets"))
					})
				})
			})
		})

		Context("when credentials are invalid", func() {
			const (
				invalidAwsAccessKeyID     = "invalid-access-key-id"
				invalidAwsSecretAccessKey = "invalid-secret-access-key"
			)

			It("fails to upload a directory", func() {
				destPathUUID, err := uuid.NewV4()
				Expect(err).ToNot(HaveOccurred())
				destPath = destPathUUID.String()
				By("Trying to upload the file to the blobstore")
				session, err := performBackup(
					invalidAwsAccessKeyID,
					invalidAwsSecretAccessKey,
					sourceFolder,
					destBucket,
					destPath,
					endpointURL,
					backupCreatorCmd,
					cleanupCmd,
					cronSchedule,
				)
				Expect(err).ToNot(HaveOccurred())
				Eventually(session.Out).Should(gbytes.Say("Service-backup Started"))
				session.Terminate().Wait()
				Eventually(session).Should(gexec.Exit())

				By("Verifying that the destPath was never created")
				Expect(s3TestClient.RemotePathExistsInBucket(destBucket, destPath)).To(BeFalse())
			})
		})

		Context("when the source folder flag is not provided", func() {
			const invalidSourceFolder = ""

			It("gracefully fails to perform the upload", func() {
				session, err := performBackup(
					awsAccessKeyID,
					awsSecretAccessKey,
					invalidSourceFolder,
					destBucket,
					destPath,
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

		Context("when the destination bucket flag is not provided", func() {
			const emptyDestBucket = ""

			It("gracefully fails to perform the upload", func() {
				session, err := performBackup(
					awsAccessKeyID,
					awsSecretAccessKey,
					sourceFolder,
					emptyDestBucket,
					destPath,
					endpointURL,
					backupCreatorCmd,
					cleanupCmd,
					cronSchedule,
				)

				Expect(err).ToNot(HaveOccurred())
				Eventually(session, awsTimeout).Should(gexec.Exit(2))
				Eventually(session.Out).Should(gbytes.Say("Flag dest-bucket not provided"))
			})
		})

		Context("when backup-creator-cmd is not provided", func() {
			const invalidBackupCreatorCmd = ""

			It("gracefully fails to perform the upload", func() {
				session, err := performBackup(
					awsAccessKeyID,
					awsSecretAccessKey,
					sourceFolder,
					destBucket,
					destPath,
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
					awsAccessKeyID,
					awsSecretAccessKey,
					sourceFolder,
					destBucket,
					destPath,
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
					awsAccessKeyID,
					awsSecretAccessKey,
					sourceFolder,
					destBucket,
					destPath,
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
					awsAccessKeyID,
					awsSecretAccessKey,
					sourceFolder,
					destBucket,
					destPath,
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

	Context("when no destination is specified", func() {
		var session *gexec.Session

		BeforeEach(func() {
			backupCmd := exec.Command(
				pathToServiceBackupBinary,
				"skip",
				"--cron-schedule", cronSchedule,
			)

			var err error
			session, err = gexec.Start(backupCmd, GinkgoWriter, GinkgoWriter)

			Expect(err).ToNot(HaveOccurred())
		})

		It("logs that backups are not enabled", func() {
			Eventually(session.Out, awsTimeout).Should(gbytes.Say("Backups Disabled"))
			session.Terminate().Wait()
			Eventually(session).Should(gexec.Exit())
		})

		It("doesn't exit until terminated", func() {
			Consistently(session, "10s").ShouldNot(gexec.Exit())
			session.Terminate().Wait()
			Eventually(session).Should(gexec.Exit())
		})
	})

	Context("when endpointURL is provided", func() {
		var (
			sourceFolder    string
			downloadFolder  string
			filesToContents map[string]string
		)

		BeforeEach(func() {
			var err error

			sourceFolder, err = ioutil.TempDir("", "")
			Expect(err).ToNot(HaveOccurred())

			downloadFolder, err = ioutil.TempDir("", "")
			Expect(err).ToNot(HaveOccurred())

			filesToContents = createFilesToUpload(sourceFolder)

			backupCreatorCmd = fmt.Sprintf(
				"%s %s",
				assetPath("create-fake-backup"),
				sourceFolder,
			)

			cleanupCmd = fmt.Sprintf(
				"rm -rf %s",
				sourceFolder,
			)
		})

		AfterEach(func() {
			os.Remove(sourceFolder)
			os.Remove(downloadFolder)
		})

		Context("when all required inputs are valid", func() {
			Context("when the bucket already exists", func() {
				AfterEach(func() {
					Expect(cephTestClient.DeleteRemotePath(destBucket, pathWithDate(destPath))).To(Succeed())
				})

				Context("using cron scheduled backup", func() {

					It("recursively uploads the contents of a directory successfully", func() {
						By("Uploading the directory contents to the blobstore")
						session, err := performBackup(
							cephAccessKeyID,
							cephSecretAccessKey,
							sourceFolder,
							destBucket,
							destPath,
							cephEndpointURL,
							backupCreatorCmd,
							cleanupCmd,
							cronSchedule,
						)
						Expect(err).ToNot(HaveOccurred())
						Eventually(session.Out, awsTimeout).Should(gbytes.Say("Cleanup completed"))

						session.Terminate().Wait()
						Eventually(session).Should(gexec.Exit())

						By("Downloading the uploaded files from the blobstore")
						err = cephTestClient.DownloadRemoteDirectory(
							destBucket,
							pathWithDate(destPath),
							downloadFolder,
						)
						Expect(err).ToNot(HaveOccurred())

						By("Validating the contents of the downloaded files")
						for fileName, contents := range filesToContents {
							downloadedFilePath := filepath.Join(downloadFolder, fileName)

							downloadedFile, err := os.Open(downloadedFilePath)
							Expect(err).ToNot(HaveOccurred())
							defer downloadedFile.Close()

							actualData := make([]byte, len(contents))
							_, err = downloadedFile.Read(actualData)
							Expect(err).ToNot(HaveOccurred())

							actualString := string(actualData)

							Expect(actualString).To(Equal(contents))
						}
					})
				})
			})
		})
	})
})
