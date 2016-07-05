package backup_test

import (
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/pivotal-cf-experimental/service-backup/backup"
	"github.com/satori/go.uuid"
)

func createFilesIn(path string) int64 {
	size := int64(0)

	for i := 0; i < 5; i++ {
		file, err := ioutil.TempFile(path, "")
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

var _ = Describe("SizeCalculator", func() {
	var calculator = &FileSystemSizeCalculator{}
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
				size, err := calculator.DirSize(path)
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
				size, err := calculator.DirSize(path)
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
					size, err := calculator.DirSize(path)
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
					size, err := calculator.DirSize(path)
					Expect(size).To(Equal(fileSize))
					Expect(err).ToNot(HaveOccurred())
				})
			})
		})
		Context("when an invalid path is provided", func() {
			It("returns an error", func() {
				_, err := calculator.DirSize("fake-path")
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
