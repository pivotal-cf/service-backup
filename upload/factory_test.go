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
			s3Cli := factory.S3(config.Destination{}, "")

			Expect(s3Cli).ToNot(BeNil())
		})
	})

	Describe("SCP", func() {
		It("returns an SCPClient", func() {
			factory := &uploaderFactory{&config.BackupConfig{}}
			scpClient := factory.SCP(config.Destination{})

			Expect(scpClient).ToNot(BeNil())
		})
	})

	Describe("Azure", func() {
		It("returns an Azure client", func() {
			factory := &uploaderFactory{&config.BackupConfig{}}
			azureClient := factory.Azure(config.Destination{})

			Expect(azureClient).ToNot(BeNil())
		})
	})

	Describe("GCP", func() {
		It("returns a GCP client", func() {
			factory := &uploaderFactory{&config.BackupConfig{}}
			azureClient := factory.GCP(config.Destination{})

			Expect(azureClient).ToNot(BeNil())
		})
	})
})
