// Copyright (C) 2016-Present Pivotal Software, Inc. All rights reserved.
// This program and the accompanying materials are made available under the terms of the under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package s3_test

import (
	"fmt"
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/service-backup/process"
	"github.com/pivotal-cf/service-backup/s3"
	"github.com/pivotal-cf/service-backup/upload"
)

var _ = Describe("S3", func() {
	Describe("default arguments", func() {
		var (
			lsCmd                                                                       *exec.Cmd
			awsCmdPath, endpointURL, region, accessKey, secretKey, systemTrustStorePath string
			processManager                                                              process.ProcessManager
		)

		BeforeEach(func() {
			awsCmdPath = "path/to/aws-cli"
			endpointURL = "http://example.com"
			region = "aws-region"
			accessKey = "access-key"
			secretKey = "secret-key"
			systemTrustStorePath = "path/to/system/trust/store"
			processManager = process.NewManager()
		})

		JustBeforeEach(func() {
			s3CLIClient := s3.New("destination-name", awsCmdPath, endpointURL, region, accessKey, secretKey, systemTrustStorePath, upload.RemotePathFunc("base-path", ""))
			lsCmd = s3CLIClient.S3Cmd("ls", "bucket-name")
		})

		It("builds an S3 command with default arguments", func() {
			Expect(lsCmd.Args).To(Equal([]string{
				awsCmdPath,
				"--endpoint-url",
				endpointURL,
				"--region",
				region,
				"--ca-bundle",
				systemTrustStorePath,
				"s3",
				"ls",
				"bucket-name",
			}))
		})

		Context("when endpoint URL is empty", func() {
			BeforeEach(func() {
				endpointURL = ""
			})

			It("builds an S3 command without specifying endpoint url", func() {
				Expect(lsCmd.Args).To(Equal([]string{
					awsCmdPath,
					"--region",
					region,
					"--ca-bundle",
					systemTrustStorePath,
					"s3",
					"ls",
					"bucket-name",
				}))
			})
		})

		Context("when region is empty", func() {
			BeforeEach(func() {
				region = ""
			})

			It("builds an S3 command without specifying region", func() {
				Expect(lsCmd.Args).To(Equal([]string{
					awsCmdPath,
					"--endpoint-url",
					endpointURL,
					"--ca-bundle",
					systemTrustStorePath,
					"s3",
					"ls",
					"bucket-name",
				}))
			})
		})

		It("adds the AWS credentials to the S3 command's env", func() {
			Expect(lsCmd.Env).To(ContainElement(fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", accessKey)))
			Expect(lsCmd.Env).To(ContainElement(fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", secretKey)))
		})
	})
})
