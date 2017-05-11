package upload

import (
	"errors"

	"code.cloudfoundry.org/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("multiUploader", func() {
	Describe("Upload", func() {
		var (
			uploaderA *fakeUploader
			uploaderB *fakeUploader
			uploader  *multiUploader

			localPath = "local/path"
			logger    = lager.NewLogger("multi-logger")
		)

		BeforeEach(func() {
			uploaderA = new(fakeUploader)
			uploaderB = new(fakeUploader)
			uploader = &multiUploader{[]Uploader{uploaderA, uploaderB}}
		})

		Context("when all uploads succeed", func() {
			It("calls upload on each uploader", func() {
				err := uploader.Upload(localPath, logger)

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
				uploaderA.uploadErr = errors.New("first backup failed because reasons")
			})

			It("returns the error from the first uploader", func() {
				err := uploader.Upload(localPath, logger)
				Expect(err).To(MatchError(ContainSubstring("first backup failed because reasons")))
			})

			It("calls upload on all the uploaders", func() {
				uploader.Upload(localPath, logger)
				Expect(len(uploaderA.uploadArgs)).To(Equal(1))
				Expect(len(uploaderB.uploadArgs)).To(Equal(1))
			})
		})

		Context("when both uploaders fail", func() {
			BeforeEach(func() {
				uploaderA.uploadErr = errors.New("first backup failed because reasons")
				uploaderB.uploadErr = errors.New("second backup failed because reasons")
			})

			It("returns the errors from both uploaders", func() {
				err := uploader.Upload(localPath, logger)
				Expect(err).To(MatchError(ContainSubstring("first backup failed because reasons")))
				Expect(err).To(MatchError(ContainSubstring("second backup failed because reasons")))
			})

			It("calls upload on all the uploaders", func() {
				uploader.Upload(localPath, logger)
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

func (f *fakeUploader) Upload(name string, logger lager.Logger) error {
	f.uploadArgs = append(f.uploadArgs, struct {
		string
		lager.Logger
	}{name, logger})
	return f.uploadErr
}

func (f *fakeUploader) Name() string {
	return f.name
}
