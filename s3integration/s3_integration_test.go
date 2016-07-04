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

	file, err := ioutil.TempFile("", "config.yml")
	Expect(err).NotTo(HaveOccurred())
	file.Write([]byte(fmt.Sprintf(`---
destinations:
- type: s3
  config:
    endpoint_url: '%s'
    bucket_name: %s
    bucket_path: %s
    access_key_id: %s
    secret_access_key: %s
source_folder: %s
source_executable: %s
aws_cli_path: aws
exit_if_in_progress: false
cron_schedule: '%s'
cleanup_executable: %s
missing_properties_message: custom message`, endpointURL, destBucket, destPath,
		awsAccessKeyID, awsSecretAccessKey, sourceFolder, backupCreatorCmd, cronSchedule, cleanupCmd,
	)))
	file.Close()

	backupCmd := exec.Command(pathToServiceBackupBinary, file.Name(), "--logLevel", "debug")
	return gexec.Start(backupCmd, GinkgoWriter, GinkgoWriter)
}

func performBackupWithName(
	name,
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

	file, err := ioutil.TempFile("", "config.yml")
	Expect(err).NotTo(HaveOccurred())
	file.Write([]byte(fmt.Sprintf(`---
destinations:
- type: s3
  name: %s
  config:
    endpoint_url: '%s'
    bucket_name: %s
    bucket_path: %s
    access_key_id: %s
    secret_access_key: %s
source_folder: %s
source_executable: %s
aws_cli_path: aws
exit_if_in_progress: false
cron_schedule: '%s'
cleanup_executable: %s
missing_properties_message: custom message`, name, endpointURL, destBucket, destPath,
		awsAccessKeyID, awsSecretAccessKey, sourceFolder, backupCreatorCmd, cronSchedule, cleanupCmd,
	)))
	file.Close()

	backupCmd := exec.Command(pathToServiceBackupBinary, file.Name(), "--logLevel", "debug")
	return gexec.Start(backupCmd, GinkgoWriter, GinkgoWriter)
}

func performBackupIfNotInProgress(
	awsAccessKeyID,
	awsSecretAccessKey,
	sourceFolder,
	destBucket,
	destPath,
	endpointURL,
	backupCreatorCmd,
	cleanupCmd,
	cronSchedule string,
	exitIfBackupInProgress bool,
) (*gexec.Session, error) {

	file, err := ioutil.TempFile("", "config.yml")
	Expect(err).NotTo(HaveOccurred())
	file.Write([]byte(fmt.Sprintf(`---
destinations:
- type: s3
  config:
    endpoint_url: %s
    bucket_name: %s
    bucket_path: %s
    access_key_id: %s
    secret_access_key: %s
source_folder: %s
source_executable: %s
aws_cli_path: aws
exit_if_in_progress: %s
cron_schedule: '%s'
cleanup_executable: %s
missing_properties_message: custom message`, endpointURL, destBucket, destPath,
		awsAccessKeyID, awsSecretAccessKey, sourceFolder, backupCreatorCmd, fmt.Sprintf("%v", exitIfBackupInProgress), cronSchedule, cleanupCmd,
	)))
	file.Close()

	backupCmd := exec.Command(pathToServiceBackupBinary, file.Name())
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

	file, err := ioutil.TempFile("", "config.yml")
	Expect(err).NotTo(HaveOccurred())
	file.Write([]byte(fmt.Sprintf(`---
destinations:
- type: s3
  config:
    endpoint_url: %s
    bucket_name: %s
    bucket_path: %s
    access_key_id: %s
    secret_access_key: %s
source_folder: %s
source_executable: %s
aws_cli_path: aws
exit_if_in_progress: false
cron_schedule: '%s'
cleanup_executable: %s
missing_properties_message: custom message`, endpointURL, destBucket, destPath,
		awsAccessKeyID, awsSecretAccessKey, sourceFolder, backupCreatorCmd, cronSchedule, cleanupCmd,
	)))
	file.Close()

	manualBackupCmd := exec.Command(pathToManualBackupBinary, file.Name())
	return gexec.Start(manualBackupCmd, GinkgoWriter, GinkgoWriter)

}

func performBackupWithServiceIdentifier(
	name,
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

	file, err := ioutil.TempFile("", "config.yml")
	Expect(err).NotTo(HaveOccurred())
	file.Write([]byte(fmt.Sprintf(`---
destinations:
- type: s3
  name: %s
  config:
    endpoint_url: %s
    bucket_name: %s
    bucket_path: %s
    access_key_id: %s
    secret_access_key: %s
source_folder: %s
source_executable: %s
aws_cli_path: aws
exit_if_in_progress: false
cron_schedule: '%s'
cleanup_executable: %s
service_identifier_executable: %s
missing_properties_message: custom message`, name, endpointURL, destBucket, destPath,
		awsAccessKeyID, awsSecretAccessKey, sourceFolder, backupCreatorCmd, cronSchedule, cleanupCmd, serviceIdentifierCmd,
	)))
	file.Close()

	backupCmd := exec.Command(pathToServiceBackupBinary, file.Name(), "--logLevel", "debug")
	return gexec.Start(backupCmd, GinkgoWriter, GinkgoWriter)
}

func pathWithDate(path string) string {
	today := time.Now()
	datePath := fmt.Sprintf("%d/%02d/%02d", today.Year(), today.Month(), today.Day())
	return path + "/" + datePath
}

func createFilesToUpload(sourceFolder string, smallFile bool) map[string]string {
	createdFiles := map[string]string{}

	var rootFile string
	var contents string

	if smallFile {
		rootFile, contents = createFileIn(sourceFolder)
	} else {
		rootFile, contents = createLargeFileIn(sourceFolder)
	}

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

func createLargeFileIn(sourceFolder string) (string, string) {
	file, err := ioutil.TempFile(sourceFolder, "")
	Expect(err).ToNot(HaveOccurred())

	fileContents := string(make([]byte, 100*1000*1024))
	_, err = file.Write([]byte(fileContents))
	Expect(err).ToNot(HaveOccurred())

	fileName := filepath.Base(file.Name())
	return fileName, fileContents
}

var _ = Describe("S3 Backup", func() {
	var (
		destBucket       string
		backupCreatorCmd string
		cleanupCmd       string
	)

	BeforeEach(func() {
		endpointURL = "https://s3.amazonaws.com"
		destBucket = existingBucketInDefaultRegion

		destPathUUID, err := uuid.NewV4()
		Expect(err).ToNot(HaveOccurred())
		destPath = destPathUUID.String()
	})

	AfterEach(func() {
		if destBucket == existingBucketInDefaultRegion || destBucket == existingBucketInNonDefaultRegion {
			Expect(s3TestClient.DeleteRemotePath(destBucket, pathWithDate(destPath))).To(Succeed())
		}
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

			filesToContents = createFilesToUpload(sourceFolder, true)

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

			Context("when the bucket already exists in the default region", func() {

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
								"",
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
							Eventually(session.Out, awsTimeout).Should(gbytes.Say("ServiceBackup.WithIdentifier.Perform backup started"))
							Eventually(session.Out, awsTimeout).Should(gbytes.Say(identifier))
							Eventually(session.Out, awsTimeout).Should(gbytes.Say("ServiceBackup.WithIdentifier.Perform backup debug info"))
							Eventually(session.Out, awsTimeout).Should(gbytes.Say(identifier))
							Eventually(session.Out, awsTimeout).Should(gbytes.Say("ServiceBackup.WithIdentifier.Perform backup completed successfully"))
							Eventually(session.Out, awsTimeout).Should(gbytes.Say(identifier))
							Eventually(session.Out, awsTimeout).Should(gbytes.Say("ServiceBackup.WithIdentifier.Upload backup started"))
							Eventually(session.Out, awsTimeout).Should(gbytes.Say(identifier))
							Eventually(session.Out, awsTimeout).Should(gbytes.Say("ServiceBackup.WithIdentifier.about to upload"))
							Consistently(session.Out, awsTimeout).ShouldNot(gbytes.Say(`"destination_name":`))
							Eventually(session.Out, awsTimeout).Should(gbytes.Say(identifier))
							Eventually(session.Out, awsTimeout).Should(gbytes.Say("ServiceBackup.WithIdentifier.s3 completed"))
							Eventually(session.Out, awsTimeout).Should(gbytes.Say(identifier))
							Eventually(session.Out, awsTimeout).Should(gbytes.Say("ServiceBackup.WithIdentifier.Upload backup completed successfully"))
							Eventually(session.Out, awsTimeout).Should(gbytes.Say(identifier))
							Eventually(session.Out, awsTimeout).Should(gbytes.Say("ServiceBackup.WithIdentifier.Cleanup completed"))
							Eventually(session.Out, awsTimeout).Should(gbytes.Say(identifier))

							session.Terminate().Wait()
							Eventually(session).Should(gexec.Exit())
						})

						Context("and a destination name is provided", func() {
							It("logs events with the service identifier and destination name", func() {
								session, err := performBackupWithServiceIdentifier(
									"bar",
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
								Eventually(session.Out, awsTimeout).Should(gbytes.Say("ServiceBackup.WithIdentifier.Perform backup started"))
								Eventually(session.Out, awsTimeout).Should(gbytes.Say(identifier))
								Eventually(session.Out, awsTimeout).Should(gbytes.Say("ServiceBackup.WithIdentifier.Perform backup debug info"))
								Eventually(session.Out, awsTimeout).Should(gbytes.Say(identifier))
								Eventually(session.Out, awsTimeout).Should(gbytes.Say("ServiceBackup.WithIdentifier.Perform backup completed successfully"))
								Eventually(session.Out, awsTimeout).Should(gbytes.Say(identifier))
								Eventually(session.Out, awsTimeout).Should(gbytes.Say("ServiceBackup.WithIdentifier.Upload backup started"))
								Eventually(session.Out, awsTimeout).Should(gbytes.Say(identifier))
								Eventually(session.Out, awsTimeout).Should(gbytes.Say("ServiceBackup.WithIdentifier.about to upload"))
								Eventually(session.Out, awsTimeout).Should(gbytes.Say(`"destination_name":"bar"`))
								Eventually(session.Out, awsTimeout).Should(gbytes.Say(identifier))
								Eventually(session.Out, awsTimeout).Should(gbytes.Say("ServiceBackup.WithIdentifier.s3 completed"))
								Eventually(session.Out, awsTimeout).Should(gbytes.Say(identifier))
								Eventually(session.Out, awsTimeout).Should(gbytes.Say("ServiceBackup.WithIdentifier.Upload backup completed successfully"))
								Eventually(session.Out, awsTimeout).Should(gbytes.Say(identifier))
								Eventually(session.Out, awsTimeout).Should(gbytes.Say("ServiceBackup.WithIdentifier.Cleanup completed"))
								Eventually(session.Out, awsTimeout).Should(gbytes.Say(identifier))

								session.Terminate().Wait()
								Eventually(session).Should(gexec.Exit())
							})
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

			Context("when the bucket already exists in a different region", func() {
				BeforeEach(func() {
					By("Not specifing a endpoint url")
					endpointURL = ""
					destBucket = existingBucketInNonDefaultRegion
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

						Consistently(session.Out, awsTimeout).ShouldNot(gbytes.Say("The bucket you are attempting to access must be addressed using the specified endpoint."))
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
			})

			Context("when the bucket does not already exist", func() {
				var strippedUUID string

				BeforeEach(func() {
					endpointURL = ""
					bucketUUID, err := uuid.NewV4()
					Expect(err).ToNot(HaveOccurred())

					strippedUUID = bucketUUID.String()
					strippedUUID = strippedUUID[:10]

					destBucket = existingBucketInDefaultRegion + strippedUUID
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
						Eventually(session.Out, awsTimeout).Should(gbytes.Say("Cleanup completed successfully"))
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

			Context("when destination name is provided", func() {
				It("logs and exits without error", func() {
					session, err := performBackupWithName(
						"foo",
						awsAccessKeyID,
						awsSecretAccessKey,
						sourceFolder,
						destBucket,
						destPath,
						endpointURL,
						backupCreatorCmd,
						"ls",
						cronSchedule,
					)
					Expect(err).ToNot(HaveOccurred())
					Eventually(session.Out, awsTimeout).Should(gbytes.Say(`"destination_name":"foo"`))
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

						By("Not specifing a endpoint url")
						endpointURL = ""
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

		Context("when the endpointURL is invalid", func() {
			const invalidEndpointURL = "http://0.0.0.0:1234/"
			It("gracefully fails to perform the upload", func() {
				session, err := performBackup(
					awsAccessKeyID,
					awsSecretAccessKey,
					sourceFolder,
					destBucket,
					destPath,
					invalidEndpointURL,
					backupCreatorCmd,
					cleanupCmd,
					cronSchedule,
				)

				Expect(err).ToNot(HaveOccurred())
				Expect(err).ToNot(HaveOccurred())
				Eventually(session.Out, awsTimeout).Should(gbytes.Say("Connection aborted"))

				session.Terminate().Wait()
				Eventually(session).Should(gexec.Exit())
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

	Context("when exit_if_in_progress is configured", func() {
		var (
			sourceFolder   string
			downloadFolder string
		)

		BeforeEach(func() {
			var err error

			sourceFolder, err = ioutil.TempDir("", "")
			Expect(err).ToNot(HaveOccurred())

			downloadFolder, err = ioutil.TempDir("", "")
			Expect(err).ToNot(HaveOccurred())

			createFilesToUpload(sourceFolder, false)

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

		Context("when exit_if_in_progress is true", func() {
			exitIfInProgress := true

			Context("when a backup is in progress", func() {
				It("accepts the first, rejects subsequent backup requests", func() {
					sessionForBackupThatGoesThrough, err1 := performBackupIfNotInProgress(
						awsAccessKeyID,
						awsSecretAccessKey,
						sourceFolder,
						destBucket,
						destPath,
						endpointURL,
						backupCreatorCmd,
						cleanupCmd,
						cronSchedule,
						exitIfInProgress,
					)

					sessionForBackupThatGetsRejected, err2 := performBackupIfNotInProgress(
						awsAccessKeyID,
						awsSecretAccessKey,
						sourceFolder,
						destBucket,
						destPath,
						endpointURL,
						backupCreatorCmd,
						cleanupCmd,
						cronSchedule,
						exitIfInProgress,
					)

					Expect(err1).ToNot(HaveOccurred())
					Expect(err2).ToNot(HaveOccurred())

					Eventually(sessionForBackupThatGoesThrough.Out, awsTimeout).Should(gbytes.Say("Perform backup started"))
					Eventually(sessionForBackupThatGetsRejected.Out, awsTimeout).Should(gbytes.Say("Backup currently in progress, exiting. Another backup will not be able to start until this is completed."))
					Eventually(sessionForBackupThatGoesThrough.Out, awsTimeout).Should(gbytes.Say("Cleanup completed"))

					sessionForBackupThatGoesThrough.Terminate().Wait()
					sessionForBackupThatGetsRejected.Terminate().Wait()

					Eventually(sessionForBackupThatGoesThrough).Should(gexec.Exit())
					Eventually(sessionForBackupThatGetsRejected).Should(gexec.Exit())
				})

			})
		})

		Context("when exit_if_in_progress is false", func() {
			exitIfInProgress := false

			Context("when a backup is in progress", func() {
				It("successfully completes new backup requests", func() {
					firstBackupRequest, err := performBackupIfNotInProgress(
						awsAccessKeyID,
						awsSecretAccessKey,
						sourceFolder,
						destBucket,
						destPath,
						endpointURL,
						backupCreatorCmd,
						cleanupCmd,
						cronSchedule,
						exitIfInProgress,
					)

					secondBackupRequest, err := performBackupIfNotInProgress(
						awsAccessKeyID,
						awsSecretAccessKey,
						sourceFolder,
						destBucket,
						destPath,
						endpointURL,
						backupCreatorCmd,
						cleanupCmd,
						cronSchedule,
						exitIfInProgress,
					)

					Expect(err).ToNot(HaveOccurred())
					Eventually(firstBackupRequest.Out, awsTimeout).Should(gbytes.Say("Perform backup started"))
					Eventually(secondBackupRequest.Out, awsTimeout).Should(gbytes.Say("Perform backup started"))
					Eventually(firstBackupRequest.Out, awsTimeout).Should(gbytes.Say("Cleanup completed"))
					Eventually(secondBackupRequest.Out, awsTimeout).Should(gbytes.Say("Cleanup completed"))
					Consistently(secondBackupRequest.Out, awsTimeout).ShouldNot(gbytes.Say("Backup currently in progress, exiting. Another backup will not be able to start until this is completed."))

					firstBackupRequest.Terminate().Wait()
					secondBackupRequest.Terminate().Wait()

					Eventually(firstBackupRequest).Should(gexec.Exit())
					Eventually(secondBackupRequest).Should(gexec.Exit())
				})
			})
		})
	})

	Context("when no destination is specified", func() {
		var session *gexec.Session

		BeforeEach(func() {
			file, err := ioutil.TempFile("", "config.yml")
			Expect(err).NotTo(HaveOccurred())
			file.Write([]byte(fmt.Sprintf(`---
cleanup_executable: ''
cron_schedule: '%s'
destinations: []
exit_if_in_progress: 'false'
aws_cli_path: "/var/vcap/packages/aws-cli/bin/aws"
azure_cli_path: "/var/vcap/packages/blobxfer/bin/blobxfer"
missing_properties_message: Provide these missing fields in your manifest.
service_identifier_executable:
source_executable:
source_folder:`, cronSchedule)))
			file.Close()

			backupCmd := exec.Command(pathToServiceBackupBinary, file.Name(), "--logLevel", "debug")
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
})
