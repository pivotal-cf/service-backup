// Copyright (C) 2016-Present Pivotal Software, Inc. All rights reserved.
// This program and the accompanying materials are made available under the terms of the under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package upload

import (
	"errors"

	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/service-backup/process"
)

var _ = Describe("multiUploader", func() {
	Describe("Upload", func() {
		var (
			uploaderA *fakeUploader
			uploaderB *fakeUploader
			uploader  *multiUploader

			processManager process.ProcessManager

			localPath = "local/path"
			logger    = lager.NewLogger("multi-logger")
		)

		BeforeEach(func() {
			uploaderA = new(fakeUploader)
			uploaderB = new(fakeUploader)
			uploader = &multiUploader{[]Uploader{uploaderA, uploaderB}}
			processManager = process.NewManager()
		})

		Context("when all uploads succeed", func() {
			It("calls upload on each uploader", func() {
				err := uploader.Upload(localPath, logger, processManager)

				Expect(err).NotTo(HaveOccurred())

				Expect(len(uploaderA.uploadArgs)).To(Equal(1))
				Expect(uploaderA.uploadArgs[0]).To(Equal(struct {
					string
					lager.Logger
				}{localPath, logger}))

				Expect(len(uploaderB.uploadArgs)).To(Equal(1))
				Expect(uploaderB.uploadArgs[0]).To(Equal(struct {
					string
					lager.Logger
				}{localPath, logger}))
			})
		})

		Context("when the first uploader fails", func() {
			BeforeEach(func() {
				uploaderA.uploadErr = errors.New("first backup failed")
			})

			It("returns the error from the first uploader", func() {
				err := uploader.Upload(localPath, logger, processManager)
				Expect(err).To(MatchError(ContainSubstring("first backup failed")))
			})

			It("calls upload on all the uploaders", func() {
				uploader.Upload(localPath, logger, processManager)
				Expect(len(uploaderA.uploadArgs)).To(Equal(1))
				Expect(len(uploaderB.uploadArgs)).To(Equal(1))
			})
		})

		Context("when both uploaders fail", func() {
			BeforeEach(func() {
				uploaderA.uploadErr = errors.New("first backup failed")
				uploaderB.uploadErr = errors.New("second backup failed")
			})

			It("returns the errors from both uploaders", func() {
				err := uploader.Upload(localPath, logger, processManager)
				Expect(err).To(MatchError(ContainSubstring("first backup failed")))
				Expect(err).To(MatchError(ContainSubstring("second backup failed")))
			})

			It("calls upload on all the uploaders", func() {
				uploader.Upload(localPath, logger, processManager)
				Expect(len(uploaderA.uploadArgs)).To(Equal(1))
				Expect(len(uploaderB.uploadArgs)).To(Equal(1))
			})
		})
	})

	Describe("Name", func() {
		It("returns the names of uploaders it warps", func() {
			multi := &multiUploader{[]Uploader{&fakeUploader{name: "a"}, &fakeUploader{name: "b"}}}
			Expect(multi.Name()).To(Equal("multi-uploader: a, b"))
		})
	})
})

type fakeUploader struct {
	uploadErr  error
	uploadArgs []struct {
		string
		lager.Logger
	}
	name string
}

func (f *fakeUploader) Upload(name string, logger lager.Logger, _ process.ProcessManager) error {
	f.uploadArgs = append(f.uploadArgs, struct {
		string
		lager.Logger
	}{name, logger})
	return f.uploadErr
}

func (f *fakeUploader) Name() string {
	return f.name
}
