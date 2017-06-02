// Copyright (C) 2016-Present Pivotal Software, Inc. All rights reserved.
// This program and the accompanying materials are made available under the terms of the under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package executor_test

import (
	"github.com/pivotal-cf/service-backup/executor"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/onsi/gomega/gbytes"

	"code.cloudfoundry.org/lager"
)

var _ = Describe("DummyExecutor", func() {
	var (
		exec   executor.Executor
		logger lager.Logger
		log    *gbytes.Buffer
	)

	BeforeEach(func() {
		logger = lager.NewLogger("ServiceBackup")
		log = gbytes.NewBuffer()
		logger.RegisterSink(lager.NewWriterSink(log, lager.INFO))
		exec = executor.NewDummyExecutor(logger)
	})

	Describe("Execute", func() {
		It("Doesn't return an error", func() {
			err := exec.Execute()
			Expect(err).To(BeNil())
		})

		It("Logs that backups are disabled", func() {
			exec.Execute()
			Expect(log).To(gbytes.Say("Backups Disabled"))
		})
	})
})
