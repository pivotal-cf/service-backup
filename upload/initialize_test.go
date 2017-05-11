package upload_test

import (
	"errors"

	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/service-backup/config"
	"github.com/pivotal-cf/service-backup/s3"
	"github.com/pivotal-cf/service-backup/upload"
	"github.com/pivotal-cf/service-backup/upload/fakes"
)

var _ = Describe("Initialize", func() {
	var (
		factory *fakes.FakeUploaderFactory
		logger  lager.Logger
	)

	BeforeEach(func() {
		factory = new(fakes.FakeUploaderFactory)
		factory.S3Returns(s3.New("fake", "", "", "", "", "", "", nil))
		logger = lager.NewLogger("parser")
		logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))
	})

	Context("when S3 is configured", func() {
		It("returns a list of 1 backuper", func() {
			expectedCACert := "/path/to/my/cert"
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

			backupConfig := &config.BackupConfig{
				Destinations: []config.Destination{expectedDestination},
			}

			uploader, err := upload.Initialize(backupConfig, logger, upload.WithUploaderFactory(factory), upload.WithCACertLocator(func() (string, error) {
				return expectedCACert, nil
			}))

			Expect(uploader).NotTo(BeNil())
			Expect(err).NotTo(HaveOccurred())

			Expect(factory.AzureCallCount()).To(Equal(0))
			Expect(factory.SCPCallCount()).To(Equal(0))
			Expect(factory.GCPCallCount()).To(Equal(0))
			Expect(factory.S3CallCount()).To(Equal(1))

			dest, caCert := factory.S3ArgsForCall(0)
			Expect(dest).To(Equal(expectedDestination))
			Expect(caCert).To(Equal(expectedCACert))

			Expect(uploader.Name()).To(Equal("multi-uploader: fake"))
		})
	})

	Context("when the ca cert lookup fails", func() {
		It("returns an error", func() {
			expectedErr := errors.New("failed")

			conf := &config.BackupConfig{
				Destinations: []config.Destination{{Type: "s3"}},
			}

			_, err := upload.Initialize(conf, logger, upload.WithCACertLocator(func() (string, error) {
				return "", expectedErr
			}))

			Expect(err).To(Equal(expectedErr))
		})
	})

	Context("when an unknown destination type is configured", func() {
		It("returns an error", func() {
			backupConfig := &config.BackupConfig{
				Destinations: []config.Destination{
					{Type: "unknown-type"},
				},
			}
			_, err := upload.Initialize(backupConfig, logger)
			Expect(err).To(MatchError("unknown destination type: unknown-type"))
		})
	})
})
