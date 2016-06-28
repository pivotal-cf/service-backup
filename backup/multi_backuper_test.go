package backup_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf-experimental/service-backup/backup"
	"github.com/pivotal-cf-experimental/service-backup/backup/backupfakes"
)

var _ = Describe("MultiBackuper", func() {
	Context("Upload", func() {
		var (
			localPath     = "local/path"
			backuperA     *backupfakes.FakeBackuper
			backuperB     *backupfakes.FakeBackuper
			multibackuper backup.MultiBackuper
			uploadErr     error
		)

		BeforeEach(func() {
			backuperA = new(backupfakes.FakeBackuper)
			backuperB = new(backupfakes.FakeBackuper)
			multibackuper = backup.MultiBackuper{backuperA, backuperB}
		})

		JustBeforeEach(func() {
			uploadErr = multibackuper.Upload(localPath)
		})

		Context("when all uploads succeed", func() {
			It("calls upload on each backuper", func() {
				Expect(uploadErr).NotTo(HaveOccurred())

				Expect(backuperA.UploadCallCount()).To(Equal(1))
				actualLocalPath := backuperA.UploadArgsForCall(0)
				Expect(actualLocalPath).To(Equal(localPath))

				Expect(backuperB.UploadCallCount()).To(Equal(1))
				actualLocalPath = backuperB.UploadArgsForCall(0)
				Expect(actualLocalPath).To(Equal(localPath))
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

	Context("SetLogSession", func() {
		It("calls SetLogSession on each backuper", func() {
			logSessionName := "session-name"
			logSessionID := "session-id"

			backuperA := new(backupfakes.FakeBackuper)
			backuperB := new(backupfakes.FakeBackuper)
			multibackuper := backup.MultiBackuper{backuperA, backuperB}

			multibackuper.SetLogSession(logSessionName, logSessionID)

			Expect(backuperA.SetLogSessionCallCount()).To(Equal(1))
			actualSessionName, actualSessionID := backuperA.SetLogSessionArgsForCall(0)
			Expect(actualSessionName).To(Equal(logSessionName))
			Expect(actualSessionID).To(Equal(logSessionID))

			Expect(backuperB.SetLogSessionCallCount()).To(Equal(1))
			actualSessionName, actualSessionID = backuperB.SetLogSessionArgsForCall(0)
			Expect(actualSessionName).To(Equal(logSessionName))
			Expect(actualSessionID).To(Equal(logSessionID))
		})
	})

	Context("CloseLogSession", func() {
		It("calls CloseLogSession on each backuper", func() {
			backuperA := new(backupfakes.FakeBackuper)
			backuperB := new(backupfakes.FakeBackuper)
			multibackuper := backup.MultiBackuper{backuperA, backuperB}

			multibackuper.CloseLogSession()

			Expect(backuperA.CloseLogSessionCallCount()).To(Equal(1))
			Expect(backuperB.CloseLogSessionCallCount()).To(Equal(1))
		})
	})
})
