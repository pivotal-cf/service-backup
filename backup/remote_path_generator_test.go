package backup_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf-experimental/service-backup/backup"
)

var _ = Describe("RemotePathGenerator", func() {
	It("generate a remote path from a base path", func() {
		basePath := "base/path"
		today := time.Now()
		datePath := fmt.Sprintf("%d/%02d/%02d", today.Year(), today.Month(), today.Day())
		generator := backup.RemotePathGenerator{}

		remotePath := generator.RemotePathWithDate(basePath)

		Expect(remotePath).To(Equal(basePath + "/" + datePath))
	})
})
