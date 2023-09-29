// Copyright (C) 2016-Present Pivotal Software, Inc. All rights reserved.
// This program and the accompanying materials are made available under the terms of the under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package upload

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("RemotePathFunc", func() {
	today := time.Now()
	datePath := fmt.Sprintf("%d/%02d/%02d", today.Year(), today.Month(), today.Day())

	DescribeTable("generates remote path with date",
		func(basePath, deploymentName, expectedRemotePath string) {
			remotePath := RemotePathFunc(basePath, deploymentName)
			Expect(remotePath()).To(Equal(expectedRemotePath))
		},
		Entry("neither base path nor deployment name", "", "", datePath),
		Entry("base path only", "base/path", "", "base/path/"+datePath),
		Entry("deployment name only", "", "deployment_name", "deployment_name/"+datePath),
		Entry("both base path and deployment name", "base/path", "deployment_name", "base/path/deployment_name/"+datePath),
	)
})
