// Copyright (C) 2016-Present Pivotal Software, Inc. All rights reserved.
// This program and the accompanying materials are made available under the terms of the under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package s3integration_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/ghttp"
	"github.com/satori/go.uuid"
)

var _ = Describe("S3 Backup", func() {
	var (
		region           string
		bucketName       string
		bucketPath       string
		backupCreatorCmd string
		cleanupCmd       string
	)

	BeforeEach(func() {
		endpointURL = "https://s3.amazonaws.com"
		region = ""
		bucketName = existingBucketInDefaultRegion
		bucketPath = uuid.NewV4().String()
	})

	AfterEach(func() {
		if bucketName == existingBucketInDefaultRegion || bucketName == existingBucketInNonDefaultRegion {
			Expect(s3TestClient.DeleteRemotePath(bucketName, pathWithDate(bucketPath), region)).To(Succeed())
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
							bucketName,
							bucketPath,
							endpointURL,
							region,
							backupCreatorCmd,
							cleanupCmd,
							cronSchedule,
							"",
						)
						Expect(err).ToNot(HaveOccurred())
						Eventually(session.Out, awsTimeout).Should(gbytes.Say("Cleanup completed"))

						session.Terminate().Wait()
						Eventually(session).Should(gexec.Exit())

						By("Downloading the uploaded files from the blobstore")
						err = s3TestClient.DownloadRemoteDirectory(
							bucketName,
							pathWithDate(bucketPath),
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
								bucketName,
								bucketPath,
								endpointURL,
								region,
								backupCreatorCmd,
								cleanupCmd,
								cronSchedule,
								serviceIdentifierCmd,
							)

							identifier := `"identifier":"FakeIdentifier"`

							Expect(err).ToNot(HaveOccurred())
							Eventually(session.Out, awsTimeout).Should(gbytes.Say("ServiceBackup.WithIdentifier.Perform backup started"))
							Eventually(session.Out, awsTimeout).Should(gbytes.Say(`"backup_guid":`))
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
									bucketName,
									bucketPath,
									endpointURL,
									region,
									backupCreatorCmd,
									cleanupCmd,
									cronSchedule,
									serviceIdentifierCmd,
								)

								identifier := `"identifier":"FakeIdentifier"`

								Expect(err).ToNot(HaveOccurred())
								Eventually(session.Out, awsTimeout).Should(gbytes.Say("ServiceBackup.WithIdentifier.Perform backup started"))
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

					Context("when add_deployment_name_to_backup_path is true", func() {
						It("uploads files to a path with deployment name", func() {

							By("Uploading the directory contents to the blobstore")
							session, err := performBackup(
								awsAccessKeyID,
								awsSecretAccessKey,
								sourceFolder,
								bucketName,
								bucketPath,
								endpointURL,
								region,
								backupCreatorCmd,
								cleanupCmd,
								cronSchedule,
								integrationTestDeploymentName,
							)
							Expect(err).ToNot(HaveOccurred())
							Eventually(session.Out, awsTimeout).Should(gbytes.Say("Cleanup completed"))

							session.Terminate().Wait()
							Eventually(session).Should(gexec.Exit())

							By("Downloading the uploaded files from the blobstore")
							err = s3TestClient.DownloadRemoteDirectory(
								bucketName,
								pathWithDeploymentAndDate(bucketPath, integrationTestDeploymentName),
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

				Context("using manually triggered backup", func() {
					It("uploads a snapshot that has been manually generated", func() {
						session, err := performManualBackup(
							awsAccessKeyID,
							awsSecretAccessKey,
							sourceFolder,
							bucketName,
							bucketPath,
							endpointURL,
							region,
							backupCreatorCmd,
							cleanupCmd,
						)
						Expect(err).ToNot(HaveOccurred())
						Eventually(session.Out, awsTimeout).Should(gbytes.Say("Cleanup completed"))

						session.Terminate().Wait()
						Eventually(session).Should(gexec.Exit())

						By("Downloading the uploaded files from the blobstore")
						err = s3TestClient.DownloadRemoteDirectory(
							bucketName,
							pathWithDate(bucketPath),
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
								bucketName,
								bucketPath,
								endpointURL,
								region,
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
					bucketName = existingBucketInDefaultRegion
				})

				Context("using cron scheduled backup", func() {
					It("recursively uploads the contents of a directory successfully", func() {
						By("Uploading the directory contents to the blobstore")
						session, err := performBackup(
							awsAccessKeyID,
							awsSecretAccessKey,
							sourceFolder,
							bucketName,
							bucketPath,
							endpointURL,
							region,
							backupCreatorCmd,
							cleanupCmd,
							cronSchedule,
							"",
						)
						Expect(err).ToNot(HaveOccurred())

						Consistently(session.Out, awsTimeout).ShouldNot(gbytes.Say("The bucket you are attempting to access must be addressed using the specified endpoint."))
						Eventually(session.Out, awsTimeout).Should(gbytes.Say("Cleanup completed"))
						session.Terminate().Wait()

						Eventually(session).Should(gexec.Exit())

						By("Downloading the uploaded files from the blobstore")
						err = s3TestClient.DownloadRemoteDirectory(
							bucketName,
							pathWithDate(bucketPath),
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

					strippedUUID = uuid.NewV4().String()
					strippedUUID = strippedUUID[:10]

					bucketName = integrationTestBucketNamePrefix + strippedUUID
					bucketPath = strippedUUID
				})

				AfterEach(func() {
					s3TestClient.DeleteBucket(bucketName, region)
				})

				It("makes the bucket", func() {
					By("Uploading the file to the blobstore")
					session, err := performBackup(
						awsAccessKeyID,
						awsSecretAccessKey,
						sourceFolder,
						bucketName,
						bucketPath,
						endpointURL,
						region,
						backupCreatorCmd,
						cleanupCmd,
						cronSchedule,
						"",
					)
					Expect(err).ToNot(HaveOccurred())
					Eventually(session.Out, awsTimeout).Should(gbytes.Say("Cleanup completed"))

					session.Terminate().Wait()
					Eventually(session).Should(gexec.Exit())

					keys, err := s3TestClient.ListRemotePath(bucketName, region)
					Expect(err).ToNot(HaveOccurred())
					Expect(keys).ToNot(BeEmpty())
				})

				Context("when the region requires a V4 signature", func() {
					BeforeEach(func() {
						endpointURL = "https://s3.eu-central-1.amazonaws.com"
						region = "eu-central-1"
					})

					It("makes the bucket", func() {
						By("Uploading the file to the blobstore")
						session, err := performBackup(
							awsAccessKeyID,
							awsSecretAccessKey,
							sourceFolder,
							bucketName,
							bucketPath,
							endpointURL,
							region,
							backupCreatorCmd,
							cleanupCmd,
							cronSchedule,
							"",
						)
						Expect(err).ToNot(HaveOccurred())
						Eventually(session.Out, awsTimeout).Should(gbytes.Say("Cleanup completed"))

						session.Terminate().Wait()
						Eventually(session).Should(gexec.Exit())

						keys, err := s3TestClient.ListRemotePath(bucketName, region)
						Expect(err).ToNot(HaveOccurred())
						Expect(keys).ToNot(BeEmpty())
					})

					Context("and the endpoint URL isn't set", func() {
						BeforeEach(func() {
							endpointURL = ""
						})

						It("makes the bucket", func() {
							By("Uploading the file to the blobstore")
							session, err := performBackup(
								awsAccessKeyID,
								awsSecretAccessKey,
								sourceFolder,
								bucketName,
								bucketPath,
								endpointURL,
								region,
								backupCreatorCmd,
								cleanupCmd,
								cronSchedule,
								"",
							)
							Expect(err).ToNot(HaveOccurred())
							Eventually(session.Out, awsTimeout).Should(gbytes.Say("Cleanup completed"))

							session.Terminate().Wait()
							Eventually(session).Should(gexec.Exit())

							keys, err := s3TestClient.ListRemotePath(bucketName, region)
							Expect(err).ToNot(HaveOccurred())
							Expect(keys).ToNot(BeEmpty())
						})
					})
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
							bucketName,
							bucketPath,
							endpointURL,
							region,
							backupCreatorCmd,
							failingCleanupCmd,
							cronSchedule,
							"",
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
							bucketName,
							bucketPath,
							endpointURL,
							region,
							backupCreatorCmd,
							cleanupCmd,
							cronSchedule,
							"",
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
						bucketName,
						bucketPath,
						endpointURL,
						region,
						backupCreatorCmd,
						emptyCleanupCmd,
						cronSchedule,
						"",
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
						bucketName,
						bucketPath,
						endpointURL,
						region,
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
							bucketName,
							bucketPath,
							endpointURL,
							region,
							backupCreatorCmd,
							cleanupCmd,
							cronSchedule,
							"",
						)
						Expect(err).ToNot(HaveOccurred())
						Eventually(session.Out, awsTimeout).Should(gbytes.Say("Cleanup completed"))

						session.Terminate().Wait()
						Eventually(session).Should(gexec.Exit())

						By("Downloading the uploaded files from the blobstore")
						err = s3TestClient.DownloadRemoteDirectory(
							bucketName,
							pathWithDate(bucketPath),
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
						bucketName = "doesnotexist" + uuid.NewV4().String()

						By("Not specifing a endpoint url")
						endpointURL = ""
					})

					It("logs an error", func() {
						session, err := performBackup(
							awsAccessKeyIDRestricted,
							awsSecretAccessKeyRestricted,
							sourceFolder,
							bucketName,
							bucketPath,
							endpointURL,
							region,
							backupCreatorCmd,
							cleanupCmd,
							cronSchedule,
							"",
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
				bucketPath = uuid.NewV4().String()
				By("Trying to upload the file to the blobstore")
				session, err := performBackup(
					invalidAwsAccessKeyID,
					invalidAwsSecretAccessKey,
					sourceFolder,
					bucketName,
					bucketPath,
					endpointURL,
					region,
					backupCreatorCmd,
					cleanupCmd,
					cronSchedule,
					"",
				)
				Expect(err).ToNot(HaveOccurred())
				Eventually(session.Out).Should(gbytes.Say("Service-backup Started"))

				By("logging the error")
				Eventually(session.Out, awsTimeout).Should(gbytes.Say("ServiceBackup.Upload backup completed with error"))
				Eventually(session.Out, awsTimeout).Should(gbytes.Say("InvalidAccessKeyId"))

				session.Terminate().Wait()
				Eventually(session).Should(gexec.Exit())

				By("Verifying that the bucketPath was never created")
				Expect(s3TestClient.RemotePathExistsInBucket(bucketName, bucketPath)).To(BeFalse())
			})
		})

		Context("when the endpointURL is invalid", func() {
			const invalidEndpointURL = "http://0.0.0.0:1234/"

			It("gracefully fails to perform the upload", func() {
				session, err := performBackup(
					awsAccessKeyID,
					awsSecretAccessKey,
					sourceFolder,
					bucketName,
					bucketPath,
					invalidEndpointURL,
					region,
					backupCreatorCmd,
					cleanupCmd,
					cronSchedule,
					"",
				)
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
					bucketName,
					bucketPath,
					endpointURL,
					region,
					failingBackupCreatorCmd,
					cleanupCmd,
					cronSchedule,
					"",
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
					bucketName,
					bucketPath,
					endpointURL,
					region,
					backupCreatorCmd,
					cleanupCmd,
					invalidCronSchedule,
					"",
				)

				Expect(err).ToNot(HaveOccurred())
				Eventually(session, awsTimeout).Should(gexec.Exit(2))
				Eventually(session.Out).Should(gbytes.Say("Error scheduling job"))
			})
		})
	})

	Context("when exit_if_in_progress is configured", func() {
		var (
			notificationServer *ghttp.Server
			uaaServer          *ghttp.Server
			cfServer           *ghttp.Server
			sourceFolder       string
			downloadFolder     string
		)

		BeforeEach(func() {
			var err error

			notificationServer = ghttp.NewServer()
			uaaServer = ghttp.NewServer()
			cfServer = ghttp.NewServer()

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

			notificationServer.Close()
			uaaServer.Close()
			cfServer.Close()
		})

		Context("when exit_if_in_progress is true", func() {
			exitIfInProgress := true

			Context("when a backup is in progress", func() {
				Context("and alerts are not configured", func() {
					It("accepts the first, rejects subsequent backup requests, and logs that alerts are not configured", func() {
						everySecond := "*/1 * * * * *"

						backupProcess, err := performBackupIfNotInProgress(
							awsAccessKeyID,
							awsSecretAccessKey,
							sourceFolder,
							bucketName,
							bucketPath,
							endpointURL,
							region,
							backupCreatorCmd,
							cleanupCmd,
							everySecond,
							exitIfInProgress,
						)
						Expect(err).ToNot(HaveOccurred())

						Eventually(backupProcess, awsTimeout).Should(gbytes.Say("Perform backup started"))
						Eventually(backupProcess, awsTimeout).Should(gbytes.Say("Backup currently in progress, exiting. Another backup will not be able to start until this is completed."))
						Eventually(backupProcess, awsTimeout).Should(gbytes.Say("Alerts not configured."))
						Eventually(backupProcess.Terminate()).Should(gexec.Exit())

						Expect(cfServer.ReceivedRequests()).To(HaveLen(0))
						Expect(uaaServer.ReceivedRequests()).To(HaveLen(0))
						Expect(notificationServer.ReceivedRequests()).To(HaveLen(0))
					})
				})

				Context("and alerts are configured", func() {
					var notificationRequestBodyFields map[string]string

					cfOrgResponseBody := `{
            "total_results": 1,
            "total_pages": 1,
            "prev_url": null,
            "next_url": null,
            "resources": [
              {
                "metadata": {
                  "guid": "org_guid",
                  "url": "/v2/organizations/org_guid",
                  "created_at": "2016-07-05T12:52:08Z",
                  "updated_at": null
                },
                "entity": {
                  "name": "test-org",
                  "billing_enabled": false,
                  "quota_definition_guid": "b2c23d15-7343-4f47-a4e9-5b50574bc746",
                  "status": "active",
                  "quota_definition_url": "/v2/quota_definitions/b2c23d15-7343-4f47-a4e9-5b50574bc746",
                  "spaces_url": "/v2/organizations/org_guid/spaces",
                  "domains_url": "/v2/organizations/org_guid/domains",
                  "private_domains_url": "/v2/organizations/org_guid/private_domains",
                  "users_url": "/v2/organizations/org_guid/users",
                  "managers_url": "/v2/organizations/org_guid/managers",
                  "billing_managers_url": "/v2/organizations/org_guid/billing_managers",
                  "auditors_url": "/v2/organizations/org_guid/auditors",
                  "app_events_url": "/v2/organizations/org_guid/app_events",
                  "space_quota_definitions_url": "/v2/organizations/org_guid/space_quota_definitions"
                }
              }
            ]
          }`

					cfSpacesForOrgResponse := `{
            "total_results": 1,
            "total_pages": 1,
            "prev_url": null,
            "next_url": null,
            "resources": [
              {
                "metadata": {
                  "guid": "space_guid",
                  "url": "/v2/spaces/3e6ca4d8-738f-46cb-989b-14290b887b47",
                  "created_at": "2016-07-05T13:12:01Z",
                  "updated_at": null
                },
                "entity": {
                  "name": "test-space",
                  "organization_guid": "org_guid",
                  "space_quota_definition_guid": null,
                  "allow_ssh": true,
                  "organization_url": "/v2/organizations/org_guid",
                  "developers_url": "/v2/spaces/3e6ca4d8-738f-46cb-989b-14290b887b47/developers",
                  "managers_url": "/v2/spaces/3e6ca4d8-738f-46cb-989b-14290b887b47/managers",
                  "auditors_url": "/v2/spaces/3e6ca4d8-738f-46cb-989b-14290b887b47/auditors",
                  "apps_url": "/v2/spaces/3e6ca4d8-738f-46cb-989b-14290b887b47/apps",
                  "routes_url": "/v2/spaces/3e6ca4d8-738f-46cb-989b-14290b887b47/routes",
                  "domains_url": "/v2/spaces/3e6ca4d8-738f-46cb-989b-14290b887b47/domains",
                  "service_instances_url": "/v2/spaces/3e6ca4d8-738f-46cb-989b-14290b887b47/service_instances",
                  "app_events_url": "/v2/spaces/3e6ca4d8-738f-46cb-989b-14290b887b47/app_events",
                  "events_url": "/v2/spaces/3e6ca4d8-738f-46cb-989b-14290b887b47/events",
                  "security_groups_url": "/v2/spaces/3e6ca4d8-738f-46cb-989b-14290b887b47/security_groups"
                }
              }
            ]
          }`

					BeforeEach(func() {
						notificationRequestBodyFields = map[string]string{}

						cfToken := "test token"
						notificationsToken := "token for notifications"

						uaaServer.AppendHandlers(
							ghttp.CombineHandlers(
								ghttp.VerifyRequest("POST", "/oauth/token", ""),
								ghttp.VerifyBasicAuth("cf", ""),
								ghttp.VerifyFormKV("grant_type", "password"),
								ghttp.VerifyFormKV("username", "admin"),
								ghttp.VerifyFormKV("password", "password"),
								ghttp.RespondWithJSONEncoded(http.StatusOK, map[string]interface{}{
									"access_token": cfToken,
									"token_type":   "bearer",
									"expires_in":   43199,
									"scope":        "cloud_controller.read",
									"jti":          "a-id-for-cf-token",
								}, http.Header{}),
							),
							ghttp.CombineHandlers(
								ghttp.VerifyRequest("POST", "/oauth/token", ""),
								ghttp.VerifyBasicAuth("client_id", "client_secret"),
								ghttp.VerifyFormKV("grant_type", "client_credentials"),
								ghttp.RespondWithJSONEncoded(http.StatusOK, map[string]interface{}{
									"access_token": notificationsToken,
									"token_type":   "bearer",
									"expires_in":   43199,
									"scope":        "clients.read password.write clients.secret clients.write uaa.admin scim.write scim.read",
									"jti":          "a-id-for-notifications-token",
								}, http.Header{}),
							),
						)

						cfInfoResponseBody := `
            {
                "name": "",
                "build": "",
                "support": "http://support.cloudfoundry.com",
                "version": 0,
                "description": "",
                "authorization_endpoint": "",
                "token_endpoint": "` + uaaServer.URL() + `",
                "min_cli_version": null,
                "min_recommended_cli_version": null,
                "api_version": "2.57.0",
                "app_ssh_endpoint": "",
                "app_ssh_host_key_fingerprint": "",
                "app_ssh_oauth_client": "",
                "logging_endpoint": "",
                "doppler_logging_endpoint": ""
            }`

						cfServer.AppendHandlers(
							ghttp.CombineHandlers(
								ghttp.VerifyRequest("GET", "/v2/info", ""),
								ghttp.RespondWith(http.StatusOK, cfInfoResponseBody, http.Header{}),
							),
							ghttp.CombineHandlers(
								ghttp.VerifyRequest("GET", "/v2/organizations", "q=name:cf_org"),
								ghttp.VerifyHeader(http.Header{
									"Authorization": {fmt.Sprintf("Bearer %s", cfToken)},
								}),
								ghttp.RespondWith(http.StatusOK, cfOrgResponseBody, http.Header{}),
							),
							ghttp.CombineHandlers(
								ghttp.VerifyRequest("GET", "/v2/organizations/org_guid/spaces", "q=name:cf_space"),
								ghttp.VerifyHeader(http.Header{
									"Authorization": {fmt.Sprintf("Bearer %s", cfToken)},
								}),
								ghttp.RespondWith(http.StatusOK, cfSpacesForOrgResponse, http.Header{}),
							),
						)

						notificationServer.AppendHandlers(ghttp.CombineHandlers(
							ghttp.VerifyRequest("POST", "/spaces/space_guid"),
							ghttp.VerifyHeader(http.Header{
								"X-NOTIFICATIONS-VERSION": {"1"},
								"Authorization":           {fmt.Sprintf("Bearer %s", notificationsToken)},
							}),
							ghttp.RespondWith(http.StatusOK, nil, http.Header{}),
							func(_ http.ResponseWriter, req *http.Request) {
								defer func() {
									GinkgoRecover()
									req.Body.Close()
								}()
								Expect(json.NewDecoder(req.Body).Decode(&notificationRequestBodyFields)).To(Succeed())
							},
						))
					})

					Context("without a service identifier command", func() {
						It("accepts the first, rejects subsequent backup requests, and sends an alert", func() {
							everySecond := "*/1 * * * * *"
							noServiceIdentifier := ""

							backupProcess, err := performBackupIfNotInProgressWithAlerts(
								awsAccessKeyID,
								awsSecretAccessKey,
								sourceFolder,
								bucketName,
								bucketPath,
								endpointURL,
								region,
								backupCreatorCmd,
								cleanupCmd,
								everySecond,
								noServiceIdentifier,
								exitIfInProgress,
								cfServer.URL(),
								notificationServer.URL(),
							)
							Expect(err).ToNot(HaveOccurred())

							Eventually(backupProcess, awsTimeout).Should(gbytes.Say("Perform backup started"))
							Eventually(backupProcess, awsTimeout).Should(gbytes.Say("Backup currently in progress, exiting. Another backup will not be able to start until this is completed."))
							Eventually(backupProcess, awsTimeout).Should(gbytes.Say("Sending alert."))
							Eventually(backupProcess, awsTimeout).Should(gbytes.Say("Sent alert."))
							Eventually(backupProcess.Terminate()).Should(gexec.Exit())

							Expect(cfServer.ReceivedRequests()).To(HaveLen(3))
							Expect(uaaServer.ReceivedRequests()).To(HaveLen(2))
							Expect(notificationServer.ReceivedRequests()).To(HaveLen(1))

							Expect(notificationRequestBodyFields["subject"]).To(ContainSubstring("SomeDB"))
							Expect(notificationRequestBodyFields["subject"]).To(ContainSubstring("Service Backup Failed"))
							Expect(notificationRequestBodyFields["text"]).To(ContainSubstring("Alert from SomeDB:"))
							Expect(notificationRequestBodyFields["text"]).To(ContainSubstring("A backup run has failed with the following error:"))
							Expect(notificationRequestBodyFields["text"]).To(ContainSubstring("Backup currently in progress, exiting. Another backup will not be able to start until this is completed."))
						})
					})

					Context("with a service identifier command", func() {
						It("accepts the first, rejects subsequent backup requests, and sends an alert", func() {
							everySecond := "*/1 * * * * *"
							serviceIdentifierCmd := assetPath("fake-service-identifier")

							backupProcess, err := performBackupIfNotInProgressWithAlerts(
								awsAccessKeyID,
								awsSecretAccessKey,
								sourceFolder,
								bucketName,
								bucketPath,
								endpointURL,
								region,
								backupCreatorCmd,
								cleanupCmd,
								everySecond,
								serviceIdentifierCmd,
								exitIfInProgress,
								cfServer.URL(),
								notificationServer.URL(),
							)
							Expect(err).ToNot(HaveOccurred())

							Eventually(backupProcess, awsTimeout).Should(gbytes.Say("Perform backup started"))
							Eventually(backupProcess, awsTimeout).Should(gbytes.Say("Backup currently in progress, exiting. Another backup will not be able to start until this is completed."))
							Eventually(backupProcess, awsTimeout).Should(gbytes.Say("Sending alert."))
							Eventually(backupProcess, awsTimeout).Should(gbytes.Say("Sent alert."))
							Eventually(backupProcess.Terminate()).Should(gexec.Exit())

							Expect(cfServer.ReceivedRequests()).To(HaveLen(3))
							Expect(uaaServer.ReceivedRequests()).To(HaveLen(2))
							Expect(notificationServer.ReceivedRequests()).To(HaveLen(1))

							Expect(notificationRequestBodyFields["subject"]).To(ContainSubstring("SomeDB"))
							Expect(notificationRequestBodyFields["subject"]).To(ContainSubstring("Service Backup Failed"))
							Expect(notificationRequestBodyFields["text"]).To(ContainSubstring("Alert from SomeDB, service instance FakeIdentifier:"))
							Expect(notificationRequestBodyFields["text"]).To(ContainSubstring("A backup run has failed with the following error:"))
							Expect(notificationRequestBodyFields["text"]).To(ContainSubstring("Backup currently in progress, exiting. Another backup will not be able to start until this is completed."))
						})
					})
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
						bucketName,
						bucketPath,
						endpointURL,
						region,
						backupCreatorCmd,
						cleanupCmd,
						cronSchedule,
						exitIfInProgress,
					)
					Expect(err).ToNot(HaveOccurred())

					secondBackupRequest, err := performBackupIfNotInProgress(
						awsAccessKeyID,
						awsSecretAccessKey,
						sourceFolder,
						bucketName,
						bucketPath,
						endpointURL,
						region,
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
exit_if_in_progress: false
aws_cli_path: "/var/vcap/packages/aws-cli/bin/aws"
azure_cli_path: "/var/vcap/packages/blobxfer/bin/blobxfer"
missing_properties_message: Provide these missing fields in your manifest.
service_identifier_executable:
source_executable:
source_folder:`, cronSchedule)))
			file.Close()

			backupCmd := exec.Command(pathToServiceBackupBinary, file.Name())
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

func performBackup(
	awsAccessKeyID,
	awsSecretAccessKey,
	sourceFolder,
	destBucket,
	destPath,
	endpointURL,
	region,
	backupCreatorCmd,
	cleanupCmd,
	cronSchedule string,
	deploymentName string,
) (*gexec.Session, error) {

	configFile, err := ioutil.TempFile("", "config.yml")
	Expect(err).NotTo(HaveOccurred())

	addDeploymentNameToPath := deploymentName != ""

	configFile.Write([]byte(fmt.Sprintf(`---
destinations:
- type: s3
  config:
    endpoint_url: '%s'
    region: '%s'
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
missing_properties_message: custom message
deployment_name: %s
add_deployment_name_to_backup_path: %t`, endpointURL, region, destBucket, destPath,
		awsAccessKeyID, awsSecretAccessKey, sourceFolder, backupCreatorCmd, cronSchedule,
		cleanupCmd, deploymentName, addDeploymentNameToPath,
	)))
	configFile.Close()

	backupCmd := exec.Command(pathToServiceBackupBinary, configFile.Name())
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
	region,
	backupCreatorCmd,
	cleanupCmd,
	cronSchedule string,
) (*gexec.Session, error) {

	configFile, err := ioutil.TempFile("", "config.yml")
	Expect(err).NotTo(HaveOccurred())
	configFile.Write([]byte(fmt.Sprintf(`---
destinations:
- type: s3
  name: %s
  config:
    endpoint_url: '%s'
    region: '%s'
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
missing_properties_message: custom message`, name, endpointURL, region, destBucket, destPath,
		awsAccessKeyID, awsSecretAccessKey, sourceFolder, backupCreatorCmd, cronSchedule, cleanupCmd,
	)))
	configFile.Close()

	backupCmd := exec.Command(pathToServiceBackupBinary, configFile.Name())
	return gexec.Start(backupCmd, GinkgoWriter, GinkgoWriter)
}

func performBackupIfNotInProgress(
	awsAccessKeyID,
	awsSecretAccessKey,
	sourceFolder,
	destBucket,
	destPath,
	endpointURL,
	region,
	backupCreatorCmd,
	cleanupCmd,
	cronSchedule string,
	exitIfBackupInProgress bool,
) (*gexec.Session, error) {

	configFile, err := ioutil.TempFile("", "config.yml")
	Expect(err).NotTo(HaveOccurred())
	configFile.Write([]byte(fmt.Sprintf(`---
destinations:
- type: s3
  config:
    endpoint_url: '%s'
    region: '%s'
    bucket_name: %s
    bucket_path: %s
    access_key_id: %s
    secret_access_key: %s
source_folder: %s
source_executable: %s
aws_cli_path: aws
exit_if_in_progress: %v
cron_schedule: '%s'
cleanup_executable: %s
missing_properties_message: custom message`, endpointURL, region, destBucket, destPath,
		awsAccessKeyID, awsSecretAccessKey, sourceFolder, backupCreatorCmd, exitIfBackupInProgress, cronSchedule, cleanupCmd,
	)))
	configFile.Close()

	backupCmd := exec.Command(pathToServiceBackupBinary, configFile.Name())
	return gexec.Start(backupCmd, GinkgoWriter, GinkgoWriter)
}

func performBackupIfNotInProgressWithAlerts(
	awsAccessKeyID,
	awsSecretAccessKey,
	sourceFolder,
	destBucket,
	destPath,
	endpointURL,
	region,
	backupCreatorCmd,
	cleanupCmd,
	cronSchedule,
	serviceIdentifierCmd string,
	exitIfBackupInProgress bool,
	cfApiURL string,
	notificationServerURL string,
) (*gexec.Session, error) {

	configFile, err := ioutil.TempFile("", "config.yml")
	Expect(err).NotTo(HaveOccurred())
	configFile.Write([]byte(fmt.Sprintf(`---
destinations:
- type: s3
  config:
    endpoint_url: %s
    region: '%s'
    bucket_name: %s
    bucket_path: %s
    access_key_id: %s
    secret_access_key: %s
source_folder: %s
source_executable: %s
aws_cli_path: aws
exit_if_in_progress: %v
cron_schedule: '%s'
cleanup_executable: %s
missing_properties_message: custom message
service_identifier_executable: %s
alerts:
  product_name: SomeDB
  config:
    cloud_controller:
      url: %s
      user: admin
      password: password
    notifications:
      service_url: %s
      cf_org: cf_org
      cf_space: cf_space
      reply_to: me@example.com
      client_id: client_id
      client_secret: client_secret
    timeout_seconds: 60
    skip_ssl_validation: false
    `, endpointURL, region, destBucket, destPath,
		awsAccessKeyID, awsSecretAccessKey, sourceFolder, backupCreatorCmd, exitIfBackupInProgress, cronSchedule, cleanupCmd, serviceIdentifierCmd, cfApiURL, notificationServerURL,
	)))
	configFile.Close()

	backupCmd := exec.Command(pathToServiceBackupBinary, configFile.Name())
	return gexec.Start(backupCmd, GinkgoWriter, GinkgoWriter)
}

func performManualBackup(
	awsAccessKeyID,
	awsSecretAccessKey,
	sourceFolder,
	destBucket,
	destPath,
	endpointURL,
	region,
	backupCreatorCmd,
	cleanupCmd string,
) (*gexec.Session, error) {

	configFile, err := ioutil.TempFile("", "config.yml")
	Expect(err).NotTo(HaveOccurred())
	configFile.Write([]byte(fmt.Sprintf(`---
destinations:
- type: s3
  config:
    endpoint_url: '%s'
    region: '%s'
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
missing_properties_message: custom message`, endpointURL, region, destBucket, destPath,
		awsAccessKeyID, awsSecretAccessKey, sourceFolder, backupCreatorCmd, cronSchedule, cleanupCmd,
	)))
	configFile.Close()

	manualBackupCmd := exec.Command(pathToManualBackupBinary, configFile.Name())
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
	region,
	backupCreatorCmd,
	cleanupCmd,
	cronSchedule,
	serviceIdentifierCmd string,
) (*gexec.Session, error) {

	configFile, err := ioutil.TempFile("", "config.yml")
	Expect(err).NotTo(HaveOccurred())
	configFile.Write([]byte(fmt.Sprintf(`---
destinations:
- type: s3
  name: %s
  config:
    endpoint_url: '%s'
    region: '%s'
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
missing_properties_message: custom message`, name, endpointURL, region, destBucket, destPath,
		awsAccessKeyID, awsSecretAccessKey, sourceFolder, backupCreatorCmd, cronSchedule, cleanupCmd, serviceIdentifierCmd,
	)))
	configFile.Close()

	backupCmd := exec.Command(pathToServiceBackupBinary, configFile.Name())
	return gexec.Start(backupCmd, GinkgoWriter, GinkgoWriter)
}

func pathWithDate(path string) string {
	today := time.Now()
	datePath := fmt.Sprintf("%d/%02d/%02d", today.Year(), today.Month(), today.Day())
	return path + "/" + datePath
}

func pathWithDeploymentAndDate(path, deploymentName string) string {
	today := time.Now()
	return fmt.Sprintf(
		"%s/%s/%d/%02d/%02d", path, deploymentName, today.Year(), today.Month(), today.Day(),
	)
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

	fileContentsUUID := uuid.NewV4()

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
