package dummy_test

import (
	"github.com/pivotal-cf-experimental/service-backup/backup"
	. "github.com/pivotal-cf-experimental/service-backup/dummy"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	"code.cloudfoundry.org/lager"
)

var _ = Describe("Dummy Executor", func() {
	var dummyExecutor backup.Executor
	var logger lager.Logger
	var log *gbytes.Buffer

	BeforeEach(func() {
		logger = lager.NewLogger("ServiceBackup")
		log = gbytes.NewBuffer()
		logger.RegisterSink(lager.NewWriterSink(log, lager.INFO))
		dummyExecutor = NewDummyExecutor(logger)
	})

	Describe("RunOnce", func() {
		var err error

		BeforeEach(func() {
			err = dummyExecutor.RunOnce()
		})

		It("Doesn't return an error", func() {
			Expect(err).To(BeNil())
		})

		It("Logs that backups are disabled", func() {
			Expect(log).To(gbytes.Say("Backups Disabled"))
		})
	})
})
