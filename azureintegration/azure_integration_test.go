package azureintegration_test

import (
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/Azure/azure-sdk-for-go/storage"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var (
	azureContainer string
)

func azureBlobService() storage.BlobStorageClient {
	azureClient, err := storage.NewBasicClient(azureAccountName, azureAccountKey)
	Expect(err).ToNot(HaveOccurred())
	return azureClient.GetBlobService()
}

func createAzureContainer(name string) {
	service := azureBlobService()
	Expect(service.CreateContainer(name, storage.ContainerAccessTypePrivate)).To(Succeed())
}

func deleteAzureContainer(name string) {
	service := azureBlobService()
	_, err := service.DeleteContainerIfExists(name)
	Expect(err).To(Succeed())
}

func runBackup(params ...string) *gexec.Session {
	backupCmd := exec.Command(pathToServiceBackupBinary, params...)
	session, err := gexec.Start(backupCmd, GinkgoWriter, GinkgoWriter)
	Expect(err).ToNot(HaveOccurred())
	return session
}

//CHANGE THE ARGUMENTS
func performBackup(sourceFolder, destinationPath string) *gexec.Session {
	file, err := ioutil.TempFile("", "config.yml")
	Expect(err).NotTo(HaveOccurred())
	file.Write([]byte(fmt.Sprintf(`---
destinations:
- type: azure
  config:
    storage_account: %s
    storage_access_key: %s
    container: %s
    path: %s
    blob_store_base_url: core.windows.net
source_folder: %s
source_executable: true
azure_cli_path: blobxfer
exit_if_in_progress: true
cron_schedule: '*/5 * * * * *'
cleanup_executable: true
missing_properties_message: custom message`, azureAccountName, azureAccountKey, azureContainer,
		destinationPath, sourceFolder,
	)))
	file.Close()

	return runBackup(file.Name())
}

func createFakeBackupFile(sourceFolder, fileName, content string) {
	filePath := sourceFolder + "/" + fileName
	Expect(os.MkdirAll(filepath.Dir(filePath), 0777)).To(Succeed())
	Expect(ioutil.WriteFile(filePath, []byte(content), 0777)).To(Succeed())
}

func downloadBlob(azureBlobService storage.BlobStorageClient, path string) []byte {
	blob, err := azureBlobService.GetBlob(azureContainer, path)
	Expect(err).ToNot(HaveOccurred())
	content, err := ioutil.ReadAll(blob)
	Expect(err).ToNot(HaveOccurred())
	return content
}

var _ = Describe("AzureClient", func() {
	Context("the client is correctly configured", func() {
		BeforeEach(func() {
			azureContainer = "ci-blobs-" + strconv.Itoa(int(time.Now().UnixNano()))
		})

		AfterEach(func() {
			deleteAzureContainer(azureContainer)
		})

		uploadsTheBackup := func() {
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

			session := performBackup(sourceFolder, destinationPath)
			Eventually(session.Out, azureTimeout).Should(gbytes.Say("Cleanup completed successfully"))
			session.Terminate().Wait()
			Eventually(session).Should(gexec.Exit())

			azureBlobService := azureBlobService()

			firstBackupBlobPath := fmt.Sprintf("%s/%d/%02d/%02d/%s", destinationPath, today.Year(), int(today.Month()), today.Day(), firstBackupFileName)
			Expect(downloadBlob(azureBlobService, firstBackupBlobPath)).To(Equal([]byte(firstBackupFileContent)))

			secondBackupBlobPath := fmt.Sprintf("%s/%d/%02d/%02d/%s", destinationPath, today.Year(), int(today.Month()), today.Day(), secondBackupFileName)
			Expect(downloadBlob(azureBlobService, secondBackupBlobPath)).To(Equal([]byte(secondBackupFileContent)))
		}

		uploadsTheLargeBackup := func() {
			sourceFolder, err := ioutil.TempDir("", "azure")
			Expect(err).ToNot(HaveOccurred())

			fileName := "bigfile.dat"
			fileContent := make([]byte, 100*1000*1024)
			_, err = rand.Read(fileContent)

			Expect(ioutil.WriteFile(filepath.Join(sourceFolder, fileName), fileContent, os.ModePerm)).To(Succeed())

			Expect(err).NotTo(HaveOccurred())

			today := time.Now()
			destinationPath := fmt.Sprintf("path/to/blobs/%d", today.Unix())

			session := performBackup(sourceFolder, destinationPath)

			Eventually(session.Out, azureTimeout).Should(gbytes.Say("Cleanup completed successfully"))
			session.Terminate().Wait()
			Eventually(session).Should(gexec.Exit())

			azureBlobService := azureBlobService()

			backupBlobPath := fmt.Sprintf("%s/%d/%02d/%02d/%s", destinationPath, today.Year(), int(today.Month()), today.Day(), fileName)

			Expect(downloadBlob(azureBlobService, backupBlobPath)).To(Equal([]byte(fileContent)))
		}

		Context("and the container already exists", func() {
			BeforeEach(func() {
				createAzureContainer(azureContainer)
			})

			It("uploads the backup", uploadsTheBackup)

			It("uploads the large backup", uploadsTheLargeBackup)
		})

		Context("and the container doesn't exist", func() {
			It("uploads the backup", uploadsTheBackup)
		})
	})
})
