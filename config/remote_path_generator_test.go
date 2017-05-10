package config_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/service-backup/config"
)

var _ = Describe("RemotePathGenerator", func() {
	It("generates a remote path with a base path", func() {
		basePath := "/base/path"
		generator := config.RemotePathGenerator{
			BasePath: basePath,
		}

		remotePath := generator.RemotePathWithDate()

		today := time.Now()
		datePath := fmt.Sprintf("%d/%02d/%02d", today.Year(), today.Month(), today.Day())
		Expect(remotePath).To(Equal(basePath + "/" + datePath))
	})

	It("generates a remote path without a base path", func() {
		generator := config.RemotePathGenerator{}

		remotePath := generator.RemotePathWithDate()

		today := time.Now()
		datePath := fmt.Sprintf("%d/%02d/%02d", today.Year(), today.Month(), today.Day())
		Expect(remotePath).To(Equal(datePath))
	})
})
