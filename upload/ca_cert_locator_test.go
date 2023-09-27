// Copyright (C) 2016-Present Pivotal Software, Inc. All rights reserved.
// This program and the accompanying materials are made available under the terms of the under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package upload

import (
	"errors"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/ginkgo/v2/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

var _ = Describe("Locator", func() {
	DescribeTable("locates certificates",
		func(caCertOnDisk string, errMatcher types.GomegaMatcher) {
			statCallCount := 0
			stat = func(name string) (os.FileInfo, error) {
				statCallCount += 1

				if caCertOnDisk == name {
					return nil, nil
				}

				return nil, &os.PathError{"stat", name, errors.New("file not found")}
			}

			found, err := CACertPath()

			Expect(found).To(Equal(caCertOnDisk))
			Expect(err).To(errMatcher)
			Expect(statCallCount).To(BeNumerically(">", 0))
		},
		Entry("Debian/Ubuntu/Gentoo etc.", "/etc/ssl/certs/ca-certificates.crt", BeNil()),
		Entry("Fedora/RHEL", "/etc/pki/tls/certs/ca-bundle.crt", BeNil()),
		Entry("OpenSUSE", "/etc/ssl/ca-bundle.pem", BeNil()),
		Entry("OpenELEC", "/etc/pki/tls/cacert.pem", BeNil()),
		Entry("Non-linux OS", "", MatchError("could not locate a known ca cert")),
	)
})
