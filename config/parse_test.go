package config_test

import (
	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf-experimental/service-backup/config"
	alerts "github.com/pivotal-cf/service-alerts-client/client"
)

var _ = Describe("Parse", func() {
	var logger lager.Logger

	BeforeEach(func() {
		logger = lager.NewLogger("parser")
		logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))
	})

	Context("with all optional fields", func() {
		It("returns a backup config", func() {
			backupConfig, err := config.Parse("fixtures/valid_config_with_optional_fields.yml", logger)
			Expect(err).NotTo(HaveOccurred())

			Expect(backupConfig.Destinations).To(Equal([]config.Destination{
				{
					Type: "s3",
					Name: "s3_destination",
					Config: map[string]interface{}{
						"endpoint_url":      "www.s3.com",
						"bucket_name":       "a_bucket",
						"bucket_path":       "a_bucket_path",
						"access_key_id":     "AKAIADCIWI@ICFIJ",
						"secret_access_key": "ASCDMIACDNI@UD937e9237aSCDAS",
					},
				},
			}))
			Expect(backupConfig.SourceFolder).To(Equal("."))
			Expect(backupConfig.SourceExecutable).To(Equal("ls"))
			Expect(backupConfig.CronSchedule).To(Equal("*/5 * * * * *"))
			Expect(backupConfig.CleanupExecutable).To(Equal("ls"))
			Expect(backupConfig.MissingPropertiesMessage).To(Equal("custom message"))
			Expect(backupConfig.ExitIfInProgress).To(BeTrue())
			Expect(backupConfig.ServiceIdentifierExecutable).To(Equal("whoami"))
			Expect(backupConfig.AwsCliPath).To(Equal("path/to/aws_cli"))
			Expect(backupConfig.AzureCliPath).To(Equal("path/to/azure_cli"))
			Expect(backupConfig.Alerts).To(Equal(&config.Alerts{
				ProductName: "MySQL",
				Config: alerts.Config{
					CloudController: alerts.CloudController{
						URL:      "https://api.cf.com",
						User:     "admin",
						Password: "password",
					},
					NotificationTarget: alerts.NotificationTarget{
						URL:               "https://notifications.cf.com",
						SkipSSLValidation: boolPointer(true),
						CFOrg:             "system",
						CFSpace:           "mysql-notifications",
						ReplyTo:           "me@example.com",
						Authentication: alerts.Authentication{
							UAA: alerts.UAA{
								URL:          "https://uaa.cf.com",
								ClientID:     "admin",
								ClientSecret: "password",
							},
						},
					},
					GlobalTimeoutSeconds: 42,
				},
			}))
		})
	})

	Context("with only mandatory fields", func() {
		It("returns a backup config", func() {
			backupConfig, err := config.Parse("fixtures/valid_minimal_config.yml", logger)
			Expect(err).NotTo(HaveOccurred())

			Expect(backupConfig.Destinations).To(Equal([]config.Destination{
				{
					Type: "s3",
					Name: "",
					Config: map[string]interface{}{
						"endpoint_url":      "www.s3.com",
						"bucket_name":       "a_bucket",
						"bucket_path":       "a_bucket_path",
						"access_key_id":     "AKAIADCIWI@ICFIJ",
						"secret_access_key": "ASCDMIACDNI@UD937e9237aSCDAS",
					},
				},
			}))
			Expect(backupConfig.SourceFolder).To(Equal("."))
			Expect(backupConfig.SourceExecutable).To(Equal(""))
			Expect(backupConfig.CronSchedule).To(Equal("*/5 * * * * *"))
			Expect(backupConfig.CleanupExecutable).To(Equal(""))
			Expect(backupConfig.MissingPropertiesMessage).To(Equal(""))
			Expect(backupConfig.ExitIfInProgress).To(BeFalse())
			Expect(backupConfig.ServiceIdentifierExecutable).To(Equal(""))
			Expect(backupConfig.AwsCliPath).To(Equal("path/to/aws_cli"))
			Expect(backupConfig.AzureCliPath).To(Equal("path/to/azure_cli"))
			Expect(backupConfig.Alerts).To(BeNil())
		})
	})

	Context("with an invalid config path", func() {
		It("returns an error", func() {
			_, err := config.Parse("238d4y238(^&*($(@)))", logger)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("with an invalid config file", func() {
		It("returns an error", func() {
			_, err := config.Parse("fixtures/invalid_config.yml", logger)
			Expect(err).To(HaveOccurred())
		})
	})
})

func boolPointer(b bool) *bool {
	return &b
}
