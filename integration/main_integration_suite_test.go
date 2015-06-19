package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

const (
	awsAccessKeyIDEnvKey     = "AWS_ACCESS_KEY_ID"
	awsSecretAccessKeyEnvKey = "AWS_SECRET_ACCESS_KEY"

	bucketName = "service-backup-integration-test"
	awsTimeout = "10s"

	awsCLIPath  = "aws"
	endpointURL = "https://s3.amazonaws.com"
)

var (
	pathToServiceBackupBinary string
	awsAccessKeyID            string
	awsSecretAccessKey        string
)

type config struct {
	AWSAccessKeyID     string `json:"awsAccessKeyID"`
	AWSSecretAccessKey string `json:"awsSecretAccessKey"`
	PathToBackupBinary string `json:"pathToBackupBinary"`
}

func TestServiceBackupBinary(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Backup Binary Suite")
}

func beforeSuiteFirstNode() []byte {
	awsAccessKeyID = os.Getenv(awsAccessKeyIDEnvKey)
	awsSecretAccessKey = os.Getenv(awsSecretAccessKeyEnvKey)

	if awsAccessKeyID == "" || awsSecretAccessKey == "" {
		Fail(fmt.Sprintf("Specify valid AWS credentials using the env variables %s and %s", awsAccessKeyIDEnvKey, awsSecretAccessKeyEnvKey))
	}

	var err error
	pathToServiceBackupBinary, err = gexec.Build("github.com/pivotal-cf-experimental/service-backup")
	Expect(err).ToNot(HaveOccurred())

	c := config{
		AWSAccessKeyID:     awsAccessKeyID,
		AWSSecretAccessKey: awsSecretAccessKey,
		PathToBackupBinary: pathToServiceBackupBinary,
	}

	data, err := json.Marshal(c)
	Expect(err).ToNot(HaveOccurred())

	createBucketIfNeeded()

	return data
}

func createBucketIfNeeded() {
	session, err := runS3Command(
		"ls",
		bucketName,
	)

	Expect(err).ToNot(HaveOccurred())
	Eventually(session, awsTimeout).Should(gexec.Exit())

	exitCode := session.ExitCode()
	if exitCode != 0 {
		errOut := string(session.Err.Contents())

		if !strings.Contains(errOut, "NoSuchBucket") {
			Fail("Unable to list bucket: " + bucketName + " - error: " + errOut)
		}

		session, err := runS3Command(
			"mb",
			"s3://"+bucketName,
		)
		Expect(err).ToNot(HaveOccurred())
		Eventually(session, awsTimeout).Should(gexec.Exit(0))
		Eventually(session.Out).Should(gbytes.Say("make_bucket: s3://" + bucketName))
	}
}

func runS3Command(args ...string) (*gexec.Session, error) {
	env := []string{}
	env = append(env, fmt.Sprintf("%s=%s", awsAccessKeyIDEnvKey, awsAccessKeyID))
	env = append(env, fmt.Sprintf("%s=%s", awsSecretAccessKeyEnvKey, awsSecretAccessKey))

	verifyBackupCmd := exec.Command(
		awsCLIPath,
		append([]string{
			"s3",
			"--region",
			"us-east-1",
		}, args...)...,
	)
	verifyBackupCmd.Env = env

	return gexec.Start(verifyBackupCmd, GinkgoWriter, GinkgoWriter)
}

func beforeSuiteOtherNodes(b []byte) {
	var c config
	err := json.Unmarshal(b, &c)
	Expect(err).ToNot(HaveOccurred())

	awsAccessKeyID = c.AWSAccessKeyID
	awsSecretAccessKey = c.AWSSecretAccessKey
	pathToServiceBackupBinary = c.PathToBackupBinary
}

var _ = SynchronizedBeforeSuite(beforeSuiteFirstNode, beforeSuiteOtherNodes)

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})
