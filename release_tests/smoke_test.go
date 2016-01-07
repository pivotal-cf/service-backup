package release_tests_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

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
		boshManifest       string

		client *s3testclient.S3TestClient
	)

	BeforeEach(func() {
		boshHost = envMustHave("BOSH_HOST")
		boshPrivateKeyFile = envMustHave("BOSH_PRIVATE_KEY_FILE")
		boshManifest = envMustHave("BOSH_MANIFEST")
		boshUsername = envMustHave("BOSH_USERNAME")
		boshPassword = envMustHave("BOSH_PASSWORD")
		awsAccessKeyID := envMustHave("AWS_ACCESS_KEY_ID")
		awsSecretKey := envMustHave("AWS_SECRET_ACCESS_KEY")

		client = s3testclient.New("https://s3-eu-west-1.amazonaws.com", awsAccessKeyID, awsSecretKey)
	})

	AfterEach(func() {
		Expect(client.DeleteRemotePath(bucketName, testPath)).To(Succeed())
	})

	It("Uploads files in the backup directory to S3", func() {
		toBackup := "to_backup.txt"
		cwd, err := os.Getwd()
		Expect(err).NotTo(HaveOccurred())
		pathToFile := filepath.Join(cwd, "test_assets", toBackup)
		cmd := exec.Command("bosh", "-n", "-d", boshManifest, "-t", fmt.Sprintf("https://%s:25555", boshHost), "-u", boshUsername, "-p", boshPassword, "scp",
			"--gateway_host", boshHost, "--gateway_user", "vcap", "--gateway_identity_file", boshPrivateKeyFile, "--upload",
			"service-backup", "0", pathToFile, "/tmp")
		cmd.Stdout = GinkgoWriter
		cmd.Stderr = GinkgoWriter
		Expect(cmd.Run()).To(Succeed())

		Eventually(func() bool {
			return client.RemotePathExistsInBucket(bucketName, fmt.Sprintf("%s/%s", pathWithDate(testPath), toBackup))
		}, time.Minute).Should(BeTrue())
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
