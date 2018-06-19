// Copyright (C) 2016-Present Pivotal Software, Inc. All rights reserved.
// This program and the accompanying materials are made available under the terms of the under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package executor

import (
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/satori/go.uuid"
)

var _ = Describe("SizeCalculator", func() {
	var path = "fakepath"

	BeforeEach(func() {
		var err error
		path, err = ioutil.TempDir("", "")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		os.Remove(path)
	})

	Describe("DirSize", func() {
		Context("when the directory is empty", func() {
			It("returns 0", func() {
				size, err := calculateDirSize(path)
				Expect(size).To(Equal(int64(0)))
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when the directory is not empty", func() {
			var fileSize int64

			BeforeEach(func() {
				fileSize = createFilesIn(path)
			})

			It("returns the sum of the files sizes", func() {
				size, err := calculateDirSize(path)
				Expect(size).To(Equal(fileSize))
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when the directory contains subdirectories", func() {
			var subdirPath string

			BeforeEach(func() {
				subdirPath = createEmptySubdirectory(path)
			})

			Context("when there are no files", func() {
				It("returns 0", func() {
					size, err := calculateDirSize(path)
					Expect(size).To(Equal(int64(0)))
					Expect(err).ToNot(HaveOccurred())
				})
			})

			Context("when there are files", func() {
				var fileSize int64

				BeforeEach(func() {
					fileSize = createFilesIn(path) + createFilesIn(subdirPath)
				})

				It("returns the sum of the files sizes", func() {
					size, err := calculateDirSize(path)
					Expect(size).To(Equal(fileSize))
					Expect(err).ToNot(HaveOccurred())
				})
			})
		})

		Context("when an invalid path is provided", func() {
			It("returns an error", func() {
				_, err := calculateDirSize("fake-path")
				Expect(err).To(HaveOccurred())
			})
		})
	})
})

func createFilesIn(path string) int64 {
	size := int64(0)

	for i := 0; i < 5; i++ {
		file, err := ioutil.TempFile(path, "executor")
		Expect(err).ToNot(HaveOccurred())

		fileContentsUUID := uuid.NewV4()

		fileContents := fileContentsUUID.String()
		_, err = file.Write([]byte(fileContents))
		Expect(err).ToNot(HaveOccurred())

		fileStat, err := file.Stat()
		Expect(err).ToNot(HaveOccurred())

		size += fileStat.Size()
	}
	return size
}

func createEmptySubdirectory(path string) string {
	subdirPath := filepath.Join(path, "dir1")
	err := os.Mkdir(subdirPath, 0777)
	Expect(err).ToNot(HaveOccurred())
	return subdirPath
}
