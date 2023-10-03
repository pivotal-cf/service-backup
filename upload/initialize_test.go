// Copyright (C) 2016-Present Pivotal Software, Inc. All rights reserved.
// This program and the accompanying materials are made available under the terms of the under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package upload_test

import (
	"errors"

	"code.cloudfoundry.org/lager/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/service-backup/azure"
	"github.com/pivotal-cf/service-backup/config"
	"github.com/pivotal-cf/service-backup/gcs"
	"github.com/pivotal-cf/service-backup/s3"
	"github.com/pivotal-cf/service-backup/scp"
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
		logger = lager.NewLogger("parser")
		logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))
	})

	Context("when generating an S3 uploader", func() {
		It("returns a list of 1 backuper", func() {
			expectedCACert := "/path/to/my/cert"
			fakeCACertLocator := func() (string, error) {
				return expectedCACert, nil
			}
			backupConfig := backupConfig("s3")
			factory.S3Returns(s3.New("s3", "", "", "", "", "", "", nil))

			uploader, err := upload.Initialize(backupConfig, logger, upload.WithUploaderFactory(factory), upload.WithCACertLocator(fakeCACertLocator))

			Expect(err).NotTo(HaveOccurred())
			Expect(uploader).NotTo(BeNil())
			Expect(uploader.Name()).To(Equal("multi-uploader: s3"))
			Expect(factory.S3CallCount()).To(Equal(1))
			dest, caCert := factory.S3ArgsForCall(0)
			Expect(dest).To(Equal(backupConfig.Destinations[0]))
			Expect(caCert).To(Equal(expectedCACert))
		})

		Context("when the ca cert lookup fails", func() {
			It("returns an error", func() {
				backupConfig := backupConfig("s3")
				expectedErr := errors.New("failed")
				fakeCACertLocator := func() (string, error) {
					return "", expectedErr
				}

				_, err := upload.Initialize(backupConfig, logger, upload.WithCACertLocator(fakeCACertLocator))

				Expect(err).To(Equal(expectedErr))
			})
		})
	})

	Context("when generating an SCP uploader", func() {
		It("returns a list of 1 backuper", func() {
			backupConfig := backupConfig("scp")
			factory.SCPReturns(scp.New("scp", "", 0, "", "", "", nil))

			uploader, err := upload.Initialize(backupConfig, logger, upload.WithUploaderFactory(factory), upload.WithCACertLocator(noopCACertLocator))

			Expect(err).NotTo(HaveOccurred())
			Expect(uploader).NotTo(BeNil())
			Expect(uploader.Name()).To(Equal("multi-uploader: scp"))
			Expect(factory.SCPCallCount()).To(Equal(1))
			Expect(factory.SCPArgsForCall(0)).To(Equal(backupConfig.Destinations[0]))
		})
	})

	Context("when generating an Azure uploader", func() {
		It("returns a list of 1 backuper", func() {
			backupConfig := backupConfig("azure")
			factory.AzureReturns(azure.New("azure", "", "", "", "", nil))

			uploader, err := upload.Initialize(backupConfig, logger, upload.WithUploaderFactory(factory), upload.WithCACertLocator(noopCACertLocator))

			Expect(err).NotTo(HaveOccurred())
			Expect(uploader).NotTo(BeNil())
			Expect(uploader.Name()).To(Equal("multi-uploader: azure"))
			Expect(factory.AzureCallCount()).To(Equal(1))
			Expect(factory.AzureArgsForCall(0)).To(Equal(backupConfig.Destinations[0]))
		})
	})

	Context("when generating an GCS uploader", func() {
		It("returns a list of 1 backuper", func() {
			backupConfig := backupConfig("gcs")
			factory.GCSReturns(gcs.New("gcs", "", "", "", nil))

			uploader, err := upload.Initialize(backupConfig, logger, upload.WithUploaderFactory(factory), upload.WithCACertLocator(noopCACertLocator))

			Expect(err).NotTo(HaveOccurred())
			Expect(uploader).NotTo(BeNil())
			Expect(uploader.Name()).To(Equal("multi-uploader: gcs"))
			Expect(factory.GCSCallCount()).To(Equal(1))
			Expect(factory.GCSArgsForCall(0)).To(Equal(backupConfig.Destinations[0]))
		})
	})

	Context("when an unknown destination type is configured", func() {
		It("returns an error", func() {
			backupConfig := backupConfig("unknown-type")

			_, err := upload.Initialize(backupConfig, logger)

			Expect(err).To(MatchError("unknown destination type: unknown-type"))
		})
	})
})

func backupConfig(destType string) *config.BackupConfig {
	dest := config.Destination{
		Type:   destType,
		Config: map[string]interface{}{},
	}
	backupConfig := &config.BackupConfig{
		Destinations: []config.Destination{dest},
	}
	return backupConfig
}

func noopCACertLocator() (string, error) {
	return "", nil
}
