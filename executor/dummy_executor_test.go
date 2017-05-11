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
