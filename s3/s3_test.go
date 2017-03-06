package s3_test

import (
	"fmt"
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/service-backup/s3"
)

var _ = Describe("S3", func() {
	Describe("default arguments", func() {
		var (
			lsCmd                                                               *exec.Cmd
			awsCmdPath, endpointURL, accessKey, secretKey, systemTrustStorePath string
		)

		BeforeEach(func() {
			awsCmdPath = "path/to/aws-cli"
			endpointURL = "http://example.com"
			accessKey = "access-key"
			secretKey = "secret-key"
			systemTrustStorePath = "path/to/system/trust/store"

			s3CLIClient := s3.New("destination-name", awsCmdPath, endpointURL, accessKey, secretKey, "base-path", systemTrustStorePath)
			lsCmd = s3CLIClient.S3Cmd("ls", "bucket-name")
		})

		It("builds an S3 command with default arguments", func() {
			Expect(lsCmd.Args).To(Equal([]string{
				awsCmdPath,
				"--endpoint-url",
				endpointURL,
				"--ca-bundle",
				systemTrustStorePath,
				"s3",
				"ls",
				"bucket-name",
			}))
		})

		It("adds the AWS credentials to the S3 command's env", func() {
			Expect(lsCmd.Env).To(ContainElement(fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", accessKey)))
			Expect(lsCmd.Env).To(ContainElement(fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", secretKey)))
		})
	})
})
