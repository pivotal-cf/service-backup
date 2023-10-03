// Copyright (C) 2016-Present Pivotal Software, Inc. All rights reserved.
// This program and the accompanying materials are made available under the terms of the under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package azureintegration_test

import (
	"crypto/rand"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/storage"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("AzureClient", func() {
	var azureContainer string

	BeforeEach(func() {
		azureContainer = "ci-blobs-" + strconv.Itoa(int(time.Now().UnixNano()))
	})

	Context("the client is correctly configured", func() {
		AfterEach(func() {
			deleteAzureContainer(azureContainer)
		})

		Context("and the container already exists", func() {
			BeforeEach(func() {
				createAzureContainer(azureContainer)
			})

			It("uploads the backup", func() {
				uploadsTheBackup(azureContainer, false)
			})

			It("uploads the large backup", func() {
				sourceFolder, err := ioutil.TempDir("", "azure")
				Expect(err).ToNot(HaveOccurred())

				fileName := "bigfile.dat"
				fileContent := make([]byte, 100*1000*1024)
				_, err = rand.Read(fileContent)
				Expect(err).NotTo(HaveOccurred())

				Expect(ioutil.WriteFile(filepath.Join(sourceFolder, fileName), fileContent, os.ModePerm)).To(Succeed())

				today := time.Now()
				deploymentName := ""
				destinationPath := fmt.Sprintf("path/to/blobs/%d", today.Unix())

				session := performBackup(sourceFolder, azureContainer, destinationPath, deploymentName, "")

				Eventually(session.Out, azureTimeout).Should(gbytes.Say("Cleanup completed successfully"))
				session.Terminate().Wait()
				Eventually(session).Should(gexec.Exit())

				azureBlobService := azureBlobService()
				backupBlobPath := remotePath(destinationPath, deploymentName, today, fileName)
				Expect(downloadBlob(azureBlobService, azureContainer, backupBlobPath)).To(Equal([]byte(fileContent)))
			})
		})

		Context("and the container doesn't exist", func() {
			It("uploads the backup", func() {
				uploadsTheBackup(azureContainer, false)
			})
		})

		Context("when add_deployment_name_to_backup_path is true", func() {
			It("uploads the backup with the deployment_name in the path", func() {
				uploadsTheBackup(azureContainer, true)
			})
		})

		Context("with endpoint configured", func() {
			It("uploads the backup", func() {
				sourceFolder := prepareBackupContents()

				destinationPath := fmt.Sprintf("path/to/blobs/%d", time.Now().Unix())

				session := performBackup(sourceFolder, azureContainer, destinationPath, "", "core.windows.net")
				Eventually(session.Out, azureTimeout).Should(gbytes.Say(`"destination_name":"foo"`))
				Eventually(session.Out, azureTimeout).Should(gbytes.Say("Cleanup completed successfully"))
				session.Terminate().Wait()

				Eventually(session).Should(gexec.Exit())
			})
		})

	})

	Context("the client endpoint is misconfigured", func() {
		It("fails to upload", func() {
			sourceFolder := prepareBackupContents()

			destinationPath := fmt.Sprintf("path/to/blobs/%d", time.Now().Unix())

			session := performBackup(sourceFolder, azureContainer, destinationPath, "", "wrong.example.com")
			Eventually(session.Out, azureTimeout).Should(gbytes.Say("Failed to establish a new connection"))
			Eventually(session.Out, azureTimeout).Should(gbytes.Say("Upload backup completed with error"))
			session.Terminate().Wait()

			Eventually(session).Should(gexec.Exit())
		})
	})
})

func prepareBackupContents() string {
	sourceFolder, err := ioutil.TempDir("", "azure")
	Expect(err).ToNot(HaveOccurred())
	firstBackupFileName := "morning/events.log"
	firstBackupFileContent := "coffee"
	secondBackupFileName := "afternoon/events.log"
	secondBackupFileContent := "ping-pong"
	createFakeBackupFile(sourceFolder, firstBackupFileName, firstBackupFileContent)
	createFakeBackupFile(sourceFolder, secondBackupFileName, secondBackupFileContent)

	return sourceFolder
}

func uploadsTheBackup(azureContainer string, addDeploymentName bool) {
	sourceFolder, err := ioutil.TempDir("", "azure")
	Expect(err).ToNot(HaveOccurred())

	firstBackupFileName := "morning/events.log"
	firstBackupFileContent := "coffee"
	secondBackupFileName := "afternoon/events.log"
	secondBackupFileContent := "ping-pong"

	createFakeBackupFile(sourceFolder, firstBackupFileName, firstBackupFileContent)
	createFakeBackupFile(sourceFolder, secondBackupFileName, secondBackupFileContent)

	today := time.Now()

	destinationPath := fmt.Sprintf("path/to/blobs/%d", today.Unix())
	deploymentName := ""
	if addDeploymentName {
		deploymentName = "deployment-name"
	}

	session := performBackup(sourceFolder, azureContainer, destinationPath, deploymentName, "")
	Eventually(session.Out, azureTimeout).Should(gbytes.Say(`"destination_name":"foo"`))
	Eventually(session.Out, azureTimeout).Should(gbytes.Say("Cleanup completed successfully"))
	session.Terminate().Wait()
	Eventually(session).Should(gexec.Exit())

	azureBlobService := azureBlobService()

	firstBackupBlobPath := remotePath(destinationPath, deploymentName, today, firstBackupFileName)
	Expect(downloadBlob(azureBlobService, azureContainer, firstBackupBlobPath)).To(Equal([]byte(firstBackupFileContent)))

	secondBackupBlobPath := remotePath(destinationPath, deploymentName, today, secondBackupFileName)
	Expect(downloadBlob(azureBlobService, azureContainer, secondBackupBlobPath)).To(Equal([]byte(secondBackupFileContent)))
}

func remotePath(destinationPath, deploymentName string, today time.Time, firstBackupFileName string) string {
	if deploymentName != "" {
		return fmt.Sprintf("%s/%s/%d/%02d/%02d/%s", destinationPath, deploymentName, today.Year(), int(today.Month()), today.Day(), firstBackupFileName)
	}
	return fmt.Sprintf("%s/%d/%02d/%02d/%s", destinationPath, today.Year(), int(today.Month()), today.Day(), firstBackupFileName)
}

func azureBlobService() storage.BlobStorageClient {
	azureClient, err := storage.NewBasicClient(azureAccountName, azureAccountKey)
	Expect(err).ToNot(HaveOccurred())
	return azureClient.GetBlobService()
}

func createAzureContainer(name string) {
	service := azureBlobService()

	containerRef := service.GetContainerReference(name)
	Expect(containerRef.Create(&storage.CreateContainerOptions{Access: storage.ContainerAccessTypePrivate})).To(Succeed())
}

func deleteAzureContainer(name string) {
	service := azureBlobService()
	containerRef := service.GetContainerReference(name)
	_, err := containerRef.DeleteIfExists(&storage.DeleteContainerOptions{})
	Expect(err).To(Succeed())
}

func runBackup(params ...string) *gexec.Session {
	backupCmd := exec.Command(pathToServiceBackupBinary, params...)
	session, err := gexec.Start(backupCmd, GinkgoWriter, GinkgoWriter)
	Expect(err).ToNot(HaveOccurred())
	return session
}

func performBackup(sourceFolder, azureContainer, destinationPath, deploymentName, endpoint string) *gexec.Session {
	file, err := ioutil.TempFile("", "config.yml")
	Expect(err).NotTo(HaveOccurred())

	addDeploymentNameToPath := deploymentName != ""

	file.Write([]byte(fmt.Sprintf(`---
destinations:
- type: azure
  name: foo
  config:
    storage_account: %s
    storage_access_key: %s
    container: %s
    path: %s
    endpoint: %s
source_folder: %s
source_executable: true
exit_if_in_progress: true
cron_schedule: '*/5 * * * * *'
cleanup_executable: true
missing_properties_message: custom message
deployment_name: %s
add_deployment_name_to_backup_path: %t`, azureAccountName, azureAccountKey, azureContainer,
		destinationPath, endpoint, sourceFolder, deploymentName, addDeploymentNameToPath,
	)))
	file.Close()

	return runBackup(file.Name())
}

func createFakeBackupFile(sourceFolder, fileName, content string) {
	filePath := sourceFolder + "/" + fileName
	Expect(os.MkdirAll(filepath.Dir(filePath), 0777)).To(Succeed())
	Expect(ioutil.WriteFile(filePath, []byte(content), 0777)).To(Succeed())
}

func downloadBlob(azureBlobService storage.BlobStorageClient, azureContainer, path string) []byte {
	blob := storage.Blob{
		Container: azureBlobService.GetContainerReference(azureContainer),
		Name:      path,
	}
	b, err := blob.Get(&storage.GetBlobOptions{})
	Expect(err).NotTo(HaveOccurred())

	content, err := ioutil.ReadAll(b)
	Expect(err).ToNot(HaveOccurred())
	return content
}
