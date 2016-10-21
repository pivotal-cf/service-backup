package config_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf-experimental/service-backup/backup"
	"github.com/pivotal-cf-experimental/service-backup/config"
	alerts "github.com/pivotal-cf/service-alerts-client/client"
)

var _ = Describe("Parsing properties", func() {
	Context("Valid backup config with alerts configured", func() {
		var (
			cron     string
			executor backup.Executor
			alerts   *alerts.ServiceAlertsClient
		)
		BeforeEach(func() {
			executor, cron, alerts, _ = config.Parse([]string{"cmd", "fixtures/valid_backup_with_alerts.yml"})
		})

		It("has the correct cron", func() {
			Expect(cron).To(Equal("*/5 * * * * *"))
		})

		It("has an executor", func() {
			Expect(executor).To(Not(BeNil()))
		})

		It("returns a valid alerts client", func() {
			Expect(alerts).To(Not(BeNil()))
		})
	})

	Context("Valid backup config without alerts configured", func() {
		var alerts *alerts.ServiceAlertsClient

		BeforeEach(func() {
			_, _, alerts, _ = config.Parse([]string{"cmd", "fixtures/valid_backup_without_alerts.yml"})
		})

		It("returns no alerts client", func() {
			Expect(alerts).To(BeNil())
		})
	})
})
