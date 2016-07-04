package backup_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf-experimental/service-backup/backup"
	"github.com/pivotal-cf-experimental/service-backup/backup/backupfakes"
	"github.com/pivotal-golang/lager"
)

var _ = Describe("MultiBackuper", func() {
	Context("Upload", func() {
		var (
			localPath = "local/path"
			backuperA *backupfakes.FakeBackuper
			backuperB *backupfakes.FakeBackuper
			uploader  backup.Uploader
			logger    lager.Logger
			uploadErr error
		)

		BeforeEach(func() {
			backuperA = new(backupfakes.FakeBackuper)
			backuperB = new(backupfakes.FakeBackuper)
			uploader = backup.Uploader{backuperA, backuperB}
		})

		JustBeforeEach(func() {
			logger = lager.NewLogger("multi-logger")
			uploadErr = uploader.Upload(localPath, logger)
		})

		Context("when all uploads succeed", func() {
			It("calls upload on each backuper", func() {
				Expect(uploadErr).NotTo(HaveOccurred())

				Expect(backuperA.UploadCallCount()).To(Equal(1))
				actualLocalPath, loggerForA := backuperA.UploadArgsForCall(0)
				Expect(actualLocalPath).To(Equal(localPath))
				Expect(loggerForA).To(Equal(logger))

				Expect(backuperB.UploadCallCount()).To(Equal(1))
				actualLocalPath, loggerForB := backuperB.UploadArgsForCall(0)
				Expect(actualLocalPath).To(Equal(localPath))
				Expect(loggerForB).To(Equal(logger))
			})
		})

		Context("when the first backuper fails", func() {
			BeforeEach(func() {
				backuperA.UploadReturns(errors.New("first backup failed because reasons"))
			})

			It("returns the error from the first backuper", func() {
				Expect(uploadErr).To(MatchError(ContainSubstring("first backup failed because reasons")))
			})

			It("calls upload on all the backupers", func() {
				Expect(backuperA.UploadCallCount()).To(Equal(1))
				Expect(backuperB.UploadCallCount()).To(Equal(1))
			})
		})

		Context("when both backupers fail", func() {
			BeforeEach(func() {
				backuperA.UploadReturns(errors.New("first backup failed because reasons"))
				backuperB.UploadReturns(errors.New("second backup failed because reasons"))
			})

			It("returns the errors from both backupers", func() {
				Expect(uploadErr).To(MatchError(ContainSubstring("first backup failed because reasons")))
				Expect(uploadErr).To(MatchError(ContainSubstring("second backup failed because reasons")))
			})

			It("calls upload on all the backupers", func() {
				Expect(backuperA.UploadCallCount()).To(Equal(1))
				Expect(backuperB.UploadCallCount()).To(Equal(1))
			})
		})
	})
})
