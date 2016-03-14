package release_tests_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/pivotal-cf-experimental/service-backup/s3testclient"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("smoke tests", func() {
	const (
		bucketName = "service-backup-test"
		testPath   = "release-tests"
	)

	var (
		boshHost           string
		boshPrivateKeyFile string
		boshUsername       string
		boshPassword       string
		toBackup           string

		boshManifest string
	)

	BeforeEach(func() {
		boshHost = envMustHave("BOSH_HOST")
		boshPrivateKeyFile = envMustHave("BOSH_PRIVATE_KEY_FILE")
		boshUsername = envMustHave("BOSH_USERNAME")
		boshPassword = envMustHave("BOSH_PASSWORD")
		toBackup = "to_backup.txt"
	})

	boshSSH := func(command string, args ...string) {
		commonArgs := []string{
			"-n",
			"-d", boshManifest,
			"-t", fmt.Sprintf("https://%s:25555", boshHost),
			"-u", boshUsername,
			"-p", boshPassword,
			command,
			"--gateway_host", boshHost,
			"--gateway_user", "vcap",
			"--gateway_identity_file", boshPrivateKeyFile,
		}
		allArgs := append(commonArgs, args...)
		cmd := exec.Command("bosh", allArgs...)
		cmd.Stdout = GinkgoWriter
		cmd.Stderr = GinkgoWriter
		Expect(cmd.Run()).To(Succeed())
	}

	JustBeforeEach(func() {
		cwd, err := os.Getwd()
		Expect(err).NotTo(HaveOccurred())
		pathToFile := filepath.Join(cwd, "test_assets", toBackup)
		boshSSH("scp", "--upload", "service-backup/0", pathToFile, "/tmp")
	})

	Context("backing up to S3", func() {
		var (
			client *s3testclient.S3TestClient
		)

		BeforeEach(func() {
			boshManifest = envMustHave("S3_BOSH_MANIFEST")

			awsAccessKeyID := envMustHave("AWS_ACCESS_KEY_ID")
			awsSecretKey := envMustHave("AWS_SECRET_ACCESS_KEY")
			client = s3testclient.New("https://s3-eu-west-1.amazonaws.com", awsAccessKeyID, awsSecretKey)
		})

		AfterEach(func() {
			boshSSH("ssh", "service-backup", "0", "rm", "/tmp/"+toBackup)

			Expect(client.DeleteRemotePath(bucketName, testPath)).To(Succeed())
		})

		It("Uploads files in the backup directory to S3", func() {
			Eventually(func() bool {
				return client.RemotePathExistsInBucket(bucketName, fmt.Sprintf("%s/%s", pathWithDate(testPath), toBackup))
			}, time.Minute).Should(BeTrue())
		})
	})

	Context("backing up to Azure", func() {
		var (
			azureBlobService storage.BlobStorageClient
		)

		BeforeEach(func() {
			boshManifest = envMustHave("AZURE_BOSH_MANIFEST")
			azureAccountName := envMustHave("AZURE_STORAGE_ACCOUNT")
			azureAccountKey := envMustHave("AZURE_STORAGE_ACCESS_KEY")
			azureClient, err := storage.NewBasicClient(azureAccountName, azureAccountKey)
			Expect(err).ToNot(HaveOccurred())
			azureBlobService = azureClient.GetBlobService()
		})

		AfterEach(func() {
			boshSSH("ssh", "service-backup", "0", "rm", "/tmp/"+toBackup)

			_, err := azureBlobService.DeleteBlobIfExists(bucketName, fmt.Sprintf("%s/%s", pathWithDate(testPath), toBackup))
			Expect(err).NotTo(HaveOccurred())
		})

		It("Uploads files in the backup directory", func() {
			Eventually(func() bool {
				exists, err := azureBlobService.BlobExists(bucketName, fmt.Sprintf("%s/%s", pathWithDate(testPath), toBackup))
				Expect(err).NotTo(HaveOccurred())
				return exists
			}, time.Minute).Should(BeTrue())
		})
	})
})

func envMustHave(key string) string {
	value := os.Getenv(key)
	Expect(value).ToNot(BeEmpty())
	return value
}

func pathWithDate(path string) string {
	today := time.Now()
	datePath := fmt.Sprintf("%d/%02d/%02d", today.Year(), today.Month(), today.Day())
	return path + "/" + datePath
}
