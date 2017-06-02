// Copyright (C) 2016-Present Pivotal Software, Inc. All rights reserved.
// This program and the accompanying materials are made available under the terms of the under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package config_test

import (
	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	alerts "github.com/pivotal-cf/service-alerts-client/client"
	"github.com/pivotal-cf/service-backup/config"
)

var _ = Describe("Parse", func() {
	var logger lager.Logger

	BeforeEach(func() {
		logger = lager.NewLogger("parser")
		logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))
	})

	Context("when the destination is GCS", func() {
		Context("with required fields", func() {
			It("returns a backup config, without alerts", func() {
				backupConfig, err := config.Parse("fixtures/valid_gcs_config_with_required_fields.yml", logger)
				Expect(err).NotTo(HaveOccurred())

				Expect(backupConfig.Destinations).To(Equal([]config.Destination{
					{
						Type: "gcs",
						Name: "google_cloud_destination",
						Config: map[string]interface{}{
							"project_id":           "my_google_project",
							"bucket_name":          "my_google_bucket",
							"service_account_json": "{\"key\":\"value\"}\n",
						},
					},
				}))
				Expect(backupConfig.SourceFolder).To(Equal("."))
				Expect(backupConfig.SourceExecutable).To(Equal("ls"))
				Expect(backupConfig.CronSchedule).To(Equal("*/5 * * * * *"))
				Expect(backupConfig.MissingPropertiesMessage).To(Equal("custom message"))
				Expect(backupConfig.ExitIfInProgress).To(BeTrue())
				Expect(backupConfig.ServiceIdentifierExecutable).To(Equal("whoami"))
				Expect(backupConfig.AwsCliPath).To(Equal("path/to/aws_cli"))
				Expect(backupConfig.AzureCliPath).To(Equal("path/to/azure_cli"))
			})
		})
	})

	Context("when the destination is s3", func() {
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
						Notifications: alerts.Notifications{
							ServiceURL:   "https://notifications.cf.com",
							CFOrg:        "system",
							CFSpace:      "mysql-notifications",
							ReplyTo:      "me@example.com",
							ClientID:     "admin",
							ClientSecret: "password",
						},
						GlobalTimeoutSeconds: 42,
						SkipSSLValidation:    boolPointer(true),
					},
				}))
				Expect(backupConfig.DeploymentName).To(Equal("deployment-name"))
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

	Context("when add_deployment_name_to_path is configured", func() {
		Context("to false", func() {
			It("unsets the deployment name", func() {
				backupConfig, err := config.Parse("fixtures/valid_with_add_deployment_name_to_path_false.yml", logger)
				Expect(err).NotTo(HaveOccurred())

				Expect(backupConfig.DeploymentName).To(Equal(""))
			})
		})

		Context("to true", func() {
			It("deployment name is still present", func() {
				backupConfig, err := config.Parse("fixtures/valid_config_with_optional_fields.yml", logger)
				Expect(err).NotTo(HaveOccurred())

				Expect(backupConfig.DeploymentName).To(Equal("deployment-name"))
			})
		})
	})
})

func boolPointer(b bool) *bool {
	return &b
}
