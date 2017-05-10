package config_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/service-backup/config"
)

var _ = Describe("RemotePathGenerator", func() {
	today := time.Now()
	datePath := fmt.Sprintf("%d/%02d/%02d", today.Year(), today.Month(), today.Day())

	DescribeTable("generates remote path with date",
		func(basePath, deploymentName, expectedRemotePath string) {
			generator := config.RemotePathGenerator{
				BasePath:       basePath,
				DeploymentName: deploymentName,
			}

			remotePath := generator.RemotePathWithDate()
			Expect(remotePath).To(Equal(expectedRemotePath))
		},
		Entry("neither base path nor deployment name", "", "", datePath),
		Entry("base path only", "base/path", "", "base/path/"+datePath),
		Entry("deployment name only", "", "deployment_name", "deployment_name/"+datePath),
		Entry("both base path and deployment name", "base/path", "deployment_name", "base/path/deployment_name/"+datePath),
	)
})
