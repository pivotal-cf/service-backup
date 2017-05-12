package upload

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/service-backup/config"
)

var _ = Describe("backupFactory", func() {
	Describe("S3", func() {
		It("returns an S3CliClient", func() {
			factory := &uploaderFactory{&config.BackupConfig{}}

			client := factory.S3(config.Destination{}, "")

			Expect(client).ToNot(BeNil())
		})
	})

	Describe("SCP", func() {
		It("returns an SCPClient", func() {
			factory := &uploaderFactory{&config.BackupConfig{}}

			client := factory.SCP(config.Destination{})

			Expect(client).ToNot(BeNil())
		})
	})

	Describe("Azure", func() {
		It("returns an Azure client", func() {
			factory := &uploaderFactory{&config.BackupConfig{}}

			client := factory.Azure(config.Destination{})

			Expect(client).ToNot(BeNil())
		})
	})

	Describe("GCS", func() {
		It("returns a GCS client", func() {
			factory := &uploaderFactory{&config.BackupConfig{}}

			client := factory.GCS(config.Destination{})

			Expect(client).ToNot(BeNil())
		})
	})
})
