package config_test

import (
	"errors"

	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/service-backup/config"
	"github.com/pivotal-cf/service-backup/config/configfakes"
)

var _ = Describe("ParseDestinations", func() {
	var (
		destinationFactory *configfakes.FakeBackuperFactory
		logger             lager.Logger
	)

	BeforeEach(func() {
		destinationFactory = new(configfakes.FakeBackuperFactory)
		logger = lager.NewLogger("parser")
		logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))
	})

	Context("when S3 is configured", func() {
		It("returns a list of 1 backuper", func() {
			expectedDestination := config.Destination{
				Type: "s3",
				Config: map[string]interface{}{
					"bucket_name":       "some-bucket",
					"bucket_path":       "some-bucket-path",
					"endpoint_url":      "some-endpoint-url",
					"region":            "a-region",
					"access_key_id":     "some-access-key-id",
					"secret_access_key": "some-secret-access-key",
				},
			}

			backupConfig := config.BackupConfig{
				Destinations: []config.Destination{expectedDestination},
			}
			backupers, err := config.ParseDestinations(
				backupConfig, destinationFactory, logger,
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(len(backupers)).To(Equal(1))
			destination, systemTrustStoreLocation := destinationFactory.S3ArgsForCall(0)
			Expect(destination).To(Equal(expectedDestination))
			Expect(systemTrustStoreLocation).NotTo(BeNil())
		})

		Context("when the system trust store cannot be located", func() {
			BeforeEach(func() {
				destinationFactory.S3Returns(nil, errors.New("could not locate system trust store"))
			})

			It("returns an error", func() {
				backupConfig := config.BackupConfig{
					Destinations: []config.Destination{
						{
							Type: "s3",
							Config: map[string]interface{}{
								"bucket_name":       "some-bucket",
								"bucket_path":       "some-bucket-path",
								"endpoint_url":      "some-endpoint-url",
								"access_key_id":     "some-access-key-id",
								"secret_access_key": "some-secret-access-key",
							},
						},
					},
				}
				_, err := config.ParseDestinations(backupConfig, destinationFactory, logger)
				Expect(err).To(MatchError("could not locate system trust store"))
			})
		})
	})

	Context("when an unknown destination type is configured", func() {
		It("returns an error", func() {
			backupConfig := config.BackupConfig{
				Destinations: []config.Destination{
					{Type: "unknown-type"},
				},
			}
			_, err := config.ParseDestinations(backupConfig, destinationFactory, logger)
			Expect(err).To(MatchError("unknown destination type: unknown-type"))
		})
	})
})
