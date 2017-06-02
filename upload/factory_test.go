// Copyright (C) 2016-Present Pivotal Software, Inc. All rights reserved.
// This program and the accompanying materials are made available under the terms of the under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

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
