package config_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/service-backup/azure"
	"github.com/pivotal-cf/service-backup/config"
	"github.com/pivotal-cf/service-backup/config/configfakes"
	"github.com/pivotal-cf/service-backup/gcp"
	"github.com/pivotal-cf/service-backup/s3"
	"github.com/pivotal-cf/service-backup/scp"
)

var _ = Describe("BackuperCreator", func() {
	Describe("S3", func() {
		It("returns an S3CliClient", func() {
			systemTrustStoreLocator := new(configfakes.FakeSystemTrustStoreLocator)
			destinationFactory := config.NewBackuperCreator(&config.BackupConfig{})

			s3Cli, err := destinationFactory.S3(config.Destination{}, systemTrustStoreLocator)

			Expect(err).NotTo(HaveOccurred())
			Expect(s3Cli).To(BeAssignableToTypeOf(&s3.S3CliClient{}))
		})

		It("returns an error when unable to locate system trust store", func() {
			systemTrustStoreLocator := new(configfakes.FakeSystemTrustStoreLocator)
			systemTrustStoreLocator.PathReturns("", errors.New("failed to locate system trust store"))
			destinationFactory := config.NewBackuperCreator(&config.BackupConfig{})

			_, err := destinationFactory.S3(config.Destination{}, systemTrustStoreLocator)

			Expect(err).To(MatchError("failed to locate system trust store"))
		})
	})

	Describe("SCP", func() {
		It("returns an SCPClient", func() {
			destinationFactory := config.NewBackuperCreator(&config.BackupConfig{})

			scpClient := destinationFactory.SCP(config.Destination{})

			Expect(scpClient).To(BeAssignableToTypeOf(&scp.SCPClient{}))
		})
	})

	Describe("Azure", func() {
		It("returns an Azure client", func() {
			destinationFactory := config.NewBackuperCreator(&config.BackupConfig{})

			azureClient := destinationFactory.Azure(config.Destination{})

			Expect(azureClient).To(BeAssignableToTypeOf(&azure.AzureClient{}))
		})
	})

	Describe("GCP", func() {
		It("returns a GCP client", func() {
			destinationFactory := config.NewBackuperCreator(&config.BackupConfig{})

			azureClient := destinationFactory.GCP(config.Destination{})

			Expect(azureClient).To(BeAssignableToTypeOf(&gcp.StorageClient{}))
		})
	})
})
