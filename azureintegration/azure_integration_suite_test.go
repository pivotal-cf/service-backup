// Copyright (C) 2016-Present Pivotal Software, Inc. All rights reserved.
// This program and the accompanying materials are made available under the terms of the under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package azureintegration_test

import (
	"encoding/json"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

const (
	azureTimeout           = "3m"
	azureAccountNameEnvKey = "AZURE_STORAGE_ACCOUNT"
	azureAccountKeyEnvKey  = "AZURE_STORAGE_ACCESS_KEY"
	azureCmd               = ""
)

var (
	azureAccountName = os.Getenv(azureAccountNameEnvKey)
	azureAccountKey  = os.Getenv(azureAccountKeyEnvKey)
)

func TestAzureIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AzureIntegration Suite")
}

type TestData struct {
	PathToServiceBackupBinary string
}

var (
	pathToServiceBackupBinary string
)

var _ = SynchronizedBeforeSuite(func() []byte {
	var err error

	pathToServiceBackupBinary, err = gexec.Build("github.com/pivotal-cf/service-backup")
	Expect(err).ToNot(HaveOccurred())

	forOtherNodes, err := json.Marshal(TestData{
		PathToServiceBackupBinary: pathToServiceBackupBinary,
	})
	Expect(err).ToNot(HaveOccurred())
	return forOtherNodes
}, func(data []byte) {
	var t TestData
	Expect(json.Unmarshal(data, &t)).To(Succeed())

	pathToServiceBackupBinary = t.PathToServiceBackupBinary
})

var _ = SynchronizedAfterSuite(func() {
}, func() {
	gexec.CleanupBuildArtifacts()
})
