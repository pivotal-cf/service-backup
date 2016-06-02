package backup_test

import (
	"os/exec"

	. "github.com/pivotal-cf-experimental/service-backup/backup"
	"github.com/pivotal-cf-experimental/service-backup/backup/backupfakes"
	"github.com/pivotal-golang/lager"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Executor", func() {
	var (
		providerFactory *backupfakes.FakeProviderFactory
		execCmd         *exec.Cmd
		executor        Executor
		backuper        *backupfakes.FakeBackuper
		logger          lager.Logger
		log             *gbytes.Buffer
		calculator      *backupfakes.FakeSizeCalculator
	)

	BeforeEach(func() {
		log = gbytes.NewBuffer()
		logger = lager.NewLogger("executor")
		logger.RegisterSink(lager.NewWriterSink(log, lager.DEBUG))

		backuper = new(backupfakes.FakeBackuper)
		calculator = new(backupfakes.FakeSizeCalculator)
		calculator.DirSizeReturns(200, nil)
	})

	Describe("RunOnce()", func() {
		var (
			runOnceErr                error
			performIdentifyServiceCmd string
		)

		BeforeEach(func() {
			providerFactory = new(backupfakes.FakeProviderFactory)
			performIdentifyServiceCmd = assetPath("fake-service-identifier")
			execCmd = exec.Command("")
		})

		JustBeforeEach(func() {
			providerFactory.ExecCommandReturns(execCmd)

			executor = NewExecutor(
				backuper,
				"source-folder",
				"remote-path",
				assetPath("fake-snapshotter"),
				assetPath("fake-cleanup"),
				performIdentifyServiceCmd,
				logger,
				providerFactory.ExecCommand,
				calculator,
			)

			runOnceErr = executor.RunOnce()
		})

		Describe("performIdentifyService", func() {
			Context("when provided service identifier", func() {
				Context("returns an identifier", func() {
					BeforeEach(func() {
						execCmd = exec.Command(assetPath("fake-service-identifier"))
					})

					It("does not return an error", func() {
						Expect(runOnceErr).ToNot(HaveOccurred())
					})

					It("makes a system call to service identifier cmd", func() {
						Expect(providerFactory.ExecCommandCallCount()).To(Equal(1))
						serviceIdentifierCmd, _ := providerFactory.ExecCommandArgsForCall(0)
						Expect(serviceIdentifierCmd).To(Equal(performIdentifyServiceCmd))
					})

					It("logs with the service identifier", func() {
						Expect(log).To(gbytes.Say("Perform backup started"))
						Expect(log).To(gbytes.Say(`"identifier":"unit-identifier"`))
					})

					It("logs with the identifier for each event", func() {
						Expect(log).To(gbytes.Say("Perform backup started"))
						Expect(log).To(gbytes.Say(`"identifier":"unit-identifier"`))
						Expect(log).To(gbytes.Say("Perform backup debug info"))
						Expect(log).To(gbytes.Say(`"identifier":"unit-identifier"`))
						Expect(log).To(gbytes.Say("Perform backup completed without error"))
						Expect(log).To(gbytes.Say(`"identifier":"unit-identifier"`))
						Expect(log).To(gbytes.Say("Upload backup started"))
						Expect(log).To(gbytes.Say(`"identifier":"unit-identifier"`))
						Expect(log).To(gbytes.Say("Upload backup completed without error"))
						Expect(log).To(gbytes.Say(`"identifier":"unit-identifier"`))
						Expect(log).To(gbytes.Say("Cleanup completed"))
						Expect(log).To(gbytes.Say(`"identifier":"unit-identifier"`))
					})
				})

				It("logs upload metadata information", func() {
					Expect(log).To(gbytes.Say(`"duration_in_seconds":\d`))
					Expect(log).To(gbytes.Say(`"size_in_bytes":200`))
				})

				Context("returns an error", func() {
					BeforeEach(func() {
						execCmd = exec.Command(assetPath("fake-error-service-identifier"))
					})

					It("does not return an error", func() {
						Expect(runOnceErr).ToNot(HaveOccurred())
					})

					It("logs that identifier command was unsuccessful", func() {
						Expect(log).To(gbytes.Say("Service identifier command returned error"))
					})

					It("does not log any identifier", func() {
						Expect(log).ToNot(gbytes.Say(`"identifier"`))
					})
				})

				Context("does not exist", func() {
					BeforeEach(func() {
						performIdentifyServiceCmd = "/path/to/nowhere"
					})

					It("does not return an error", func() {
						Expect(runOnceErr).ToNot(HaveOccurred())
					})

					It("logs that identifier command did not exist", func() {
						Expect(log).To(gbytes.Say("Service identifier command not found"))
					})

					It("does not log any identifier", func() {
						Expect(log).ToNot(gbytes.Say(`"identifier"`))
					})
				})
			})

			Context("when no service identifier command provided", func() {
				BeforeEach(func() {
					performIdentifyServiceCmd = ""
				})

				It("does not return an error", func() {
					Expect(runOnceErr).ToNot(HaveOccurred())
				})

				It("logs do not mention identifier at all", func() {
					Expect(log).ToNot(gbytes.Say("identifier"))
				})
			})
		})
	})
})
