package release_tests_test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/pivotal-cf-experimental/service-backup/s3testclient"

	"github.com/Azure/azure-sdk-for-go/storage"
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
		boshUsername       string
		boshPassword       string
		boshPrivateKeyFile string
		toBackup           string

		boshManifest string
	)

	BeforeEach(func() {
		boshHost = envMustHave("BOSH_HOST")
		boshUsername = envMustHave("BOSH_USERNAME")
		boshPassword = envMustHave("BOSH_PASSWORD")
		boshPrivateKeyFile = envMustHave("BOSH_PRIVATE_KEY_FILE")
		toBackup = "to_backup.txt"
	})

	boshCmdWithGateway := func(stdout io.Writer, command string, args ...string) {
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
		GinkgoWriter.Write([]byte(fmt.Sprintf("running BOSH SSH command %s\n", allArgs)))
		cmd := exec.Command("bosh", allArgs...)
		cmd.Stdout = stdout
		cmd.Stderr = GinkgoWriter
		Expect(cmd.Run()).To(Succeed())
	}

	boshSSH := func(args ...string) string {
		buf := new(bytes.Buffer)
		writer := io.MultiWriter(GinkgoWriter, buf)

		boshCmdWithGateway(writer, "ssh", append([]string{"service-backup", "0"}, args...)...)
		return buf.String()
	}

	boshSCP := func(source, destination string) {
		boshCmdWithGateway(GinkgoWriter, "scp", "--upload", "service-backup/0", source, destination)
	}

	JustBeforeEach(func() {
		cwd, err := os.Getwd()
		Expect(err).NotTo(HaveOccurred())
		pathToFile := filepath.Join(cwd, "test_assets", toBackup)
		boshSCP(pathToFile, "/tmp")
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
			boshSSH("rm", "/tmp/"+toBackup)
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
			boshSSH("rm", "/tmp/"+toBackup)

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

	Context("backing up to SCP", func() {
		BeforeEach(func() {
			boshManifest = envMustHave("SCP_BOSH_MANIFEST")

			publicKeyFile := strings.Replace(boshManifest, ".yml", ".pub", -1)
			publicKeyBytes, err := ioutil.ReadFile(publicKeyFile)
			Expect(err).NotTo(HaveOccurred())
			publicKey := strings.TrimSpace(string(publicKeyBytes))

			boshSSH(fmt.Sprintf("echo %s | sudo tee -a ~vcap/.ssh/authorized_keys", publicKey))
		})

		AfterEach(func() {
			boshSSH("rm", "/tmp/"+toBackup)
		})

		It("Uploads files in the backup directory", func() {
			Eventually(func() bool {
				return strings.Contains(boshSSH("find", "/home/vcap/backups", "'-type'", "f"), toBackup)
			}, time.Minute).Should(BeTrue())
		})
	})
})

func envMustHave(key string) string {
	value := os.Getenv(key)
	Expect(value).ToNot(BeEmpty(), key)
	return value
}

func pathWithDate(path string) string {
	today := time.Now()
	datePath := fmt.Sprintf("%d/%02d/%02d", today.Year(), today.Month(), today.Day())
	return path + "/" + datePath
}
