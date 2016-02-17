package azureintegration_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/Azure/azure-sdk-for-go/storage"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var (
	azureContainer = "ci-blobs"
)

func runBackup(params ...string) *gexec.Session {
	backupCmd := exec.Command(pathToServiceBackupBinary, params...)
	session, err := gexec.Start(backupCmd, GinkgoWriter, GinkgoWriter)
	Expect(err).ToNot(HaveOccurred())
	return session
}

func performBackup(sourceFolder, destinationPath string) *gexec.Session {
	return runBackup(
		"--source-folder", sourceFolder,
		"--dest-path", destinationPath,
		"--azure-storage-access-key", azureAccountKey,
		"--azure-storage-account", azureAccountName,
		"--azure-container", azureContainer,
		"--cron-schedule", "*/5 * * * * *", // every 5 seconds
		"--backup-creator-cmd", "true",
		"--cleanup-cmd", "true",
	)
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

		It("uploads the backup", func() {
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
			Eventually(session.Out, azureTimeout).Should(gbytes.Say("Cleanup completed without error"))
			session.Terminate().Wait()
			Eventually(session).Should(gexec.Exit())

			azureClient, err := storage.NewBasicClient(azureAccountName, azureAccountKey)
			Expect(err).ToNot(HaveOccurred())
			azureBlobService := azureClient.GetBlobService()

			firstBackupBlobPath := fmt.Sprintf("%s/%d/%02d/%02d/%s", destinationPath, today.Year(), int(today.Month()), today.Day(), firstBackupFileName)
			Expect(downloadBlob(azureBlobService, firstBackupBlobPath)).To(Equal([]byte(firstBackupFileContent)))

			secondBackupBlobPath := fmt.Sprintf("%s/%d/%02d/%02d/%s", destinationPath, today.Year(), int(today.Month()), today.Day(), secondBackupFileName)
			Expect(downloadBlob(azureBlobService, secondBackupBlobPath)).To(Equal([]byte(secondBackupFileContent)))
		})
	})

	Context("when the client is wrongly configured", func() {
		It("exits with non-zero", func() {
			session := runBackup(
				"--source-folder", "does/not/matter",
				"--dest-path", "does/not/matter_either",
				"--azure-storage-access-key", azureAccountKey,
				"--azure-storage-account", azureAccountName,
				// --azure-container
				"--cron-schedule", "*/5 * * * * *", // every 5 seconds
				"--backup-creator-cmd", "true",
				"--cleanup-cmd", "true",
			)

			Expect(session.Wait(time.Second).ExitCode()).ToNot(Equal(0))
			Expect(string(session.Out.Contents())).To(ContainSubstring("Flag azure-container not provided"))
		})
	})

})
